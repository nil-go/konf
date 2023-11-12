// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package fs_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	pfs "github.com/ktong/konf/provider/fs"
)

func BenchmarkNew(b *testing.B) {
	mapFS := fstest.MapFS{
		"config.json": {
			Data: []byte(`{"k":"v"}`),
		},
	}
	b.ResetTimer()

	var loader pfs.FS
	for i := 0; i < b.N; i++ {
		loader = pfs.New(mapFS, "config.json")
	}
	b.StopTimer()

	values, err := loader.Load()
	require.NoError(b, err)
	require.Equal(b, "v", values["k"])
}

func BenchmarkLoad(b *testing.B) {
	fs := fstest.MapFS{
		"config.json": {
			Data: []byte(`{"k":"v"}`),
		},
	}
	loader := pfs.New(fs, "config.json")
	b.ResetTimer()

	var (
		values map[string]any
		err    error
	)
	for i := 0; i < b.N; i++ {
		values, err = loader.Load()
	}
	b.StopTimer()

	require.NoError(b, err)
	require.Equal(b, "v", values["k"])
}
