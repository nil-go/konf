// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import "context"

// Loader is the interface that wraps the Load method.
//
// Load loads configuration and returns as a nested map[string]any.
// It requires that the string keys should be nested like `{parent: {child: {key: 1}}}`.
// The key in returned map should be case-insensitive,
// otherwise random overridden exists.
type Loader interface {
	Load() (map[string]any, error)
}

// Watcher is the interface that wraps the Watch method.
//
// Watch watches configuration and triggers a callback with full new configurations
// as a nested map[string]any when it changes.
// It blocks until ctx is done, or the service returns an error.
type Watcher interface {
	Watch(context.Context, func(map[string]any)) error
}

// ConfigAware is the interface that wraps the WithConfig method.
//
// WithConfig enables provider loads configuration from providers
// before it in Load and Watch methods.
//
// It ensures the WithConfig is called before Load and Watch.
type ConfigAware interface {
	WithConfig(*Config)
}
