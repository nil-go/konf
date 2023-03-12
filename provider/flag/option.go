// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package flag

import "flag"

// WithDelimiter provides the delimiter when specifying flag name with nested hierarchy.
//
// The default delimiter is `.`, which makes flag name like `parent.child.key`.
func WithDelimiter(delimiter string) Option {
	return func(flag *options) {
		flag.delimiter = delimiter
	}
}

// WithPrefix enables only loads flags with the given prefix.
//
// E.g. if the given prefix is "server", it only loads flags
// which name starts with "server".
func WithPrefix(prefix string) Option {
	return func(flag *options) {
		flag.prefix = prefix
	}
}

// WithFlagSet provides the flag set that loads configuration from.
//
// The default flag set is [flag.CommandLine].
func WithFlagSet(set *flag.FlagSet) Option {
	return func(flag *options) {
		flag.set = set
	}
}

// Option configures the give Flag.
type Option func(*options)

type options Flag

func apply(opts []Option) options {
	option := &options{
		delimiter: ".",
	}
	for _, opt := range opts {
		opt(option)
	}

	return *option
}
