// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package flag

import "flag"

// WithDelimiter provides the delimiter when specifying flag name with nested hierarchy.
//
// The default delimiter is `.`, which makes flag name like `parent.child.key`.
func WithDelimiter(delimiter string) Option {
	return func(flag *Flag) {
		flag.delimiter = delimiter
	}
}

// WithPrefix enables only loads flags with the given prefix.
func WithPrefix(prefix string) Option {
	return func(flag *Flag) {
		flag.prefix = prefix
	}
}

// WithFlagSet provides the flag set that loads configuration from.
//
// The default flag set is [flag.CommandLine].
func WithFlagSet(set *flag.FlagSet) Option {
	return func(flag *Flag) {
		flag.set = set
	}
}

// Option configures the give Flag.
type Option func(*Flag)
