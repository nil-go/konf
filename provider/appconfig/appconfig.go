// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package appconfig loads configuration from AWS [AppConfig].
//
// # Change notification
//
// By default, it periodically polls the configuration only. if the SNS topic is provided,
// it will also listen to the events sent from AWS AppConfig using following extensions.
//   - [EventBridge extension] With SNS target
//   - [SNS extension]
//
// Only ON_DEPLOYMENT_ROLLED_BACK events [Fanout to Amazon SQS queues] and trigger polling the configuration
// and other type of events are ignored.
//
// # Permission
//
// It requires following permissions to access AWS AppConfig:
//   - appconfig:StartConfigurationSession
//   - appconfig:GetLatestConfiguration
//
// [AppConfig]: https://aws.amazon.com/systems-manager/features/appconfig
// [EventBridge extension]: https://docs.aws.amazon.com/appconfig/latest/userguide/working-with-appconfig-extensions-about-predefined-notification-eventbridge.html
// [SNS extension]: https://docs.aws.amazon.com/appconfig/latest/userguide/working-with-appconfig-extensions-about-predefined-notification-sns.html
// [Fanout to Amazon SQS queues]: https://docs.aws.amazon.com/sns/latest/dg/sns-sqs-as-subscriber.html
package appconfig

import (
	"context"
	"encoding/json"
	"errors"
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
	unmarshal    func([]byte, any) error
	pollInterval time.Duration

	onStatus func(bool, error)
	client   clientProxy
}

// New creates an AppConfig with the given application, environment, profile and Option(s).
//
// The application and environment must be the id (not name) if push mode has been enabled.
func New(application, environment, profile string, opts ...Option) *AppConfig {
	option := &options{
		client: clientProxy{
			application: application,
			environment: environment,
			profile:     profile,
		},
	}
	for _, opt := range opts {
		opt(option)
	}
	option.client.timeout = option.pollInterval / 2 //nolint:gomnd

	return (*AppConfig)(option)
}

var errNil = errors.New("nil AppConfig")

func (a *AppConfig) Load() (map[string]any, error) {
	if a == nil {
		return nil, errNil
	}

	values, _, err := a.load(context.Background())

	return values, err
}

func (a *AppConfig) Watch(ctx context.Context, onChange func(map[string]any)) error {
	if a == nil {
		return errNil
	}

	pollInterval := a.pollInterval
	if pollInterval == 0 {
		pollInterval = time.Minute
	}
	var tickCh <-chan time.Time
	if pollInterval > 0 {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		tickCh = ticker.C
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tickCh:
			values, changed, err := a.load(ctx)
			if a.onStatus != nil {
				a.onStatus(changed, err)
			}
			if changed {
				onChange(values)
			}
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
	config      aws.Config
	application string
	environment string
	profile     string

	client *appconfigdata.Client

	timeout       time.Duration
	nextPollToken atomic.Pointer[string]
	nextPollTime  atomic.Pointer[time.Time]
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

	if nextPollTime := p.nextPollTime.Load(); nextPollTime != nil && time.Now().Before(*nextPollTime) {
		// Have to wait until the next poll time.
		time.Sleep(time.Until(*nextPollTime))
	}

	ctx, cancel := context.WithTimeout(ctx, max(p.timeout, 10*time.Second)) //nolint:gomnd
	defer cancel()

	if p.nextPollToken.Load() == nil {
		session, err := p.client.StartConfigurationSession(ctx, &appconfigdata.StartConfigurationSessionInput{
			ApplicationIdentifier:                aws.String(p.application),
			ConfigurationProfileIdentifier:       aws.String(p.profile),
			EnvironmentIdentifier:                aws.String(p.environment),
			RequiredMinimumPollIntervalInSeconds: aws.Int32(15), //nolint:gomnd // The minimum interval supported.
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
