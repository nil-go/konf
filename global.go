// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"log/slog"
	"reflect"

	"github.com/ktong/konf/provider/env"
)

// Get retrieves the value given the path to use.
// It returns zero value if there is an error while getting configuration.
//
// The path is case-insensitive.
func Get[T any](path string) T { //nolint:ireturn
	var value T
	if err := global.Unmarshal(path, &value); err != nil {
		slog.Error(
			"Could not read config, return empty value instead.",
			"error", err,
			"path", path,
			"type", reflect.TypeOf(value),
		)

		return *new(T)
	}

	return value
}

// Unmarshal loads configuration under the given path into the given object
// pointed to by target. It supports [konf] tags on struct fields for customized field name.
//
// The path is case-insensitive.
func Unmarshal(path string, target any) error {
	return global.Unmarshal(path, target)
}

// OnChange executes the given onChange function while the value of any given path
// (or any value is no paths) have been changed.
//
// It requires Watch has been called.
func OnChange(onChange func(), paths ...string) {
	global.OnChange(func(Unmarshaler) { onChange() }, paths...)
}

// SetGlobal makes config as the global Config. After this call,
// the konf package's functions (e.g. konf.Get) will read from the global config.
// This method is not concurrency-safe.
//
// The default global config only loads configuration from environment variables.
func SetGlobal(config Config) {
	global = config
}

var global, _ = New(WithLoader(env.New())) //nolint:gochecknoglobals
