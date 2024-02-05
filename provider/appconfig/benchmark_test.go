// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package appconfig_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	"github.com/aws/smithy-go/middleware"

	"github.com/nil-go/konf/provider/appconfig"
	"github.com/nil-go/konf/provider/appconfig/internal/assert"
)

func BenchmarkNew(b *testing.B) {
	var loader *appconfig.AppConfig
	for i := 0; i < b.N; i++ {
		loader = appconfig.New("app", "env", "profile")
	}
	b.StopTimer()

	assert.Equal(b, "appConfig:app-env-profile", loader.String())
}

func BenchmarkLoad(b *testing.B) {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithAPIOptions([]func(*middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc(
						"mock", func(
							ctx context.Context,
							input middleware.FinalizeInput,
							handler middleware.FinalizeHandler,
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
					),
					middleware.Before,
				)
			},
		}),
	)
	assert.NoError(b, err)

	loader := appconfig.New("app", "env", "profiler", appconfig.WithAWSConfig(&cfg))
	b.ResetTimer()

	var values map[string]any
	for i := 0; i < b.N; i++ {
		values, err = loader.Load()
	}
	b.StopTimer()

	assert.NoError(b, err)
	assert.Equal(b, "v", values["k"])
}
