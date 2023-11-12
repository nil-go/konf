// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ktong/konf"
	"github.com/ktong/konf/internal/assert"
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
				assert.NoError(t, config.Unmarshal("config", &cfg))
				assert.Equal(t, "", cfg)
			},
		},
		{
			description: "nil loader",
			opts:        []konf.Option{konf.WithLoader(nil)},
			assert: func(config konf.Config) {
				var cfg string
				assert.NoError(t, config.Unmarshal("config", &cfg))
				assert.Equal(t, "", cfg)
			},
		},
		{
			description: "for primary type",
			opts:        []konf.Option{konf.WithLoader(mapLoader{"config": "string"})},
			assert: func(config konf.Config) {
				var cfg string
				assert.NoError(t, config.Unmarshal("config", &cfg))
				assert.Equal(t, "string", cfg)
			},
		},
		{
			description: "config for struct",
			opts:        []konf.Option{konf.WithLoader(mapLoader{"config": "struct"})},
			assert: func(config konf.Config) {
				var cfg struct {
					Config string
				}
				assert.NoError(t, config.Unmarshal("", &cfg))
				assert.Equal(t, "struct", cfg.Config)
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
				assert.NoError(t, config.Unmarshal("config.nest", &cfg))
				assert.Equal(t, "string", cfg)
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
				assert.NoError(t, config.Unmarshal("config_nest", &cfg))
				assert.Equal(t, "string", cfg)
			},
		},
		{
			description: "non string key",
			opts: []konf.Option{
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
				assert.NoError(t, config.Unmarshal("config.nest", &cfg))
				assert.Equal(t, "", cfg)
			},
		},
		{
			description: "configer",
			opts: []konf.Option{
				konf.WithLoader(mapLoader{}),
			},
			assert: func(config konf.Config) {
				var configured bool
				assert.NoError(t, config.Unmarshal("configured", &configured))
				assert.True(t, configured)
			},
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			config, err := konf.New(testcase.opts...)
			assert.NoError(t, err)
			testcase.assert(config)
		})
	}
}

type mapLoader map[string]any

func (m mapLoader) WithConfig(
	interface {
		Unmarshal(path string, target any) error
	},
) {
	m["configured"] = true
}

func (m mapLoader) Load() (map[string]any, error) {
	return m, nil
}

func TestConfig_Watch(t *testing.T) {
	t.Parallel()

	watcher := mapWatcher(make(chan map[string]any))
	config, err := konf.New(konf.WithLoader(watcher))
	assert.NoError(t, err)

	var cfg string
	assert.NoError(t, config.Unmarshal("config", &cfg))
	assert.Equal(t, "string", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		assert.NoError(t, config.Watch(ctx))
	}()

	var newCfg atomic.Value
	config.OnChange(func(unmarshaler konf.Unmarshaler) {
		var cfg string
		assert.NoError(t, config.Unmarshal("config", &cfg))
		newCfg.Store(cfg)
	})
	watcher.change(map[string]any{"config": "changed"})
	assert.Equal(t, "changed", newCfg.Load())
}

type mapWatcher chan map[string]any

func (m mapWatcher) Load() (map[string]any, error) {
	return map[string]any{"config": "string"}, nil
}

func (m mapWatcher) Watch(ctx context.Context, fn func(map[string]any)) error {
	for {
		select {
		case values := <-m:
			fn(values)
		case <-ctx.Done():
			return nil
		}
	}
}

func (m mapWatcher) change(values map[string]any) {
	m <- values

	time.Sleep(time.Second) // Wait for change gets propagated.
}

func TestConfig_Watch_error(t *testing.T) {
	t.Parallel()

	config, err := konf.New(konf.WithLoader(errorWatcher{}))
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	assert.EqualError(t, config.Watch(ctx), "[konf] watch configuration change: watch error")
}

type errorWatcher struct{}

func (errorWatcher) Load() (map[string]any, error) {
	return make(map[string]any), nil
}

func (errorWatcher) Watch(context.Context, func(map[string]any)) error {
	return errors.New("watch error")
}

func TestConfig_error(t *testing.T) {
	t.Parallel()

	_, err := konf.New(konf.WithLoader(errorLoader{}))
	assert.EqualError(t, err, "[konf] load configuration: load error")
}

type errorLoader struct{}

func (errorLoader) Load() (map[string]any, error) {
	return nil, errors.New("load error")
}
