// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nil-go/konf/provider/file"
	"github.com/nil-go/konf/provider/file/internal/assert"
)

func TestFile_Watch_loadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watch.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{"k":"old"}`), 0o600))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	loading := make(chan struct{})
	resume := make(chan struct{})
	loadErr := errors.New("incomplete write")
	first := true
	loader := file.New(path, file.WithUnmarshal(func(data []byte, value any) error {
		if first {
			first = false
			close(loading)
			select {
			case <-resume:
			case <-ctx.Done():
				return ctx.Err()
			}

			return loadErr
		}

		return json.Unmarshal(data, value)
	}))
	statuses := make(chan error, 10)
	loader.Status(func(_ bool, err error) {
		select {
		case statuses <- err:
		case <-ctx.Done():
		}
	})
	values := make(chan map[string]any, 10)
	done := make(chan error, 1)
	go func() {
		done <- loader.Watch(ctx, func(value map[string]any) {
			select {
			case values <- value:
			case <-ctx.Done():
			}
		})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("timed out stopping file watcher")
		}
	})

	// Wait for an actual load instead of assuming the watcher is ready after a sleep.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
started:
	for {
		select {
		case <-loading:
			break started
		case <-ticker.C:
			assert.NoError(t, os.WriteFile(path, nil, 0o600))
		case <-ctx.Done():
			t.Fatal("timed out starting file load")
		}
	}

	// Complete the write while the first load is still returning a parse error.
	// The following write event must be processed even if it matches the first.
	assert.NoError(t, os.WriteFile(path, []byte(`{"k":"new"}`), 0o600))
	close(resume)
	select {
	case err := <-statuses:
		if !errors.Is(err, loadErr) {
			t.Errorf("expected load error, got %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for load error")
	}
	select {
	case value := <-values:
		assert.Equal(t, map[string]any{"k": "new"}, value)
	case <-ctx.Done():
		t.Fatal("timed out waiting for completed write")
	}
}

func TestFile_Watch(t *testing.T) {
	testcases := []struct {
		description string
		action      func(string) error
		expected    map[string]any
	}{
		{
			description: "write",
			action: func(path string) error {
				err := os.WriteFile(path, []byte(`{"p": {"k": "c"}}`), 0o600)
				time.Sleep(time.Second) // wait for the file to be written

				return err
			},
			expected: map[string]any{"p": map[string]any{"k": "c"}},
		},
		{
			description: "remove",
			action: func(path string) error {
				err := os.Remove(path)
				for _, e := os.Stat(path); os.IsExist(e); _, e = os.Stat(path) { //nolint:revive
					// wait for the file to be removed
				}

				return err
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "watch.json")
			assert.NoError(t, os.WriteFile(tmpFile, []byte(`{"p": {"k": "v"}}`), 0o600))
			for _, e := os.Stat(tmpFile); os.IsNotExist(e); _, e = os.Stat(tmpFile) { //nolint:revive
				// wait for the file to be written
			}

			values := make(chan map[string]any)

			started := make(chan struct{})
			ctx := t.Context()
			loader := file.New(tmpFile)
			go func() {
				close(started)
				err := loader.Watch(ctx, func(changed map[string]any) {
					values <- changed
				})
				assert.NoError(t, err)
			}()
			<-started
			time.Sleep(time.Second) // wait for the watcher to start

			assert.NoError(t, testcase.action(tmpFile))
			select {
			case value := <-values:
				assert.Equal(t, testcase.expected, value)
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for file change")
			}
		})
	}
}
