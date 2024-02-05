// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/provider/env"
)

func TestConfig_Load_panic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, r.(string), "cannot load config from nil loader")
		}
	}()
	_ = konf.New().Load(nil)
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
			assert: func(config *konf.Config) {
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
			assert: func(config *konf.Config) {
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
			assert: func(config *konf.Config) {
				var value string
				assert.NoError(t, config.Unmarshal("config.nest", &value))
				assert.Equal(t, "", value)
			},
		},
	}

	for i := range testcases {
		testcase := testcases[i]

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
	t.Setenv("CONFIG_NEST", "env")
	config := konf.New()
	err := config.Load(env.New())
	assert.NoError(t, err)
	err = config.Load(mapLoader{"owner": "map", "config": map[string]any{"nest": "map"}})
	assert.NoError(t, err)

	assert.Equal(t, "non-exist has no configuration.\n\n", config.Explain("non-exist"))
	assert.Equal(t,
		"owner has value[map] that is loaded by loader[map].\n\n",
		config.Explain("owner", konf.WithValueFormatter(
			func(_ string, _ konf.Loader, value any) string {
				return value.(string)
			},
		)),
	)
	expected := `config.nest has value[map] that is loaded by loader[map].
Here are other value(loader)s:
  - env(env)

`
	assert.Equal(t, expected, config.Explain("config"))
}
