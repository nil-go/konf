// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package s3 loads configuration from AWS [S3].
//
// # Poll/Push Mode
//
// By default, it's poll only mode which periodically polls the configuration.
// You can switch to push only mode by providing the SNS topic,
// or hybrid mode by providing both the SNS topic and the poll interval.
//
// if the SNS topic is provided, it will listen to the events sent from AWS S3 using following setup.
//   - [EventBridge] with SNS target
//   - [SNS]
//
// Only ObjectCreated:* events [Fanout to Amazon SQS queues] and trigger polling the configuration
// and other type of events are ignored.
//
// # Permission
//
// It requires following permissions to access object from AWS S3:
//   - s3:GetObject
//
// [S3]: https://aws.amazon.com/s3/
// [EventBridge]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/EventBridge.html
// [SNS]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/how-to-enable-disable-notification-intro.html
// [Fanout to Amazon SQS queues]: https://docs.aws.amazon.com/sns/latest/dg/sns-sqs-as-subscriber.html
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

	onStatus func(bool, error)
	client   clientProxy
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
	}
	for _, opt := range opts {
		opt(option)
	}
	option.client.timeout = option.pollInterval / 2 //nolint:gomnd

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

	pollInterval := time.Minute
	if a.pollInterval > 0 {
		pollInterval = a.pollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			values, changed, err := a.load(ctx)
			if a.onStatus != nil {
				a.onStatus(changed, err)
			}
			if changed {
				onChange(values)
			}
		case <-ctx.Done():
			return nil
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
	if e := unmarshal(resp, &values); e != nil {
		return nil, false, fmt.Errorf("unmarshal: %w", e)
	}

	return values, true, nil
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
			if p.config, err = config.LoadDefaultConfig(ctx); err != nil {
				return nil, false, fmt.Errorf("load default AWS config: %w", err)
			}
		}
		p.client = s3.NewFromConfig(p.config)
	}

	ctx, cancel := context.WithTimeout(ctx, max(p.timeout, 10*time.Second)) //nolint:gomnd
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
