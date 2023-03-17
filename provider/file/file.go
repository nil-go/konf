// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package file loads configuration from files.
//
// File loads file with given path from OS file system and returns nested map[string]any
// that is parsed as json.
//
// The default behavior can be changed with following options:
//   - WithFS provides the fs.FS that config file is loaded from.
//     E.g. `WithFS(cfg)` will load configuration from embed file while cfg is embed.FS.
//   - WithUnmarshal provides the function that parses config file.
//     E.g. `WithUnmarshal(yaml.Unmarshal)` will parse the file as yaml.
//   - IgnoreFileNotExit ignores the error if config file does not exist.
package file

import (
	"fmt"
	"io/fs"
	"log"
	"os"
)

// File is a Provider that loads configuration from file.
type File struct {
	_              [0]func() // Ensure it's incomparable.
	fs             fs.FS
	path           string
	unmarshal      func([]byte, any) error
	ignoreNotExist bool
}

// New returns a File with the given path and Option(s).
func New(path string, opts ...Option) File {
	return File(apply(path, opts))
}

func (f File) Load() (map[string]any, error) {
	var (
		bytes []byte
		err   error
	)
	if f.fs == nil {
		bytes, err = os.ReadFile(f.path)
	} else {
		bytes, err = fs.ReadFile(f.fs, f.path)
	}
	if err != nil {
		if f.ignoreNotExist && os.IsNotExist(err) {
			log.Printf("Config file %s does not exist.", f.path)

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
	if f.fs == nil {
		return "os file:" + f.path
	}

	return "fs file:" + f.path
}
