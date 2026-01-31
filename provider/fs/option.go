// Copyright (c) 2026 The konf authors
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

type (
	// Option configures the a FS with specific options.
	Option  func(file *options)
	options FS
)
