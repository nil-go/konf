// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package fs loads configuration from file system.
//
// FS loads a file with the given path from the file system and returns
// a nested map[string]any that is parsed with the given unmarshal function.
//
// The unmarshal function must be able to unmarshal the file content into a map[string]any.
// For example, with the default json.Unmarshal, the file is parsed as JSON.
package fs

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// FS is a Provider that loads configuration from file system.
//
// To create a new FS, call [New].
type FS struct {
	unmarshal func([]byte, any) error
	fs        fs.FS
	path      string
}

// New creates a FS with the given fs.FS, path and Option(s).
//
// It panics if the fs is nil or the path is empty.
func New(fs fs.FS, path string, opts ...Option) FS { //nolint:varnamelen
	if fs == nil {
		panic("cannot create FS with nil fs")
	}
	if path == "" {
		panic("cannot create FS with empty path")
	}

	option := &options{
		fs:   fs,
		path: path,
	}
	for _, opt := range opts {
		opt(option)
	}
	if option.unmarshal == nil {
		option.unmarshal = json.Unmarshal
	}

	return FS(*option)
}

func (f FS) Load() (map[string]any, error) {
	bytes, err := fs.ReadFile(f.fs, f.path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var out map[string]any
	if err := f.unmarshal(bytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return out, nil
}

func (f FS) String() string {
	return "fs:" + f.path
}
