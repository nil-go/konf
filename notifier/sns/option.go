// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package sns

import (
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// WithAWSConfig provides the AWS Config for the AWS SDK.
//
// By default, it loads the default AWS Config.
func WithAWSConfig(config aws.Config) Option {
	return func(options *options) {
		options.config = config
	}
}

// WithLogHandler provides the slog.Handler for logs from notifier.
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
	// Option configures the Notifier with specific options.
	Option  func(options *options)
	options Notifier
)
