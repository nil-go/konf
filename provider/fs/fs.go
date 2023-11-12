// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package fs loads configuration from file system.
//
// FS loads file with given path from file system
// and returns nested map[string]any that is parsed as json.
//
// The default behavior can be changed with following options:
//   - WithUnmarshal provides the function that parses config file.
//     E.g. `WithUnmarshal(yaml.Unmarshal)` will parse the file as yaml.
//   - IgnoreFileNotExit ignores the error if config file does not exist.
package fs

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
)

// FS is a Provider that loads configuration from file system.
type FS struct {
	fs             fs.FS
	path           string
	unmarshal      func([]byte, any) error
	ignoreNotExist bool
}

// New returns a FS with the given fs.FS, path and Option(s).
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
