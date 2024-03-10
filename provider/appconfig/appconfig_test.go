// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package appconfig_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	"github.com/aws/smithy-go/middleware"
	"github.com/aws/smithy-go/transport/http"

	"github.com/nil-go/konf/provider/appconfig"
	"github.com/nil-go/konf/provider/appconfig/internal/assert"
)

func TestAppConfig_empty(t *testing.T) {
	var loader appconfig.AppConfig
	values, err := loader.Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assert.Equal(t, nil, values)
}

func TestAppConfig_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		middleware  func(
			context.Context,
			middleware.FinalizeInput,
			middleware.FinalizeHandler,
		) (middleware.FinalizeOutput, middleware.Metadata, error)
		unmarshal func([]byte, any) error
		expected  map[string]any
		err       string
	}{
		{
			description: "appconfig",
			middleware: func(
				ctx context.Context,
				input middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.StartConfigurationSessionOutput{
							InitialConfigurationToken: aws.String("initial-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					if ct := input.Request.(*http.Request).URL.Query().Get("configuration_token"); ct == "next-nextPollToken" {
						return middleware.FinalizeOutput{
							Result: &appconfigdata.GetLatestConfigurationOutput{
								Configuration:              []byte{},
								NextPollConfigurationToken: aws.String("next-nextPollToken"),
								NextPollIntervalInSeconds:  60,
							},
						}, middleware.Metadata{}, nil
					}

					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte(`{"k":"v"}`),
							NextPollConfigurationToken: aws.String("next-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "start session error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("start session error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			err: "start configuration session: operation error AppConfigData: StartConfigurationSession, start session error",
		},
		{
			description: "get configuration error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.StartConfigurationSessionOutput{
							InitialConfigurationToken: aws.String("initial-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("get configuration error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			err: "get latest configuration: operation error AppConfigData: GetLatestConfiguration, get configuration error",
		},
		{
			description: "unmarshal error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.StartConfigurationSessionOutput{
							InitialConfigurationToken: aws.String("initial-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte(`{"k":"v"}`),
							NextPollConfigurationToken: aws.String("next-nextPollToken"),
							NextPollIntervalInSeconds:  60,
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			unmarshal: func([]byte, any) error {
				return errors.New("unmarshal error")
			},
			err: "unmarshal: unmarshal error",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.LoadDefaultConfig(
				context.Background(),
				config.WithAPIOptions([]func(*middleware.Stack) error{
					func(stack *middleware.Stack) error {
						return stack.Finalize.Add(
							middleware.FinalizeMiddlewareFunc(
								"mock",
								testcase.middleware,
							),
							middleware.Before,
						)
					},
				}),
			)
			assert.NoError(t, err)

			loader := appconfig.New(
				"app", "env", "profiler",
				appconfig.WithAWSConfig(cfg),
				appconfig.WithUnmarshal(testcase.unmarshal),
			)
			values, err := loader.Load()
			if testcase.err != "" {
				assert.EqualError(t, err, testcase.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testcase.expected, values)
				values, err = loader.Load()
				assert.NoError(t, err)
				assert.Equal(t, nil, values)
			}
		})
	}
}

func TestAppConfig_Watch(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		middleware  func(
			context.Context,
			middleware.FinalizeInput,
			middleware.FinalizeHandler,
		) (middleware.FinalizeOutput, middleware.Metadata, error)
		unmarshal func([]byte, any) error
		expected  map[string]any
		err       string
	}{
		{
			description: "latest configuration",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.StartConfigurationSessionOutput{
							InitialConfigurationToken: aws.String("initial-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte(`{"k":"v"}`),
							NextPollConfigurationToken: aws.String("next-nextPollToken"),
							NextPollIntervalInSeconds:  1,
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			expected: map[string]any{"k": "v"},
		},
		{
			description: "empty configuration",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.StartConfigurationSessionOutput{
							InitialConfigurationToken: aws.String("initial-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte{},
							NextPollConfigurationToken: aws.String("next-nextPollToken"),
							NextPollIntervalInSeconds:  1,
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
		},
		{
			description: "get configuration error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.StartConfigurationSessionOutput{
							InitialConfigurationToken: aws.String("initial-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("get latest configuration error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			err: "get latest configuration: operation error AppConfigData: GetLatestConfiguration, get latest configuration error",
		},
		{
			description: "unmarshal error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.StartConfigurationSessionOutput{
							InitialConfigurationToken: aws.String("initial-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte(`{"k":"v"}`),
							NextPollConfigurationToken: aws.String("next-nextPollToken"),
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			unmarshal: func([]byte, any) error {
				return errors.New("unmarshal error")
			},
			err: "unmarshal: unmarshal error",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			cfg, cerr := config.LoadDefaultConfig(
				context.Background(),
				config.WithAPIOptions([]func(*middleware.Stack) error{
					func(stack *middleware.Stack) error {
						return stack.Finalize.Add(
							middleware.FinalizeMiddlewareFunc(
								"mock",
								testcase.middleware,
							),
							middleware.Before,
						)
					},
				}),
			)
			assert.NoError(t, cerr)

			loader := appconfig.New(
				"app", "env", "profiler",
				appconfig.WithAWSConfig(cfg),
				appconfig.WithUnmarshal(testcase.unmarshal),
			)
			_, _ = loader.Load()

			var err atomic.Pointer[error]
			loader.Status(func(_ bool, e error) {
				if e != nil {
					err.Store(&e)
				}
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			values := make(chan map[string]any)
			started := make(chan struct{})
			go func() {
				close(started)

				e := loader.Watch(ctx, func(changed map[string]any) {
					values <- changed
				})
				assert.NoError(t, e)
			}()
			<-started

			time.Sleep(time.Second) // wait for the first tick, but not the second
			select {
			case val := <-values:
				assert.Equal(t, testcase.expected, val)
			default:
				if testcase.err == "" {
					assert.Equal(t, nil, err.Load())
				} else {
					assert.EqualError(t, *err.Load(), testcase.err)
				}
			}
		})
	}
}

func TestAppConfig_String(t *testing.T) {
	t.Parallel()

	loader := appconfig.New("app", "env", "profile")
	assert.Equal(t, "appconfig://app/profile", loader.String())
}
