// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import "context"

// Loader is the interface that wraps the Load method.
//
// Load loads latest configuration and returns as a nested map[string]any.
// It requires that the string keys should be nested like `{parent: {child: {key: 1}}}`.
// The key in returned map should be case-insensitive, otherwise random overridden exists.
type Loader interface {
	Load() (map[string]any, error)
}

// Watcher is the interface that wraps the Watch method.
//
// Watch watches configuration and triggers onChange callback with latest
// full configurations as a nested map[string]any when it changes.
// It blocks until ctx is done, or the watching returns an error.
type Watcher interface {
	Watch(ctx context.Context, onChange func(map[string]any)) error
}
