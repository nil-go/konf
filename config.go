// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/ktong/konf/internal/maps"
)

// Config is a registry which holds configuration loaded by Loader(s).
type Config struct {
	delimiter string

	values    *provider
	providers []*provider
	watchOnce sync.Once
}

// New returns a Config with the given Option(s).
func New(opts ...Option) (*Config, error) {
	option := apply(opts)
	config := option.Config
	config.values = &provider{values: make(map[string]any)}
	config.providers = make([]*provider, 0, len(option.loaders))

	for _, loader := range option.loaders {
		if loader == nil {
			continue
		}

		if configAware, ok := loader.(ConfigAware); ok {
			configAware.WithConfig(config)
		}

		values, err := loader.Load()
		if err != nil {
			return nil, fmt.Errorf("[konf] load configuration: %w", err)
		}
		maps.Merge(config.values.values, values)
		slog.Info(
			"Configuration has been loaded.",
			"loader", loader,
		)

		provider := &provider{
			values: values,
		}
		if w, ok := loader.(Watcher); ok {
			provider.watcher = w
		}
		config.providers = append(config.providers, provider)
	}

	return config, nil
}

// Unmarshal loads configuration under the given path into the given object
// pointed to by target. It supports [mapstructure] tags on struct fields.
//
// The path is case-insensitive.
func (c *Config) Unmarshal(path string, target any) error {
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Metadata:         nil,
			Result:           target,
			WeaklyTypedInput: true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
				mapstructure.TextUnmarshallerHookFunc(),
			),
		},
	)
	if err != nil {
		return fmt.Errorf("[konf] new decoder: %w", err)
	}

	if err := decoder.Decode(c.sub(path)); err != nil {
		return fmt.Errorf("[konf] decode: %w", err)
	}

	return nil
}

func (c *Config) sub(path string) any {
	if path == "" {
		return c.values.values
	}

	var next any = c.values.values
	for _, key := range strings.Split(strings.ToLower(path), c.delimiter) {
		mp, ok := next.(map[string]any)
		if !ok {
			return nil
		}

		val, exist := mp[key]
		if !exist {
			return nil
		}
		next = val
	}

	return next
}

// Watch watches configuration and triggers callbacks when it changes.
// It blocks until ctx is done, or the service returns an error.
//
// It only can be called once. Call after first returns an error.
func (c *Config) Watch(ctx context.Context, fns ...func(*Config)) error { //nolint:funlen
	var first bool
	c.watchOnce.Do(func() {
		first = true
	})
	if !first {
		return errOnlyOnce
	}

	changeChan := make(chan struct{})
	defer close(changeChan)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		for {
			select {
			case <-changeChan:
				values := make(map[string]any)
				for _, w := range c.providers {
					maps.Merge(values, w.values)
				}
				c.values.values = values

				for _, fn := range fns {
					fn(c)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	var (
		firstErr error
		errOnce  sync.Once
	)
	for _, provider := range c.providers {
		if provider.watcher != nil {
			provider := provider

			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()

				if err := provider.watcher.Watch(
					ctx,
					func(values map[string]any) {
						provider.values = values
						slog.Info(
							"Configuration has been changed.",
							"watcher", provider.watcher,
						)
						changeChan <- struct{}{}
					},
				); err != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("[konf] watch configuration change: %w", err)
						cancel()
					})
				}
			}()
		}
	}
	waitGroup.Wait()

	return firstErr
}

var errOnlyOnce = errors.New("[konf] Watch only can be called once")

type provider struct {
	watcher Watcher
	values  map[string]any
}
