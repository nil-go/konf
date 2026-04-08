// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package sns_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/middleware"

	ksns "github.com/nil-go/konf/notifier/sns"
	"github.com/nil-go/konf/notifier/sns/internal/assert"
)

func TestNotifier_nil(t *testing.T) {
	t.Parallel()

	var n *ksns.Notifier
	n.Register(nil) // no panic
	err := n.Start(context.Background())
	assert.EqualError(t, err, "nil Notifier")
}

//nolint:dupl,gocognit,gocyclo,maintidx
func TestNotifier(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		errLoader   error
		middleware  func(
			context.Context,
			middleware.FinalizeInput,
			middleware.FinalizeHandler,
		) (middleware.FinalizeOutput, middleware.Metadata, error)
		notified bool
		error    string
		log      string
	}{
		{
			description: "success",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{
						Result: &sns.SubscribeOutput{
							SubscriptionArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic:12345678901234567890123456789012"),
						},
					}, middleware.Metadata{}, nil
				case "Unsubscribe":
					return middleware.FinalizeOutput{
						Result: &sns.UnsubscribeOutput{},
					}, middleware.Metadata{}, nil
				case "ReceiveMessage":
					return middleware.FinalizeOutput{
						Result: &sqs.ReceiveMessageOutput{
							Messages: []types.Message{
								{
									MessageId:     aws.String("message-id"),
									ReceiptHandle: aws.String("receipt-handle"),
									Body:          aws.String("message"),
								},
							},
						},
					}, middleware.Metadata{}, nil
				case "DeleteMessageBatch":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteMessageBatchOutput{},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: true,
		},
		{
			description: "empty message",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{
						Result: &sns.SubscribeOutput{
							SubscriptionArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic:12345678901234567890123456789012"),
						},
					}, middleware.Metadata{}, nil
				case "Unsubscribe":
					return middleware.FinalizeOutput{
						Result: &sns.UnsubscribeOutput{},
					}, middleware.Metadata{}, nil
				case "ReceiveMessage":
					return middleware.FinalizeOutput{
						Result: &sqs.ReceiveMessageOutput{},
					}, middleware.Metadata{}, nil
				case "DeleteMessageBatch":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteMessageBatchOutput{},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: false,
		},
		{
			description: "unsupported message",
			errLoader:   fmt.Errorf("unsupported message: %w", errors.ErrUnsupported),
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{
						Result: &sns.SubscribeOutput{
							SubscriptionArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic:12345678901234567890123456789012"),
						},
					}, middleware.Metadata{}, nil
				case "Unsubscribe":
					return middleware.FinalizeOutput{
						Result: &sns.UnsubscribeOutput{},
					}, middleware.Metadata{}, nil
				case "ReceiveMessage":
					return middleware.FinalizeOutput{
						Result: &sqs.ReceiveMessageOutput{
							Messages: []types.Message{
								{
									MessageId:     aws.String("message-id"),
									ReceiptHandle: aws.String("receipt-handle"),
									Body:          aws.String("message"),
								},
							},
						},
					}, middleware.Metadata{}, nil
				case "DeleteMessageBatch":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteMessageBatchOutput{},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: true,
			log: `level=INFO msg="Start watching SNS topic." topic=topic queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue
level=INFO msg="Received messages from SNS topic." topic=topic count=1
level=WARN msg="No loader to process message." msg=message
`,
		},
		{
			description: "process message error",
			errLoader:   errors.New("process message error"),
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{
						Result: &sns.SubscribeOutput{
							SubscriptionArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic:12345678901234567890123456789012"),
						},
					}, middleware.Metadata{}, nil
				case "Unsubscribe":
					return middleware.FinalizeOutput{
						Result: &sns.UnsubscribeOutput{},
					}, middleware.Metadata{}, nil
				case "ReceiveMessage":
					return middleware.FinalizeOutput{
						Result: &sqs.ReceiveMessageOutput{
							Messages: []types.Message{
								{
									MessageId:     aws.String("message-id"),
									ReceiptHandle: aws.String("receipt-handle"),
									Body:          aws.String("message"),
								},
							},
						},
					}, middleware.Metadata{}, nil
				case "DeleteMessageBatch":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteMessageBatchOutput{},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: true,
			log: `level=INFO msg="Start watching SNS topic." topic=topic queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue
level=INFO msg="Received messages from SNS topic." topic=topic count=1
level=WARN msg="Fail to process message." msg=message loader=loader error="process message error"
`,
		},
		{
			description: "GetCallerIdentity error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("get caller identity error")
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: false,
			error:    "get caller identity: operation error STS: GetCallerIdentity, get caller identity error",
		},
		{
			description: "CreateTopic error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "CreateTopic":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("create topic error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: false,
			error:    "get sns topic ARN: operation error SNS: CreateTopic, create topic error",
		},
		{
			description: "CreateQueue error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("create queue error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: false,
			error:    "create sqs queue: operation error SQS: CreateQueue, create queue error",
		},
		{
			description: "GetQueueAttributes error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("get queue attributes error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: false,
			error:    "get sqs queue attributes: operation error SQS: GetQueueAttributes, get queue attributes error",
		},
		{
			description: "DeleteQueue error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("delete queue error")
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{
						Result: &sns.SubscribeOutput{
							SubscriptionArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic:12345678901234567890123456789012"),
						},
					}, middleware.Metadata{}, nil
				case "Unsubscribe":
					return middleware.FinalizeOutput{
						Result: &sns.UnsubscribeOutput{},
					}, middleware.Metadata{}, nil
				case "ReceiveMessage":
					return middleware.FinalizeOutput{
						Result: &sqs.ReceiveMessageOutput{
							Messages: []types.Message{
								{
									MessageId:     aws.String("message-id"),
									ReceiptHandle: aws.String("receipt-handle"),
									Body:          aws.String("message"),
								},
							},
						},
					}, middleware.Metadata{}, nil
				case "DeleteMessageBatch":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteMessageBatchOutput{},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: true,
			log: `level=INFO msg="Start watching SNS topic." topic=topic queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue
level=INFO msg="Received messages from SNS topic." topic=topic count=1
level=WARN msg="Fail to delete sqs queue." queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue error="operation error SQS: DeleteQueue, delete queue error"
`,
		},
		{
			description: "Subscribe error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("subscribe error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: false,
			error:    "subscribe sns topic topic: operation error SNS: Subscribe, subscribe error",
		},
		{
			description: "Unsubscribe error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{
						Result: &sns.SubscribeOutput{
							SubscriptionArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic:12345678901234567890123456789012"),
						},
					}, middleware.Metadata{}, nil
				case "Unsubscribe":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("unsubscribe error")
				case "ReceiveMessage":
					return middleware.FinalizeOutput{
						Result: &sqs.ReceiveMessageOutput{
							Messages: []types.Message{
								{
									MessageId:     aws.String("message-id"),
									ReceiptHandle: aws.String("receipt-handle"),
									Body:          aws.String("message"),
								},
							},
						},
					}, middleware.Metadata{}, nil
				case "DeleteMessageBatch":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteMessageBatchOutput{},
					}, middleware.Metadata{}, nil
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: true,
			log: `level=INFO msg="Start watching SNS topic." topic=topic queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue
level=INFO msg="Received messages from SNS topic." topic=topic count=1
level=WARN msg="Fail to unsubscribe sns topic." topic=topic error="operation error SNS: Unsubscribe, unsubscribe error"
`,
		},
		{
			description: "ReceiveMessage error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{
						Result: &sns.SubscribeOutput{
							SubscriptionArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic:12345678901234567890123456789012"),
						},
					}, middleware.Metadata{}, nil
				case "Unsubscribe":
					return middleware.FinalizeOutput{
						Result: &sns.UnsubscribeOutput{},
					}, middleware.Metadata{}, nil
				case "ReceiveMessage":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("receive message error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: false,
			log: `level=INFO msg="Start watching SNS topic." topic=topic queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue
level=WARN msg="Fail to receive sqs message." queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue error="operation error SQS: ReceiveMessage, receive message error"
`,
		},
		{
			description: "DeleteMessageBatch error",
			middleware: func(
				ctx context.Context,
				_ middleware.FinalizeInput,
				_ middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				switch awsMiddleware.GetOperationName(ctx) {
				case "GetCallerIdentity":
					return middleware.FinalizeOutput{
						Result: &sts.GetCallerIdentityOutput{
							Arn: aws.String("arn:aws:sts::123456789012:assumed-role/role-name/session-name"),
						},
					}, middleware.Metadata{}, nil
				case "CreateTopic":
					return middleware.FinalizeOutput{
						Result: &sns.CreateTopicOutput{
							TopicArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic"),
						},
					}, middleware.Metadata{}, nil
				case "CreateQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.CreateQueueOutput{
							QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
						},
					}, middleware.Metadata{}, nil
				case "DeleteQueue":
					return middleware.FinalizeOutput{
						Result: &sqs.DeleteQueueOutput{},
					}, middleware.Metadata{}, nil
				case "GetQueueAttributes":
					return middleware.FinalizeOutput{
						Result: &sqs.GetQueueAttributesOutput{
							Attributes: map[string]string{
								"QueueArn": "arn:aws:sqs:us-west-2:123456789012:MyQueue",
							},
						},
					}, middleware.Metadata{}, nil
				case "Subscribe":
					return middleware.FinalizeOutput{
						Result: &sns.SubscribeOutput{
							SubscriptionArn: aws.String("arn:aws:sns:us-west-2:123456789012:MyTopic:12345678901234567890123456789012"),
						},
					}, middleware.Metadata{}, nil
				case "Unsubscribe":
					return middleware.FinalizeOutput{
						Result: &sns.UnsubscribeOutput{},
					}, middleware.Metadata{}, nil
				case "ReceiveMessage":
					return middleware.FinalizeOutput{
						Result: &sqs.ReceiveMessageOutput{
							Messages: []types.Message{
								{
									MessageId:     aws.String("message-id"),
									ReceiptHandle: aws.String("receipt-handle"),
									Body:          aws.String("message"),
								},
							},
						},
					}, middleware.Metadata{}, nil
				case "DeleteMessageBatch":
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("delete message error")
				default:
					return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
				}
			},
			notified: true,
			log: `level=INFO msg="Start watching SNS topic." topic=topic queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue
level=INFO msg="Received messages from SNS topic." topic=topic count=1
level=WARN msg="Fail to delete sqs message." queue=https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue error="operation error SQS: DeleteMessageBatch, delete message error"
`,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cfg, err := config.LoadDefaultConfig(ctx,
				config.WithAPIOptions([]func(*middleware.Stack) error{
					func(stack *middleware.Stack) error {
						return stack.Finalize.Add(
							middleware.FinalizeMiddlewareFunc(
								"mock",
								testcase.middleware,
							),
							middleware.Before,
						)
					},
				}),
			)
			assert.NoError(t, err)

			opts := []ksns.Option{
				ksns.WithAWSConfig(cfg),
			}
			buf := &buffer{}
			if testcase.log != "" {
				opts = append(opts, ksns.WithLogHandler(logHandler(buf)))
			}
			notifier := ksns.NewNotifier("topic", opts...)
			loader := &loader{
				cancel: cancel,
				err:    testcase.errLoader,
			}
			notifier.Register(loader)

			done := make(chan struct{})
			var startErr error
			go func() {
				startErr = notifier.Start(ctx)
				close(done)
			}()

			select {
			case <-done:
				if testcase.error == "" {
					assert.NoError(t, startErr)
				} else {
					assert.EqualError(t, startErr, testcase.error)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("timeout waiting for notifier.Start to return")
			}

			assert.Equal(t, testcase.notified, loader.notified.Load())
			assert.Equal(t, testcase.log, buf.String())
		})
	}
}

type loader struct {
	notified atomic.Bool
	cancel   context.CancelFunc
	err      error
}

func (l *loader) OnEvent([]byte) error {
	l.notified.Store(true)
	l.cancel()

	return l.err
}

func (l *loader) String() string {
	return "loader"
}

func logHandler(buf *buffer) *slog.TextHandler {
	return slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == slog.TimeKey {
				return slog.Attr{}
			}

			return attr
		},
	})
}

type buffer struct {
	b bytes.Buffer
	m sync.RWMutex
}

func (b *buffer) Read(p []byte) (int, error) {
	b.m.RLock()
	defer b.m.RUnlock()

	return b.b.Read(p)
}

func (b *buffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p)
}

func (b *buffer) String() string {
	b.m.RLock()
	defer b.m.RUnlock()

	return b.b.String()
}
