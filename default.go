// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"log/slog"
	"reflect"
	"sync/atomic"

	"github.com/ktong/konf/provider/env"
)

// Get returns the value under the given path.
// It returns zero value if there is an error.
// The path is case-insensitive.
func Get[T any](path string) T { //nolint:ireturn
	var value T
	if err := Unmarshal(path, &value); err != nil {
		slog.Error(
			"Could not read config, return empty value instead.",
			"error", err,
			"path", path,
			"type", reflect.TypeOf(value),
		)
	}

	return value
}

// Unmarshal reads configuration under the given path
// into the given object pointed to by target.
// The path is case-insensitive.
func Unmarshal(path string, target any) error {
	return defaultConfig.Load().Unmarshal(path, target)
}

// OnChange executes the given onChange function
// while the value of any given path have been changed.
// It requires Config.Watch has been called first.
// The paths are case-insensitive.
//
// This method is concurrency-safe.
func OnChange(onChange func(), paths ...string) {
	defaultConfig.Load().OnChange(func(*Config) { onChange() }, paths...)
}

// SetDefault makes c the default [Config].
// After this call, the konf package's top functions (e.g. konf.Get)
// will read from the default config.
func SetDefault(c *Config) {
	defaultConfig.Store(c)
}

var defaultConfig atomic.Pointer[Config] //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	config := New()
	// Ignore error as env loader does not return error.
	_ = config.Load(env.New())
	defaultConfig.Store(config)
}
