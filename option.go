// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

// Option configures how it loads configuration.
type Option func(*Config)

func WithDelimiter(delimiter string) Option {
	return func(config *Config) {
		config.delimiter = delimiter
	}
}

func WithLogger(logger Logger) Option {
	return func(config *Config) {
		config.logger = logger
	}
}
