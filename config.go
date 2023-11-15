// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"encoding"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/ktong/konf/internal/maps"
)

// Config is a registry which holds configuration loaded by Loader(s).
type Config struct {
	decodeHook mapstructure.DecodeHookFunc
	delimiter  string
	tagName    string

	onChanges *onChanges
	values    *provider
	providers []*provider
}

type Unmarshaler interface {
	Unmarshal(path string, target any) error
}

// New returns a Config with the given Option(s).
func New(opts ...Option) (Config, error) {
	option := &options{
		Config: Config{
			delimiter: ".",
			tagName:   "konf",
			decodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				textUnmarshalerHookFunc(),
			),
		},
	}
	for _, opt := range opts {
		opt(option)
	}
	option.values = &provider{values: make(map[string]any)}
	option.providers = make([]*provider, 0, len(option.loaders))
	option.onChanges = &onChanges{onChanges: make(map[string][]func(Unmarshaler))}

	for _, loader := range option.loaders {
		if loader == nil {
			continue
		}

		values, err := loader.Load()
		if err != nil {
			return Config{}, fmt.Errorf("[konf] load configuration: %w", err)
		}
		maps.Merge(option.values.values, values)
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
		option.providers = append(option.providers, provider)
	}

	return option.Config, nil
}

// Watch watches and updates configuration when it changes.
// It blocks until ctx is done, or the service returns an error.
//
// It only can be called once. Call after first has no effects.
func (c Config) Watch(ctx context.Context) error { //nolint:cyclop,funlen
	changeChan := make(chan []func(Unmarshaler))
	defer close(changeChan)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		firstErr   error
		errOnce    sync.Once
		waitGroup  sync.WaitGroup
		hasWatcher bool
	)
	for _, p := range c.providers {
		if p.watcher != nil {
			watcher := p

			watcher.watchOnce.Do(func() {
				hasWatcher = true

				waitGroup.Add(1)
				go func() {
					defer waitGroup.Done()

					onChange := func(values map[string]any) {
						slog.Info(
							"Configuration has been changed.",
							"watcher", watcher.watcher,
						)

						// Find the onChanges should be triggered.
						oldValues := &provider{values: watcher.values}
						newValues := &provider{values: values}
						onChanges := c.onChanges.filter(func(path string) bool {
							return oldValues.sub(path, c.delimiter) != nil || newValues.sub(path, c.delimiter) != nil
						})
						watcher.values = values
						changeChan <- onChanges
					}
					if err := watcher.watcher.Watch(ctx, onChange); err != nil {
						errOnce.Do(func() {
							firstErr = fmt.Errorf("[konf] watch configuration change: %w", err)
							cancel()
						})
					}
				}()
			})
		}
	}

	if !hasWatcher {
		return nil
	}

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		for {
			select {
			case onChanges := <-changeChan:
				values := make(map[string]any)
				for _, w := range c.providers {
					maps.Merge(values, w.values)
				}
				c.values.values = values

				for _, onChange := range onChanges {
					onChange(c)
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	waitGroup.Wait()

	return firstErr
}

type provider struct {
	values    map[string]any
	watcher   Watcher
	watchOnce sync.Once
}

func (p *provider) sub(path string, delimiter string) any {
	if path == "" {
		return p.values
	}

	var next any = p.values
	for _, key := range strings.Split(strings.ToLower(path), delimiter) {
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

// OnChange executes the given onChange function while the value of any given path
// (or any value is no paths) have been changed.
//
// It requires Config.Watch has been called.
func (c Config) OnChange(onchange func(Unmarshaler), paths ...string) {
	c.onChanges.append(onchange, paths)
}

type onChanges struct {
	onChanges map[string][]func(Unmarshaler)
	mutex     sync.RWMutex
}

func (c *onChanges) append(onchange func(Unmarshaler), paths []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(paths) == 0 {
		paths = []string{""}
	}

	for _, path := range paths {
		c.onChanges[path] = append(c.onChanges[path], onchange)
	}
}

func (c *onChanges) filter(predict func(string) bool) []func(Unmarshaler) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var callbacks []func(Unmarshaler)
	for path, onChanges := range c.onChanges {
		if predict(path) {
			callbacks = append(callbacks, onChanges...)
		}
	}

	return callbacks
}

// Unmarshal loads configuration under the given path into the given object
// pointed to by target. It supports [konf] tags on struct fields for customized field name.
//
// The path is case-insensitive.
func (c Config) Unmarshal(path string, target any) error {
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Result:           target,
			WeaklyTypedInput: true,
			DecodeHook:       c.decodeHook,
			TagName:          c.tagName,
		},
	)
	if err != nil {
		return fmt.Errorf("[konf] new decoder: %w", err)
	}

	if err := decoder.Decode(c.values.sub(path, c.delimiter)); err != nil {
		return fmt.Errorf("[konf] decode: %w", err)
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
