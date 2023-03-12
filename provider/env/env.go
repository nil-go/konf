// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package env loads configuration from environment variables.
//
// Env loads all environment variables and returns nested map[string]any.
// by splitting the names by `_`. E.g. the environment variable
// `PARENT_CHILD_KEY="1"` is loaded as `{PARENT: {CHILD: {KEY: "1"}}}`.
// The environment variables with empty value are treated as unset.
//
// The default behavior can be changed with following options:
//   - WithPrefix enables loads environment variables with the given prefix in the name.
//   - WithDelimiter provides the delimiter when splitting environment variable name to nested keys.
package env

import (
	"os"
	"strings"

	"github.com/ktong/konf/internal/maps"
)

// Env is a Provider that loads configuration from environment variables.
type Env struct {
	_         [0]func() // Ensure it's incomparable.
	prefix    string
	delimiter string
}

// New returns an Env with the given Option(s).
func New(opts ...Option) Env {
	return Env(apply(opts))
}

func (e Env) Load() (map[string]any, error) {
	values := make(map[string]any)
	for _, env := range os.Environ() {
		if e.prefix == "" || strings.HasPrefix(env, e.prefix) {
			key, value, _ := strings.Cut(env, "=")
			if value == "" {
				// The environment variable with empty value is treated as unset.
				continue
			}
			maps.Insert(values, strings.Split(key, e.delimiter), value)
		}
	}

	return values, nil
}

func (e Env) String() string {
	if e.prefix == "" {
		return "env"
	}

	return "env:" + e.prefix
}
