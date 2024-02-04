// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package appconfig //nolint:testpackage
import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"

	"github.com/nil-go/konf/provider/appconfig/internal/assert"
)

func BenchmarkNew(b *testing.B) {
	var loader *AppConfig
	for i := 0; i < b.N; i++ {
		loader = New("app", "env", "profile")
	}
	b.StopTimer()

	assert.Equal(b, "appConfig:app-env-profile", loader.String())
}

func BenchmarkLoad(b *testing.B) {
	loader := &AppConfig{
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
	}
	b.ResetTimer()

	var (
		values map[string]any
		err    error
	)
	for i := 0; i < b.N; i++ {
		values, err = loader.Load()
	}
	b.StopTimer()

	assert.NoError(b, err)
	assert.Equal(b, "v", values["k"])
}
