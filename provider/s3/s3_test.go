// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package s3_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"

	ks3 "github.com/nil-go/konf/provider/s3"
	"github.com/nil-go/konf/provider/s3/internal/assert"
)

func TestS3_empty(t *testing.T) {
	var loader *ks3.S3
	values, err := loader.Load()
	assert.EqualError(t, err, "nil S3")
	assert.Equal(t, nil, values)
	err = loader.Watch(context.Background(), nil)
	assert.EqualError(t, err, "nil S3")
	err = loader.OnEvent([]byte{})
	assert.EqualError(t, err, "nil S3")
}

func TestS3_Load(t *testing.T) {
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

			loader := ks3.New(
				"bucket/key",
				append(testcase.opts, ks3.WithAWSConfig(cfg))...,
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

func TestS3_Watch(t *testing.T) {
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
			loader := ks3.New(
				"bucket/key",
				append(testcase.opts, ks3.WithAWSConfig(cfg))...,
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
	opts        []ks3.Option
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
			description: "pull only",
			opts: []ks3.Option{
				ks3.WithPollInterval(10 * time.Millisecond),
			},
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetObject":
					return middleware.FinalizeOutput{
						Result: &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(`{"k":"v"}`)),
							ETag: aws.String("k42"),
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
			description: "get object error",
			opts: []ks3.Option{
				ks3.WithPollInterval(10 * time.Millisecond),
			},
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetObject":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("get object error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			err: "get object: operation error S3: GetObject, get object error",
		},
		{
			description: "unmarshal error",
			opts: []ks3.Option{
				ks3.WithPollInterval(10 * time.Millisecond),
				ks3.WithUnmarshal(func([]byte, any) error {
					return errors.New("unmarshal error")
				}),
			},
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetObject":
					return middleware.FinalizeOutput{
						Result: &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(`{"k":"v"}`)),
							ETag: aws.String("k42"),
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			err: "unmarshal: unmarshal error",
		},
	}
}

func watchcases() []testcase {
	return []testcase{
		{
			description: "object created (sns)",
			event: []byte(`
{
   "Records":[
      {
         "eventSource":"aws:s3",
         "eventName":"ObjectCreated:Put",
         "s3":{
            "bucket":{
               "name":"bucket"
            },
            "object":{
               "key":"key"
            }
         }
      }
   ]
}`),
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetObject":
					return middleware.FinalizeOutput{
						Result: &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(`{"k":"v"}`)),
							ETag: aws.String("k42"),
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
			description: "object created (event bridge)",
			event: []byte(`
{
  "detail-type": "Object Created",
  "source": "aws.s3",
  "detail": {
    "bucket": {
      "name": "bucket"
    },
    "object": {
      "key": "key"
    }
  }
}`),
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetObject":
					return middleware.FinalizeOutput{
						Result: &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(`{"k":"v"}`)),
							ETag: aws.String("k42"),
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
			description: "object deleted (sns)",
			event: []byte(`
{
   "Records":[
      {
         "eventSource":"aws:s3",
         "eventName":"ObjectRemoved:Delete",
         "s3":{
            "bucket":{
               "name":"bucket"
            },
            "object":{
               "key":"key"
            }
         }
      }
   ]
}`),
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetObject":
					return middleware.FinalizeOutput{
						Result: &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(`{"k":"v"}`)),
							ETag: aws.String("k42"),
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
			description: "object deleted (event bridge)",
			event: []byte(`
{
  "detail-type": "Object Deleted",
  "source": "aws.s3",
  "detail": {
    "bucket": {
      "name": "bucket"
    },
    "object": {
      "key": "key"
    }
  }
}`),
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetObject":
					return middleware.FinalizeOutput{
						Result: &s3.GetObjectOutput{
							Body: io.NopCloser(strings.NewReader(`{"k":"v"}`)),
							ETag: aws.String("k42"),
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
   "Records":[
      {
         "eventSource":"aws:s3",
         "eventName":"s3:ObjectCreated:Put",
         "s3":{
            "bucket":{
               "name":"another-bucket"
            },
            "object":{
               "key":"key"
            }
         }
      }
   ]
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
			err: "unsupported s3 event: unsupported operation",
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
			err: "unmarshal s3 event: invalid character 'o' in literal null (expecting 'u')",
		},
	}
}

func TestS3_String(t *testing.T) {
	t.Parallel()

	loader := ks3.New("bucket/key")
	assert.Equal(t, "s3://bucket/key", loader.String())

	loader = ks3.New("s3://bucket/key")
	assert.Equal(t, "s3://bucket/key", loader.String())
}
