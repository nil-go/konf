// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package env

// WithDelimiter provides the delimiter when specifying environment variable name with nested hierarchy.
//
// The default delimiter is `_`, which makes environment variable name like `PARENT_CHILD_KEY`.
func WithDelimiter(delimiter string) Option {
	return func(env *options) {
		env.delimiter = delimiter
	}
}

// WithPrefix enables only loads environment variables with the given prefix.
func WithPrefix(prefix string) Option {
	return func(env *options) {
		env.prefix = prefix
	}
}

// Option configures the given Env.
type Option func(*options)

type options Env

func apply(opts []Option) options {
	option := &options{
		delimiter: "_",
	}
	for _, opt := range opts {
		opt(option)
	}

	return *option
}
