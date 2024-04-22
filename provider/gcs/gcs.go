// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package gcs loads configuration from GCP [Cloud Storage].
//
// It requires following roles on the target GCS object:
//   - roles/storage.objectViewer
//
// # Change notification
//
// By default, it periodically polls the configuration only.
// It also listens to change events by register it to PubSub notifier with [Pub/Sub notifications for Cloud Storage].
//
// Only OBJECT_FINALIZE events trigger polling the configuration and other type of events are ignored.
//
// [Cloud Storage]: https://cloud.google.com/storage
// [Pub/Sub notifications for Cloud Storage]: https://cloud.google.com/storage/docs/pubsub-notifications
package gcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// GCS is a Provider that loads configuration from GCP Cloud Storage.
//
// To create a new GCS, call [New].
type GCS struct {
	pollInterval time.Duration
	unmarshal    func([]byte, any) error

	onStatus  func(bool, error)
	changedCh chan struct{}
	client    clientProxy
}

// New creates a GCS with the given endpoint and Option(s).
func New(uri string, opts ...Option) *GCS {
	uri = strings.TrimPrefix(uri, "gs:")
	uri = strings.TrimLeft(uri, "/")
	bucket, object, _ := strings.Cut(uri, "/")

	option := &options{
		client: clientProxy{
			bucket: bucket,
			object: object,
		},
		changedCh: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		switch o := opt.(type) {
		case *optionFunc:
			o.fn(option)
		default:
			option.client.opts = append(option.client.opts, o)
		}
	}

	return (*GCS)(option)
}

var errNil = errors.New("nil GCS")

func (g *GCS) Load() (map[string]any, error) {
	if g == nil {
		return nil, errNil
	}

	values, _, err := g.load(context.Background())

	return values, err
}

func (g *GCS) Watch(ctx context.Context, onChange func(map[string]any)) error { //nolint:cyclop
	if g == nil {
		return errNil
	}
	if g.changedCh == nil {
		g.changedCh = make(chan struct{}, 1)
	}

	defer func() {
		if g.client.client != nil {
			_ = g.client.client.Close()
		}
	}()

	pollInterval := time.Minute
	if g.pollInterval > 0 {
		pollInterval = g.pollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.changed()
		case <-g.changedCh:
			values, changed, err := g.load(ctx)
			if g.onStatus != nil {
				g.onStatus(changed, err)
			}
			if changed {
				onChange(values)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (g *GCS) changed() {
	select {
	case g.changedCh <- struct{}{}:
	default:
		// Ignore if the channel is full since it's already triggered.
	}
}

func (g *GCS) load(ctx context.Context) (map[string]any, bool, error) {
	resp, changed, err := g.client.load(ctx)
	if !changed || err != nil {
		return nil, false, err
	}

	unmarshal := g.unmarshal
	if unmarshal == nil {
		unmarshal = json.Unmarshal
	}

	var values map[string]any
	if e := unmarshal(resp, &values); e != nil {
		return nil, false, fmt.Errorf("unmarshal: %w", e)
	}

	return values, true, nil
}

func (g *GCS) OnEvent(attributes map[string]string) error {
	if g == nil {
		return errNil
	}

	if attributes["bucketId"] == g.client.bucket &&
		attributes["objectId"] == g.client.object {
		if attributes["eventType"] == "OBJECT_FINALIZE" {
			g.changed()
		}

		return nil
	}

	return fmt.Errorf("unsupported gcs event: %w", errors.ErrUnsupported)
}

func (g *GCS) Status(onStatus func(bool, error)) {
	g.onStatus = onStatus
}

func (g *GCS) String() string {
	return "gs://" + g.client.bucket + "/" + g.client.object
}

type clientProxy struct {
	bucket string
	object string

	client         *storage.Client
	opts           []option.ClientOption
	lastGeneration atomic.Int64
}

func (p *clientProxy) load(ctx context.Context) ([]byte, bool, error) {
	if p.client == nil {
		var err error
		if p.client, err = storage.NewClient(ctx, append(p.opts, storage.WithJSONReads())...); err != nil {
			return nil, false, fmt.Errorf("create GCS client: %w", err)
		}
	}

	object := p.client.Bucket(p.bucket).Object(p.object)
	if generation := p.lastGeneration.Load(); generation > 0 {
		object = object.If(storage.Conditions{GenerationNotMatch: generation})
	}
	reader, err := object.NewReader(ctx)
	if err != nil {
		var ge *googleapi.Error
		if errors.As(err, &ge) && ge.Code == http.StatusNotModified {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("create object reader: %w", err)
	}
	defer func() {
		// Ignore error: it could do nothing on this error.
		_ = reader.Close()
	}()

	if reader.Attrs.Generation == p.lastGeneration.Load() {
		return nil, false, nil
	}
	p.lastGeneration.Store(reader.Attrs.Generation)

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, false, fmt.Errorf("read object: %w", err)
	}

	return bytes, true, nil
}
