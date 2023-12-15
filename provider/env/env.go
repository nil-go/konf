// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package env loads configuration from environment variables.
//
// Env loads environment variables whose names starts with the given prefix
// and returns them as a nested map[string]any.
// Environment variables with empty values are treated as unset.
//
// It splits the names by delimiter. For example, with the default delimiter "_",
// the environment variable `PARENT_CHILD_KEY="1"` is loaded as `{PARENT: {CHILD: {KEY: "1"}}}`.
package env

import (
	"os"
	"strings"

	"github.com/ktong/konf/internal/maps"
)

// Env is a Provider that loads configuration from environment variables.
//
// To create a new Env, call New.
type Env struct {
	_         [0]func() // Ensure it's incomparable.
	prefix    string
	delimiter string
}

// New creates an Env with the given Option(s).
func New(opts ...Option) Env {
	option := &options{
		delimiter: "_",
	}
	for _, opt := range opts {
		opt(option)
	}

	return Env(*option)
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
