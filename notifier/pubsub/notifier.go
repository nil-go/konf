// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package pubsub

import (
	"log/slog"
	"sync"

	"google.golang.org/api/option"
)

// Notifier that watches change events on given PubSub topic.
//
// To create a new Notifier, call [NewNotifier].
type Notifier struct {
	topic  string
	logger *slog.Logger

	clientOpts   []option.ClientOption
	loaders      []loader
	loadersMutex sync.RWMutex
}

type loader interface{ OnEvent([]byte) error }

// NewNotifier creates a Notifier with the given PubSub topic.
func NewNotifier(topic string, opts ...Option) *Notifier {
	option := &options{
		topic: topic,
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case *optionFunc:
			o.fn(option)
		default:
			option.clientOpts = append(option.clientOpts, o)
		}
	}

	return (*Notifier)(option)
}

// Register registers a loader to the Notifier.
func (n *Notifier) Register(loader loader) {
	if n == nil {
		return
	}

	n.loadersMutex.Lock()
	defer n.loadersMutex.Unlock()
	n.loaders = append(n.loaders, loader)
}
