// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package env_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf/provider/env"
)

func BenchmarkNew(b *testing.B) {
	var loader env.Env
	for i := 0; i < b.N; i++ {
		loader = env.New()
	}
	b.StopTimer()

	values, err := loader.Load()
	require.NoError(b, err)
	require.NotEmpty(b, values["USER"])
}

func BenchmarkLoad(b *testing.B) {
	loader := env.New()
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
	require.NotEmpty(b, values["USER"])
}
