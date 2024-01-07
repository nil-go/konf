// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package fs

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

// Option configures the a FS with specific options.
type Option func(file *options)

type options FS
