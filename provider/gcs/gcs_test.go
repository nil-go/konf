// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package gcs_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/api/option"

	"github.com/nil-go/konf/provider/gcs"
	"github.com/nil-go/konf/provider/gcs/internal/assert"
)

func TestGCS_empty(t *testing.T) {
	var loader *gcs.GCS
	values, err := loader.Load()
	assert.EqualError(t, err, "nil GCS")
	assert.Equal(t, nil, values)
	err = loader.Watch(context.Background(), nil)
	assert.EqualError(t, err, "nil GCS")
	err = loader.OnEvent(map[string]string{})
	assert.EqualError(t, err, "nil GCS")
}

func TestGCS_Load(t *testing.T) {
	t.Parallel()

	for _, testcase := range testcases() {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			loader := gcs.New(
				"bucket/file",
				option.WithHTTPClient(&http.Client{
					Transport: roundTripFunc(func(request *http.Request) *http.Response {
						assert.Equal(t, "/storage/v1/b/bucket/o/file", request.URL.Path)
						switch request.URL.Query().Get("alt") {
						case "media":
							return testcase.object
						default:
							return &http.Response{
								StatusCode: http.StatusNotFound,
							}
						}
					}),
				}),
				gcs.WithUnmarshal(testcase.unmarshal),
			)
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

func TestGCS_Watch(t *testing.T) {
	t.Parallel()

	for _, testcase := range append(testcases(), watchcases()...) {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			var err atomic.Pointer[error]
			loader := gcs.New(
				"bucket/file",
				option.WithHTTPClient(&http.Client{
					Transport: roundTripFunc(func(request *http.Request) *http.Response {
						assert.Equal(t, "/storage/v1/b/bucket/o/file", request.URL.Path)
						switch request.URL.Query().Get("alt") {
						case "media":
							return testcase.object
						default:
							return &http.Response{
								StatusCode: http.StatusNotFound,
							}
						}
					}),
				}),
				gcs.WithPollInterval(10*time.Millisecond),
				gcs.WithUnmarshal(testcase.unmarshal),
			)
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

			if len(testcase.event) > 0 {
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
				if testcase.err == "" {
					assert.Equal(t, nil, err.Load())
				} else {
					assert.EqualError(t, *err.Load(), testcase.err)
				}
			}
		})
	}
}

type testcase struct {
	description string
	object      *http.Response
	event       map[string]string
	unmarshal   func([]byte, any) error
	expected    map[string]any
	err         string
}

func testcases() []testcase {
	return []testcase{
		{
			description: "gcs",
			object: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"k": "v"}`)),
				Header:     http.Header{"X-Goog-Generation": []string{"42"}},
			},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "create object reader error",
			object: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
				Header:     make(http.Header),
			},
			err: "create object reader: storage: object doesn't exist",
		},
		{
			description: "unmarshal error",
			object: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"k": "v"}`)),
				Header:     http.Header{"X-Goog-Generation": []string{"42"}},
			},
			unmarshal: func([]byte, any) error {
				return errors.New("unmarshal error")
			},
			err: "unmarshal: unmarshal error",
		},
	}
}

func watchcases() []testcase {
	return []testcase{
		{
			description: "OBJECT_FINALIZE",
			object: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"k": "v"}`)),
				Header:     http.Header{"X-Goog-Generation": []string{"42"}},
			},
			event: map[string]string{
				"eventType": "OBJECT_FINALIZE",
				"bucketId":  "bucket",
				"objectId":  "file",
			},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "OBJECT_DELETE",
			object: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"k": "v"}`)),
				Header:     http.Header{"X-Goog-Generation": []string{"42"}},
			},
			event: map[string]string{
				"eventType": "OBJECT_DELETE",
				"bucketId":  "bucket",
				"objectId":  "file",
			},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "unmatched event",
			object: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"k": "v"}`)),
				Header:     http.Header{"X-Goog-Generation": []string{"42"}},
			},
			event: map[string]string{
				"EventType": "OBJECT_DELETE",
				"bucketId":  "bucket",
				"objectId":  "another file",
			},
			expected: map[string]any{
				"k": "v",
			},
			err: "unsupported gcs event: unsupported operation",
		},
	}
}

func TestGCS_String(t *testing.T) {
	t.Parallel()

	loader := gcs.New("gs://bucket/file")
	assert.Equal(t, "gs://bucket/file", loader.String())
}

type roundTripFunc func(*http.Request) *http.Response

func (r roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req), nil
}
