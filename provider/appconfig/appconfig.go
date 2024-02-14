// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package appconfig loads configuration from AWS AppConfig.
//
// AppConfig loads configuration from AWS AppConfig with the given application, environment, profile
// and returns a nested map[string]any that is parsed with the given unmarshal function.
//
// The unmarshal function must be able to unmarshal the configuration into a map[string]any.
// For example, with the default json.Unmarshal, the configuration is parsed as JSON.
package appconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
)

// AppConfig is a Provider that loads configuration from AWS AppConfig.
//
// To create a new AppConfig, call [New].
type AppConfig struct {
	logger    *slog.Logger
	unmarshal func([]byte, any) error

	client       *clientProxy
	application  string
	environment  string
	profile      string
	pollInterval time.Duration

	token atomic.Pointer[string]
}

// New creates an AppConfig with the given application, environment, profile and Option(s).
func New(application, environment, profile string, opts ...Option) *AppConfig {
	if application == "" {
		panic("cannot create AppConfig with empty application")
	}
	if environment == "" {
		panic("cannot create AppConfig with empty environment")
	}
	if profile == "" {
		panic("cannot create AppConfig with empty profile")
	}

	option := &options{
		AppConfig: AppConfig{
			application: application,
			environment: environment,
			profile:     profile,
		},
	}
	for _, opt := range opts {
		opt(option)
	}
	if option.logger == nil {
		option.logger = slog.Default()
	}
	if option.unmarshal == nil {
		option.unmarshal = json.Unmarshal
	}
	if option.pollInterval <= 0 {
		option.pollInterval = time.Minute
	}
	option.client = &clientProxy{config: option.awsConfig}

	return &option.AppConfig
}

func (a *AppConfig) Load() (map[string]any, error) {
	ctx := context.Background()

	if a.token.Load() == nil {
		input := &appconfigdata.StartConfigurationSessionInput{
			ApplicationIdentifier:                aws.String(a.application),
			ConfigurationProfileIdentifier:       aws.String(a.profile),
			EnvironmentIdentifier:                aws.String(a.environment),
			RequiredMinimumPollIntervalInSeconds: aws.Int32(int32(max(a.pollInterval.Seconds()/2, 1))), //nolint:gomnd
		}
		output, err := a.client.StartConfigurationSession(ctx, input)
		if err != nil {
			return nil, err
		}
		a.token.Store(output.InitialConfigurationToken)
	}
	values, _, err := a.load(ctx)

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
					ctx, "Error when reloading from AWS AppConfig",
					"application", a.application,
					"environment", a.environment,
					"profile", a.profile,
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

func (a *AppConfig) load(ctx context.Context) (map[string]any, bool, error) {
	input := &appconfigdata.GetLatestConfigurationInput{
		ConfigurationToken: a.token.Load(),
	}
	output, err := a.client.GetLatestConfiguration(ctx, input)
	if err != nil {
		return nil, false, err
	}
	a.token.Store(output.NextPollConfigurationToken)

	if len(output.Configuration) == 0 {
		// It may return empty configuration data
		// if the client already has the latest version.
		return nil, false, nil
	}

	var out map[string]any
	if e := a.unmarshal(output.Configuration, &out); e != nil {
		return nil, false, fmt.Errorf("unmarshal: %w", e)
	}

	return out, true, nil
}

func (a *AppConfig) String() string {
	return "appConfig:" + a.application + "-" + a.environment + "-" + a.profile
}

type clientProxy struct {
	config aws.Config

	client     *appconfigdata.Client
	clientOnce sync.Once

	timeout time.Duration
}

func (c *clientProxy) StartConfigurationSession(
	ctx context.Context,
	params *appconfigdata.StartConfigurationSessionInput,
	optFns ...func(*appconfigdata.Options),
) (*appconfigdata.StartConfigurationSessionOutput, error) {
	client, err := c.loadClient(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	session, err := client.StartConfigurationSession(ctx, params, optFns...)
	if err != nil {
		return nil, fmt.Errorf("start configuration session: %w", err)
	}

	return session, nil
}

func (c *clientProxy) GetLatestConfiguration(
	ctx context.Context,
	params *appconfigdata.GetLatestConfigurationInput,
	optFns ...func(*appconfigdata.Options),
) (*appconfigdata.GetLatestConfigurationOutput, error) {
	client, err := c.loadClient(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	configuration, err := client.GetLatestConfiguration(ctx, params, optFns...)
	if err != nil {
		return nil, fmt.Errorf("get latest configuration: %w", err)
	}

	return configuration, nil
}

func (c *clientProxy) loadClient(ctx context.Context) (*appconfigdata.Client, error) {
	var err error

	c.clientOnce.Do(func() {
		if c.timeout == 0 {
			c.timeout = 10 * time.Second //nolint:gomnd
		}
		if reflect.ValueOf(c.config).IsZero() {
			if c.config, err = config.LoadDefaultConfig(ctx); err != nil {
				err = fmt.Errorf("load default AWS config: %w", err)

				return
			}
		}

		c.client = appconfigdata.NewFromConfig(c.config)
	})

	return c.client, err
}
