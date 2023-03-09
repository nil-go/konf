// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/ktong/konf"
)

func TestConfig_Unmarshal(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []konf.Option
		assert      func(*konf.Config)
	}{
		{
			description: "empty values",
			assert: func(config *konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config", &cfg))
				require.Equal(t, "", cfg)
			},
		},
		{
			description: "nil errorLoader",
			opts:        []konf.Option{konf.WithLoader(nil)},
			assert: func(config *konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config", &cfg))
				require.Equal(t, "", cfg)
			},
		},
		{
			description: "for primary type",
			opts:        []konf.Option{konf.WithLoader(mapLoader{"config": "string"})},
			assert: func(config *konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config", &cfg))
				require.Equal(t, "string", cfg)
			},
		},
		{
			description: "config for struct",
			opts:        []konf.Option{konf.WithLoader(mapLoader{"config": "struct"})},
			assert: func(config *konf.Config) {
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
			assert: func(config *konf.Config) {
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
			assert: func(config *konf.Config) {
				var cfg string
				require.NoError(t, config.Unmarshal("config_nest", &cfg))
				require.Equal(t, "string", cfg)
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
			assert: func(config *konf.Config) {
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

func TestConfig_Watch(t *testing.T) {
	t.Parallel()

	watcher := mapWatcher(make(chan map[string]any))
	config, err := konf.New(konf.WithLoader(watcher))
	require.NoError(t, err)

	var cfg string
	require.NoError(t, config.Unmarshal("config", &cfg))
	require.Equal(t, "string", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		err := config.Watch(ctx, func(config *konf.Config) {
			defer waitGroup.Done()

			require.NoError(t, config.Unmarshal("config", &cfg))
		})
		require.NoError(t, err)
	}()

	watcher.change(map[string]any{"config": "changed"})
	waitGroup.Wait()

	require.Equal(t, "changed", cfg)
}

func TestConfig_Watch_twice(t *testing.T) {
	t.Parallel()

	config, err := konf.New(konf.WithLoader(mapWatcher(make(chan map[string]any))))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return config.Watch(ctx)
	})
	group.Go(func() error {
		return config.Watch(ctx)
	})

	require.EqualError(t, group.Wait(), "[konf] Watch only can be called once")
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
}

func TestConfig_Watch_error(t *testing.T) {
	t.Parallel()

	config, err := konf.New(konf.WithLoader(errorWatcher{}))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.EqualError(t, config.Watch(ctx), "[konf] watch configuration change: watch error")
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
	require.EqualError(t, err, "[konf] load configuration: load error")
}

type errorLoader struct{}

func (errorLoader) Load() (map[string]any, error) {
	return nil, errors.New("load error")
}

func TestConfig_logger(t *testing.T) {
	t.Parallel()

	logger := &logger{}
	_, err := konf.New(konf.WithLogger(logger), konf.WithLoader(mapLoader{}))
	require.NoError(t, err)

	require.Equal(t, "Loaded configuration.", logger.message)
	require.Equal(t, []any{"errorLoader", mapLoader{}}, logger.keyAndValues)
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
