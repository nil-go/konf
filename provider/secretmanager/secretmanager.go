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
	"log/slog"
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
	logger       *slog.Logger

	client *clientProxy
}

// New creates a SecretManager with the given endpoint and Option(s).
func New(opts ...Option) SecretManager {
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

	if option.pollInterval <= 0 {
		option.pollInterval = time.Minute
	}
	if option.splitter == nil {
		option.splitter = func(s string) []string { return strings.Split(s, "-") }
	}
	if option.logger == nil {
		option.logger = slog.Default()
	}

	return SecretManager(*option)
}

func (a SecretManager) Load() (map[string]any, error) {
	values, _, err := a.load(context.Background())

	return values, err
}

func (a SecretManager) Watch(ctx context.Context, onChange func(map[string]any)) error {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			values, changed, err := a.load(ctx)
			if err != nil {
				a.logger.WarnContext(
					ctx, "Error when reloading from GCP Secret Manager",
					"project", a.client.project,
					"filter", a.client.filter,
					"error", err,
				)

				continue
			}

			if changed {
				onChange(values)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (a SecretManager) load(ctx context.Context) (map[string]any, bool, error) {
	resp, changed, err := a.client.load(ctx)
	if !changed || err != nil {
		return nil, false, err
	}

	values := make(map[string]any)
	for key, value := range resp {
		keys := a.splitter(key)
		if len(keys) == 0 || len(keys) == 1 && keys[0] == "" {
			continue
		}

		imaps.Insert(values, keys, value)
	}

	return values, true, nil
}

func (a SecretManager) String() string {
	return "secretManager:" + a.client.project
}

type clientProxy struct {
	project string
	filter  string
	opts    []option.ClientOption

	client     *secretmanager.Client
	clientOnce sync.Once

	lastETags atomic.Pointer[map[string]string]
}

func (p *clientProxy) load(ctx context.Context) (map[string]string, bool, error) { //nolint:cyclop,funlen
	client, err := p.loadClient(ctx)
	if err != nil {
		return nil, false, err
	}

	eTags := make(map[string]string)
	iter := client.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{
		Parent: "projects/" + p.project,
		Filter: p.filter,
	})
	for resp, e := iter.Next(); !errors.Is(e, iterator.Done); resp, e = iter.Next() {
		if e != nil {
			return nil, false, fmt.Errorf("list secrets on %s: %w", p.project, e)
		}

		eTags[resp.GetName()] = resp.GetEtag()
	}

	var changed bool
	if last := p.lastETags.Load(); last == nil || !maps.Equal(*last, eTags) {
		changed = true
		p.lastETags.Store(&eTags)
	}
	if !changed {
		return nil, false, nil
	}

	secretsCh := make(chan *secretmanagerpb.AccessSecretVersionResponse, len(eTags))
	errChan := make(chan error, len(eTags))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(eTags))
	for name := range eTags {
		name := name

		go func() {
			defer waitGroup.Done()

			resp, e := p.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
				Name: name + "/versions/latest",
			})
			if e != nil {
				errChan <- fmt.Errorf("access secret %s: %w", strings.Split(name, "/")[3], e)
				cancel()

				return
			}
			secretsCh <- resp
		}()
	}
	waitGroup.Wait()
	close(secretsCh)
	close(errChan)

	for e := range errChan {
		if !errors.Is(e, ctx.Err()) {
			err = errors.Join(e)
		}
	}
	if err != nil {
		return nil, false, err
	}

	values := make(map[string]string, len(eTags))
	for resp := range secretsCh {
		data := resp.GetPayload().GetData()
		values[strings.Split(resp.GetName(), "/")[3]] = unsafe.String(unsafe.SliceData(data), len(data))
	}

	return values, true, nil
}

func (p *clientProxy) loadClient(ctx context.Context) (*secretmanager.Client, error) {
	var err error

	p.clientOnce.Do(func() {
		if p.project == "" {
			if p.project, err = metadata.ProjectID(); err != nil {
				err = fmt.Errorf("get GCP project ID: %w", err)

				return
			}
		}

		if p.client, err = secretmanager.NewClient(ctx, p.opts...); err != nil {
			err = fmt.Errorf("create GCP secret manager client: %w", err)

			return
		}
	})

	return p.client, err
}