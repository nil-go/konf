// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
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
		slog.Error(
			"Could not read config, return empty value instead.",
			"error", err,
			"path", path,
			"type", reflect.TypeOf(value),
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

// Explain provides information about how Config resolve each value
// from loaders for the given path.
// The path is case-insensitive.
//
// If there are sensitive information (e.g. password, secret) which should not be exposed,
// you can use [WithValueFormatter] to pass a value formatter to blur the information.
func Explain(path string, opts ...ExplainOption) string {
	return defaultConfig.Load().Explain(path, opts...)
}

// OnChange registers a callback function that is executed
// when the value of any given path in the default Config changes.
// The paths are case-insensitive.
//
// The onChange function must be non-blocking and usually completes instantly.
// If it requires a long time to complete, it should be executed in a separate goroutine.
//
// This method is concurrency-safe.
// It panics if onChange is nil.
func OnChange(onChange func(), paths ...string) {
	defaultConfig.Load().OnChange(func(*Config) { onChange() }, paths...)
}

// SetDefault sets the given Config as the default Config.
// After this call, the konf package's top functions (e.g. konf.Get)
// will interact with the given Config.
//
// It panics if config is nil.
func SetDefault(config *Config) {
	if config == nil {
		panic("cannot set default with nil config")
	}

	defaultConfig.Store(config)
}

var defaultConfig atomic.Pointer[Config] //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	config := New()
	// Ignore error as env loader does not return error.
	_ = config.Load(env.New())
	defaultConfig.Store(config)
}
