// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package env

// WithPrefix enables loads environment variables with the given prefix in the name.
//
// E.g. if the given prefix is "server", it only loads environment variables
// which name starts with "server".
func WithPrefix(prefix string) Option {
	return func(env *options) {
		env.prefix = prefix
	}
}

// WithDelimiter provides the delimiter when splitting environment variable name to nested keys.
//
// The default delimiter is `_`, which loads the environment variable `PARENT_CHILD_KEY="1"`
// as `{PARENT: {CHILD: {KEY: "1"}}}`.
func WithDelimiter(delimiter string) Option {
	return func(env *options) {
		env.delimiter = delimiter
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
