// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package azblob

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// WithCredential provides the azcore.TokenCredential for Azure authentication.
//
// By default, it uses azidentity.DefaultAzureCredential.
func WithCredential(credential azcore.TokenCredential) Option {
	return func(options *options) {
		options.client.credential = credential
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
	// Option configures the Blob with specific options.
	Option  func(options *options)
	options Blob
)
