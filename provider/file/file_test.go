// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file_test

import (
	"errors"
	"testing"

	"github.com/nil-go/konf/provider/file"
	"github.com/nil-go/konf/provider/file/internal/assert"
)

func TestFile_New_panic(t *testing.T) {
	t.Parallel()

	defer func() {
		assert.Equal(t, "cannot create File with empty path", recover().(string))
	}()

	file.New("")
	t.Fail()
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
		testcase := testcase

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

	assert.Equal(t, "file:config.json", file.New("config.json").String())
}
