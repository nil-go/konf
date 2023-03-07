// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package env

import (
	"os"
	"strings"

	"github.com/ktong/konf/internal/maps"
)

// Env is a Provider that loads configuration from environment variables.
//
// The name of environment variable is case-insensitive.
type Env struct {
	prefix    string
	delimiter string
}

// New returns an Env with the given Option(s).
func New(opts ...Option) Env {
	env := Env{
		delimiter: "_",
	}
	for _, opt := range opts {
		opt(&env)
	}

	return env
}

func (e Env) Load() (map[string]any, error) {
	config := make(map[string]any)
	for _, env := range os.Environ() {
		if e.prefix == "" || strings.HasPrefix(env, e.prefix) {
			parts := strings.SplitN(env, "=", 2) //nolint:gomnd
			maps.Insert(config, strings.Split(strings.ToLower(parts[0]), e.delimiter), parts[1])
		}
	}

	return config, nil
}
