// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

// Loader is the interface that wraps the basic Load method.
//
// Load loads configuration and returns as a nested map[string]any.
// It requires that the string keys should be nested like `{parent: {child: {key: 1}}}`.
type Loader interface {
	Load() (map[string]any, error)
}
