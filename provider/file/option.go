// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file

import "log/slog"

// WithUnmarshal provides the function used to parses the configuration file.
// The unmarshal function must be able to unmarshal the file content into a map[string]any.
//
// The default function is json.Unmarshal.
func WithUnmarshal(unmarshal func([]byte, any) error) Option {
	return func(options *options) {
		options.unmarshal = unmarshal
	}
}

// IgnoreFileNotExit ignores the error and return an empty map instead if the configuration file is not found.
func IgnoreFileNotExit() Option {
	return func(options *options) {
		options.ignoreNotExist = true
	}
}

// WithLogger provides the slog.Logger for File loader.
//
// By default, it uses slog.Default().
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

type (
	// Option configures the a File with specific options.
	Option  func(options *options)
	options File
)
