// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package sns provides a notifier that subscribes to an SNS topic that watches change of configuration on AWS.
//
// It [Fanout SNS topic to Amazon SQS queues], which requires following permissions:
//   - sns:Subscribe
//   - sns:Unsubscribe
//   - sqs:CreateQueue
//
// [Fanout SNS topic to Amazon SQS queues]: https://docs.aws.amazon.com/sns/latest/dg/sns-sqs-as-subscriber.html
package sns

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/rand"
)

// Notifier that watches change events on given SNS topic.
//
// To create a new Notifier, call [NewNotifier].
type Notifier struct {
	topic  string
	config aws.Config
	logger *slog.Logger

	loaders      []loader
	loadersMutex sync.RWMutex
}

type loader interface{ OnEvent([]byte) error }

// NewNotifier creates a Notifier with the given SNS topic ARN.
func NewNotifier(topic string, opts ...Option) *Notifier {
	option := &options{
		topic: topic,
	}
	for _, opt := range opts {
		opt(option)
	}

	return (*Notifier)(option)
}

// Register registers a loader to the Notifier.
// The loader is required to implement `OnEvent([]byte) error`.
func (n *Notifier) Register(loader loader) {
	if n == nil {
		return
	}

	n.loadersMutex.Lock()
	defer n.loadersMutex.Unlock()
	n.loaders = append(n.loaders, loader)
}

var errNil = errors.New("nil Notifier")

// Start starts watching events on given SNS topic and fanout to registered loaders.
// It blocks until ctx is done, or it returns an error.
func (n *Notifier) Start(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
	if n == nil {
		return errNil
	}

	logger := n.logger
	if n.logger == nil {
		logger = slog.Default()
	}

	if reflect.ValueOf(n.config).IsZero() {
		var err error
		if n.config, err = config.LoadDefaultConfig(ctx); err != nil {
			return fmt.Errorf("load default AWS config: %w", err)
		}
	}

	stsClient := sts.NewFromConfig(n.config)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("get caller identity: %w", err)
	}
	policy := fmt.Sprintf(`
{
	"Version":"2012-10-17",
	"Statement":[
		{
			"Effect":"Allow",
			"Principal":{
				"Service": "sns.amazonaws.com"
			},
			"Action":"sqs:SendMessage",
			"Resource":"*",
			"Condition":{
				"ArnEquals":{
					"aws:SourceArn":"%s"
				}
			}
		},
		{
			"Effect":"Allow",
			"Principal":{
				"AWS":"%s"
			},
			"Action": [
				"sqs:GetQueueAttributes",
				"sqs:DeleteQueue",
				"sqs:ReceiveMessage",
				"sqs:DeleteMessage"
			],
			"Resource":"*"
		}
	]
}`, n.topic, aws.ToString(identity.Arn))

	sqsClient := sqs.NewFromConfig(n.config)
	uuid, err := rand.NewUUID(rand.Reader).GetUUID()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}
	queue, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(uuid),
		Attributes: map[string]string{
			"Policy": policy,
		},
	})
	if err != nil {
		return fmt.Errorf("create sqs queue: %w", err)
	}
	defer func() {
		if _, derr := sqsClient.DeleteQueue(context.WithoutCancel(ctx), &sqs.DeleteQueueInput{
			QueueUrl: queue.QueueUrl,
		}); derr != nil {
			logger.LogAttrs(ctx, slog.LevelWarn,
				"Fail to delete sqs queue.",
				slog.String("queue", *queue.QueueUrl),
				slog.Any("error", derr),
			)
		}
	}()
	queueAttrs, err := sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       queue.QueueUrl,
		AttributeNames: []types.QueueAttributeName{"QueueArn"},
	})
	if err != nil {
		return fmt.Errorf("get sqs queue attributes: %w", err)
	}
	queueArn := queueAttrs.Attributes["QueueArn"]

	snsClient := sns.NewFromConfig(n.config)
	Subscription, err := snsClient.Subscribe(ctx, &sns.SubscribeInput{
		TopicArn:              aws.String(n.topic),
		Protocol:              aws.String("sqs"),
		Endpoint:              aws.String(queueArn),
		Attributes:            map[string]string{"RawMessageDelivery": "true"},
		ReturnSubscriptionArn: true,
	})
	if err != nil {
		return fmt.Errorf("subscribe sns topic %s: %w", n.topic, err)
	}
	defer func() {
		if _, derr := snsClient.Unsubscribe(context.WithoutCancel(ctx), &sns.UnsubscribeInput{
			SubscriptionArn: Subscription.SubscriptionArn,
		}); derr != nil {
			logger.LogAttrs(ctx, slog.LevelWarn,
				"Fail to unsubscribe sns topic.",
				slog.String("topic", n.topic),
				slog.Any("error", derr),
			)
		}
	}()
	logger.LogAttrs(ctx, slog.LevelInfo,
		"Subscribed sqs queue to sns topic.",
		slog.String("queue", *queue.QueueUrl),
		slog.String("topic", n.topic),
	)

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			messages, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            queue.QueueUrl,
				MaxNumberOfMessages: 10, //nolint:gomnd // The maximum number of messages to return.
				WaitTimeSeconds:     20, //nolint:gomnd // The maximum amount of time for waiting messages.
			})
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					logger.LogAttrs(ctx, slog.LevelWarn,
						"Fail to receive sqs message.",
						slog.String("queue", *queue.QueueUrl),
						slog.Any("error", err),
					)
				}
				timer.Reset(20 * time.Second) //nolint:gomnd // Retry after 20 seconds to avoid busy loop.

				continue
			}

			timer.Reset(time.Second) // Reset timer for next polling.
			if len(messages.Messages) == 0 {
				continue
			}

			n.loadersMutex.RLock()
			loaders := slices.Clone(n.loaders)
			n.loadersMutex.RUnlock()
			for _, msg := range messages.Messages {
				bytes := []byte(*msg.Body)
				var errM error
				for _, loader := range loaders {
					errM = loader.OnEvent(bytes)
					if errors.Is(errM, errors.ErrUnsupported) {
						continue
					}

					if errM != nil {
						logger.LogAttrs(ctx, slog.LevelWarn,
							"Fail to process message.",
							slog.String("msg", *msg.Body),
							slog.Any("loader", loader),
							slog.Any("error", errM),
						)
					}

					break
				}
				if errors.Is(errM, errors.ErrUnsupported) {
					logger.LogAttrs(ctx, slog.LevelWarn,
						"No loader to process message.",
						slog.String("msg", *msg.Body),
					)
				}
			}

			entries := make([]types.DeleteMessageBatchRequestEntry, 0, len(messages.Messages))
			for _, msg := range messages.Messages {
				entries = append(entries, types.DeleteMessageBatchRequestEntry{
					Id:            msg.MessageId,
					ReceiptHandle: msg.ReceiptHandle,
				})
			}
			if _, err = sqsClient.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
				QueueUrl: queue.QueueUrl,
				Entries:  entries,
			}); err != nil && !errors.Is(err, context.Canceled) {
				logger.LogAttrs(ctx, slog.LevelWarn,
					"Fail to delete sqs message.",
					slog.String("queue", *queue.QueueUrl),
					slog.Any("error", err),
				)
			}
		}
	}
}
