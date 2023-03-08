// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file_test

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf/provider/file"
)

func TestFile(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		path        string
		opts        []file.Option
		expected    map[string]any
		err         string
	}{
		{
			description: "os file",
			path:        "testdata/config.json",
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
		{
			description: "os file (not exist)",
			path:        "not_found.json",
			err:         "[konf] read file: open not_found.json: no such file or directory",
		},
		{
			description: "os file (ignore not exist)",
			path:        "not_found.json",
			opts:        []file.Option{file.IgnoreFileNotExit()},
			expected:    map[string]any{},
		},
		{
			description: "fs file",
			path:        "config.json",
			opts: []file.Option{
				file.WithFS(fstest.MapFS{
					"config.json": {
						Data: []byte(`{"p":{"k":"v"}}`),
					},
				}),
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
		{
			description: "fs file (not exist)",
			path:        "not_found.json",
			opts: []file.Option{
				file.WithFS(fstest.MapFS{}),
			},
			err: "[konf] read file: open not_found.json: file does not exist",
		},
		{
			description: "fs file (ignore not exist)",
			path:        "not_found.json",
			opts: []file.Option{
				file.WithFS(fstest.MapFS{}),
				file.IgnoreFileNotExit(),
			},
			expected: map[string]any{},
		},
		{
			description: "unmarshal error",
			path:        "testdata/config.json",
			opts: []file.Option{
				file.WithUnmarshal(func([]byte, any) error {
					return errors.New("unmarshal error")
				}),
			},
			err: "[konf] unmarshal: unmarshal error",
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			loader, err := file.New(testcase.path, testcase.opts...).Load()
			if err != nil {
				require.EqualError(t, err, testcase.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testcase.expected, loader)
			}
		})
	}
}

func TestFile_log(t *testing.T) {
	var log []any
	_, err := file.New("not_found.json", file.IgnoreFileNotExit(), file.WithLog(func(a ...any) {
		log = append(log, a...)
	})).Load()
	require.NoError(t, err)
	require.Equal(t, []any{"Config file not_found.json does not exist."}, log)
}
