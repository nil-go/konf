// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package secretmanager loads configuration from GCP [Secret Manager].
//
// [Secret Manager]: https://cloud.google.com/security/products/secret-manager
package secretmanager

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"cloud.google.com/go/compute/metadata"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	imaps "github.com/nil-go/konf/provider/secretmanager/internal/maps"
)

// SecretManager is a Provider that loads configuration from GCP Secret Manager.
//
// To create a new SecretManager, call [New].
type SecretManager struct {
	pollInterval time.Duration
	splitter     func(string) []string

	onStatus func(bool, error)
	client   *clientProxy
}

// New creates a SecretManager with the given endpoint and Option(s).
func New(opts ...Option) *SecretManager {
	option := &options{
		client: &clientProxy{},
	}
	for _, opt := range opts {
		switch o := opt.(type) {
		case *optionFunc:
			o.fn(option)
		default:
			option.client.opts = append(option.client.opts, o)
		}
	}

	return (*SecretManager)(option)
}

func (m *SecretManager) Load() (map[string]any, error) {
	values, _, err := m.load(context.Background())

	return values, err
}

func (m *SecretManager) Watch(ctx context.Context, onChange func(map[string]any)) error {
	pollInterval := time.Minute
	if m.pollInterval > 0 {
		pollInterval = m.pollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			values, changed, err := m.load(ctx)
			if m.onStatus != nil {
				m.onStatus(changed, err)
			}
			if changed {
				onChange(values)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (m *SecretManager) load(ctx context.Context) (map[string]any, bool, error) {
	resp, changed, err := m.client.load(ctx)
	if !changed || err != nil {
		return nil, false, err
	}

	splitter := m.splitter
	if splitter == nil {
		splitter = func(s string) []string {
			return strings.Split(s, "-")
		}
	}

	values := make(map[string]any)
	for key, value := range resp {
		keys := splitter(key)
		if len(keys) == 0 || len(keys) == 1 && keys[0] == "" {
			continue
		}

		imaps.Insert(values, keys, value)
	}

	return values, true, nil
}

func (m *SecretManager) Status(onStatus func(bool, error)) {
	m.onStatus = onStatus
}

func (m *SecretManager) String() string {
	return "secretManager:" + m.client.project
}

type clientProxy struct {
	project string
	filter  string

	client    *secretmanager.Client
	opts      []option.ClientOption
	lastETags atomic.Pointer[map[string]string]
}

func (p *clientProxy) load(ctx context.Context) (map[string]string, bool, error) { //nolint:cyclop,funlen
	if p == nil {
		p = &clientProxy{}
	}

	if p.project == "" {
		var err error
		if p.project, err = metadata.ProjectID(); err != nil {
			return nil, false, fmt.Errorf("get GCP project ID: %w", err)
		}
	}
	if p.client == nil {
		var err error
		if p.client, err = secretmanager.NewClient(ctx, p.opts...); err != nil {
			return nil, false, fmt.Errorf("create GCP secret manager client: %w", err)
		}
	}

	eTags := make(map[string]string)
	iter := p.client.ListSecrets(ctx,
		&secretmanagerpb.ListSecretsRequest{
			Parent: "projects/" + p.project,
			Filter: p.filter,
		},
	)
	for {
		resp, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, false, fmt.Errorf("list secrets on %s: %w", p.project, err)
		}
		eTags[resp.GetName()] = resp.GetEtag()
	}

	if last := p.lastETags.Load(); last != nil && maps.Equal(*last, eTags) {
		return nil, false, nil
	}
	p.lastETags.Store(&eTags)

	secretChan := make(chan *secretmanagerpb.AccessSecretVersionResponse, len(eTags))
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(eTags))
	for name := range eTags {
		name := name

		go func() {
			defer waitGroup.Done()

			resp, err := p.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
				Name: name + "/versions/latest",
			})
			if err != nil {
				cancel(fmt.Errorf("access secret %s: %w", strings.Split(name, "/")[3], err))

				return
			}
			secretChan <- resp
		}()
	}
	waitGroup.Wait()
	close(secretChan)

	if err := context.Cause(ctx); err != nil && !errors.Is(err, ctx.Err()) {
		return nil, false, err //nolint:wrapcheck
	}

	values := make(map[string]string, len(eTags))
	for resp := range secretChan {
		data := resp.GetPayload().GetData()
		values[strings.Split(resp.GetName(), "/")[3]] = unsafe.String(unsafe.SliceData(data), len(data))
	}

	return values, true, nil
}
