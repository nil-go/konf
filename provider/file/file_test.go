// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package file_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf/provider/file"
)

func TestFile_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		path        string
		opts        []file.Option
		expected    map[string]any
		err         string
	}{
		{
			description: "file",
			path:        "testdata/config.json",
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "file (not exist)",
			path:        "not_found.json",
			err:         "read file: open not_found.json: ",
		},
		{
			description: "file (ignore not exist)",
			path:        "not_found.json",
			opts:        []file.Option{file.IgnoreFileNotExit()},
			expected:    map[string]any{},
		},
		{
			description: "unmarshal error",
			path:        "testdata/config.json",
			opts: []file.Option{
				file.WithUnmarshal(func([]byte, any) error {
					return errors.New("unmarshal error")
				}),
			},
			err: "unmarshal: unmarshal error",
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			values, err := file.New(testcase.path, testcase.opts...).Load()
			if err != nil {
				require.True(t, strings.HasPrefix(err.Error(), testcase.err))
			} else {
				require.NoError(t, err)
				require.Equal(t, testcase.expected, values)
			}
		})
	}
}

func TestFile_Watch(t *testing.T) {
	testcases := []struct {
		description string
		action      func(string) error
		expacted    map[string]any
	}{
		{
			description: "create",
			action: func(path string) error {
				return os.WriteFile(path, []byte(`{"p": {"k": "v"}}`), 0o600)
			},
			expacted: map[string]any{"p": map[string]any{"k": "v"}},
		},
		{
			description: "write",
			action: func(path string) error {
				return os.WriteFile(path, []byte(`{"p": {"k": "c"}}`), 0o600)
			},
			expacted: map[string]any{"p": map[string]any{"k": "c"}},
		},
		{
			description: "remove",
			action:      os.Remove,
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "watch.json")
			require.NoError(t, os.WriteFile(tmpFile, []byte(`{"p": {"k": "v"}}`), 0o600))

			loader := file.New(tmpFile)
			var values map[string]any
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var waitGroup sync.WaitGroup
			waitGroup.Add(1)
			go func() {
				err := loader.Watch(ctx, func(changed map[string]any) {
					defer waitGroup.Done()
					values = changed
				})
				require.NoError(t, err)
			}()

			time.Sleep(time.Second)
			require.NoError(t, testcase.action(tmpFile))
			waitGroup.Wait()
			require.Equal(t, testcase.expacted, values)
		})
	}
}

func TestFile_String(t *testing.T) {
	t.Parallel()

	require.Equal(t, "file:config.json", file.New("config.json").String())
}
