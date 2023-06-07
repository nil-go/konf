// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"reflect"
	"sync"

	"github.com/ktong/konf/provider/env"
)

// Get retrieves the value given the path to use.
// It returns zero value if there is an error while getting configuration.
//
// The path is case-insensitive.
func Get[T any](path string) T { //nolint:ireturn
	mux.RLock()
	defer mux.RUnlock()

	var value T
	if err := global.Unmarshal(path, &value); err != nil {
		global.logger.Error(
			"Could not read config, return empty value instead.",
			err,
			"path", path,
			"type", reflect.TypeOf(value),
		)

		return *new(T)
	}

	return value
}

// Unmarshal loads configuration under the given path into the given object
// pointed to by target. It supports [mapstructure] tags on struct fields.
//
// The path is case-insensitive.
func Unmarshal(path string, target any) error {
	mux.RLock()
	defer mux.RUnlock()

	return global.Unmarshal(path, target)
}

// Watch watches configuration and triggers callbacks when it changes.
// It blocks until ctx is done, or the service returns an error.
//
// It only can be called once. Call after first returns an error.
func Watch(ctx context.Context, fns ...func()) error {
	mux.RLock()
	defer mux.RUnlock()

	return global.Watch(
		ctx,
		func(*Config) {
			for _, fn := range fns {
				fn()
			}
		},
	)
}

// SetGlobal makes c the global Config. After this call,
// the konf package's functions (e.g. konf.Get) will read from the global config.
//
// The default global config only loads configuration from environment variables.
func SetGlobal(config *Config) {
	mux.Lock()
	defer mux.Unlock()

	global = config
}

//nolint:gochecknoglobals
var (
	global, _ = New(WithLoader(env.New()))
	mux       sync.RWMutex
)
