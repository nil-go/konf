// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:ireturn
package secretmanager

import (
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/option/internaloption"
)

// WithProject provides GCP project ID.
//
// By default, it fetches project ID from metadata server.
func WithProject(project string) Option {
	return &optionFunc{
		fn: func(options *options) {
			options.client.project = project
		},
	}
}

// WithFilter provides [filter] that will be used to select a set of secrets.
//
// [filter]: // https://cloud.google.com/secret-manager/docs/filtering
func WithFilter(filter string) Option {
	return &optionFunc{
		fn: func(options *options) {
			options.client.filter = filter
		},
	}
}

// WithNameSplitter provides the function used to split secret names into nested keys.
// If it returns an nil/[]string{}/[]string{""}, the secret will be ignored.
//
// For example, with the default splitter, an secret name like "PARENT-CHILD-KEY"
// would be split into "PARENT", "CHILD", and "KEY".
func WithNameSplitter(splitter func(string) []string) Option {
	return &optionFunc{
		fn: func(options *options) {
			options.splitter = splitter
		},
	}
}

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

type (
	Option     = option.ClientOption
	optionFunc struct {
		internaloption.EmbeddableAdapter
		fn func(options *options)
	}
	options SecretManager
)
