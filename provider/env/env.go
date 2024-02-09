// Copyright (c) 2024 The konf authors
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

	"github.com/nil-go/konf/internal/maps"
)

// Env is a Provider that loads configuration from environment variables.
//
// To create a new Env, call New.
type Env struct {
	_        [0]func() // Ensure it's incomparable.
	prefix   string
	splitter func(string) []string
}

// New creates an Env with the given Option(s).
func New(opts ...Option) Env {
	option := &options{}
	for _, opt := range opts {
		opt(option)
	}
	if option.splitter == nil {
		option.splitter = func(s string) []string { return strings.Split(s, "_") }
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
			maps.Insert(values, e.splitter(key), value)
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
