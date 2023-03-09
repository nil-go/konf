// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf"
)

func TestConfig_Unmarshal(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []konf.Option
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
			description: "nil loader",
			opts:        []konf.Option{konf.WithLoader(nil)},
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config", &cfg))
				require.Equal(t, "", cfg)
			},
		},
		{
			description: "for primary type",
			opts:        []konf.Option{konf.WithLoader(mapLoader{"config": "string"})},
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config", &cfg))
				require.Equal(t, "string", cfg)
			},
		},
		{
			description: "config for struct",
			opts:        []konf.Option{konf.WithLoader(mapLoader{"config": "struct"})},
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
			opts: []konf.Option{
				konf.WithLoader(
					mapLoader{
						"config": map[string]any{
							"nest": "string",
						},
					},
				),
			},
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config.nest", &cfg))
				require.Equal(t, "string", cfg)
			},
		},
		{
			description: "customized delimiter",
			opts: []konf.Option{
				konf.WithDelimiter("_"),
				konf.WithLoader(
					mapLoader{
						"config": map[string]any{
							"nest": "string",
						},
					},
				),
			},
			assert: func(config konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config_nest", &cfg))
				require.Equal(t, "string", cfg)
			},
		},
		{
			description: "non string key",
			opts: []konf.Option{
				konf.WithDelimiter("_"),
				konf.WithLoader(
					mapLoader{
						"config": map[int]any{
							1: "string",
						},
					},
				),
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

			config, err := konf.New(testcase.opts...)
			require.NoError(t, err)
			testcase.assert(config)
		})
	}
}

type mapLoader map[string]any

func (m mapLoader) Load() (map[string]any, error) {
	return m, nil
}

func TestConfig_error(t *testing.T) {
	t.Parallel()

	_, err := konf.New(konf.WithLoader(loader{}))
	require.EqualError(t, err, "[konf] load configuration: error")
}

type loader struct{}

func (loader) Load() (map[string]any, error) {
	return nil, errors.New("error")
}

func TestConfig_logger(t *testing.T) {
	t.Parallel()

	logger := &logger{}
	_, err := konf.New(konf.WithLogger(logger), konf.WithLoader(mapLoader{}))
	require.NoError(t, err)

	require.Equal(t, "Loaded configuration.", logger.message)
	require.Equal(t, []any{"loader", mapLoader{}}, logger.keyAndValues)
}

type logger struct {
	konf.Logger
	message      string
	keyAndValues []any
}

func (l *logger) Info(message string, keyAndValues ...any) {
	l.message = message
	l.keyAndValues = keyAndValues
}
