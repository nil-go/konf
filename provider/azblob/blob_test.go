// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package azblob_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/messaging"

	"github.com/nil-go/konf/provider/azblob"
	"github.com/nil-go/konf/provider/azblob/internal/assert"
)

func TestBlob_empty(t *testing.T) {
	var loader *azblob.Blob
	values, err := loader.Load()
	assert.EqualError(t, err, "nil Blob")
	assert.Equal(t, nil, values)
	err = loader.Watch(context.Background(), nil)
	assert.EqualError(t, err, "nil Blob")
	err = loader.OnEvent(messaging.CloudEvent{})
	assert.EqualError(t, err, "nil Blob")
}

func TestBlob(t *testing.T) {
	t.Parallel()

	loader := azblob.New("", "", "")
	values, err := loader.Load()
	assert.Equal(t, nil, values)
	assert.EqualError(t, err, "get blob: no Host in request URL")
}

func TestBlob_Load(t *testing.T) {
	t.Parallel()

	for _, testcase := range testcases() {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(testcase.handler))
			defer server.Close()

			loader := azblob.New(server.URL, "container", "blob",
				append(testcase.opts, azblob.WithUnmarshal(testcase.unmarshal))...)
			values, err := loader.Load()
			if testcase.err != "" {
				if strings.Contains(testcase.err, "%s") {
					assert.EqualError(t, err, fmt.Sprintf(testcase.err, server.URL))
				} else {
					assert.EqualError(t, err, testcase.err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testcase.expected, values)
				values, err = loader.Load()
				assert.NoError(t, err)
				assert.Equal(t, nil, values)
			}
		})
	}
}

func TestBlob_Watch(t *testing.T) {
	t.Parallel()

	for _, testcase := range append(testcases(), watchcases()...) {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(testcase.handler))
			defer server.Close()

			loader := azblob.New(
				server.URL,
				"container",
				"blob",
				append(
					testcase.opts,
					azblob.WithUnmarshal(testcase.unmarshal),
					azblob.WithPollInterval(10*time.Millisecond),
				)...,
			)
			var err atomic.Pointer[error]
			loader.Status(func(_ bool, e error) {
				if e != nil {
					err.Store(&e)
				}
			})

			values := make(chan map[string]any)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			started := make(chan struct{})
			go func() {
				close(started)

				e := loader.Watch(ctx, func(changed map[string]any) {
					values <- changed
				})
				assert.NoError(t, e)
			}()
			<-started

			if !reflect.ValueOf(testcase.event).IsZero() {
				testcase.event.Data = []byte(fmt.Sprintf(string(testcase.event.Data.([]byte)), server.URL))
				eerr := loader.OnEvent(testcase.event)
				if testcase.err == "" {
					assert.NoError(t, eerr)
				} else {
					assert.EqualError(t, eerr, testcase.err)
				}
			}

			time.Sleep(15 * time.Millisecond) // wait for the first tick, but not the second
			select {
			case val := <-values:
				assert.Equal(t, testcase.expected, val)
			default:
				if strings.Contains(testcase.err, "%s") {
					assert.EqualError(t, *err.Load(), fmt.Sprintf(testcase.err, server.URL))
				} else {
					assert.EqualError(t, *err.Load(), testcase.err)
				}
			}
		})
	}
}

type testcase struct {
	description string
	opts        []azblob.Option
	handler     func(http.ResponseWriter, *http.Request)
	event       messaging.CloudEvent
	unmarshal   func([]byte, any) error
	expected    map[string]any
	err         string
}

func testcases() []testcase {
	return []testcase{
		{
			description: "blob",
			opts: []azblob.Option{
				azblob.WithCredential(nil),
			},
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("ETag", "k42")
				_, _ = writer.Write([]byte(`{"k":"v"}`))
			},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "download blob error",
			opts: []azblob.Option{
				azblob.WithCredential(nil),
			},
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				http.Error(writer, "download blob error", http.StatusNotFound)
			},
			err: `get blob: GET %s/container/blob
--------------------------------------------------------------------------------
RESPONSE 404: 404 Not Found
ERROR CODE UNAVAILABLE
--------------------------------------------------------------------------------
download blob error

--------------------------------------------------------------------------------
`,
		},
		{
			description: "unmarshal error",
			opts: []azblob.Option{
				azblob.WithCredential(nil),
			},
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("ETag", "k42")
				_, _ = writer.Write([]byte(`{"k":"v"}`))
			},
			unmarshal: func([]byte, any) error {
				return errors.New("unmarshal error")
			},
			err: "unmarshal: unmarshal error",
		},
		{
			description: "default credential",
			err:         "get blob: authenticated requests are not permitted for non TLS protected (https) endpoints",
		},
	}
}

func watchcases() []testcase {
	return []testcase{
		{
			description: "BlobCreated",
			opts: []azblob.Option{
				azblob.WithCredential(nil),
			},
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("ETag", "k42")
				_, _ = writer.Write([]byte(`{"k":"v"}`))
			},
			event: messaging.CloudEvent{
				Type: "Microsoft.Storage.BlobCreated",
				Data: []byte(`{"url":"%s/container/blob"}`),
			},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "BlobDeleted",
			opts: []azblob.Option{
				azblob.WithCredential(nil),
			},
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("ETag", "k42")
				_, _ = writer.Write([]byte(`{"k":"v"}`))
			},
			event: messaging.CloudEvent{
				Type: "Microsoft.Storage.Deleted",
				Data: []byte(`{"url":"%s/container/blob"}`),
			},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "unmatched event",
			opts: []azblob.Option{
				azblob.WithCredential(nil),
			},
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("ETag", "k42")
				_, _ = writer.Write([]byte(`{"k":"v"}`))
			},
			event: messaging.CloudEvent{
				Type: "Microsoft.Storage.BlobCreated",
				Data: []byte(`{"url":"%s/another_container/blob"}`),
			},
			expected: map[string]any{
				"k": "v",
			},
			err: "unsupported blob storage event: unsupported operation",
		},
	}
}

func TestBlob_String(t *testing.T) {
	t.Parallel()

	loader := azblob.New("https://azblob.io", "container", "blob")
	assert.Equal(t, "https://azblob.io/container/blob", loader.String())
}
