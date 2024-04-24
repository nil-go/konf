// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package azservicebus

import (
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// WithCredential provides the azcore.TokenCredential for Azure authentication.
//
// By default, it uses azidentity.DefaultAzureCredential.
func WithCredential(credential azcore.TokenCredential) Option {
	return func(options *options) {
		options.credential = credential
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
