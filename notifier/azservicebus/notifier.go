// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package azservicebus provides a notifier that subscribes to an Service Bus topic that watches change of configuration on Azure.
//
// It requires following roles:
//   - Azure Service Bus Data Owner
//   - Azure Service Bus Data Receiver
package azservicebus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/messaging"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
	"github.com/google/uuid"
)

// Notifier that watches change events on given Service Bus topic.
//
// To create a new Notifier, call [NewNotifier].
type Notifier struct {
	namespace  string
	topic      string
	credential azcore.TokenCredential
	logger     *slog.Logger

	loaders      []loader
	loadersMutex sync.RWMutex
}

type loader interface {
	OnEvent(messaging.CloudEvent) error
}

// NewNotifier creates a Notifier with the given Service Bus namespace and topic.
func NewNotifier(namespace, topic string, opts ...Option) *Notifier {
	option := &options{
		namespace: namespace,
		topic:     topic,
		// Place holder for the default credential.
		credential: &azidentity.DefaultAzureCredential{},
	}
	for _, opt := range opts {
		opt(option)
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

// Start starts watching events on given Service Bus topic and fanout to registered loaders.
// It blocks until ctx is done, or it returns an error.
func (n *Notifier) Start(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
	if n == nil {
		return errNil
	}

	if token, ok := n.credential.(*azidentity.DefaultAzureCredential); ok && reflect.ValueOf(*token).IsZero() {
		var err error
		n.credential, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return fmt.Errorf("load default Azure credential: %w", err)
		}
	}

	logger := n.logger
	if n.logger == nil {
		logger = slog.Default()
	}

	adminClient, err := admin.NewClient(n.namespace, n.credential, nil)
	if err != nil {
		return fmt.Errorf("create Azure Service Bus admin client: %w", err)
	}
	subscription, err := adminClient.CreateSubscription(ctx, n.topic, "konf-"+uuid.NewString(), nil)
	if err != nil {
		return fmt.Errorf("create Azure Service Bus subscription: %w", err)
	}
	subscriptionName := subscription.SubscriptionName
	defer func() {
		_, err = adminClient.DeleteSubscription(context.WithoutCancel(ctx),
			n.topic, subscriptionName, nil,
		)
		if err != nil {
			logger.LogAttrs(ctx, slog.LevelWarn,
				"Fail to delete service bus subscription.",
				slog.String("topic", n.topic),
				slog.String("subscription", subscriptionName),
				slog.Any("error", err),
			)
		}
	}()

	client, err := azservicebus.NewClient(n.namespace, n.credential, nil)
	if err != nil {
		return fmt.Errorf("create Azure Service Bus client: %w", err)
	}
	receiver, err := client.NewReceiverForSubscription(n.topic, subscriptionName, &azservicebus.ReceiverOptions{
		ReceiveMode: azservicebus.ReceiveModeReceiveAndDelete,
	})
	if err != nil {
		return fmt.Errorf("create Azure Service Bus receiver: %w", err)
	}
	logger.LogAttrs(ctx, slog.LevelInfo,
		"Start watching service bus topic.",
		slog.String("topic", n.topic),
		slog.String("subscription", subscriptionName),
	)

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			messages, err := receiver.ReceiveMessages(ctx, 10, nil) //nolint:mnd // default maximum.
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					logger.LogAttrs(ctx, slog.LevelWarn,
						"Fail to receive service bus message.",
						slog.String("subscription", subscriptionName),
						slog.Any("error", err),
					)
				}
				timer.Reset(20 * time.Second) //nolint:mnd // Retry after 20 seconds to avoid busy loop.

				continue
			}

			timer.Reset(time.Second) // Reset timer for next polling.
			if len(messages) == 0 {
				continue
			}

			logger.LogAttrs(ctx, slog.LevelInfo,
				"Received messages from service bus topic.",
				slog.String("topic", n.topic),
				slog.Int("count", len(messages)),
			)

			n.loadersMutex.RLock()
			loaders := slices.Clone(n.loaders)
			n.loadersMutex.RUnlock()
			for _, msg := range messages {
				if len(msg.Body) == 0 {
					continue
				}
				var event messaging.CloudEvent
				err := event.UnmarshalJSON(msg.Body)
				if err != nil {
					logger.LogAttrs(ctx, slog.LevelWarn,
						"Fail to unmarshal message.",
						slog.String("msg", string(msg.Body)),
						slog.Any("error", err),
					)

					continue
				}

				var errM error
				for _, loader := range loaders {
					errM = loader.OnEvent(event)
					if errors.Is(errM, errors.ErrUnsupported) {
						continue
					}

					if errM != nil {
						logger.LogAttrs(ctx, slog.LevelWarn,
							"Fail to process message.",
							slog.Any("event", event),
							slog.Any("loader", loader),
							slog.Any("error", errM),
						)
					}

					break
				}
				if errors.Is(errM, errors.ErrUnsupported) {
					logger.LogAttrs(ctx, slog.LevelWarn,
						"No loader to process message.",
						slog.Any("event", event),
					)
				}
			}
		}
	}
}
