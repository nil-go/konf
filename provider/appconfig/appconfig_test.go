// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package appconfig_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	"github.com/aws/smithy-go/middleware"

	"github.com/nil-go/konf/provider/appconfig"
	"github.com/nil-go/konf/provider/appconfig/internal/assert"
)

func TestAppConfig_New_panic(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		call        func()
		err         string
	}{
		{
			description: "application",
			call: func() {
				appconfig.New("", "env", "profile")
			},
			err: "cannot create AppConfig with empty application",
		},
		{
			description: "environment",
			call: func() {
				appconfig.New("app", "", "profile")
			},
			err: "cannot create AppConfig with empty environment",
		},
		{
			description: "profile",
			call: func() {
				appconfig.New("app", "env", "")
			},
			err: "cannot create AppConfig with empty profile",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					assert.Equal(t, r.(string), testcase.err)
				}
			}()
			testcase.call()
		})
	}
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
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "StartConfigurationSession":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.StartConfigurationSessionOutput{
							InitialConfigurationToken: aws.String("initial-token"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte(`{"k":"v"}`),
							NextPollConfigurationToken: aws.String("next-token"),
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			unmarshal: json.Unmarshal,
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
			unmarshal: json.Unmarshal,
			err: "start configuration session: operation error AppConfigData: StartConfigurationSession," +
				" start session error",
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
							InitialConfigurationToken: aws.String("initial-token"),
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
							InitialConfigurationToken: aws.String("initial-token"),
						},
					}, middleware.Metadata{}, nil
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte(`{"k":"v"}`),
							NextPollConfigurationToken: aws.String("next-token"),
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
		log       string
	}{
		{
			description: "latest configuration",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte(`{"k":"v"}`),
							NextPollConfigurationToken: aws.String("next-token"),
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
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte{},
							NextPollConfigurationToken: aws.String("next-token"),
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
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("get latest configuration error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			log: `level=WARN msg="Error when reloading from AWS AppConfig"` +
				` application=app environment=env profile=profiler` +
				` error="get latest configuration: operation error AppConfigData: GetLatestConfiguration,` +
				` get latest configuration error"` + "\n",
		},
		{
			description: "unmarshal error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetLatestConfiguration":
					return middleware.FinalizeOutput{
						Result: &appconfigdata.GetLatestConfigurationOutput{
							Configuration:              []byte(`{"k":"v"}`),
							NextPollConfigurationToken: aws.String("next-token"),
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			unmarshal: func([]byte, any) error {
				return errors.New("unmarshal error")
			},
			log: `level=WARN msg="Error when reloading from AWS AppConfig"` +
				` application=app environment=env profile=profiler error="unmarshal: unmarshal error"` + "\n",
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
						return stack.Initialize.Add(
							middleware.InitializeMiddlewareFunc(
								"mock", func(
									ctx context.Context,
									input middleware.InitializeInput,
									handler middleware.InitializeHandler,
								) (middleware.InitializeOutput, middleware.Metadata, error) {
									if v, ok := input.Parameters.(*appconfigdata.GetLatestConfigurationInput); ok {
										if v.ConfigurationToken == nil {
											v.ConfigurationToken = aws.String("initial-token")
										}
									}

									return handler.HandleInitialize(ctx, input)
								},
							),
							middleware.Before,
						)
					},
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

			buf := new(buffer)
			loader := appconfig.New(
				"app", "env", "profiler",
				appconfig.WithAWSConfig(cfg),
				appconfig.WithPollInterval(100*time.Millisecond),
				appconfig.WithLogHandler(logHandler(buf)),
				appconfig.WithUnmarshal(testcase.unmarshal),
			)
			var values atomic.Value
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var waitGroup sync.WaitGroup
			waitGroup.Add(1)
			go func() {
				waitGroup.Done()

				err := loader.Watch(ctx, func(changed map[string]any) {
					values.Store(changed)
				})
				assert.NoError(t, err)
			}()
			waitGroup.Wait()

			time.Sleep(150 * time.Millisecond)
			if val, ok := values.Load().(map[string]any); ok {
				assert.Equal(t, testcase.expected, val)
			} else {
				assert.Equal(t, testcase.log, buf.String())
			}
		})
	}
}

func TestAppConfig_String(t *testing.T) {
	t.Parallel()

	loader := appconfig.New("app", "env", "profile")
	assert.Equal(t, "appConfig:app-env-profile", loader.String())
}

func logHandler(buf *buffer) *slog.TextHandler {
	return slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == slog.TimeKey {
				return slog.Attr{}
			}

			return attr
		},
	})
}

type buffer struct {
	b bytes.Buffer
	m sync.RWMutex
}

func (b *buffer) Read(p []byte) (int, error) {
	b.m.RLock()
	defer b.m.RUnlock()

	return b.b.Read(p)
}

func (b *buffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p)
}

func (b *buffer) String() string {
	b.m.RLock()
	defer b.m.RUnlock()

	return b.b.String()
}
