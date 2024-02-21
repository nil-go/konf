// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package secretmanager_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	pb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/nil-go/konf/provider/secretmanager"
	"github.com/nil-go/konf/provider/secretmanager/internal/assert"
)

func TestSecretManager_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []option.ClientOption
		service     pb.SecretManagerServiceServer
		expected    map[string]any
		err         string
	}{
		{
			description: "secrets",
			service: &secretManagerService{
				values: map[string]string{
					"projects/test/secrets/p-k": "v",
					"projects/test/secrets/p-d": ".",
				},
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
		},
		{
			description: "with filter",
			opts: []option.ClientOption{
				secretmanager.WithFilter(`name ~ "p-*"`),
			},
			service: &secretManagerService{
				values: map[string]string{
					"projects/test/secrets/p-k": "v",
				},
				assert: func(m proto.Message) {
					switch request := m.(type) {
					case *pb.ListSecretsRequest:
						assert.Equal(t, "projects/test", request.GetParent())
						assert.Equal(t, `name ~ "p-*"`, request.GetFilter())
					case *pb.AccessSecretVersionRequest:
						assert.Equal(t, "projects/test/secrets/p-k/versions/latest", request.GetName())
					}
				},
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
		{
			description: "with nil splitter",
			opts: []option.ClientOption{
				secretmanager.WithNameSplitter(func(string) []string { return nil }),
			},
			service: &secretManagerService{
				values: map[string]string{
					"projects/test/secrets/p-k": "v",
				},
			},
			expected: map[string]any{},
		},
		{
			description: "with empty splitter",
			opts: []option.ClientOption{
				secretmanager.WithNameSplitter(func(string) []string { return []string{""} }),
			},
			service: &secretManagerService{
				values: map[string]string{
					"projects/test/secrets/p-k": "v",
				},
			},
			expected: map[string]any{},
		},
		{
			description: "list secrets error",
			service:     &faultySecretManagerService{method: "ListSecrets"},
			err:         "list secrets on test: rpc error: code = Unknown desc = list secrets error",
		},
		{
			description: "access secret error",
			service:     &faultySecretManagerService{method: "AccessSecretVersion"},
			err:         "access secret p-k: rpc error: code = Unknown desc = access secret error",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			conn, closer := grpcServer(t, testcase.service)
			defer closer()

			loader := secretmanager.New(append(
				testcase.opts,
				secretmanager.WithProject("test"),
				option.WithGRPCConn(conn),
			)...)
			var values map[string]any
			values, err := loader.Load()
			if testcase.err != "" {
				assert.EqualError(t, err, testcase.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testcase.expected, values)
			}
		})
	}
}

func TestSecretManager_Watch(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []option.ClientOption
		service     pb.SecretManagerServiceServer
		expected    map[string]any
		log         string
	}{
		{
			description: "success",
			service: &secretManagerService{
				values: map[string]string{
					"projects/test/secrets/p-k": "v",
					"projects/test/secrets/p-d": ".",
				},
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
		},
		{
			description: "list secrets error",
			service:     &faultySecretManagerService{method: "ListSecrets"},
			log: `level=WARN msg="Error when reloading from GCP Secret Manager" project=test filter=""` +
				` error="list secrets on test: rpc error: code = Unknown desc = list secrets error"` + "\n",
		},
		{
			description: "access secret error",
			service:     &faultySecretManagerService{method: "AccessSecretVersion"},
			log: `level=WARN msg="Error when reloading from GCP Secret Manager" project=test filter=""` +
				` error="access secret p-k: rpc error: code = Unknown desc = access secret error"` + "\n",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			conn, closer := grpcServer(t, testcase.service)
			defer closer()

			buf := &buffer{}
			loader := secretmanager.New(append(
				testcase.opts,
				secretmanager.WithProject("test"),
				option.WithGRPCConn(conn),
				secretmanager.WithLogHandler(logHandler(buf)),
				secretmanager.WithPollInterval(10*time.Millisecond),
			)...)

			values := make(chan map[string]any)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := make(chan struct{})
			go func() {
				close(started)

				err := loader.Watch(ctx, func(changed map[string]any) {
					values <- changed
				})
				assert.NoError(t, err)
			}()
			<-started

			time.Sleep(15 * time.Millisecond) // wait for the first tick, but not the second
			select {
			case val := <-values:
				assert.Equal(t, testcase.expected, val)
			default:
				assert.Equal(t, testcase.log, buf.String())
			}
		})
	}
}

func grpcServer(t *testing.T, service pb.SecretManagerServiceServer) (*grpc.ClientConn, func()) {
	t.Helper()

	server := grpc.NewServer()
	pb.RegisterSecretManagerServiceServer(server, service)

	started := make(chan struct{})
	endpoint := t.TempDir() + "/load.sock"
	go func() {
		_ = os.RemoveAll(endpoint)
		listener, e := net.Listen("unix", endpoint)
		assert.NoError(t, e)
		close(started)

		assert.NoError(t, server.Serve(listener))
	}()
	<-started

	conn, err := grpc.Dial("unix:"+endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)

	return conn, func() {
		_ = conn.Close()
		server.GracefulStop()
	}
}

func TestSecretManager_String(t *testing.T) {
	t.Parallel()

	loader := secretmanager.New(secretmanager.WithProject("test"))
	assert.Equal(t, "secretManager:test", loader.String())
}

type secretManagerService struct {
	pb.UnimplementedSecretManagerServiceServer

	values map[string]string
	assert func(proto.Message)
}

func (s *secretManagerService) ListSecrets(
	_ context.Context,
	request *pb.ListSecretsRequest,
) (*pb.ListSecretsResponse, error) {
	if s.assert != nil {
		s.assert(request)
	}

	resp := &pb.ListSecretsResponse{TotalSize: int32(len(s.values))}
	for name := range s.values {
		resp.Secrets = append(resp.Secrets, &pb.Secret{Name: name})
	}

	return resp, nil
}

func (s *secretManagerService) AccessSecretVersion(
	_ context.Context,
	request *pb.AccessSecretVersionRequest,
) (*pb.AccessSecretVersionResponse, error) {
	if s.assert != nil {
		s.assert(request)
	}

	name := request.GetName()

	return &pb.AccessSecretVersionResponse{
		Name:    strings.Replace(name, "/versions/latest", "/versions/1", 1),
		Payload: &pb.SecretPayload{Data: []byte(s.values[strings.TrimSuffix(name, "/versions/latest")])},
	}, nil
}

type faultySecretManagerService struct {
	pb.UnimplementedSecretManagerServiceServer

	method string
}

func (f *faultySecretManagerService) ListSecrets(
	context.Context,
	*pb.ListSecretsRequest,
) (*pb.ListSecretsResponse, error) {
	if f.method == "ListSecrets" {
		return nil, errors.New("list secrets error")
	}

	return &pb.ListSecretsResponse{Secrets: []*pb.Secret{{Name: "projects/test/secrets/p-k"}}}, nil
}

func (f *faultySecretManagerService) AccessSecretVersion(
	context.Context,
	*pb.AccessSecretVersionRequest,
) (*pb.AccessSecretVersionResponse, error) {
	if f.method == "AccessSecretVersion" {
		return nil, errors.New("access secret error")
	}

	return &pb.AccessSecretVersionResponse{}, nil
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
