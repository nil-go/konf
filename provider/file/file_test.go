// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package file_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nil-go/konf/provider/file"
	"github.com/nil-go/konf/provider/file/internal/assert"
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
			if testcase.err != "" {
				assert.True(t, strings.HasPrefix(err.Error(), testcase.err))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testcase.expected, values)
			}
		})
	}
}

func TestFile_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "file:config.json", file.New("config.json").String())
}
