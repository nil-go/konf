// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package azappconfig

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// WithKeyFilter provides [key filter] that will be used to select a set of configuration setting entities.
//
// [key filter]: https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-key-value#supported-filters
func WithKeyFilter(filter string) Option {
	return func(options *options) {
		options.client.keyFilter = filter
	}
}

// WithLabelFilter provides [label filter] that will be used to select a set of configuration setting entities.
//
// [label filter]: https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-key-value#supported-filters
func WithLabelFilter(filter string) Option {
	return func(options *options) {
		options.client.labelFilter = filter
	}
}

// WithCredential provides the azcore.TokenCredential for Azure authentication.
//
// By default, it uses azidentity.DefaultAzureCredential.
func WithCredential(credential azcore.TokenCredential) Option {
	return func(options *options) {
		options.client.credential = credential
	}
}

// WithKeySplitter provides the function used to split setting key into nested path.
// If it returns an nil/[]string{}/[]string{""}, the variable will be ignored.
//
// For example, with the default splitter, a key like "parent/child/key"
// would be split into "parent", "child", and "key".
func WithKeySplitter(splitter func(string) []string) Option {
	return func(options *options) {
		options.splitter = splitter
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

type (
	// Option configures the AppConfig with specific options.
	Option  func(options *options)
	options AppConfig
)
