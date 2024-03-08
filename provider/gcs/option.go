// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:ireturn
package gcs

import (
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/option/internaloption"
)

// WithPollInterval provides the interval for polling the configuration.
//
// The default interval is 1 minute.
func WithPollInterval(interval time.Duration) Option {
	return &optionFunc{
		fn: func(options *options) {
			options.pollInterval = interval
		},
	}
}

// WithUnmarshal provides the function used to parses the configuration.
// The unmarshal function must be able to unmarshal the configuration into a map[string]any.
//
// The default function is json.Unmarshal.
func WithUnmarshal(unmarshal func([]byte, any) error) Option {
	return &optionFunc{
		fn: func(options *options) {
			options.unmarshal = unmarshal
		},
	}
}

type (
	Option     = option.ClientOption
	optionFunc struct {
		internaloption.EmbeddableAdapter
		fn func(options *options)
	}
	options GCS
)
