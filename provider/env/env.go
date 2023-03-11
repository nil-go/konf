// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package env loads configuration from environment variables.
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
	_         [0]func() // Ensure it's incomparable.
	prefix    string
	delimiter string
}

// New returns an Env with the given Option(s).
func New(opts ...Option) Env {
	return Env(apply(opts))
}

func (e Env) Load() (map[string]any, error) {
	config := make(map[string]any)
	for _, env := range os.Environ() {
		if e.prefix == "" || strings.HasPrefix(env, e.prefix) {
			key, value, _ := strings.Cut(env, "=")
			maps.Insert(config, strings.Split(strings.ToLower(key), e.delimiter), value)
		}
	}

	return config, nil
}

func (e Env) String() string {
	if e.prefix == "" {
		return "env"
	}

	return "env:" + e.prefix
}
