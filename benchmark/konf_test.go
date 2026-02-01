// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package benchmark_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nil-go/konf"
)

func BenchmarkKonf_Get(b *testing.B) {
	assert.Equal(b, os.Getenv("USER"), konf.Get[string]("user"))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = konf.Get[string]("user")
		}
	})
}

func BenchmarkKonf_Unmarshal(b *testing.B) {
	var value struct {
		User string
	}
	err := konf.Unmarshal("", &value)
	require.NoError(b, err)
	assert.Equal(b, os.Getenv("USER"), value.User)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = konf.Unmarshal("", &value)
		}
	})
}
