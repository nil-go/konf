// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/nil-go/konf/internal/maps"
)

type (
	values struct {
		values    map[string]any
		providers []provider
		watched   atomic.Bool
	}
	provider struct {
		loader Loader
		values map[string]any
	}
)

func (v *values) sub(paths []string) any {
	return maps.Sub(v.values, paths)
}

type onChange struct {
	onChanges      map[string][]func(Config)
	onChangesMutex sync.RWMutex
}

func (c *onChange) register(onChange func(Config), paths []string) {
	if c.onChanges == nil {
		c.onChanges = make(map[string][]func(Config))
	}

	c.onChangesMutex.Lock()
	defer c.onChangesMutex.Unlock()

	for _, path := range paths {
		path = strings.ToLower(path)
		c.onChanges[path] = append(c.onChanges[path], onChange)
	}
}

func (c *onChange) walk(fn func(path string, onChanges []func(Config))) {
	c.onChangesMutex.RLock()
	defer c.onChangesMutex.RUnlock()

	for path, onChanges := range c.onChanges {
		fn(path, onChanges)
	}
}
