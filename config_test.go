// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"testing"
	"time"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/provider/env"
)

func TestConfig_nil(t *testing.T) {
	var config *konf.Config

	assert.True(t, !config.Exists([]string{"key"}))
	assert.Equal(t, "key has no configuration.\n\n", config.Explain("key"))
	var value string
	assert.NoError(t, config.Unmarshal("key", &value))
	assert.Equal(t, "", value)
}

func TestConfig_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		loader      konf.Loader
		err         string
	}{
		{
			description: "error",
			loader:      &errorLoader{},
			err:         "load configuration: load error",
		},
		{
			description: "nil loader",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			var config konf.Config
			err := config.Load(testcase.loader)
			if testcase.err == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, testcase.err, err.Error())
			}
		})
	}
}

func TestConfig_Unmarshal(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []konf.Option
		loaders     []konf.Loader
		assert      func(*konf.Config)
	}{
		{
			description: "empty values",
			assert: func(config *konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, "", value)
			},
		},
		{
			description: "for primary type",
			loaders:     []konf.Loader{mapLoader{"config": "string"}},
			assert: func(config *konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, "string", value)
			},
		},
		{
			description: "config for map",
			loaders:     []konf.Loader{mapLoader{"Config": "struct"}},
			assert: func(config *konf.Config) {
				var value map[string]string
				assert.NoError(t, config.Unmarshal("", &value))
				assert.Equal(t, "struct", value["Config"])
			},
		},
		{
			description: "config for struct",
			loaders:     []konf.Loader{mapLoader{"config": "struct"}},
			assert: func(config *konf.Config) {
				var value struct {
					Config string
				}
				assert.NoError(t, config.Unmarshal("", &value))
				assert.Equal(t, "struct", value.Config)
			},
		},
		{
			description: "config for struct (case sensitive)",
			opts:        []konf.Option{konf.WithCaseSensitive()},
			loaders:     []konf.Loader{mapLoader{"ConfigValue": "struct"}},
			assert: func(config *konf.Config) {
				var value struct {
					ConfigValue string
					Configvalue string
				}
				assert.NoError(t, config.Unmarshal("", &value))
				assert.Equal(t, "struct", value.ConfigValue)
				assert.Equal(t, "", value.Configvalue)
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
			assert: func(config *konf.Config) {
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
			assert: func(config *konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config_nest", &value))
				assert.Equal(t, "string", value)
			},
		},
		{
			description: "decode hook",
			loaders: []konf.Loader{
				mapLoader{
					"config": map[string]any{
						"nest": "sky",
					},
				},
			},
			assert: func(config *konf.Config) {
				var value struct {
					N Enum `konf:"nest"`
				}
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, Sky, value.N)
			},
		},
		{
			description: "customized decode hook",
			opts: []konf.Option{
				konf.WithDecodeHook[string, time.Duration](time.ParseDuration),
			},
			loaders: []konf.Loader{
				mapLoader{
					"config": map[string]any{
						"nest": "1s",
					},
				},
			},
			assert: func(config *konf.Config) {
				var value struct {
					N time.Duration `konf:"nest"`
				}
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, time.Second, value.N)
			},
		},
		{
			description: "tag name",
			loaders: []konf.Loader{
				mapLoader{
					"config": map[string]any{
						"nest": "string",
					},
				},
			},
			assert: func(config *konf.Config) {
				var value struct {
					N string `konf:"nest"`
				}
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, "string", value.N)
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
						"nest": "a,b,c",
					},
				},
			},
			assert: func(config *konf.Config) {
				var value struct {
					N []string `test:"nest"`
				}
				assert.NoError(t, config.Unmarshal("config", &value))
				assert.Equal(t, []string{"a", "b", "c"}, value.N)
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
			assert: func(config *konf.Config) {
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

			var config konf.Config
			if len(testcase.opts) > 0 {
				config = *konf.New(testcase.opts...)
			}
			for _, loader := range testcase.loaders {
				assert.NoError(t, config.Load(loader))
			}
			testcase.assert(&config)
		})
	}
}

func TestConfigCopyPanic(t *testing.T) {
	defer func() {
		assert.Equal(t, recover(), "illegal use of non-zero Config copied by value")
	}()

	var config konf.Config
	assert.NoError(t, config.Load(mapLoader{}))
	configCopy := config //nolint:govet
	assert.NoError(t, configCopy.Load(mapLoader{}))

	t.Fail()
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

	var config konf.Config
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
		"config":   map[string]any{"Nest": "map"},
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
			expected: `config.Nest has value[map] that is loaded by loader[map].
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
}

type Enum int

const (
	Unknown Enum = iota
	Sky
)

func (e *Enum) UnmarshalText(text []byte) error {
	switch string(text) {
	case "sky":
		*e = Sky
	default:
		*e = Unknown
	}

	return nil
}
