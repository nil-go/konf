// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package fs_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/nil-go/konf/internal/assert"
	kfs "github.com/nil-go/konf/provider/fs"
)

func TestFS_New_panic(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		call        func()
		err         string
	}{
		{
			description: "fs",
			call: func() {
				kfs.New(nil, "config.json")
			},
			err: "cannot create FS with nil fs",
		},
		{
			description: "path",
			call: func() {
				kfs.New(fstest.MapFS{}, "")
			},
			err: "cannot create FS with empty path",
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					assert.Equal(t, r.(string), testcase.err)
				}
			}()
			testcase.call()
		})
	}
}

func TestFS_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		fs          fs.FS
		path        string
		opts        []kfs.Option
		expected    map[string]any
		err         string
	}{
		{
			description: "fs file",
			fs: fstest.MapFS{
				"config.json": {
					Data: []byte(`{"p":{"k":"v"}}`),
				},
			},
			path: "config.json",
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
		{
			description: "fs file (not exist)",
			fs:          fstest.MapFS{},
			path:        "not_found.json",
			err:         "read file: open not_found.json: file does not exist",
		},
		{
			description: "unmarshal error",
			fs: fstest.MapFS{
				"config.json": {
					Data: []byte(`{"p":{"k":"v"}}`),
				},
			},
			path: "config.json",
			opts: []kfs.Option{
				kfs.WithUnmarshal(func([]byte, any) error {
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

			values, err := kfs.New(testcase.fs, testcase.path, testcase.opts...).Load()
			if testcase.err != "" {
				assert.EqualError(t, err, testcase.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testcase.expected, values)
			}
		})
	}
}

func TestFS_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fs:config.json", kfs.New(fstest.MapFS{}, "config.json").String())
}
