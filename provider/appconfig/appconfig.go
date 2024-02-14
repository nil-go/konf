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
	unmarshal    func([]byte, any) error
	pollInterval time.Duration
	logger       *slog.Logger

	client *clientProxy
}

// New creates an AppConfig with the given application, environment, profile and Option(s).
func New(application, environment, profile string, opts ...Option) AppConfig {
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
		client: &clientProxy{
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
	option.client.pollInterval = max(option.pollInterval/2, time.Second) //nolint:gomnd

	return AppConfig(*option)
}

func (a AppConfig) Load() (map[string]any, error) {
	values, _, err := a.load(context.Background())

	return values, err
}

func (a AppConfig) Watch(ctx context.Context, onChange func(map[string]any)) error {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			values, changed, err := a.load(ctx)
			if err != nil {
				a.logger.WarnContext(
					ctx, "Error when reloading from AWS AppConfig",
					"application", a.client.application,
					"environment", a.client.environment,
					"profile", a.client.profile,
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

func (a AppConfig) load(ctx context.Context) (map[string]any, bool, error) {
	resp, changed, err := a.client.load(ctx)
	if !changed || err != nil {
		return nil, false, err
	}

	var values map[string]any
	if e := a.unmarshal(resp, &values); e != nil {
		return nil, false, fmt.Errorf("unmarshal: %w", e)
	}

	return values, changed, nil
}

func (a AppConfig) String() string {
	return "appConfig:" + a.client.application + "-" + a.client.environment + "-" + a.client.profile
}

type clientProxy struct {
	config       aws.Config
	application  string
	environment  string
	profile      string
	pollInterval time.Duration

	client     *appconfigdata.Client
	clientOnce sync.Once

	timeout time.Duration
	token   atomic.Pointer[string]
}

func (c *clientProxy) load(ctx context.Context) ([]byte, bool, error) {
	client, err := c.loadClient(ctx)
	if err != nil {
		return nil, false, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.GetLatestConfiguration(ctx,
		&appconfigdata.GetLatestConfigurationInput{ConfigurationToken: c.token.Load()},
	)
	if err != nil {
		return nil, false, fmt.Errorf("get latest configuration: %w", err)
	}
	c.token.Store(resp.NextPollConfigurationToken)

	// It may return empty configuration data if the client already has the latest version.
	return resp.Configuration, len(resp.Configuration) > 0, nil
}

func (c *clientProxy) loadClient(ctx context.Context) (*appconfigdata.Client, error) {
	var err error

	c.clientOnce.Do(func() {
		if c.timeout <= 0 {
			c.timeout = 10 * time.Second //nolint:gomnd
		}

		cctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		if reflect.ValueOf(c.config).IsZero() {
			if c.config, err = config.LoadDefaultConfig(cctx); err != nil {
				err = fmt.Errorf("load default AWS config: %w", err)

				return
			}
		}
		c.client = appconfigdata.NewFromConfig(c.config)

		var session *appconfigdata.StartConfigurationSessionOutput
		if session, err = c.client.StartConfigurationSession(cctx, &appconfigdata.StartConfigurationSessionInput{
			ApplicationIdentifier:                aws.String(c.application),
			ConfigurationProfileIdentifier:       aws.String(c.profile),
			EnvironmentIdentifier:                aws.String(c.environment),
			RequiredMinimumPollIntervalInSeconds: aws.Int32(int32(c.pollInterval.Seconds())),
		}); err != nil {
			err = fmt.Errorf("start configuration session: %w", err)

			return
		}
		c.token.Store(session.InitialConfigurationToken)
	})

	return c.client, err
}
