// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/go-viper/mapstructure/v2"

	"github.com/nil-go/konf/internal/maps"
)

// Config reads configuration from appropriate sources.
//
// To create a new Config, call [New].
type Config struct {
	decodeHook mapstructure.DecodeHookFunc
	delimiter  string
	tagName    string

	values    map[string]any
	providers []*provider

	onChanges      map[string][]func(*Config)
	onChangesMutex sync.RWMutex
	watchOnce      sync.Once
}

type provider struct {
	loader Loader
	values map[string]any
}

// New creates a new Config with the given Option(s).
func New(opts ...Option) *Config {
	option := &options{
		delimiter: ".",
		tagName:   "konf",
		decodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.TextUnmarshallerHookFunc(),
		),
		values:    make(map[string]any),
		onChanges: make(map[string][]func(*Config)),
	}
	for _, opt := range opts {
		opt(option)
	}

	return (*Config)(option)
}

// Load loads configuration from the given loaders.
// Each loader takes precedence over the loaders before it.
//
// This method can be called multiple times but it is not concurrency-safe.
func (c *Config) Load(loaders ...Loader) error {
	for _, loader := range loaders {
		if loader == nil {
			continue
		}

		values, err := loader.Load()
		if err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		maps.Merge(c.values, values)

		// Merged to empty map to convert to lower case.
		provider := &provider{
			loader: loader,
			values: make(map[string]any),
		}
		maps.Merge(provider.values, values)
		c.providers = append(c.providers, provider)

		slog.Info(
			"Configuration has been loaded.",
			"loader", loader,
		)
	}

	return nil
}

// Watch watches and updates configuration when it changes.
// It blocks until ctx is done, or the service returns an error.
// WARNING: All loaders passed in Load after calling Watch do not get watched.
//
// It only can be called once. Call after first has no effects.
func (c *Config) Watch(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
	initialized := true
	c.watchOnce.Do(func() {
		initialized = false
	})
	if initialized {
		return nil
	}

	onChangesChannel := make(chan []func(*Config))
	defer close(onChangesChannel)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		for {
			select {
			case onChanges := <-onChangesChannel:
				values := make(map[string]any)
				for _, w := range c.providers {
					maps.Merge(values, w.values)
				}
				c.values = values

				for _, onChange := range onChanges {
					onChange(c)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	errChan := make(chan error, len(c.providers))
	for _, provider := range c.providers {
		if _, ok := provider.loader.(Watcher); ok {
			provider := provider

			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()

				onChange := func(values map[string]any) {
					// Merged to empty map to convert to lower case.
					newValues := make(map[string]any)
					maps.Merge(newValues, values)

					oldValues := provider.values
					provider.values = newValues

					// Find the onChanges should be triggered.
					onChanges := func() []func(*Config) {
						c.onChangesMutex.RLock()
						defer c.onChangesMutex.RUnlock()

						var callbacks []func(*Config)
						for path, onChanges := range c.onChanges {
							if sub(oldValues, path, c.delimiter) != nil || sub(newValues, path, c.delimiter) != nil {
								callbacks = append(callbacks, onChanges...)
							}
						}

						return callbacks
					}
					onChangesChannel <- onChanges()

					slog.Info(
						"Configuration has been changed.",
						"provider", provider.loader,
					)
				}
				if watcher, ok := provider.loader.(Watcher); ok {
					if err := watcher.Watch(ctx, onChange); err != nil {
						errChan <- fmt.Errorf("watch configuration change: %w", err)
						cancel()
					}
				}
			}()
		}
	}
	waitGroup.Wait()
	close(errChan)

	var err error
	for e := range errChan {
		err = errors.Join(e)
	}

	return err
}

func sub(values map[string]any, path string, delimiter string) any {
	if path == "" {
		return values
	}

	var next any = values
	for _, key := range strings.Split(path, delimiter) {
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

// OnChange registers a callback function that is executed
// when the value of any given path in the Config changes.
// It requires Config.Watch has been called first.
// The paths are case-insensitive.
//
// This method is concurrency-safe.
func (c *Config) OnChange(onchange func(*Config), paths ...string) {
	c.onChangesMutex.Lock()
	defer c.onChangesMutex.Unlock()

	if len(paths) == 0 {
		paths = []string{""}
	}

	for _, path := range paths {
		path = strings.ToLower(path)
		c.onChanges[path] = append(c.onChanges[path], onchange)
	}
}

// Unmarshal reads configuration under the given path from the Config
// and decodes it into the given object pointed to by target.
// The path is case-insensitive.
func (c *Config) Unmarshal(path string, target any) error {
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Result:           target,
			WeaklyTypedInput: true,
			DecodeHook:       c.decodeHook,
			TagName:          c.tagName,
		},
	)
	if err != nil {
		return fmt.Errorf("new decoder: %w", err)
	}

	if err := decoder.Decode(sub(c.values, strings.ToLower(path), c.delimiter)); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

// Explain provides information about how Config resolve each value  from loaders for the given path.
// The path is case-insensitive.
func (c *Config) Explain(path string) string {
	explanation := &strings.Builder{}
	c.explain(explanation, path, sub(c.values, strings.ToLower(path), c.delimiter))

	return explanation.String()
}

func (c *Config) explain(explanation *strings.Builder, path string, value any) {
	if values, ok := value.(map[string]any); ok {
		for k, v := range values {
			c.explain(explanation, path+c.delimiter+k, v)
		}
	}

	var (
		loader  string
		loaders []string
	)
	for _, provider := range c.providers {
		if v := sub(provider.values, path, c.delimiter); v != nil {
			loaders = append(loaders, fmt.Sprintf("%v[%v]", v, provider.loader))
			loader = fmt.Sprintf("%v", provider.loader)
		}
	}
	slices.Reverse(loaders)

	if loader == "" {
		_, _ = fmt.Fprintf(explanation, "%s has no configuration", path)

		return
	}
	_, _ = fmt.Fprintf(explanation, "%s has value [%v] is loaded by %v.\n", path, value, loader)
	if len(loaders) > 1 {
		_, _ = fmt.Fprintf(explanation, "Here are other value[loader]s: %v\n", loaders[1:])
	}
	explanation.WriteString("\n")
}
