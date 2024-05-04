// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package azblob loads configuration from Azure [Blob Storage].
//
// It requires following roles to access blob from Azure Blob Storage:
// - Storage Blob Data Reader
//
// # Change notification
//
// By default, it periodically polls the configuration only.
// It also listens to change events by register it to notifier with [Cloud Event schema].
//
// Only Microsoft.Storage.BlobCreated events trigger polling the configuration and other type of events are ignored.
//
// [Blob Storage]: https://azure.microsoft.com/en-us/products/storage/blobs
// [Cloud Event schema]: https://learn.microsoft.com/en-us/azure/event-grid/event-schema-blob-storage?tabs=cloud-event-schema
package azblob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/messaging"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
)

// Blob is a Provider that loads configuration from Azure Blob Storage.
//
// To create a new Blob, call [New].
type Blob struct {
	pollInterval time.Duration
	unmarshal    func([]byte, any) error

	onStatus  func(bool, error)
	changedCh chan struct{}
	client    clientProxy
}

// New creates an Blob with the given endpoint and Option(s).
func New(endpoint, container, blob string, opts ...Option) *Blob {
	option := &options{
		client: clientProxy{
			// Place holder for the default credential.
			credential: &azidentity.DefaultAzureCredential{},
			endpoint:   endpoint,
			container:  container,
			blob:       blob,
		},
		changedCh: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(option)
	}
	option.client.timeout = option.pollInterval / 2 //nolint:mnd

	return (*Blob)(option)
}

var errNil = errors.New("nil Blob")

func (b *Blob) Load() (map[string]any, error) {
	if b == nil {
		return nil, errNil
	}

	values, _, err := b.load(context.Background())

	return values, err
}

func (b *Blob) Watch(ctx context.Context, onChange func(map[string]any)) error {
	if b == nil {
		return errNil
	}
	if b.changedCh == nil {
		b.changedCh = make(chan struct{}, 1)
	}

	pollInterval := time.Minute
	if b.pollInterval > 0 {
		pollInterval = b.pollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.changed()
		case <-b.changedCh:
			values, changed, err := b.load(ctx)
			if b.onStatus != nil {
				b.onStatus(changed, err)
			}
			if changed {
				onChange(values)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *Blob) changed() {
	select {
	case b.changedCh <- struct{}{}:
	default:
		// Ignore if the channel is full since it's already triggered.
	}
}

func (b *Blob) load(ctx context.Context) (map[string]any, bool, error) {
	resp, changed, err := b.client.load(ctx)
	if !changed || err != nil {
		return nil, false, err
	}

	unmarshal := b.unmarshal
	if unmarshal == nil {
		unmarshal = json.Unmarshal
	}
	var values map[string]any
	if e := unmarshal(resp, &values); e != nil {
		return nil, false, fmt.Errorf("unmarshal: %w", e)
	}

	return values, true, nil
}

var errNonBytesData = errors.New("event data should be []byte")

func (b *Blob) OnEvent(event messaging.CloudEvent) error {
	if b == nil {
		return errNil
	}

	var data struct {
		URL string `json:"url"`
	}
	bytes, ok := event.Data.([]byte)
	if !ok {
		return errNonBytesData
	}
	if e := json.Unmarshal(bytes, &data); e != nil {
		return fmt.Errorf("unmarshal event data: %w", e)
	}

	if data.URL == b.String() {
		if event.Type == "Microsoft.Storage.BlobCreated" {
			b.changed()
		}

		return nil
	}

	return fmt.Errorf("unsupported blob storage event: %w", errors.ErrUnsupported)
}

func (b *Blob) Status(onStatus func(bool, error)) {
	b.onStatus = onStatus
}

func (b *Blob) String() string {
	return b.client.url()
}

type clientProxy struct {
	endpoint   string
	container  string
	blob       string
	credential azcore.TokenCredential

	client *blob.Client

	timeout time.Duration
	eTag    atomic.Pointer[azcore.ETag]
}

func (p *clientProxy) load(ctx context.Context) ([]byte, bool, error) { //nolint:cyclop
	if p.client == nil {
		if token, ok := p.credential.(*azidentity.DefaultAzureCredential); ok && reflect.ValueOf(*token).IsZero() {
			var err error
			if p.credential, err = azidentity.NewDefaultAzureCredential(nil); err != nil {
				return nil, false, fmt.Errorf("load default Azure credential: %w", err)
			}
		}

		client, err := azblob.NewClient(p.endpoint, p.credential, nil)
		if err != nil {
			return nil, false, fmt.Errorf("create Azure blob client: %w", err)
		}
		p.client = client.ServiceClient().NewContainerClient(p.container).NewBlobClient(p.blob)
	}

	ctx, cancel := context.WithTimeout(ctx, max(p.timeout, 10*time.Second)) //nolint:mnd
	defer cancel()

	resp, err := p.client.DownloadStream(ctx, &azblob.DownloadStreamOptions{
		AccessConditions: &azblob.AccessConditions{
			ModifiedAccessConditions: &blob.ModifiedAccessConditions{
				IfNoneMatch: p.eTag.Load(),
			},
		},
	})
	if err != nil {
		return nil, false, fmt.Errorf("get blob: %w", err)
	}
	defer func() {
		// Ignore error: it could do nothing on this error.
		_ = resp.Body.Close()
	}()

	if resp.ErrorCode != nil && *resp.ErrorCode == string(bloberror.ConditionNotMet) {
		return nil, false, nil
	}
	if eTag := p.eTag.Load(); eTag != nil && eTag.Equals(*resp.ETag) {
		return nil, false, nil
	}
	p.eTag.Store(resp.ETag)

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("read blob: %w", err)
	}

	return bytes, true, nil
}

func (p *clientProxy) url() string {
	return p.endpoint + "/" + p.container + "/" + p.blob
}
