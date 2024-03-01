// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"log/slog"
	"reflect"
	"sync/atomic"

	"github.com/nil-go/konf/provider/env"
)

// Get retrieves the value under the given path from the default Config.
// It returns the zero value of the expected type if there is an error.
// The path is case-insensitive.
func Get[T any](path string) T { //nolint:ireturn
	var value T
	if err := Unmarshal(path, &value); err != nil {
		logger := defaultConfig.Load().logger
		if logger == nil {
			logger = slog.Default()
		}

		logger.LogAttrs(
			context.Background(), slog.LevelWarn,
			"Could not read config, return empty value instead.",
			slog.String("path", path),
			slog.Any("type", reflect.TypeOf(value)),
			slog.Any("error", err),
		)
	}

	return value
}

// Unmarshal reads configuration under the given path from the default Config
// and decodes it into the given object pointed to by target.
// The path is case-insensitive.
func Unmarshal(path string, target any) error {
	return defaultConfig.Load().Unmarshal(path, target)
}

// OnChange registers a callback function that is executed
// when the value of any given path in the default Config changes.
// The paths are case-insensitive.
//
// The register function must be non-blocking and usually completes instantly.
// If it requires a long time to complete, it should be executed in a separate goroutine.
//
// This method is concurrency-safe.
func OnChange(onChange func(), paths ...string) {
	defaultConfig.Load().OnChange(func(*Config) { onChange() }, paths...)
}

// Explain provides information about how default Config resolve each value
// from loaders for the given path. It blur sensitive information.
// The path is case-insensitive.
func Explain(path string) string {
	return defaultConfig.Load().Explain(path)
}

// SetDefault sets the given Config as the default Config.
// After this call, the konf package's top functions (e.g. konf.Get)
// will interact with the given Config.
func SetDefault(config *Config) {
	if config != nil {
		defaultConfig.Store(config)
	}
}

var defaultConfig atomic.Pointer[Config] //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	var config Config
	// Ignore error as env loader does not return error.
	_ = config.Load(env.Env{})
	defaultConfig.Store(&config)
}
