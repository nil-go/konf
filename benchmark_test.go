// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"os"
	"testing"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
)

func BenchmarkLoad(b *testing.B) {
	var config konf.Config
	err := config.Load(mapLoader{"k": "v"})
	assert.NoError(b, err)
	var value string
	assert.NoError(b, config.Unmarshal("k", &value))
	assert.Equal(b, "v", value)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var cfg konf.Config
			_ = cfg.Load(mapLoader{"k": "v"})
		}
	})
}

func BenchmarkGet(b *testing.B) {
	assert.Equal(b, os.Getenv("USER"), konf.Get[Value]("").User)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = konf.Get[Value]("")
		}
	})
}

func BenchmarkUnmarshal(b *testing.B) {
	var value Value
	err := konf.Unmarshal("", &value)
	assert.NoError(b, err)
	assert.Equal(b, os.Getenv("USER"), value.User)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = konf.Unmarshal("", &value)
		}
	})
}

type Value struct {
	User string
}
