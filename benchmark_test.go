// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"testing"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
)

func BenchmarkLoad(b *testing.B) {
	var config konf.Config

	var err error
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = config.Load(mapLoader{"k": "v"})
	}
	b.StopTimer()

	assert.NoError(b, err)
	var value string
	assert.NoError(b, config.Unmarshal("k", &value))
	assert.Equal(b, "v", value)
}

func BenchmarkGet(b *testing.B) {
	var config konf.Config
	assert.NoError(b, config.Load(mapLoader{"k": "v"}))
	konf.SetDefault(&config)

	var value string
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value = konf.Get[string]("k")
	}
	b.StopTimer()

	assert.Equal(b, "v", value)
}

func BenchmarkUnmarshal(b *testing.B) {
	var config konf.Config
	assert.NoError(b, config.Load(mapLoader{"k": "v"}))
	konf.SetDefault(&config)

	var (
		value string
		err   error
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = konf.Unmarshal("k", &value)
	}
	b.StopTimer()

	assert.NoError(b, err)
	assert.Equal(b, "v", value)
}
