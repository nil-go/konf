// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"log/slog"

	"github.com/go-viper/mapstructure/v2"
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

// WithTagName provides the tag name that [mapstructure] reads for field names.
// The tag name is used by mapstructure when decoding configuration into structs.
//
// For example, with the default tag name `konf`, mapstructure would look for `konf` tags on struct fields.
func WithTagName(tagName string) Option {
	return func(options *options) {
		options.tagName = tagName
	}
}

// WithDecodeHook provides the decode hook for [mapstructure] decoding.
// The decode hook is a function that can transform or customize how values are decoded.
//
// By default, it composes mapstructure.StringToTimeDurationHookFunc,
// mapstructure.StringToSliceHookFunc(",") and mapstructure.TextUnmarshallerHookFunc.
func WithDecodeHook(decodeHook mapstructure.DecodeHookFunc) Option {
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

// WithValueFormatter provides the value formatter for Config.Explain.
// It's for hiding sensitive information (e.g. password, secret) which should not be exposed.
//
// By default, it uses fmt.Sprint to format the value.
func WithValueFormatter(valueFormatter func(path string, loader Loader, value any) string) ExplainOption {
	return func(options *explainOptions) {
		options.valueFormatter = valueFormatter
	}
}

type (
	// ExplainOption configures Config.Explain with specific options.
	ExplainOption  func(*explainOptions)
	explainOptions struct {
		valueFormatter func(path string, loader Loader, value any) string
	}
)
