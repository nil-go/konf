// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package file loads configuration from files.
//
// File loads file with given path from OS file system
// and returns nested map[string]any that is parsed as json.
//
// The default behavior can be changed with following options:
//   - WithUnmarshal provides the function that parses config file.
//     E.g. `WithUnmarshal(yaml.Unmarshal)` will parse the file as yaml.
//   - IgnoreFileNotExit ignores the error if config file does not exist.
package file

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// File is a Provider that loads configuration from file.
type File struct {
	path           string
	unmarshal      func([]byte, any) error
	ignoreNotExist bool
}

// New returns a File with the given path and Option(s).
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
