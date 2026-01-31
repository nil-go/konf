// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package s3 loads configuration from AWS [S3].
//
// It requires following permissions to access object from AWS S3:
//   - s3:GetObject
//
// # Change notification
//
// By default, it's periodically polls the configuration.
// It also listens to change events by register it to SNS notifier with one of following setups:
//   - [EventBridge] with SNS target
//   - [SNS]
//
// Only ObjectCreated:* events trigger polling the configuration and other type of events are ignored.
//
// [S3]: https://aws.amazon.com/s3/
// [EventBridge]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/EventBridge.html
// [SNS]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/how-to-enable-disable-notification-intro.html
package s3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// S3 is a Provider that loads configuration from AWS S3.
//
// To create a new S3, call [New].
type S3 struct {
	unmarshal    func([]byte, any) error
	pollInterval time.Duration

	onStatus  func(bool, error)
	changedCh chan struct{}
	client    clientProxy
}

// New creates an S3 with the given uri and Option(s).
func New(uri string, opts ...Option) *S3 {
	uri = strings.TrimPrefix(uri, "s3:")
	uri = strings.TrimLeft(uri, "/")
	bucket, key, _ := strings.Cut(uri, "/")

	option := &options{
		client: clientProxy{
			bucket: bucket,
			key:    key,
		},
		changedCh: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(option)
	}
	option.client.timeout = option.pollInterval / 2 //nolint:mnd

	return (*S3)(option)
}

var errNil = errors.New("nil S3")

func (a *S3) Load() (map[string]any, error) {
	if a == nil {
		return nil, errNil
	}

	values, _, err := a.load(context.Background())

	return values, err
}

func (a *S3) Watch(ctx context.Context, onChange func(map[string]any)) error {
	if a == nil {
		return errNil
	}
	if a.changedCh == nil {
		a.changedCh = make(chan struct{}, 1)
	}

	pollInterval := time.Minute
	if a.pollInterval > 0 {
		pollInterval = a.pollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			a.changed()
		case <-a.changedCh:
			values, changed, err := a.load(ctx)
			if a.onStatus != nil {
				a.onStatus(changed, err)
			}
			if changed {
				onChange(values)
			}
		}
	}
}

func (a *S3) load(ctx context.Context) (map[string]any, bool, error) {
	resp, changed, err := a.client.load(ctx)
	if !changed || err != nil {
		return nil, false, err
	}

	unmarshal := a.unmarshal
	if unmarshal == nil {
		unmarshal = json.Unmarshal
	}
	var values map[string]any
	e := unmarshal(resp, &values)
	if e != nil {
		return nil, false, fmt.Errorf("unmarshal: %w", e)
	}

	return values, true, nil
}

func (a *S3) OnEvent(msg []byte) error { //nolint:cyclop
	if a == nil {
		return errNil
	}

	//nolint:tagliatelle
	type (
		s3Object struct {
			Bucket struct {
				Name string `json:"name"`
			} `json:"bucket"`
			Object struct {
				Key string `json:"key"`
			} `json:"object"`
		}
		s3Event struct {
			// From EventBridge: https://docs.aws.amazon.com/AmazonS3/latest/userguide/ev-events.html
			DetailType string   `json:"detail-type"`
			Source     string   `json:"source"`
			Detail     s3Object `json:"detail"`

			// From SNS: https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-content-structure.html
			Records []struct {
				EventSource string   `json:"eventSource"`
				EventName   string   `json:"eventName"`
				S3          s3Object `json:"s3"`
			} `json:"Records"`
		}
	)

	var event s3Event
	err := json.Unmarshal(msg, &event)
	if err != nil {
		return fmt.Errorf("unmarshal s3 event: %w", err)
	}

	if event.Source == "aws.s3" &&
		event.Detail.Bucket.Name == a.client.bucket &&
		event.Detail.Object.Key == a.client.key {
		if event.DetailType == "Object Created" {
			// Trigger to reload the configuration.
			a.changed()
		}

		return nil
	}

	if len(event.Records) > 0 {
		record := event.Records[0]
		if record.EventSource == "aws:s3" &&
			record.S3.Bucket.Name == a.client.bucket &&
			record.S3.Object.Key == a.client.key {
			if strings.HasPrefix(record.EventName, "ObjectCreated:") {
				// Trigger to reload the configuration.
				a.changed()
			}

			return nil
		}
	}

	return fmt.Errorf("unsupported s3 event: %w", errors.ErrUnsupported)
}

func (a *S3) changed() {
	select {
	case a.changedCh <- struct{}{}:
	default:
		// Ignore if the channel is full since it's already triggered.
	}
}

func (a *S3) Status(onStatus func(bool, error)) {
	a.onStatus = onStatus
}

func (a *S3) String() string {
	return "s3://" + path.Join(a.client.bucket, a.client.key)
}

type clientProxy struct {
	config aws.Config
	bucket string
	key    string

	client *s3.Client

	timeout time.Duration
	eTag    atomic.Pointer[string]
}

func (p *clientProxy) load(ctx context.Context) ([]byte, bool, error) {
	if p.client == nil {
		if reflect.ValueOf(p.config).IsZero() {
			var err error
			p.config, err = config.LoadDefaultConfig(ctx)
			if err != nil {
				return nil, false, fmt.Errorf("load default AWS config: %w", err)
			}
		}
		p.client = s3.NewFromConfig(p.config)
	}

	ctx, cancel := context.WithTimeout(ctx, max(p.timeout, 10*time.Second)) //nolint:mnd
	defer cancel()

	resp, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket:      &p.bucket,
		Key:         &p.key,
		IfNoneMatch: p.eTag.Load(),
	})
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) && ae.ErrorCode() == "NotModified" {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("get object: %w", err)
	}
	defer func() {
		// Ignore error: it could do nothing on this error.
		_ = resp.Body.Close()
	}()

	if resp.ETag == p.eTag.Load() {
		return nil, false, nil
	}
	p.eTag.Store(resp.ETag)

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("read object: %w", err)
	}

	return bytes, true, nil
}
