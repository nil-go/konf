// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import "context"

// Loader is the interface that wraps the Load method.
//
// Load loads the latest configuration and returns it as a nested map[string]any.
// The keys in the returned map should be case-insensitive to avoid random overriding.
// The keys should be nested like `{parent: {child: {key: 1}}}`.
type Loader interface {
	Load() (map[string]any, error)
}

// Watcher is the interface that wraps the Watch method.
//
// Watch watches the configuration and triggers the onChange callback with the latest
// full configurations as a nested map[string]any when it changes.
// It blocks until ctx is done, or the watching returns an error.
type Watcher interface {
	Watch(ctx context.Context, onChange func(map[string]any)) error
}
