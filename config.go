// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/ktong/konf/internal/maps"
)

// Config is a registry which holds configuration loaded by Loader(s).
type Config struct {
	delimiter string
	logger    Logger
	loaders   []Loader

	values map[string]any
}

// New returns a Config with the given Option(s).
func New(opts ...Option) (Config, error) {
	config := &Config{
		delimiter: ".",
		logger:    stdlog{},
		values:    make(map[string]any),
	}
	for _, opt := range opts {
		opt(config)
	}

	for _, loader := range config.loaders {
		if err := config.load(loader); err != nil {
			return Config{}, err
		}
	}

	return *config, nil
}

func (c Config) load(loader Loader) error {
	if loader == nil {
		return nil
	}

	values, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	maps.Merge(c.values, values)
	c.logger.Info(
		"Loaded configuration.",
		"loader", loader,
	)

	return nil
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
		return c.values
	}

	var next any = c.values
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
