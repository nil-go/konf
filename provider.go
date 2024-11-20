// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"strings"
)

// Loader is the interface that wraps the Load method.
//
// Load loads the latest configuration and returns it as a nested map[string]any.
// The keys should be nested like `{parent: {child: {key: 1}}}`.
type Loader interface {
	Load() (map[string]any, error)
}

// Watcher is the interface that wraps the Watch method.
//
// Watch watches the configuration and triggers the register callback with the latest
// full configurations as a nested map[string]any when it changes.
// It blocks until ctx is done, or the watching returns an error.
type Watcher interface {
	Watch(ctx context.Context, onChange func(map[string]any)) error
}

// Statuser is the interface that wraps the Status method.
//
// Status enables providers report the status of configuration watching.
type Statuser interface {
	Status(onStatus func(changed bool, err error))
}

// Exists tests if the given path exist in the configuration.
//
// It's used by the loader to check if the configuration has been set by other loaders.
func (c *Config) Exists(path []string) bool {
	if c == nil { // To support nil
		return false
	}
	c.nocopy.Check()

	value := c.providers.value()
	if value == nil {
		return false // To support zero Config
	}

	return c.sub(value, strings.Join(path, c.delim())) != nil
}
