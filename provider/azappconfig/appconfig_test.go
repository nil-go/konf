// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package azappconfig_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nil-go/konf/provider/azappconfig"
	"github.com/nil-go/konf/provider/azappconfig/internal/assert"
)

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
		testcase := testcase

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
			}
		})
	}
}

func TestAppConfig_Watch(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		opts        []azappconfig.Option
		expected    map[string]any
		log         string
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
			log: `level=WARN msg="Error when reloading from Azure App Configuration" endpoint=%s` +
				` keyFilter="" labelFilter=error error="next page of list settings: GET %s/kv\n` +
				`--------------------------------------------------------------------------------\n` +
				`RESPONSE 400: 400 Bad Request\nERROR CODE UNAVAILABLE\n` +
				`--------------------------------------------------------------------------------\n` +
				`list settings error\n\n--------------------------------------------------------------------------------\n"` +
				"\n",
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			server := httpServer()
			defer server.Close()

			buf := &buffer{}
			loader := azappconfig.New(
				server.URL,
				append(
					testcase.opts,
					azappconfig.WithLogHandler(logHandler(buf)),
					azappconfig.WithCredential(nil),
					azappconfig.WithPollInterval(10*time.Millisecond),
				)...,
			)
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
				assert.Equal(t, fmt.Sprintf(testcase.log, server.URL, server.URL), buf.String())
			}
		})
	}
}

func TestAppConfig_String(t *testing.T) {
	t.Parallel()

	loader := azappconfig.New("https://appconfig.azconfig.io")
	assert.Equal(t, "azAppConfig:https://appconfig.azconfig.io", loader.String())
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
					"etag":  "",
				},
			}
		case request.URL.Query().Get("key") != "":
			items = []map[string]string{
				{
					"key":   "p/k",
					"value": "v",
					"etag":  "",
				},
			}
		default:
			items = []map[string]string{
				{
					"key":   "p/k",
					"value": "v",
					"etag":  "",
				},
				{
					"key":   "p/d",
					"value": ".",
					"etag":  "",
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
