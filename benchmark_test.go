// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/ktong/konf"
	"github.com/ktong/konf/internal/assert"
)

func BenchmarkNew(b *testing.B) {
	var (
		config konf.Config
		err    error
	)
	for i := 0; i < b.N; i++ {
		config, err = konf.New(konf.WithLoader(mapLoader{"k": "v"}))
	}
	b.StopTimer()

	konf.SetGlobal(config)
	assert.NoError(b, err)
	assert.Equal(b, "v", konf.Get[string]("k"))
}

func BenchmarkGet(b *testing.B) {
	config, err := konf.New(konf.WithLoader(mapLoader{"k": "v"}))
	assert.NoError(b, err)
	konf.SetGlobal(config)
	b.ResetTimer()

	var value string
	for i := 0; i < b.N; i++ {
		value = konf.Get[string]("k")
	}
	b.StopTimer()

	assert.Equal(b, "v", value)
}

func BenchmarkWatch(b *testing.B) {
	watcher := mapWatcher(make(chan map[string]any))
	config, err := konf.New(konf.WithLoader(watcher))
	assert.NoError(b, err)
	konf.SetGlobal(config)

	assert.Equal(b, "string", konf.Get[string]("config"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		assert.NoError(b, konf.Watch(ctx))
	}()
	b.ResetTimer()

	var cfg atomic.Value
	konf.OnChange(func() {
		cfg.Store(konf.Get[string]("config"))
	})
	for i := 0; i < b.N; i++ {
		watcher.change(map[string]any{"config": "changed"})
	}
	b.StopTimer()
	assert.Equal(b, "changed", cfg.Load())
}
