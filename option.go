// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

// WithLoader provides the loaders that configuration is loaded from.
//
// Each loader takes precedence over the loaders before it
// while multiple loaders are specified.
func WithLoader(loaders ...Loader) Option {
	return func(config *options) {
		config.loaders = append(config.loaders, loaders...)
	}
}

// WithDelimiter provides the delimiter when specifying config path.
//
// The default delimiter is `.`, which makes config path like `parent.child.key`.
func WithDelimiter(delimiter string) Option {
	return func(config *options) {
		config.delimiter = delimiter
	}
}

// WithLogger provides a Logger implementation to logger.
//
// The default implementation is using standard [log].
func WithLogger(logger Logger) Option {
	return func(config *options) {
		config.logger = logger
	}
}

// Option configures the given Config.
type Option func(*options)

type options struct {
	*Config

	loaders []Loader
}

func apply(opts []Option) options {
	option := &options{
		Config: &Config{
			delimiter: ".",
			logger:    stdlog{},
			values:    make(map[string]any),
		},
	}
	for _, opt := range opts {
		opt(option)
	}

	return *option
}
