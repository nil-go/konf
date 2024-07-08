// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package pubsub provides a notifier that subscribes to an PubSub topic that watches change of configuration on GCP.
//
// It requires following roles on the target project:
//   - roles/pubsub.editor
package pubsub

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

// Notifier that watches change events on given PubSub topic.
//
// To create a new Notifier, call [NewNotifier].
type Notifier struct {
	topic   string
	project string
	logger  *slog.Logger

	clientOpts   []option.ClientOption
	loaders      []loader
	loadersMutex sync.RWMutex
}

type loader interface{ OnEvent(map[string]string) error }

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
func (n *Notifier) Register(loaders ...loader) {
	if n == nil {
		return
	}

	n.loadersMutex.Lock()
	defer n.loadersMutex.Unlock()
	n.loaders = append(n.loaders, loaders...)
}

var errNil = errors.New("nil Notifier")

// Start starts watching events on given PubSub topic and fanout to registered loaders.
// It blocks until ctx is done, or it returns an error.
func (n *Notifier) Start(ctx context.Context) error { //nolint:cyclop,funlen
	if n == nil {
		return errNil
	}

	project := n.project
	if project == "" {
		var err error
		if project, err = metadata.ProjectIDWithContext(ctx); err != nil {
			return fmt.Errorf("get GCP project ID: %w", err)
		}
	}

	logger := n.logger
	if n.logger == nil {
		logger = slog.Default()
	}

	client, err := pubsub.NewClient(ctx, project, n.clientOpts...)
	if err != nil {
		return fmt.Errorf("create PubSub client: %w", err)
	}
	defer func() {
		if derr := client.Close(); derr != nil {
			logger.LogAttrs(ctx, slog.LevelWarn,
				"Fail to close pubsub client.",
				slog.String("project", project),
				slog.Any("error", derr),
			)
		}
	}()
	subscription, err := client.CreateSubscription(ctx, "konf-"+uuid.NewString(), pubsub.SubscriptionConfig{
		Topic: client.Topic(n.topic),
	})
	if err != nil {
		return fmt.Errorf("create PubSub subscription: %w", err)
	}
	defer func() {
		if derr := subscription.Delete(context.WithoutCancel(ctx)); derr != nil {
			logger.LogAttrs(ctx, slog.LevelWarn,
				"Fail to delete pubsub subscription.",
				slog.String("topic", n.topic),
				slog.String("subscription", subscription.String()),
				slog.Any("error", derr),
			)
		}
	}()
	logger.LogAttrs(ctx, slog.LevelInfo,
		"Start watching PubSub topic.",
		slog.String("topic", n.topic),
		slog.String("subscription", subscription.String()),
	)

	err = subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		attributes := msg.Attributes
		logger.LogAttrs(ctx, slog.LevelInfo,
			"Received PubSub message.",
			slog.String("topic", n.topic),
			slog.Any("eventType", attributes["eventType"]),
		)

		for _, loader := range n.loaders {
			err = loader.OnEvent(attributes)
			if errors.Is(err, errors.ErrUnsupported) {
				continue
			}

			if err != nil {
				logger.LogAttrs(ctx, slog.LevelError,
					"Fail to fanout event to loader.",
					slog.Any("msg", msg.Attributes),
					slog.Any("loader", loader),
					slog.Any("error", err),
				)
			}

			break
		}
		if errors.Is(err, errors.ErrUnsupported) {
			logger.LogAttrs(ctx, slog.LevelWarn,
				"No loader to process message.",
				slog.String("topic", n.topic),
				slog.Any("msg", msg.Attributes),
			)
		}

		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("receive PubSub message: %w", err)
	}

	return nil
}
