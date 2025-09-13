// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package pubsub_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/apiv1/pubsubpb" //nolint:staticcheck
	"cloud.google.com/go/pubsub/pstest"         //nolint:staticcheck
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"

	kpubsub "github.com/nil-go/konf/notifier/pubsub"
	"github.com/nil-go/konf/notifier/pubsub/internal/assert"
)

func TestNotifier_nil(t *testing.T) {
	t.Parallel()

	var n *kpubsub.Notifier
	n.Register(nil) // no panic
	err := n.Start(context.Background())
	assert.EqualError(t, err, "nil Notifier")
}

func TestNotifier(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []pstest.ServerReactorOption
		errLoader   error
		notified    bool
		error       string
		log         string
	}{
		{
			description: "success",
			notified:    true,
		},
		{
			description: "unsupported message",
			errLoader:   fmt.Errorf("unsupported message: %w", errors.ErrUnsupported),
			notified:    true,
			log: `level=INFO msg="Start watching PubSub topic." topic=topic subscription=projects/test/subscriptions/konf-
level=INFO msg="Received PubSub message." topic=topic eventType=test
level=WARN msg="No loader to process message." topic=topic msg=map[eventType:test]
`,
		},
		{
			description: "process message error",
			errLoader:   errors.New("process message error"),
			notified:    true,
			log: `level=INFO msg="Start watching PubSub topic." topic=topic subscription=projects/test/subscriptions/konf-
level=INFO msg="Received PubSub message." topic=topic eventType=test
level=ERROR msg="Fail to fanout event to loader." msg=map[eventType:test] loader=loader error="process message error"
`,
		},
		{
			description: "create subscription error",
			opts: []pstest.ServerReactorOption{
				pstest.WithErrorInjection("CreateSubscription", codes.AlreadyExists, "already exists"),
			},
			error: "create PubSub subscription: rpc error: code = AlreadyExists desc = already exists",
		},
		{
			description: "delete subscription error",
			opts: []pstest.ServerReactorOption{
				pstest.WithErrorInjection("DeleteSubscription", codes.Internal, "internal error"),
			},
			notified: true,
			log: `level=INFO msg="Start watching PubSub topic." topic=topic subscription=projects/test/subscriptions/konf-
level=INFO msg="Received PubSub message." topic=topic eventType=test
level=WARN msg="Fail to delete pubsub subscription." topic=topic subscription=projects/test/subscriptions/konf- error="rpc error: code = Internal desc = internal error"
`,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// Start a fake pubsub server running locally.
			srv := pstest.NewServer(testcase.opts...)
			defer func() {
				_ = srv.Close()
			}()
			topic := "projects/test/topics/topic"
			_, err := srv.GServer.CreateTopic(ctx, &pubsubpb.Topic{Name: topic}) //nolint:staticcheck
			assert.NoError(t, err)

			// Connect to the server without using TLS.
			conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			assert.NoError(t, err)
			defer func() {
				_ = conn.Close()
			}()

			opts := []kpubsub.Option{
				kpubsub.WithProject("test"),
				option.WithGRPCConn(conn),
			}
			buf := &buffer{}
			if testcase.log != "" {
				opts = append(opts, kpubsub.WithLogHandler(logHandler(buf)))
			}
			notifier := kpubsub.NewNotifier("topic", opts...)
			loader := &loader{
				cancel: cancel,
				err:    testcase.errLoader,
			}
			notifier.Register(loader)
			var waitgroup sync.WaitGroup
			waitgroup.Add(1)
			go func() {
				defer waitgroup.Done()
				err = notifier.Start(ctx)
				if testcase.error == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, testcase.error)
				}
			}()
			time.Sleep(10 * time.Millisecond) // Wait for notifier starts.
			srv.Publish(topic, []byte{}, map[string]string{"eventType": "test"})
			waitgroup.Wait()

			assert.Equal(t, testcase.notified, loader.notified.Load())
			re := regexp.MustCompile(`konf-[0-9a-f-]+`)
			assert.Equal(t, testcase.log, re.ReplaceAllString(buf.String(), "konf-"))
		})
	}
}

type loader struct {
	notified atomic.Bool
	cancel   context.CancelFunc
	err      error
}

func (l *loader) OnEvent(map[string]string) error {
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
