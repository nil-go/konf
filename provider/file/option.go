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

// WithLogHandler provides the slog.Handler for logs from watch.
//
// By default, it uses handler from slog.Default().
func WithLogHandler(handler slog.Handler) Option {
	return func(options *options) {
		if handler != nil {
			options.logger = slog.New(handler)
		}
	}
}

type (
	// Option configures the a File with specific options.
	Option  func(options *options)
	options File
)
