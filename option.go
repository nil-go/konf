// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

// Option configures the given Config.
type Option func(*Config)

// WithLoader provides the loaders that configuration is loaded from.
//
// Each loader takes precedence over the loaders before it
// while multiple loader are specified.
func WithLoader(loaders ...Loader) Option {
	return func(config *Config) {
		config.loaders = append(config.loaders, loaders...)
	}
}

// WithDelimiter provides the delimiter when specifying config path.
//
// The default delimiter is `.`, which makes config path like `parent.child.key`.
func WithDelimiter(delimiter string) Option {
	return func(config *Config) {
		config.delimiter = delimiter
	}
}

// WithLogger provides a Logger implementation to logger.
//
// The default implementation is using standard [log].
func WithLogger(logger Logger) Option {
	return func(config *Config) {
		config.logger = logger
	}
}
