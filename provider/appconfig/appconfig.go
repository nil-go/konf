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
	option.client.timeout = max(option.pollInterval/2, 10*time.Second)   //nolint:gomnd

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
				a.logger.LogAttrs(
					ctx, slog.LevelWarn,
					"Error when reloading from AWS AppConfig",
					slog.String("application", a.client.application),
					slog.String("environment", a.client.environment),
					slog.String("profile", a.client.profile),
					slog.Any("error", err),
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

	return values, true, nil
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

	client *appconfigdata.Client

	timeout time.Duration
	token   atomic.Pointer[string]
}

func (p *clientProxy) load(ctx context.Context) ([]byte, bool, error) {
	if p.client == nil {
		if reflect.ValueOf(p.config).IsZero() {
			var err error
			if p.config, err = config.LoadDefaultConfig(ctx); err != nil {
				return nil, false, fmt.Errorf("load default AWS config: %w", err)
			}
		}
		p.client = appconfigdata.NewFromConfig(p.config)
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	if p.token.Load() == nil {
		session, err := p.client.StartConfigurationSession(ctx, &appconfigdata.StartConfigurationSessionInput{
			ApplicationIdentifier:                aws.String(p.application),
			ConfigurationProfileIdentifier:       aws.String(p.profile),
			EnvironmentIdentifier:                aws.String(p.environment),
			RequiredMinimumPollIntervalInSeconds: aws.Int32(int32(p.pollInterval.Seconds())),
		})
		if err != nil {
			return nil, false, fmt.Errorf("start configuration session: %w", err)
		}
		p.token.Store(session.InitialConfigurationToken)
	}

	resp, err := p.client.GetLatestConfiguration(ctx,
		&appconfigdata.GetLatestConfigurationInput{ConfigurationToken: p.token.Load()},
	)
	if err != nil {
		return nil, false, fmt.Errorf("get latest configuration: %w", err)
	}
	p.token.Store(resp.NextPollConfigurationToken)

	// It may return empty configuration data if the client already has the latest version.
	return resp.Configuration, len(resp.Configuration) > 0, nil
}
