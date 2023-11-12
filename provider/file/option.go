// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file

// WithUnmarshal provides the function that parses config file.
//
// The default function is json.Unmarshal.
func WithUnmarshal(unmarshal func([]byte, any) error) Option {
	return func(options *options) {
		options.unmarshal = unmarshal
	}
}

// IgnoreFileNotExit ignores the error if config file does not exist.
func IgnoreFileNotExit() Option {
	return func(options *options) {
		options.ignoreNotExist = true
	}
}

// Option configures the given File.
type Option func(options *options)

type options File
