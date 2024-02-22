// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/provider/env"
)

func TestConfig_Load_error(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []konf.LoadOption
		err         string
	}{
		{
			description: "error",
			err:         "load configuration: load error",
		},
		{
			description: "continue on error",
			opts:        []konf.LoadOption{konf.ContinueOnError()},
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			config := konf.New()
			err := config.Load(&errorLoader{}, testcase.opts...)
			if testcase.err == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, testcase.err, err.Error())
			}
		})
	}
}

func TestConfig_Load_panic(t *testing.T) {
	t.Parallel()

	defer func() {
		assert.Equal(t, "cannot load config from nil loader", recover().(string))
	}()

	_ = konf.New().Load(nil)
	t.Fail()
}

func TestConfig_Unmarshal(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []konf.Option
		loaders     []konf.Loader
		assert      func(konf.Config)
	}{
		{
			description: "empty values",
			assert: func(config konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, "", value)
			},
		},
		{
			description: "for primary type",
			loaders:     []konf.Loader{mapLoader{"config": "string"}},
			assert: func(config konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, "string", value)
			},
		},
		{
			description: "config for struct",
			loaders:     []konf.Loader{mapLoader{"config": "struct"}},
			assert: func(config konf.Config) {
				var value struct {
					Config string
				}
				assert.NoError(t, config.Unmarshal("", &value))
				assert.Equal(t, "struct", value.Config)
			},
		},
		{
			description: "default delimiter",
			loaders: []konf.Loader{
				mapLoader{
					"config": map[string]any{
						"nest": "string",
					},
				},
			},
			assert: func(config konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config.nest", &value))
				assert.Equal(t, "string", value)
			},
		},
		{
			description: "customized delimiter",
			opts: []konf.Option{
				konf.WithDelimiter("_"),
			},
			loaders: []konf.Loader{
				mapLoader{
					"config": map[string]any{
						"nest": "string",
					},
				},
			},
			assert: func(config konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config_nest", &value))
				assert.Equal(t, "string", value)
			},
		},
		{
			description: "customized decode hook",
			opts: []konf.Option{
				konf.WithDecodeHook(mapstructure.StringToTimeDurationHookFunc()),
			},
			loaders: []konf.Loader{
				mapLoader{
					"config": map[string]any{
						"nest": "1s",
					},
				},
			},
			assert: func(config konf.Config) {
				var value time.Duration
				assert.NoError(t, config.Unmarshal("config.nest", &value))
				assert.Equal(t, time.Second, value)
			},
		},
		{
			description: "customized tag name",
			opts: []konf.Option{
				konf.WithTagName("test"),
			},
			loaders: []konf.Loader{
				mapLoader{
					"config": map[string]any{
						"nest": "string",
					},
				},
			},
			assert: func(config konf.Config) {
				var value struct {
					N string `test:"nest"`
				}
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, "string", value.N)
			},
		},
		{
			description: "non string key",
			loaders: []konf.Loader{
				mapLoader{
					"config": map[int]any{
						1: "string",
					},
				},
			},
			assert: func(config konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config.nest", &value))
				assert.Equal(t, "", value)
			},
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			config := konf.New(testcase.opts...)
			for _, loader := range testcase.loaders {
				assert.NoError(t, config.Load(loader))
			}
			testcase.assert(config)
		})
	}
}

type mapLoader map[string]any

func (m mapLoader) Load() (map[string]any, error) {
	return m, nil
}

func (m mapLoader) String() string {
	return "map"
}

func TestConfig_Explain(t *testing.T) {
	t.Parallel()

	config := konf.New()
	err := config.Load(env.New())
	assert.NoError(t, err)
	err = config.Load(mapLoader{
		"config": map[string]any{"nest": "env"},
	})
	assert.NoError(t, err)
	err = config.Load(mapLoader{
		"number":   123,
		"password": "password",
		"key":      []byte("AKIA9SKKLKSKKSKKSKK8"),
		"config":   map[string]any{"nest": "map"},
	})
	assert.NoError(t, err)

	testcases := []struct {
		description string
		path        string
		expected    string
	}{
		{
			description: "non-exist",
			path:        "non-exist",
			expected:    "non-exist has no configuration.\n\n",
		},
		{
			description: "number",
			path:        "number",
			expected:    "number has value[123] that is loaded by loader[map].\n\n",
		},
		{
			description: "password",
			path:        "password",
			expected:    "password has value[******] that is loaded by loader[map].\n\n",
		},
		{
			description: "API key",
			path:        "key",
			expected:    "key has value[AWS API Key] that is loaded by loader[map].\n\n",
		},
		{
			description: "config",
			path:        "config",
			expected: `config.nest has value[map] that is loaded by loader[map].
Here are other value(loader)s:
  - env(map)

`,
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testcase.expected, config.Explain(testcase.path))
		})
	}

	t.Run("with value formatter", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			"number has value[value:123] that is loaded by loader[map].\n\n",
			config.Explain("number", konf.WithValueFormatter(
				func(_ konf.Loader, _ string, value any) string {
					return fmt.Sprintf("value:%v", value)
				},
			)),
		)
	})
}
