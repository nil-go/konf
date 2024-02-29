// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import "log/slog"

// WithDelimiter provides the delimiter used when specifying config paths.
// The delimiter is used to separate keys in the path.
//
// For example, with the default delimiter `.`, a config path might look like `parent.child.key`.
func WithDelimiter(delimiter string) Option {
	return func(options *options) {
		options.delimiter = delimiter
	}
}

// WithTagName provides the tag name that reads for field names.
// The tag name is used when decoding configuration into structs.
//
// For example, with the default tag name `konf`, it would look for `konf` tags on struct fields.
func WithTagName(tagName string) Option {
	return func(options *options) {
		options.tagName = tagName
	}
}

// WithDecodeHook provides the decode hook for decoding.
// The decode hook is a function that can transform or customize how values are decoded.
//
// By default, it composes StringToTimeDurationHookFunc, StringToSliceHookFunc(",") and TextUnmarshallerHookFunc.
func WithDecodeHook(decodeHook DecodeHook) Option {
	return func(options *options) {
		options.decodeHook = decodeHook
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
	// Option configures a Config with specific options.
	Option  func(*options)
	options Config
)
