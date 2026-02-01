// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/nil-go/konf/provider/file"
	"github.com/nil-go/konf/provider/file/internal/assert"
)

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
			tmpFile := path.Join(t.TempDir(), "watch.json")
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
			assert.Equal(t, testcase.expected, <-values)
		})
	}
}
