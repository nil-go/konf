// Copyright (c) 2025 The konf authors
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

// WithNameSplitter provides the function used to split environment variable names into nested keys.
// If it returns an nil/[]string{}/[]string{""}, the variable will be ignored.
//
// For example, with the default splitter, an environment variable name like "PARENT_CHILD_KEY"
// would be split into "PARENT", "CHILD", and "KEY".
func WithNameSplitter(splitter func(string) []string) Option {
	return func(options *options) {
		options.splitter = splitter
	}
}

type (
	// Option configures an Env with specific options.
	Option  func(*options)
	options Env
)
