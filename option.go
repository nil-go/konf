// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"log/slog"

	"github.com/nil-go/konf/internal/convert"
)

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
// The decode hook is a function that can customize how configuration are decoded.
//
// It can be either `func(F) (T, error)` which returns the converted value,
// or `func(F, T) error` which sets the converted value inline.
//
// By default, it composes string to time.Duration, string to []string split by `,`
// and string to encoding.TextUnmarshaler.
func WithDecodeHook[F, T any, FN func(F) (T, error) | func(F, T) error](hook FN) Option {
	return func(options *options) {
		options.convertOpts = append(options.convertOpts, convert.WithHook[F, T](hook))
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

// WithOnStatus provides the callback for monitoring status of configuration loading/watching.
func WithOnStatus(onStatus func(loader Loader, changed bool, err error)) Option {
	return func(options *options) {
		options.onStatus = onStatus
	}
}

// WithCaseSensitive enables the case sensitivity of the configuration keys.
func WithCaseSensitive() Option {
	return func(options *options) {
		options.caseSensitive = true
	}
}

// WithMapKeyCaseSensitive enables the case sensitivity of the map keys.
func WithMapKeyCaseSensitive() Option {
	return func(options *options) {
		options.mapKeyCaseSensitive = true
	}
}

type (
	// Option configures a Config with specific options.
	Option  func(*options)
	options struct {
		Config

		tagName     string
		convertOpts []convert.Option
	}
)
