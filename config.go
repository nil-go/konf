// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// Config is a registry which holds configuration loaded by Loader(s).
type Config struct {
	delimiter string
	logger    Logger

	values map[string]any
}

// New initializes a Config with the given Option(s).
func New(opts ...Option) Config {
	config := &Config{
		delimiter: ".",
		logger:    stdlog{},
		values:    make(map[string]any),
	}
	for _, opt := range opts {
		opt(config)
	}

	return *config
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
