// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import "reflect"

var global = New() //nolint:gochecknoglobals

func Unmarshal(path string, target any) error {
	return global.Unmarshal(path, target)
}

func Get[T any](path string) T { //nolint:ireturn
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

// SetGlobal makes c the global Config. After this call,
// the konf package's functions (e.g. konf.Get) will read from c.
func SetGlobal(c Config) {
	global = c
}
