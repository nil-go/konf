// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package parameterstore loads configuration from AWS [Parameter Store].
//
// It requires following permissions to access object from AWS S3:
//   - ssm:GetParametersByPath
//
// # Change notification
//
// By default, it's periodically polls the configuration.
// It also listens to change events by register it to SNS notifier with one of following setups:
//   - [EventBridge] with SNS target
//
// Only following events trigger polling the configuration and other type of events are ignored:
//   - Parameter Store Change
//
// [Parameter Store]: https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html
// [EventBridge]: https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-paramstore-cwe.html
package parameterstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	imaps "github.com/nil-go/konf/provider/parameterstore/internal/maps"
)

type ParameterStore struct {
	pollInterval time.Duration
	splitter     func(string) []string

	onStatus  func(bool, error)
	changedCh chan struct{}
	client    clientProxy
}

// New creates a ParameterStore with the given endpoint and Option(s).
func New(opts ...Option) *ParameterStore {
	option := &options{
		client:    clientProxy{},
		changedCh: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(option)
	}
	if option.client.path == "" {
		option.client.path = "/"
	}

	return (*ParameterStore)(option)
}

var errNil = errors.New("nil ParameterStore")

func (p *ParameterStore) Load() (map[string]any, error) {
	if p == nil {
		return nil, errNil
	}

	values, _, err := p.load(context.Background())

	return values, err
}

func (p *ParameterStore) Watch(ctx context.Context, onChange func(map[string]any)) error {
	if p == nil {
		return errNil
	}
	if p.changedCh == nil {
		p.changedCh = make(chan struct{}, 1)
	}

	pollInterval := time.Minute
	if p.pollInterval > 0 {
		pollInterval = p.pollInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.changed()
		case <-p.changedCh:
			values, changed, err := p.load(ctx)
			if p.onStatus != nil {
				p.onStatus(changed, err)
			}
			if changed {
				onChange(values)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (p *ParameterStore) changed() {
	select {
	case p.changedCh <- struct{}{}:
	default:
		// Ignore if the channel is full since it's already triggered.
	}
}

func (p *ParameterStore) load(ctx context.Context) (map[string]any, bool, error) {
	resp, changed, err := p.client.load(ctx)
	if !changed || err != nil {
		return nil, false, err
	}

	splitter := p.splitter
	if splitter == nil {
		splitter = func(s string) []string {
			parts := strings.Split(s, "/")
			if len(parts) > 0 && parts[0] == "" {
				parts = parts[1:]
			}

			return parts
		}
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

func (p *ParameterStore) OnEvent(msg []byte) error {
	if p == nil {
		return errNil
	}

	//nolint:tagliatelle
	var event struct {
		Source     string `json:"source"`
		DetailType string `json:"detail-type"`
	}
	err := json.Unmarshal(msg, &event)
	if err != nil {
		return fmt.Errorf("unmarshal parameter store event: %w", err)
	}

	if event.Source == "aws.ssm" {
		if event.DetailType == "Parameter Store Change" {
			// Trigger to reload the configuration.
			p.changed()
		}

		return nil
	}

	return fmt.Errorf("unsupported parameter store event: %w", errors.ErrUnsupported)
}

func (p *ParameterStore) Status(onStatus func(bool, error)) {
	p.onStatus = onStatus
}

func (p *ParameterStore) String() string {
	return "parameter-store:" + p.client.path
}

type clientProxy struct {
	path    string
	filters []types.ParameterStringFilter
	config  aws.Config

	client       *ssm.Client
	lastVersions atomic.Pointer[map[string]int64]
}

func (p *clientProxy) load(ctx context.Context) (map[string]string, bool, error) { //nolint:cyclop
	if p.client == nil {
		if reflect.ValueOf(p.config).IsZero() {
			var err error
			p.config, err = config.LoadDefaultConfig(ctx)
			if err != nil {
				return nil, false, fmt.Errorf("load default AWS config: %w", err)
			}
		}
		p.client = ssm.NewFromConfig(p.config)
	}
	if p.path == "" {
		p.path = "/"
	}

	var (
		parameters []types.Parameter
		nextToken  *string
	)
	for {
		output, err := p.client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:             aws.String(p.path),
			ParameterFilters: p.filters,
			Recursive:        aws.Bool(true),
			WithDecryption:   aws.Bool(true),
			NextToken:        nextToken,
		})
		if err != nil {
			return nil, false, fmt.Errorf("get parameters: %w", err)
		}
		parameters = append(parameters, output.Parameters...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	versions := make(map[string]int64, len(parameters))
	values := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		versions[*parameter.Name] = parameter.Version
		values[*parameter.Name] = *parameter.Value
	}

	if last := p.lastVersions.Load(); last != nil && maps.Equal(*last, versions) {
		return nil, false, nil
	}
	p.lastVersions.Store(&versions)

	return values, true, nil
}
