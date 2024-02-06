// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package appconfig

import (
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// WithAWSConfig provides the AWS Config for the AWS SDK.
//
// By default, it loads the default AWS Config.
func WithAWSConfig(awsConfig aws.Config) Option {
	return func(options *options) {
		options.awsConfig = awsConfig
	}
}

// WithPollInterval provides the interval for polling the configuration.
//
// The default interval is 1 minute.
func WithPollInterval(pollInterval time.Duration) Option {
	return func(options *options) {
		options.pollInterval = pollInterval
	}
}

// WithUnmarshal provides the function used to parses the configuration.
// The unmarshal function must be able to unmarshal the configuration into a map[string]any.
//
// The default function is json.Unmarshal.
func WithUnmarshal(unmarshal func([]byte, any) error) Option {
	return func(options *options) {
		options.unmarshal = unmarshal
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
	// Option configures the a AppConfig with specific options.
	Option  func(options *options)
	options struct {
		AppConfig

		awsConfig aws.Config
	}
)
