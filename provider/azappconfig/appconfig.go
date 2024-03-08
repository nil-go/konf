// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package azappconfig loads configuration from Azure [App Configuration].
//
// [App Configuration]: https://docs.microsoft.com/en-us/azure/azure-app-configuration/
package azappconfig

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"

	imaps "github.com/nil-go/konf/provider/azappconfig/internal/maps"
)

// AppConfig is a Provider that loads configuration from Azure App Configuration.
//
// To create a new AppConfig, call [New].
type AppConfig struct {
	splitter     func(string) []string
	pollInterval time.Duration

	onStatus func(bool, error)
	client   *clientProxy
}

// New creates an AppConfig with the given endpoint and Option(s).
func New(endpoint string, opts ...Option) *AppConfig {
	option := &options{
		client: &clientProxy{
			// Place holder for the default credential.
			credential: &azidentity.DefaultAzureCredential{},
			endpoint:   endpoint,
		},
	}
	for _, opt := range opts {
		opt(option)
	}

	if option.splitter == nil {
		option.splitter = func(s string) []string { return strings.Split(s, "/") }
	}
	option.client.timeout = option.pollInterval / 2 //nolint:gomnd

	return (*AppConfig)(option)
}

func (a *AppConfig) Load() (map[string]any, error) {
	values, _, err := a.load(context.Background())

	return values, err
}

func (a *AppConfig) Watch(ctx context.Context, onChange func(map[string]any)) error {
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

func (a *AppConfig) load(ctx context.Context) (map[string]any, bool, error) {
	resp, changed, err := a.client.load(ctx)
	if !changed || err != nil {
		return nil, false, err
	}

	splitter := a.splitter
	if splitter == nil {
		splitter = func(s string) []string { return strings.Split(s, "-") }
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

func (a *AppConfig) Status(onStatus func(bool, error)) {
	a.onStatus = onStatus
}

func (a *AppConfig) String() string {
	return "azAppConfig:" + a.client.endpoint
}

type clientProxy struct {
	endpoint    string
	keyFilter   string
	labelFilter string
	credential  azcore.TokenCredential

	client *azappconfig.Client

	timeout   time.Duration
	lastETags atomic.Pointer[map[string]azcore.ETag]
}

func (p *clientProxy) load(ctx context.Context) (map[string]string, bool, error) { //nolint:cyclop,funlen
	if p == nil {
		p = &clientProxy{}
	}

	if p.client == nil {
		if token, ok := p.credential.(*azidentity.DefaultAzureCredential); ok && reflect.ValueOf(*token).IsZero() {
			var err error
			credentialOptions := &azidentity.DefaultAzureCredentialOptions{}
			if p.credential, err = azidentity.NewDefaultAzureCredential(credentialOptions); err != nil {
				return nil, false, fmt.Errorf("load default Azure credential: %w", err)
			}
		}

		var err error
		clientOptions := &azappconfig.ClientOptions{}
		if p.client, err = azappconfig.NewClient(p.endpoint, p.credential, clientOptions); err != nil {
			return nil, false, fmt.Errorf("create Azure app configuration client: %w", err)
		}
	}

	selector := azappconfig.SettingSelector{
		Fields: []azappconfig.SettingFields{
			azappconfig.SettingFieldsKey,
			azappconfig.SettingFieldsValue,
			azappconfig.SettingFieldsETag,
		},
	}
	if p.keyFilter != "" {
		selector.KeyFilter = &p.keyFilter
	}
	if p.labelFilter != "" {
		selector.LabelFilter = &p.labelFilter
	}
	pager := p.client.NewListSettingsPager(selector, &azappconfig.ListSettingsOptions{})

	var (
		values = make(map[string]string)
		eTags  = make(map[string]azcore.ETag)

		nextPage = func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, max(p.timeout, 10*time.Second)) //nolint:gomnd
			defer cancel()

			page, err := pager.NextPage(ctx)
			if err != nil {
				return fmt.Errorf("next page of list settings: %w", err)
			}

			for _, setting := range page.Settings {
				values[*setting.Key] = *setting.Value
				eTags[*setting.Key] = *setting.ETag
			}

			return nil
		}
	)
	for pager.More() {
		if err := nextPage(ctx); err != nil {
			return nil, false, err
		}
	}

	var changed bool
	if last := p.lastETags.Load(); last == nil || !maps.Equal(*last, eTags) {
		changed = true
		p.lastETags.Store(&eTags)
	}

	return values, changed, nil
}
