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
	unmarshal func([]byte, any) error

	onStatus func(bool, error)
	client   *clientProxy
}

// New creates an AppConfig with the given application, environment, profile and Option(s).
func New(application, environment, profile string, opts ...Option) *AppConfig {
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

	return (*AppConfig)(option)
}

func (a *AppConfig) Load() (map[string]any, error) {
	values, _, err := a.load(context.Background())

	return values, err
}

func (a *AppConfig) Watch(ctx context.Context, onChange func(map[string]any)) error {
	timer := time.NewTimer(a.client.nextPollDuration())
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			values, changed, err := a.load(ctx)
			if a.onStatus != nil {
				a.onStatus(changed, err)
			}
			if changed {
				onChange(values)
			}
			timer.Reset(a.client.nextPollDuration())
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

func (a *AppConfig) Status(onStatus func(bool, error)) {
	a.onStatus = onStatus
}

func (a *AppConfig) String() string {
	return "appconfig://" + a.client.application + "/" + a.client.profile
}

type clientProxy struct {
	config       aws.Config
	application  string
	environment  string
	profile      string
	pollInterval time.Duration

	client *appconfigdata.Client

	nextPollToken atomic.Pointer[string]
	nextPollTime  atomic.Pointer[time.Time]
}

func (p *clientProxy) load(ctx context.Context) ([]byte, bool, error) {
	if p == nil {
		// Use empty instance instead to avoid nil pointer dereference,
		// Assignment propagates only to callee but not to caller.
		p = &clientProxy{}
	}

	if p.client == nil {
		if reflect.ValueOf(p.config).IsZero() {
			var err error
			if p.config, err = config.LoadDefaultConfig(ctx); err != nil {
				return nil, false, fmt.Errorf("load default AWS config: %w", err)
			}
		}
		p.client = appconfigdata.NewFromConfig(p.config)
	}
	// The minimum interval required by AWS AppConfig SDK is 15 seconds.
	p.pollInterval = max(p.pollInterval, 15*time.Second) //nolint:gomnd

	ctx, cancel := context.WithTimeout(ctx, p.pollInterval)
	defer cancel()

	if p.nextPollToken.Load() == nil {
		session, err := p.client.StartConfigurationSession(ctx, &appconfigdata.StartConfigurationSessionInput{
			ApplicationIdentifier:                aws.String(p.application),
			ConfigurationProfileIdentifier:       aws.String(p.profile),
			EnvironmentIdentifier:                aws.String(p.environment),
			RequiredMinimumPollIntervalInSeconds: aws.Int32(int32(p.pollInterval.Seconds())),
		})
		if err != nil {
			return nil, false, fmt.Errorf("start configuration session: %w", err)
		}
		p.nextPollToken.Store(session.InitialConfigurationToken)
	}

	resp, err := p.client.GetLatestConfiguration(ctx,
		&appconfigdata.GetLatestConfigurationInput{ConfigurationToken: p.nextPollToken.Load()},
	)
	if err != nil {
		return nil, false, fmt.Errorf("get latest configuration: %w", err)
	}
	p.nextPollToken.Store(resp.NextPollConfigurationToken)
	nextPollTime := time.Now().Add(time.Duration(resp.NextPollIntervalInSeconds) * time.Second)
	p.nextPollTime.Store(&nextPollTime)

	// It may return empty configuration data if the client already has the latest version.
	return resp.Configuration, len(resp.Configuration) > 0, nil
}

func (p *clientProxy) nextPollDuration() time.Duration {
	if nextPollTime := p.nextPollTime.Load(); nextPollTime != nil {
		if duration := time.Until(*nextPollTime); duration > 0 {
			return duration
		}
	}

	return p.pollInterval
}
