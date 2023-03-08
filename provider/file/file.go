// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file

import (
	"fmt"
	"io/fs"
	"os"
)

// File is a Provider that loads configuration from file.
type File struct {
	fs             fs.FS
	path           string
	unmarshal      func([]byte, any) error
	log            func(...any)
	ignoreNotExist bool
}

// New returns a File with the given path and Option(s).
func New(path string, opts ...Option) File {
	return File(apply(path, opts))
}

func (f File) Load() (map[string]any, error) {
	if f.fs == nil {
		bytes, err := os.ReadFile(f.path)
		if err != nil {
			return f.notExist(err)
		}

		return f.parse(bytes)
	}

	bytes, err := fs.ReadFile(f.fs, f.path)
	if err != nil {
		return f.notExist(err)
	}

	return f.parse(bytes)
}

func (f File) notExist(err error) (map[string]any, error) {
	if f.ignoreNotExist && os.IsNotExist(err) {
		f.log(fmt.Sprintf("Config file %s does not exist.", f.path))

		return make(map[string]any), nil
	}

	return nil, fmt.Errorf("[konf] read file: %w", err)
}

func (f File) parse(bytes []byte) (map[string]any, error) {
	var out map[string]any
	if err := f.unmarshal(bytes, &out); err != nil {
		return nil, fmt.Errorf("[konf] unmarshal: %w", err)
	}

	return out, nil
}
