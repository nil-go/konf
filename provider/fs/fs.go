// Copyright (c) 2026 The konf authors
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
	"os"
)

// FS is a Provider that loads configuration from file system.
//
// To create a new FS, call [New].
type FS struct {
	fs        fs.FS
	path      string
	unmarshal func([]byte, any) error
}

// New creates a FS with the given fs.FS, path and Option(s).
func New(fs fs.FS, path string, opts ...Option) FS {
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
	ffs := f.fs
	if ffs == nil {
		// Ignore error: It uses whatever returned.
		path, _ := os.Getwd()
		ffs = os.DirFS(path)
	}

	bytes, err := fs.ReadFile(ffs, f.path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var out map[string]any
	err = f.unmarshal(bytes, &out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return out, nil
}

func (f FS) String() string {
	return "fs:///" + f.path
}
