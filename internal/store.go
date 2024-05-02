// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"strings"

	kmaps "github.com/nil-go/konf/internal/maps"
)

type Store struct {
	values    map[string]any
	keyMap    func(s string) string
	delimiter string
}

func NewStore(values map[string]any, delimiter string, keyMap func(s string) string) *Store {
	return &Store{
		values:    values,
		delimiter: delimiter,
		keyMap:    keyMap,
	}
}

func (s *Store) Merge(store *Store) {
	kmaps.Merge(s.values, store.values, s.keyMap)
}

func (s *Store) Sub(path string) any {
	return kmaps.Sub(s.values, strings.Split(path, s.delimiter), s.keyMap)
}

func (s *Store) Exist(path []string) bool {
	return kmaps.Sub(s.values, path, s.keyMap) != nil
}
