// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []konf.Option
		loader      konf.Loader
		assert      func(konf.Config)
	}{
		{
			description: "empty values",
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config", &cfg))
				require.Equal(t, "", cfg)
			},
		},
		{
			description: "for primary type",
			loader: mapLoader{
				"config": "string",
			},
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config", &cfg))
				require.Equal(t, "string", cfg)
			},
		},
		{
			description: "config for struct",
			loader: mapLoader{
				"config": "struct",
			},
			assert: func(config konf.Config) {
				var cfg struct {
					Config string
				}
				require.NoError(t, config.Unmarshal("", &cfg))
				require.Equal(t, "struct", cfg.Config)
			},
		},
		{
			description: "default delimiter",
			loader: mapLoader{
				"config": map[string]any{
					"nest": "string",
				},
			},
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config.nest", &cfg))
				require.Equal(t, "string", cfg)
			},
		},
		{
			description: "customized delimiter",
			opts:        []konf.Option{konf.WithDelimiter("_")},
			loader: mapLoader{
				"config": map[string]any{
					"nest": "string",
				},
			},
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config_nest", &cfg))
				require.Equal(t, "string", cfg)
			},
		},
		{
			description: "non string key",
			loader: mapLoader{
				"config": map[int]any{
					1: "string",
				},
			},
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config.nest", &cfg))
				require.Equal(t, "", cfg)
			},
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			config := konf.New(testcase.opts...)
			require.NoError(t, config.Load(testcase.loader))
			testcase.assert(config)
		})
	}
}
