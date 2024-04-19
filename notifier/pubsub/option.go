// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:ireturn
package pubsub

import (
	"log/slog"

	"google.golang.org/api/option"
	"google.golang.org/api/option/internaloption"
)

// WithLogHandler provides the slog.Handler for logs from notifier.
//
// By default, it uses handler from slog.Default().
func WithLogHandler(handler slog.Handler) Option {
	return &optionFunc{
		fn: func(options *options) {
			if handler != nil {
				options.logger = slog.New(handler)
			}
		},
	}
}

type (
	// Option configures the Notifier with specific options.
	Option     = option.ClientOption
	optionFunc struct {
		internaloption.EmbeddableAdapter
		fn func(options *options)
	}
	options Notifier
)
