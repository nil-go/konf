// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package parameterstore

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// WithPath provides the hierarchy for loading parameters from Parameter Store.
// Only parameters under the given path will be loaded.
// Hierarchies start with a forward slash (/).
// The hierarchy is the parameter name except the last part of the parameter.
//
// For example, if the prefix is "/server", only parameters starts with "/server/" will be loaded.
// By default, the path is "/" for all parameters.
func WithPath(path string) Option {
	return func(options *options) {
		options.client.path = path
	}
}

// WithFilter provides [filter] that will be used to select a set of parameters.
//
// Filters to limit the request results. The following Key values are supported
// for GetParametersByPath : Type, KeyId, and Label. The following Key values
// aren't supported for GetParametersByPath : tag, DataType, Name, Path, and Tier .
func WithFilter(filters ...types.ParameterStringFilter) Option {
	return func(options *options) {
		options.client.filters = append(options.client.filters, filters...)
	}
}

// WithNameSplitter provides the function used to split parameter names into nested keys.
// If it returns an nil/[]string{}/[]string{""}, the parameter will be ignored.
//
// For example, with the default splitter, an parameter name like "PARENT/CHILD/KEY"
// would be split into "PARENT", "CHILD", and "KEY".
func WithNameSplitter(splitter func(string) []string) Option {
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

// WithAWSConfig provides the AWS Config for the AWS SDK.
//
// By default, it loads the default AWS Config.
func WithAWSConfig(config aws.Config) Option {
	return func(options *options) {
		options.client.config = config
	}
}

type (
	// Option configures the a ParameterStore with specific options.
	Option  func(options *options)
	options ParameterStore
)
