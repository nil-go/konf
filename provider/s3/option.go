// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package s3

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// WithAWSConfig provides the AWS Config for the AWS SDK.
//
// By default, it loads the default AWS Config.
func WithAWSConfig(config aws.Config) Option {
	return func(options *options) {
		options.client.config = config
	}
}

// WithPollInterval provides the interval for polling the configuration.
//
// The default interval is 1 minute.
func WithPollInterval(interval time.Duration) Option {
	return func(options *options) {
		options.pollInterval = interval
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

type (
	// Option configures the a S3 with specific options.
	Option  func(options *options)
	options S3
)
