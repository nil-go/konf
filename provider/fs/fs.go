// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package fs loads configuration from file system.
//
// FS loads a file with the given path from the file system and returns
// a nested map[string]any that is parsed with the given unmarshal function.
//
// The unmarshal function must be able to unmarshal the file content into a map[string]any.
// For example, with the default json.Unmarshal, the file is parsed as JSON.
//
// By default, it returns error while loading if the file is not found.
// IgnoreFileNotExit can override the behavior to return an empty map[string]any.
package fs

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
)

// FS is a Provider that loads configuration from file system.
//
// To create a new FS, call [New].
type FS struct {
	unmarshal      func([]byte, any) error
	fs             fs.FS
	path           string
	ignoreNotExist bool
}

// New creates a FS with the given fs.FS, path and Option(s).
func New(fs fs.FS, path string, opts ...Option) FS {
	option := &options{
		fs:        fs,
		path:      path,
		unmarshal: json.Unmarshal,
	}
	for _, opt := range opts {
		opt(option)
	}

	return FS(*option)
}

func (f FS) Load() (map[string]any, error) {
	bytes, err := fs.ReadFile(f.fs, f.path)
	if err != nil {
		if f.ignoreNotExist && os.IsNotExist(err) {
			slog.Warn("Config file does not exist.", "file", f.path)

			return make(map[string]any), nil
		}

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
