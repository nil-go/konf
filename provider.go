// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"fmt"

	"github.com/ktong/konf/internal/maps"
)

// Loader is the interface that wraps the basic Load method.
//
// Load loads configuration and returns as a nested map[string]any.
// It requires that the string keys should be nested like `{parent: {child: {key: 1}}}`.
type Loader interface {
	Load() (map[string]any, error)
}

// Load uses the given Loader to load configuration.
//
// It could not be used in multiple goroutines as it's not thread safe.
func (c Config) Load(loader Loader) error {
	if loader == nil {
		return nil
	}

	values, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	maps.Merge(c.values, values)
	c.logger.Info(
		"Loaded configuration.",
		"loader", loader,
	)

	return nil
}
