// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package appconfig loads configuration from AWS AppConfig.
//
// AppConfig loads configuration from AWS AppConfig with the given application, environment, profile
// and returns a nested map[string]any that is parsed with the given unmarshal function.
//
// The unmarshal function must be able to unmarshal the file content into a map[string]any.
// For example, with the default json.Unmarshal, the file is parsed as JSON.
package appconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

	client       appConfigClient
	application  string
	environment  string
	profile      string
	pollInterval time.Duration

	token atomic.Pointer[string]
}

type appConfigClient interface {
	StartConfigurationSession(
		ctx context.Context,
		params *appconfigdata.StartConfigurationSessionInput,
		optFns ...func(*appconfigdata.Options),
	) (*appconfigdata.StartConfigurationSessionOutput, error)
	GetLatestConfiguration(
		ctx context.Context,
		params *appconfigdata.GetLatestConfigurationInput,
		optFns ...func(*appconfigdata.Options),
	) (*appconfigdata.GetLatestConfigurationOutput, error)
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
	if option.awsConfig == nil {
		awsConfig, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			panic(fmt.Sprintf("cannot load AWS default config: %v", err))
		}
		option.awsConfig = &awsConfig
	}
	option.AppConfig.client = appconfigdata.NewFromConfig(*option.awsConfig)

	return &option.AppConfig
}

func (a *AppConfig) Load() (map[string]any, error) {
	if a.token.Load() == nil {
		input := &appconfigdata.StartConfigurationSessionInput{
			ApplicationIdentifier:                aws.String(a.application),
			ConfigurationProfileIdentifier:       aws.String(a.profile),
			EnvironmentIdentifier:                aws.String(a.environment),
			RequiredMinimumPollIntervalInSeconds: aws.Int32(int32(max(a.pollInterval.Seconds()/2, 1))), //nolint:gomnd
		}
		output, err := a.client.StartConfigurationSession(context.Background(), input)
		if err != nil {
			return nil, fmt.Errorf("start configuration session: %w", err)
		}
		a.token.Store(output.InitialConfigurationToken)
	}

	input := &appconfigdata.GetLatestConfigurationInput{
		ConfigurationToken: a.token.Load(),
	}
	output, err := a.client.GetLatestConfiguration(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("get latest configuration: %w", err)
	}
	a.token.Store(output.NextPollConfigurationToken)

	var out map[string]any
	if err := a.unmarshal(output.Configuration, &out); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return out, nil
}

func (a *AppConfig) Watch(ctx context.Context, onChange func(map[string]any)) error {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			input := &appconfigdata.GetLatestConfigurationInput{
				ConfigurationToken: a.token.Load(),
			}
			output, err := a.client.GetLatestConfiguration(ctx, input)
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
			a.token.Store(output.NextPollConfigurationToken)

			if len(output.Configuration) == 0 {
				// It may return empty configuration data
				// if the client already has the latest version.
				continue
			}

			var out map[string]any
			if err := a.unmarshal(output.Configuration, &out); err != nil {
				a.logger.WarnContext(
					ctx, "Error when unmarshalling config from AWS AppConfig",
					"application", a.application,
					"environment", a.environment,
					"profile", a.profile,
					"error", err,
				)

				continue
			}

			onChange(out)
		case <-ctx.Done():
			return nil
		}
	}
}
