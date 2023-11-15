// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import "github.com/mitchellh/mapstructure"

// WithLoader provides the loaders that configuration is loaded from.
//
// Each loader takes precedence over the loaders before it
// while multiple loaders are specified.
func WithLoader(loaders ...Loader) Option {
	return func(options *options) {
		options.loaders = append(options.loaders, loaders...)
	}
}

// WithDelimiter provides the delimiter when specifying config path.
//
// The default delimiter is `.`, which makes config path like `parent.child.key`.
func WithDelimiter(delimiter string) Option {
	return func(options *options) {
		options.delimiter = delimiter
	}
}

// WithTagName provides the tag name that [mapstructure] reads for field names.
//
// The default tag name is `konf`.
func WithTagName(tagName string) Option {
	return func(options *options) {
		options.tagName = tagName
	}
}

// WithDecodeHook provides the decode hook for [mapstructure] decoding.
func WithDecodeHook(decodeHook mapstructure.DecodeHookFunc) Option {
	return func(options *options) {
		options.decodeHook = decodeHook
	}
}

// Option configures the given Config.
type Option func(*options)

type options struct {
	loaders []Loader
	Config
}
