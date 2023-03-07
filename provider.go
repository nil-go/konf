// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"fmt"

	"github.com/ktong/konf/internal/maps"
)

type Loader interface {
	Load() (map[string]any, error)
}

// Load is not thread safe.
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
