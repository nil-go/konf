// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/sync/errgroup"

	"github.com/ktong/konf/internal/maps"
)

// Config is a registry which holds configuration loaded by Loader(s).
type Config struct {
	delimiter string
	logger    Logger

	values   *map[string]any // Use pointer of map for switching while configuration changes.
	watchers []*watcher
}

type watcher struct {
	watcher Watcher
	values  map[string]any
}

// New returns a Config with the given Option(s).
func New(opts ...Option) (Config, error) {
	option := apply(opts)
	config := option.Config
	config.watchers = make([]*watcher, 0, len(option.loaders))

	for _, loader := range option.loaders {
		if loader == nil {
			continue
		}

		values, err := loader.Load()
		if err != nil {
			return Config{}, fmt.Errorf("[konf] load configuration: %w", err)
		}
		maps.Merge(*config.values, values)
		config.logger.Info(
			"Loaded configuration.",
			"loader", loader,
		)

		provider := &watcher{
			values: values,
		}
		if w, ok := loader.(Watcher); ok {
			provider.watcher = w
		}
		config.watchers = append(config.watchers, provider)
	}

	return config, nil
}

// Unmarshal loads configuration under the given path into the given object pointed to by target.
// It supports [mapstructure] tags.
func (c Config) Unmarshal(path string, target any) error {
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

	if err = decoder.Decode(c.sub(path)); err != nil {
		return fmt.Errorf("[konf] decode: %w", err)
	}

	return nil
}

func (c Config) sub(path string) any {
	if path == "" {
		return *c.values
	}

	var next any = *c.values
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
// It blocks until ctx is done, or the service returns a non-retryable error.
func (c Config) Watch(ctx context.Context, fns ...func(Config)) error { //nolint:gocognit
	changeChan := make(chan struct{})
	defer close(changeChan)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-changeChan:
				values := make(map[string]any)
				for _, w := range c.watchers {
					maps.Merge(values, w.values)
				}
				*c.values = values

				for _, fn := range fns {
					fn(c)
				}
			}
		}
	})

	for _, watcher := range c.watchers {
		watcher := watcher
		if watcher.watcher != nil {
			group.Go(func() error {
				if err := watcher.watcher.Watch(
					ctx,
					func(values map[string]any) {
						watcher.values = values
						c.logger.Info(
							"Configuration has been changed.",
							"loader", watcher.watcher,
						)
						changeChan <- struct{}{}
					},
				); err != nil {
					return fmt.Errorf("[konf] watch configuration change: %w", err)
				}

				return nil
			})
		}
	}

	return group.Wait() //nolint:wrapcheck
}
