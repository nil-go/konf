// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package appconfig loads configuration from AWS [AppConfig].
//
// It requires following permissions to access AWS AppConfig:
//   - appconfig:StartConfigurationSession
//   - appconfig:GetLatestConfiguration
//
// If change notification is enabled, it also requires following permissions:
//   - appconfig:GetApplication
//   - appconfig:GetEnvironment
//
// # Change notification
//
// By default, it periodically polls the configuration only.
// It also listens to change events by register it to SNS notifier with one of following extensions:
//   - [EventBridge extension] With SNS target
//   - [SNS extension]
//
// Only ON_DEPLOYMENT_ROLLED_BACK events trigger polling the configuration and other type of events are ignored.
//
// [AppConfig]: https://aws.amazon.com/systems-manager/features/appconfig
// [EventBridge extension]: https://docs.aws.amazon.com/appconfig/latest/userguide/working-with-appconfig-extensions-about-predefined-notification-eventbridge.html
// [SNS extension]: https://docs.aws.amazon.com/appconfig/latest/userguide/working-with-appconfig-extensions-about-predefined-notification-sns.html
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
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
)

// AppConfig is a Provider that loads configuration from AWS AppConfig.
//
// To create a new AppConfig, call [New].
type AppConfig struct {
	unmarshal    func([]byte, any) error
	pollInterval time.Duration

	onStatus  func(bool, error)
	changedCh chan struct{}
	client    clientProxy
}

// New creates an AppConfig with the given application (ID or Name),
// environment (ID or Name), profile (ID or Name) and Option(s).
func New(application, environment, profile string, opts ...Option) *AppConfig {
	option := &options{
		client: clientProxy{
			application: application,
			environment: environment,
			profile:     profile,
		},
		changedCh: make(chan struct{}, 1),
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
	if a.changedCh == nil {
		a.changedCh = make(chan struct{}, 1)
	}

	pollInterval := a.pollInterval
	if pollInterval == 0 {
		pollInterval = time.Minute
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			a.changed()
		case <-a.changedCh:
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

func (a *AppConfig) changed() {
	select {
	case a.changedCh <- struct{}{}:
	default:
		// Ignore if the channel is full since it's already triggered.
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

func (a *AppConfig) OnEvent(msg []byte) error { //nolint:cyclop,funlen
	if a == nil {
		return errNil
	}
	if msg == nil {
		return nil
	}

	//nolint:tagliatelle
	type (
		appConfig struct {
			Type        string `json:"Type"`
			Application struct {
				ID string `json:"Id"`
			} `json:"Application"`
			Environment struct {
				ID string `json:"Id"`
			} `json:"Environment"`
			ConfigurationProfile struct {
				ID   string `json:"Id"`
				Name string `json:"Name"`
			} `json:"ConfigurationProfile"`
		}
		appConfigEvent struct {
			// From EventBridge: https://docs.aws.amazon.com/appconfig/latest/userguide/working-with-appconfig-extensions-about-predefined-notification-eventbridge.html
			Source string    `json:"source"`
			Detail appConfig `json:"detail"`

			// From SNS: https://docs.aws.amazon.com/appconfig/latest/userguide/working-with-appconfig-extensions-about-predefined-notification-sns.html
			appConfig
		}
	)

	var event appConfigEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("unmarshal appconfig event: %w", err)
	}

	applicationID := event.Detail.Application.ID
	if applicationID == "" {
		applicationID = event.Application.ID
	}
	if err := a.client.ensureApplicationID(applicationID); err != nil {
		return err
	}
	environmentID := event.Detail.Environment.ID
	if environmentID == "" {
		environmentID = event.Environment.ID
	}
	if err := a.client.ensureEnvironmentID(environmentID); err != nil {
		return err
	}

	if event.Source == "aws.appconfig" &&
		event.Detail.Application.ID == a.client.applicationID &&
		event.Detail.Environment.ID == a.client.environmentID &&
		(event.Detail.ConfigurationProfile.ID == a.client.profile ||
			event.Detail.ConfigurationProfile.Name == a.client.profile) {
		if event.Detail.Type == "OnDeploymentRolledBack" {
			// Trigger to reload the configuration.
			a.changed()
		}

		return nil
	}

	if event.Application.ID == a.client.applicationID &&
		event.Environment.ID == a.client.environmentID &&
		(event.ConfigurationProfile.ID == a.client.profile ||
			event.ConfigurationProfile.Name == a.client.profile) {
		if event.Type == "OnDeploymentRolledBack" {
			// Trigger to reload the configuration.
			a.changed()
		}

		return nil
	}

	return fmt.Errorf("unsupported appconfig event: %w", errors.ErrUnsupported)
}

func (a *AppConfig) Status(onStatus func(bool, error)) {
	a.onStatus = onStatus
}

func (a *AppConfig) String() string {
	return "appconfig://" + a.client.application + "/" + a.client.profile
}

type clientProxy struct {
	config        aws.Config
	application   string
	applicationID string
	environment   string
	environmentID string
	profile       string

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

func (p *clientProxy) ensureApplicationID(applicationID string) error {
	if p.applicationID != "" || applicationID == "" {
		return nil
	}
	if applicationID == p.application {
		p.applicationID = applicationID

		return nil
	}

	client := appconfig.NewFromConfig(p.config)
	application, err := client.GetApplication(context.Background(), &appconfig.GetApplicationInput{
		ApplicationId: aws.String(applicationID),
	})
	if err != nil {
		return fmt.Errorf("get application: %w", err)
	}
	if aws.ToString(application.Name) == p.application {
		p.applicationID = applicationID
	}

	return nil
}

func (p *clientProxy) ensureEnvironmentID(environmentID string) error {
	if p.environmentID != "" || p.applicationID == "" || environmentID == "" {
		return nil
	}
	if environmentID == p.environment {
		p.environmentID = environmentID

		return nil
	}

	client := appconfig.NewFromConfig(p.config)
	environment, err := client.GetEnvironment(context.Background(), &appconfig.GetEnvironmentInput{
		ApplicationId: aws.String(p.applicationID),
		EnvironmentId: aws.String(environmentID),
	})
	if err != nil {
		return fmt.Errorf("get environment: %w", err)
	}
	if aws.ToString(environment.Name) == p.environment {
		p.environmentID = environmentID
	}

	return nil
}
