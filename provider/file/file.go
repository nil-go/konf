// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package file loads configuration from OS file.
//
// File loads a file with the given path from the OS file system and returns
// a nested map[string]any that is parsed with the given unmarshal function.
//
// The unmarshal function must be able to unmarshal the file content into a map[string]any.
// For example, with the default json.Unmarshal, the file is parsed as JSON.
//
// By default, it returns error while loading if the file is not found.
// IgnoreFileNotExit can override the behavior to return an empty map[string]any.
package file

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// File is a Provider that loads configuration from a OS file.
//
// To create a new File, call [New].
type File struct {
	path           string
	unmarshal      func([]byte, any) error
	ignoreNotExist bool
}

// New creates a File with the given path and Option(s).
func New(path string, opts ...Option) File {
	option := &options{
		path:      path,
		unmarshal: json.Unmarshal,
	}
	for _, opt := range opts {
		opt(option)
	}

	return File(*option)
}

func (f File) Load() (map[string]any, error) {
	bytes, err := os.ReadFile(f.path)
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

func (f File) String() string {
	return "file:" + f.path
}
