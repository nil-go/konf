// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
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
			description: "nil loader",
			opts:        []konf.Option{},
			loaders:     []konf.Loader{nil},
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

func TestConfig_Watch(t *testing.T) {
	t.Parallel()

	config := konf.New()
	watcher := mapWatcher(make(chan map[string]any))
	err := config.Load(watcher)
	assert.NoError(t, err)

	var value string
	assert.NoError(t, config.Unmarshal("config", &value))
	assert.Equal(t, "string", value)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		assert.NoError(t, config.Watch(ctx))
	}()

	var newValue atomic.Value
	config.OnChange(func(config *konf.Config) {
		var value string
		assert.NoError(t, config.Unmarshal("config", &value))
		newValue.Store(value)
	}, "config")
	watcher.change(map[string]any{"Config": "changed"})
	assert.Equal(t, "changed", newValue.Load())
}

type mapWatcher chan map[string]any

func (m mapWatcher) Load() (map[string]any, error) {
	return map[string]any{"Config": "string"}, nil
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

	config := konf.New()
	err := config.Load(errorWatcher{})
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	assert.EqualError(t, config.Watch(ctx), "watch configuration change: watch error")
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

	config := konf.New()
	err := config.Load(errorLoader{})
	assert.EqualError(t, err, "load configuration: load error")
}

type errorLoader struct{}

func (errorLoader) Load() (map[string]any, error) {
	return nil, errors.New("load error")
}
