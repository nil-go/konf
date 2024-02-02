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
	logger         *slog.Logger
	path           string
	unmarshal      func([]byte, any) error
	ignoreNotExist bool
}

// New creates a File with the given path and Option(s).
//
// It panics if the path is empty.
func New(path string, opts ...Option) File {
	if path == "" {
		panic("cannot create File with empty path")
	}

	option := &options{
		path: path,
	}
	for _, opt := range opts {
		opt(option)
	}
	if option.logger == nil {
		option.logger = slog.Default()
	}
	option.logger = option.logger.WithGroup("konf.file")
	if option.unmarshal == nil {
		option.unmarshal = json.Unmarshal
	}

	return File(*option)
}

func (f File) Load() (map[string]any, error) {
	bytes, err := os.ReadFile(f.path)
	if err != nil {
		if f.ignoreNotExist && os.IsNotExist(err) {
			f.logger.Warn("Config file does not exist.", "file", f.path)

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
