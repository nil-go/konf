// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf"
)

func BenchmarkNew(b *testing.B) {
	var (
		config *konf.Config
		err    error
	)
	for i := 0; i < b.N; i++ {
		config, err = konf.New(konf.WithLoader(mapLoader{"k": "v"}))
	}
	b.StopTimer()

	konf.SetGlobal(config)
	require.NoError(b, err)
	require.Equal(b, "v", konf.Get[string]("k"))
}

func BenchmarkGet(b *testing.B) {
	config, err := konf.New(konf.WithLoader(mapLoader{"k": "v"}))
	require.NoError(b, err)
	konf.SetGlobal(config)
	b.ResetTimer()

	var value string
	for i := 0; i < b.N; i++ {
		value = konf.Get[string]("k")
	}
	b.StopTimer()

	require.Equal(b, "v", value)
}

func BenchmarkWatch(b *testing.B) {
	watcher := mapWatcher(make(chan map[string]any))
	config, err := konf.New(konf.WithLoader(watcher))
	require.NoError(b, err)
	konf.SetGlobal(config)

	cfg := konf.Get[string]("config")
	require.Equal(b, "string", cfg)
	var waitGroup sync.WaitGroup
	waitGroup.Add(b.N)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := konf.Watch(ctx, func() {
			defer waitGroup.Done()

			cfg = konf.Get[string]("config")
		})
		require.NoError(b, err)
	}()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		watcher.change(map[string]any{"config": "changed"})
	}
	waitGroup.Wait()
	b.StopTimer()

	require.Equal(b, "changed", cfg)
}
