// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package pflag

import "github.com/spf13/pflag"

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

// WithFlagSet provides the [pflag.FlagSet] that loads configuration from.
//
// The default flag set is [pflag.CommandLine] plus [flag.CommandLine].
func WithFlagSet(set *pflag.FlagSet) Option {
	return func(options *options) {
		options.set = set
	}
}

// WithDelimiter provides the delimiter used when splitting flag names into nested keys.
//
// For example, with the default delimiter ".", an flag name like "parent.child.key"
// would be split into "parent", "child", and "key".
func WithDelimiter(delimiter string) Option {
	return func(options *options) {
		options.delimiter = delimiter
	}
}

// Option configures the a PFlag with specific options.
type Option func(*options)

type options PFlag
