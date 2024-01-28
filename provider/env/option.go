// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package env

// WithPrefix provides the prefix used when loading environment variables.
// Only environment variables with names that start with the prefix will be loaded.
//
// For example, if the prefix is "server", only environment variables whose names start with "server" will be loaded.
// By default, it has no prefix which loads all environment variables.
func WithPrefix(prefix string) Option {
	return func(options *options) {
		options.prefix = prefix
	}
}

// WithDelimiter provides the delimiter used when splitting environment variable names into nested keys.
//
// For example, with the default delimiter "_", an environment variable name like "PARENT_CHILD_KEY"
// would be split into "PARENT", "CHILD", and "KEY".
func WithDelimiter(delimiter string) Option {
	return func(options *options) {
		options.delimiter = delimiter
	}
}

type (
	// Option configures an Env with specific options.
	Option  func(*options)
	options Env
)
