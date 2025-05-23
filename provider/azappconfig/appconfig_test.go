// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package azappconfig_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/messaging"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	"github.com/nil-go/konf/provider/azappconfig"
	"github.com/nil-go/konf/provider/azappconfig/internal/assert"
)

func TestAppConfig_empty(t *testing.T) {
	var loader *azappconfig.AppConfig
	values, err := loader.Load()
	assert.EqualError(t, err, "nil AppConfig")
	assert.Equal(t, nil, values)
	err = loader.Watch(context.Background(), nil)
	assert.EqualError(t, err, "nil AppConfig")
	err = loader.OnEvent(messaging.CloudEvent{})
	assert.EqualError(t, err, "nil AppConfig")
}

func TestAppConfig(t *testing.T) {
	t.Parallel()

	loader := azappconfig.New("")
	values, err := loader.Load()
	assert.Equal(t, nil, values)
	assert.EqualError(t, err, "next page of list settings: no Host in request URL")
}

func TestAppConfig_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []azappconfig.Option
		expected    map[string]any
		err         string
	}{
		{
			description: "app config",
			opts: []azappconfig.Option{
				azappconfig.WithCredential(nil),
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
		},
		{
			description: "with key filter",
			opts: []azappconfig.Option{
				azappconfig.WithKeyFilter("p*"),
				azappconfig.WithCredential(nil),
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
		{
			description: "with label filter",
			opts: []azappconfig.Option{
				azappconfig.WithLabelFilter("q"),
				azappconfig.WithKeySplitter(func(s string) []string { return strings.Split(s, "_") }),
				azappconfig.WithCredential(nil),
			},
			expected: map[string]any{
				"q": map[string]any{
					"k": "v",
				},
			},
		},
		{
			description: "with nil splitter",
			opts: []azappconfig.Option{
				azappconfig.WithKeyFilter("p_*"),
				azappconfig.WithKeySplitter(func(string) []string { return nil }),
				azappconfig.WithCredential(nil),
			},
			expected: map[string]any{},
		},
		{
			description: "with empty splitter",
			opts: []azappconfig.Option{
				azappconfig.WithKeyFilter("p_*"),
				azappconfig.WithKeySplitter(func(string) []string { return []string{""} }),
				azappconfig.WithCredential(nil),
			},
			expected: map[string]any{},
		},
		{
			description: "default credential",
			err:         "next page of list settings: authenticated requests are not permitted for non TLS protected (https) endpoints",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			server := httpServer()
			defer server.Close()

			loader := azappconfig.New(server.URL, testcase.opts...)
			values, err := loader.Load()
			if testcase.err != "" {
				assert.EqualError(t, err, testcase.err)
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

func TestAppConfig_Watch(t *testing.T) {
	t.Parallel()

	server := httpServer()
	t.Cleanup(server.Close)

	testcases := []struct {
		description string
		opts        []azappconfig.Option
		event       messaging.CloudEvent
		expected    map[string]any
		err         string
	}{
		{
			description: "success",
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
		},
		{
			description: "error",
			opts: []azappconfig.Option{
				azappconfig.WithLabelFilter("error"),
			},
			err: `next page of list settings: GET %s/kv
--------------------------------------------------------------------------------
RESPONSE 400: 400 Bad Request
ERROR CODE UNAVAILABLE
--------------------------------------------------------------------------------
list settings error

--------------------------------------------------------------------------------
`,
		},
		{
			description: "KeyValueModified",
			event: messaging.CloudEvent{
				Type:    "Microsoft.AppConfiguration.KeyValueModified",
				Subject: to.Ptr(server.URL + "/kv/k"),
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
		},
		{
			description: "KeyValueDeleted",
			event: messaging.CloudEvent{
				Type:    "Microsoft.AppConfiguration.KeyValueDeleted",
				Subject: to.Ptr(server.URL + "/kv/k"),
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
		},
		{
			description: "unmatched event",
			event: messaging.CloudEvent{
				Type:    "Microsoft.Storage.BlobCreated",
				Subject: to.Ptr("https://another.azconfig.io/kv/"),
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
			err: "unsupported app configuration event: unsupported operation",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			loader := azappconfig.New(
				server.URL,
				append(
					testcase.opts,
					azappconfig.WithCredential(nil),
					azappconfig.WithPollInterval(10*time.Millisecond),
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
				if e := err.Load(); e != nil {
					assert.EqualError(t, *e, fmt.Sprintf(testcase.err, server.URL))
				}
			}
		})
	}
}

func TestAppConfig_String(t *testing.T) {
	t.Parallel()

	loader := azappconfig.New("https://appconfig.azconfig.io")
	assert.Equal(t, "https://appconfig.azconfig.io", loader.String())
}

func httpServer() *httptest.Server {
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("label") == "error" {
			http.Error(writer, "list settings error", http.StatusBadRequest)

			return
		}
		var items []map[string]string
		switch {
		case request.URL.Query().Get("label") != "":
			items = []map[string]string{
				{
					"key":   "q_k",
					"value": "v",
					"etag":  "qk42",
				},
			}
		case request.URL.Query().Get("key") != "":
			items = []map[string]string{
				{
					"key":   "p/k",
					"value": "v",
					"etag":  "pk42",
				},
			}
		default:
			items = []map[string]string{
				{
					"key":   "p/k",
					"value": "v",
					"etag":  "pk42",
				},
				{
					"key":   "p/d",
					"value": ".",
					"etag":  "pd42",
				},
			}
		}

		bytes, err := json.Marshal(map[string][]map[string]string{"items": items})
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}

		writer.Header().Set("Sync-Token", "jtqGc1I4=MDoyOA==;sn=28")
		_, _ = writer.Write(bytes)
	})

	return httptest.NewServer(handler)
}
