// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/nil-go/konf/provider/file"
	"github.com/nil-go/konf/provider/file/internal/assert"
)

func TestFS_empty(t *testing.T) {
	var loader *file.File
	values, err := loader.Load()
	assert.EqualError(t, err, "nil File")
	assert.Equal(t, nil, values)
	err = loader.Watch(context.Background(), nil)
	assert.EqualError(t, err, "nil File")
}

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
			description: "empty path",
			err:         "read file: open : no such file or directory",
		},
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
			err:         "read file: open not_found.json: no such file or directory",
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

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			values, err := file.New(testcase.path, testcase.opts...).Load()
			if testcase.err != "" {
				assert.EqualError(t, err, testcase.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testcase.expected, values)
			}
		})
	}
}

func TestFile_String(t *testing.T) {
	t.Parallel()

	path, err := filepath.Abs("config.json")
	assert.NoError(t, err)
	assert.Equal(t, "file://"+path, file.New("config.json").String())
}
