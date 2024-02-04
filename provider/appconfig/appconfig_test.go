// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package appconfig //nolint:testpackage
import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"

	"github.com/nil-go/konf/provider/appconfig/internal/assert"
)

func TestAppConfig_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		client      appConfigClient
		unmarshal   func([]byte, any) error
		expected    map[string]any
		err         string
	}{
		{
			description: "appconfig",
			client: fakeAppConfigClient{
				getLatestConfiguration: func(
					context.Context,
					*appconfigdata.GetLatestConfigurationInput,
					...func(*appconfigdata.Options),
				) (*appconfigdata.GetLatestConfigurationOutput, error) {
					return &appconfigdata.GetLatestConfigurationOutput{
						Configuration:              []byte(`{"k":"v"}`),
						NextPollConfigurationToken: aws.String("next-token"),
					}, nil
				},
			},
			unmarshal: json.Unmarshal,
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "start session error",
			client: fakeAppConfigClient{
				startConfigurationSession: func(
					context.Context,
					*appconfigdata.StartConfigurationSessionInput,
					...func(*appconfigdata.Options),
				) (*appconfigdata.StartConfigurationSessionOutput, error) {
					return nil, errors.New("start session error")
				},
			},
			unmarshal: json.Unmarshal,
			err:       "start configuration session: start session error",
		},
		{
			description: "get configuration error",
			client: fakeAppConfigClient{
				getLatestConfiguration: func(
					context.Context,
					*appconfigdata.GetLatestConfigurationInput,
					...func(*appconfigdata.Options),
				) (*appconfigdata.GetLatestConfigurationOutput, error) {
					return nil, errors.New("get configuration error")
				},
			},
			err: "get latest configuration: get configuration error",
		},
		{
			description: "unmarshal error",
			client:      fakeAppConfigClient{},
			unmarshal: func([]byte, any) error {
				return errors.New("unmarshal error")
			},
			err: "unmarshal: unmarshal error",
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			values, err := (&AppConfig{client: testcase.client, unmarshal: testcase.unmarshal}).Load()
			if testcase.err != "" {
				assert.EqualError(t, err, testcase.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testcase.expected, values)
			}
		})
	}
}

func TestAppConfig_Watch(t *testing.T) {
	testcases := []struct {
		description string
		client      appConfigClient
		expected    map[string]any
	}{
		{
			description: "get latest configuration",
			client: fakeAppConfigClient{
				getLatestConfiguration: func(
					context.Context,
					*appconfigdata.GetLatestConfigurationInput,
					...func(*appconfigdata.Options),
				) (*appconfigdata.GetLatestConfigurationOutput, error) {
					return &appconfigdata.GetLatestConfigurationOutput{
						Configuration:              []byte(`{"k":"v"}`),
						NextPollConfigurationToken: aws.String("next-token"),
					}, nil
				},
			},
			expected: map[string]any{"k": "v"},
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			loader := &AppConfig{client: testcase.client, unmarshal: json.Unmarshal, pollInterval: time.Second}
			var values map[string]any
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var waitGroup sync.WaitGroup
			waitGroup.Add(1)
			go func() {
				err := loader.Watch(ctx, func(changed map[string]any) {
					defer waitGroup.Done()
					values = changed
				})
				assert.NoError(t, err)
			}()

			waitGroup.Wait()
			assert.Equal(t, testcase.expected, values)
		})
	}
}

type fakeAppConfigClient struct {
	startConfigurationSession func(
		ctx context.Context,
		params *appconfigdata.StartConfigurationSessionInput,
		optFns ...func(*appconfigdata.Options),
	) (*appconfigdata.StartConfigurationSessionOutput, error)
	getLatestConfiguration func(
		ctx context.Context,
		params *appconfigdata.GetLatestConfigurationInput,
		optFns ...func(*appconfigdata.Options),
	) (*appconfigdata.GetLatestConfigurationOutput, error)
}

func (f fakeAppConfigClient) StartConfigurationSession(
	ctx context.Context,
	params *appconfigdata.StartConfigurationSessionInput,
	optFns ...func(*appconfigdata.Options),
) (*appconfigdata.StartConfigurationSessionOutput, error) {
	if f.startConfigurationSession != nil {
		return f.startConfigurationSession(ctx, params, optFns...)
	}

	return &appconfigdata.StartConfigurationSessionOutput{InitialConfigurationToken: aws.String("initial-token")}, nil
}

func (f fakeAppConfigClient) GetLatestConfiguration(
	ctx context.Context,
	params *appconfigdata.GetLatestConfigurationInput,
	optFns ...func(*appconfigdata.Options),
) (*appconfigdata.GetLatestConfigurationOutput, error) {
	if f.getLatestConfiguration != nil {
		return f.getLatestConfiguration(ctx, params, optFns...)
	}

	return &appconfigdata.GetLatestConfigurationOutput{
		NextPollConfigurationToken: aws.String("next-token"),
		Configuration:              make([]byte, 0),
	}, nil
}

func TestAppConfig_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "appConfig:app-env-profile", New("app", "env", "profile").String())
}
