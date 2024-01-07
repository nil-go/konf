// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"encoding"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/ktong/konf/internal/maps"
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
	values  map[string]any
	watcher Watcher
}

// New creates a new Config with the given Option(s).
func New(opts ...Option) *Config {
	option := &options{
		delimiter: ".",
		tagName:   "konf",
		decodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			textUnmarshalerHookFunc(),
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
			values: make(map[string]any),
		}
		maps.Merge(provider.values, values)
		if w, ok := loader.(Watcher); ok {
			provider.watcher = w
		}
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
		if provider.watcher != nil {
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
						"provider", provider.watcher,
					)
				}
				if err := provider.watcher.Watch(ctx, onChange); err != nil {
					errChan <- fmt.Errorf("watch configuration change: %w", err)
					cancel()
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

// textUnmarshalerHookFunc is a fixed version of mapstructure.TextUnmarshallerHookFunc.
// This hook allows to additionally unmarshal text into custom string types
// that implement the encoding.Text(Un)Marshaler interface(s).
//
//nolint:wrapcheck
func textUnmarshalerHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		from reflect.Type,
		to reflect.Type, //nolint:varnamelen
		data interface{},
	) (interface{}, error) {
		if from.Kind() != reflect.String {
			return data, nil
		}
		result := reflect.New(to).Interface()
		unmarshaller, ok := result.(encoding.TextUnmarshaler)
		if !ok {
			return data, nil
		}

		// default text representation is the actual value of the `from` string
		var (
			dataVal = reflect.ValueOf(data)
			text    = []byte(dataVal.String())
		)
		if from.Kind() == to.Kind() { //nolint:nestif
			// source and target are of underlying type string
			var (
				err    error
				ptrVal = reflect.New(dataVal.Type())
			)
			if !ptrVal.Elem().CanSet() {
				// cannot set, skip, this should not happen
				if err := unmarshaller.UnmarshalText(text); err != nil {
					return nil, err
				}

				return result, nil
			}
			ptrVal.Elem().Set(dataVal)

			// We need to assert that both, the value type and the pointer type
			// do (not) implement the TextMarshaller interface before proceeding and simply
			// using the string value of the string type.
			// it might be the case that the internal string representation differs from
			// the (un)marshaled string.
			for _, v := range []reflect.Value{dataVal, ptrVal} {
				if marshaller, ok := v.Interface().(encoding.TextMarshaler); ok {
					text, err = marshaller.MarshalText()
					if err != nil {
						return nil, err
					}

					break
				}
			}
		}

		// text is either the source string's value or the source string type's marshaled value
		// which may differ from its internal string value.
		if err := unmarshaller.UnmarshalText(text); err != nil {
			return nil, err
		}

		return result, nil
	}
}
