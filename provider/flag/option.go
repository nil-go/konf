// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package flag

import (
	"flag"
)

// WithPrefix provides the prefix used when loading flags.
// Only flags with names that start with the prefix will be loaded.
//
// For example, if the prefix is "server", only flags whose names start with "server" will be loaded.
// By default, it has no prefix which loads all flags.
func WithPrefix(prefix string) Option {
	return func(options *options) {
		options.prefix = prefix
	}
}

// WithFlagSet provides the [flag.FlagSet] that loads configuration from.
//
// The default flag set is [flag.CommandLine].
func WithFlagSet(set *flag.FlagSet) Option {
	return func(options *options) {
		options.set = set
	}
}

// WithNameSplitter provides the function used to split flag names into nested keys.
// If it returns an nil/[]string{}/[]string{""}, the variable will be ignored.
//
// For example, with the default splitter, an flag name like "parent.child.key"
// would be split into "parent", "child", and "key".
func WithNameSplitter(splitter func(string) []string) Option {
	return func(options *options) {
		options.splitter = splitter
	}
}

type (
	// Option configures the a Flag with specific options.
	Option  func(*options)
	options Flag
)
