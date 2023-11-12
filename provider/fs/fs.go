// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package fs loads configuration from files.
//
// FS loads file with given path from OS file system and returns nested map[string]any
// that is parsed as json.
//
// The default behavior can be changed with following options:
//   - WithFS provides the fs.FS that config file is loaded from.
//     E.g. `WithFS(cfg)` will load configuration from embed file while cfg is embed.FS.
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

// FS is a Provider that loads configuration from file.
type FS struct {
	fs             fs.FS
	path           string
	unmarshal      func([]byte, any) error
	ignoreNotExist bool
}

// New returns a FS with the given path and Option(s).
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
