// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package parameterstore_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go/middleware"

	"github.com/nil-go/konf/provider/parameterstore"
	"github.com/nil-go/konf/provider/parameterstore/internal/assert"
)

func TestParameterStore_empty(t *testing.T) {
	var loader *parameterstore.ParameterStore
	values, err := loader.Load()
	assert.EqualError(t, err, "nil ParameterStore")
	assert.Equal(t, nil, values)
	err = loader.Watch(context.Background(), nil)
	assert.EqualError(t, err, "nil ParameterStore")
	err = loader.OnEvent([]byte{})
	assert.EqualError(t, err, "nil ParameterStore")
}

func TestParameterStore_Load(t *testing.T) {
	t.Parallel()

	for _, testcase := range testcases() {
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

			loader := parameterstore.New(
				append(testcase.opts, parameterstore.WithAWSConfig(cfg))...,
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

func TestParameterStore_Watch(t *testing.T) {
	t.Parallel()

	for _, testcase := range append(testcases(), watchcases()...) {
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

			var err atomic.Pointer[error]
			loader := parameterstore.New(
				append(testcase.opts, parameterstore.WithAWSConfig(cfg))...,
			)
			loader.Status(func(_ bool, e error) {
				if e != nil {
					err.Store(&e)
				}
			})

			values := make(chan map[string]any)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			started := make(chan struct{})
			go func() {
				close(started)

				e := loader.Watch(ctx, func(changed map[string]any) {
					values <- changed
				})
				assert.NoError(t, e)
			}()
			<-started
			if testcase.event != nil {
				err := loader.OnEvent(testcase.event)
				if testcase.err != "" {
					assert.EqualError(t, err, testcase.err)
				} else {
					assert.NoError(t, err)
				}
			}

			time.Sleep(15 * time.Millisecond) // wait for the first tick, but not the second
			select {
			case val := <-values:
				assert.Equal(t, testcase.expected, val)
			default:
				if e := err.Load(); e != nil {
					assert.EqualError(t, *e, testcase.err)
				}
			}
		})
	}
}

type testcase struct {
	description string
	opts        []parameterstore.Option
	event       []byte
	middleware  func(
		context.Context,
		middleware.FinalizeInput,
		middleware.FinalizeHandler,
	) (middleware.FinalizeOutput, middleware.Metadata, error)
	expected map[string]any
	err      string
}

func testcases() []testcase {
	return []testcase{
		{
			description: "parameters",
			opts: []parameterstore.Option{
				parameterstore.WithPollInterval(10 * time.Millisecond),
			},
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetParametersByPath":
					return middleware.FinalizeOutput{
						Result: &ssm.GetParametersByPathOutput{
							Parameters: []types.Parameter{
								{
									Name:    aws.String("/k"),
									Value:   aws.String("v"),
									Version: 1,
								},
								{
									Name:    aws.String("d"),
									Value:   aws.String("."),
									Version: 1,
								},
							},
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			expected: map[string]any{
				"k": "v",
				"d": ".",
			},
		},
		{
			description: "with path and filter",
			opts: []parameterstore.Option{
				parameterstore.WithPath("/"),
				parameterstore.WithFilter(types.ParameterStringFilter{
					Key:    aws.String("Type"),
					Option: aws.String("Equals"),
					Values: []string{"String"},
				}),
				parameterstore.WithPollInterval(10 * time.Millisecond),
			},
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetParametersByPath":
					return middleware.FinalizeOutput{
						Result: &ssm.GetParametersByPathOutput{
							Parameters: []types.Parameter{
								{
									Name:    aws.String("/k"),
									Value:   aws.String("v"),
									Version: 1,
								},
							},
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
			description: "with nil splitter",
			opts: []parameterstore.Option{
				parameterstore.WithPollInterval(10 * time.Millisecond),
				parameterstore.WithNameSplitter(func(string) []string { return nil }),
			},
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetParametersByPath":
					return middleware.FinalizeOutput{
						Result: &ssm.GetParametersByPathOutput{
							Parameters: []types.Parameter{
								{
									Name:    aws.String("/k"),
									Value:   aws.String("v"),
									Version: 1,
								},
							},
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			expected: map[string]any{},
		},
		{
			description: "with empty splitter",
			opts: []parameterstore.Option{
				parameterstore.WithPollInterval(10 * time.Millisecond),
				parameterstore.WithNameSplitter(func(string) []string { return []string{""} }),
			},
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetParametersByPath":
					return middleware.FinalizeOutput{
						Result: &ssm.GetParametersByPathOutput{
							Parameters: []types.Parameter{
								{
									Name:    aws.String("/k"),
									Value:   aws.String("v"),
									Version: 1,
								},
							},
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			expected: map[string]any{},
		},
		{
			description: "GetParametersByPath error",
			opts: []parameterstore.Option{
				parameterstore.WithPollInterval(10 * time.Millisecond),
			},
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetParametersByPath":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("get parameters by path error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			err: "get parameters: operation error SSM: GetParametersByPath, get parameters by path error",
		},
	}
}

func watchcases() []testcase {
	return []testcase{
		{
			description: "oParameter Store Change",
			event: []byte(`
{
  "detail-type": "Parameter Store Change",
  "source": "aws.ssm"
}`),
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetParametersByPath":
					return middleware.FinalizeOutput{
						Result: &ssm.GetParametersByPathOutput{
							Parameters: []types.Parameter{
								{
									Name:    aws.String("/k"),
									Value:   aws.String("v"),
									Version: 1,
								},
							},
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
			description: "unmatched message created",
			event: []byte(`
{
  "detail-type": "Object Created",
  "source": "aws.s3"
}`),
			middleware: func(
				context.Context,
				middleware.FinalizeInput,
				middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
			},
			expected: map[string]any{
				"k": "v",
			},
			err: "unsupported parameter store event: unsupported operation",
		},
		{
			description: "no-json messages",
			event:       []byte(`not a json`),
			middleware: func(
				context.Context,
				middleware.FinalizeInput,
				middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
			},
			err: "unmarshal parameter store event: invalid character 'o' in literal null (expecting 'u')",
		},
	}
}

func TestS3_String(t *testing.T) {
	t.Parallel()

	loader := parameterstore.New()
	assert.Equal(t, "parameter-store:/", loader.String())

	loader = parameterstore.New(parameterstore.WithPath("/path"))
	assert.Equal(t, "parameter-store:/path", loader.String())
}
