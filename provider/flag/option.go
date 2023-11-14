// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package flag

import "flag"

// WithPrefix enables only loads flags with the given prefix in the name.
//
// E.g. if the given prefix is "server", it only loads flags
// which name starts with "server".
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

// WithDelimiter provides the delimiter when splitting flag name to nested keys.
//
// The default delimiter is `_`, which loads the flag `parent.child.key` with value 1
// as `{parent: {child: {key: 1}}}`.
func WithDelimiter(delimiter string) Option {
	return func(options *options) {
		options.delimiter = delimiter
	}
}

// Option configures the give Flag.
type Option func(*options)

type options Flag
