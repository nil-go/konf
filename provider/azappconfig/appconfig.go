// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package azappconfig loads configuration from Azure [App Configuration].
//
// [App Configuration]: https://docs.microsoft.com/en-us/azure/azure-app-configuration/
package azappconfig

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"

	imaps "github.com/nil-go/konf/provider/azappconfig/internal/maps"
)

// AppConfig is a Provider that loads configuration from Azure App Configuration.
//
// To create a new AppConfig, call [New].
type AppConfig struct {
	logger   *slog.Logger
	splitter func(string) []string

	client       *clientProxy
	lastETags    atomic.Pointer[map[string]azcore.ETag]
	keyFilter    string
	labelFilter  string
	pollInterval time.Duration
	timeout      time.Duration
}

// New creates an AppConfig with the given endpoint and Option(s).
func New(endpoint string, opts ...Option) *AppConfig {
	if endpoint == "" {
		panic("cannot create Azure AppConfig with empty endpoint")
	}

	option := &options{
		AppConfig: AppConfig{
			client: &clientProxy{
				endpoint: endpoint,
				// Place holder for the default credential.
				credential: &azidentity.DefaultAzureCredential{},
			},
			timeout: 30 * time.Second, //nolint:gomnd
		},
	}
	for _, opt := range opts {
		opt(option)
	}

	if option.splitter == nil {
		option.splitter = func(s string) []string { return strings.Split(s, "/") }
	}
	if option.logger == nil {
		option.logger = slog.Default()
	}
	if option.pollInterval <= 0 {
		option.pollInterval = time.Minute
	}

	return &option.AppConfig
}

func (a *AppConfig) Load() (map[string]any, error) {
	values, _, err := a.load(context.Background())

	return values, err
}

func (a *AppConfig) Watch(ctx context.Context, onChange func(map[string]any)) error {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			values, changed, err := a.load(ctx)
			if err != nil {
				a.logger.WarnContext(
					ctx, "Error when reloading from Azure App Configuration",
					"keyFilter", a.keyFilter,
					"labelFilter", a.labelFilter,
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

func (a *AppConfig) load(ctx context.Context) (map[string]any, bool, error) { //nolint:cyclop
	selector := azappconfig.SettingSelector{
		Fields: []azappconfig.SettingFields{
			azappconfig.SettingFieldsKey,
			azappconfig.SettingFieldsValue,
			azappconfig.SettingFieldsETag,
		},
	}
	if a.keyFilter != "" {
		selector.KeyFilter = &a.keyFilter
	}
	if a.labelFilter != "" {
		selector.LabelFilter = &a.labelFilter
	}
	pager, err := a.client.NewListSettingsPager(selector, &azappconfig.ListSettingsOptions{})
	if err != nil {
		return nil, false, err
	}

	var (
		values = make(map[string]any)
		eTags  = make(map[string]azcore.ETag)

		nextPage = func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, a.timeout)
			defer cancel()

			page, err := pager.NextPage(ctx)
			if err != nil {
				return fmt.Errorf("next page of list settings: %w", err)
			}

			for _, setting := range page.Settings {
				keys := a.splitter(*setting.Key)
				if len(keys) == 0 || len(keys) == 1 && keys[0] == "" {
					continue
				}

				imaps.Insert(values, keys, *setting.Value)
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
	if last := a.lastETags.Load(); last == nil || !maps.Equal(*last, eTags) {
		changed = true
		a.lastETags.Store(&eTags)
	}

	return values, changed, nil
}

func (a *AppConfig) String() string {
	return "azAppConfig:" + a.client.endpoint
}

type clientProxy struct {
	endpoint   string
	credential azcore.TokenCredential

	client     *azappconfig.Client
	clientOnce sync.Once
}

func (c *clientProxy) NewListSettingsPager(
	selector azappconfig.SettingSelector,
	options *azappconfig.ListSettingsOptions,
) (*runtime.Pager[azappconfig.ListSettingsPageResponse], error) {
	client, err := c.loadClient()
	if err != nil {
		return nil, err
	}

	return client.NewListSettingsPager(selector, options), nil
}

func (c *clientProxy) loadClient() (*azappconfig.Client, error) {
	var err error

	c.clientOnce.Do(func() {
		if defaultToken, ok := c.credential.(*azidentity.DefaultAzureCredential); ok {
			empty := azidentity.DefaultAzureCredential{}
			if empty == *defaultToken {
				credentialOptions := &azidentity.DefaultAzureCredentialOptions{}
				if c.credential, err = azidentity.NewDefaultAzureCredential(credentialOptions); err != nil {
					err = fmt.Errorf("load default Azure credential: %w", err)

					return
				}
			}
		}

		clientOptions := &azappconfig.ClientOptions{}
		if c.client, err = azappconfig.NewClient(c.endpoint, c.credential, clientOptions); err != nil {
			err = fmt.Errorf("create Azure app configuration client: %w", err)

			return
		}
	})

	return c.client, err
}
