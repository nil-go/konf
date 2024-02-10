// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nil-go/konf/provider/file"
	"github.com/nil-go/konf/provider/file/internal/assert"
)

func TestFile_Watch(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		action      func(string) error
		expected    map[string]any
	}{
		{
			description: "write",
			action: func(path string) error {
				return os.WriteFile(path, []byte(`{"p": {"k": "c"}}`), 0o600)
			},
			expected: map[string]any{"p": map[string]any{"k": "c"}},
		},
		{
			description: "remove",
			action:      os.Remove,
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			tmpFile := filepath.Join(t.TempDir(), "watch.json")
			assert.NoError(t, os.WriteFile(tmpFile, []byte(`{"p": {"k": "v"}}`), 0o600))

			loader := file.New(tmpFile)
			var values atomic.Pointer[map[string]any]
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var waitGroup sync.WaitGroup
			waitGroup.Add(1)
			go func() {
				waitGroup.Done()
				err := loader.Watch(ctx, func(changed map[string]any) {
					values.Store(&changed)
				})
				assert.NoError(t, err)
			}()
			waitGroup.Wait()

			time.Sleep(100 * time.Millisecond)
			assert.NoError(t, testcase.action(tmpFile))
			time.Sleep(100 * time.Millisecond)
			assert.Equal(t, testcase.expected, *values.Load())
		})
	}
}
