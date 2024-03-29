// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package gcs loads configuration from GCP [Cloud Storage].
//
// [Cloud Storage]: https://cloud.google.com/storage
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

	onStatus func(bool, error)
	client   *clientProxy
}

// New creates a GCS with the given endpoint and Option(s).
func New(uri string, opts ...Option) *GCS {
	uri = strings.TrimPrefix(uri, "gs:")
	uri = strings.TrimLeft(uri, "/")
	bucket, object, _ := strings.Cut(uri, "/")

	option := &options{
		client: &clientProxy{
			bucket: bucket,
			object: object,
		},
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

func (g *GCS) Load() (map[string]any, error) {
	values, _, err := g.load(context.Background())

	return values, err
}

func (g *GCS) Watch(ctx context.Context, onChange func(map[string]any)) error {
	pollInterval := time.Minute
	if g.pollInterval > 0 {
		pollInterval = g.pollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
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

func (g *GCS) Status(onStatus func(bool, error)) {
	g.onStatus = onStatus
}

func (g *GCS) Close() error {
	if err := g.client.client.Close(); err != nil {
		return fmt.Errorf("close GCS client: %w", err)
	}

	return nil
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
	if p == nil {
		// Use empty instance instead to avoid nil pointer dereference,
		// Assignment propagates only to callee but not to caller.
		p = &clientProxy{}
	}

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
