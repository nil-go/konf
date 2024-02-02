// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"testing"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/provider/env"
)

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
			err := config.Load(testcase.loaders...)
			assert.NoError(t, err)
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
	err := config.Load(env.New(), mapLoader{"owner": "map", "config": map[string]any{"nest": "map"}})
	assert.NoError(t, err)

	assert.Equal(t, "non-exist has no configuration.\n\n", config.Explain("non-exist"))
	assert.Equal(t, "owner has value[map] that is loaded by loader[map].\n\n", config.Explain("owner"))
	expected := `config.nest has value[map] that is loaded by loader[map].
Here are other value(loader)s:
  - env(env)

`
	assert.Equal(t, expected, config.Explain("config"))
}
