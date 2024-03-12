// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package azblob loads configuration from Azure [Blob Storage].
//
// [Blob Storage]: https://azure.microsoft.com/en-us/products/storage/blobs
package azblob

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
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

	onStatus func(bool, error)
	client   *clientProxy
}

// New creates an Blob with the given endpoint and Option(s).
func New(endpoint, container, blob string, opts ...Option) *Blob {
	option := &options{
		client: &clientProxy{
			// Place holder for the default credential.
			credential: &azidentity.DefaultAzureCredential{},
			endpoint:   endpoint,
			container:  container,
			blob:       blob,
		},
	}
	for _, opt := range opts {
		opt(option)
	}
	option.client.timeout = option.pollInterval / 2 //nolint:gomnd

	return (*Blob)(option)
}

func (a *Blob) Load() (map[string]any, error) {
	values, _, err := a.load(context.Background())

	return values, err
}

func (a *Blob) Watch(ctx context.Context, onChange func(map[string]any)) error {
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

func (a *Blob) load(ctx context.Context) (map[string]any, bool, error) {
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

func (a *Blob) Status(onStatus func(bool, error)) {
	a.onStatus = onStatus
}

func (a *Blob) String() string {
	return a.client.endpoint + "/" + a.client.container + "/" + a.client.blob
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
	if p == nil {
		p = &clientProxy{}
	}

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

	ctx, cancel := context.WithTimeout(ctx, max(p.timeout, 10*time.Second)) //nolint:gomnd
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
