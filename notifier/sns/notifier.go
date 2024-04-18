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
	topic string

	config       aws.Config
	logger       *slog.Logger
	loaders      []func([]byte) error
	loadersMutex sync.RWMutex
}

// NewNotifier creates a Notifier with the given SNS topic ARN.
func NewNotifier(topic string, opts ...Option) *Notifier {
	option := &options{}
	for _, opt := range opts {
		opt(option)
	}

	return &Notifier{
		topic: topic,
	}
}

// Register registers a loader to the Notifier.
// The loader is required to implement `OnEvent([]byte) error`.
func (n *Notifier) Register(loader interface{ OnEvent([]byte) error }) {
	n.loadersMutex.Lock()
	defer n.loadersMutex.Unlock()
	n.loaders = append(n.loaders, loader.OnEvent)
}

// Start starts watching events on given SNS topic and fanout to registered loaders.
// It blocks until ctx is done, or it returns an error.
func (n *Notifier) Start(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
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
		deleteQueueInput := &sqs.DeleteQueueInput{QueueUrl: queue.QueueUrl}
		if _, e := sqsClient.DeleteQueue(context.WithoutCancel(ctx), deleteQueueInput); e != nil {
			logger.LogAttrs(ctx, slog.LevelWarn,
				"Fail to delete sqs queue.",
				slog.String("queue", *queue.QueueUrl),
				slog.Any("error", e),
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

	snsClient := sns.NewFromConfig(n.config)
	Subscription, err := snsClient.Subscribe(ctx, &sns.SubscribeInput{
		Protocol:              aws.String("sqs"),
		TopicArn:              aws.String(n.topic),
		Endpoint:              aws.String(queueAttrs.Attributes["QueueArn"]),
		Attributes:            map[string]string{"RawMessageDelivery": "true"},
		ReturnSubscriptionArn: true,
	})
	if err != nil {
		return fmt.Errorf("subscribe sns topic %s: %w", n.topic, err)
	}
	defer func() {
		unsubscribeInput := &sns.UnsubscribeInput{SubscriptionArn: Subscription.SubscriptionArn}
		if _, e := snsClient.Unsubscribe(context.WithoutCancel(ctx), unsubscribeInput); e != nil {
			logger.LogAttrs(ctx, slog.LevelWarn,
				"Fail to unsubscribe sns topic.",
				slog.String("topic", n.topic),
				slog.Any("error", e),
			)
		}
	}()
	slog.LogAttrs(ctx, slog.LevelInfo,
		"Subscribed sqs queue to sns topic.",
		slog.String("queue", *queue.QueueUrl),
		slog.String("topic", n.topic),
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
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

				continue
			}
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
					errM = loader(bytes)
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
