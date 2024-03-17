// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package benchmark_test

import (
	"os"
	"strings"
	"testing"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkKoanf_Get(b *testing.B) {
	assert.Equal(b, os.Getenv("USER"), k.Get("user"))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = k.Get("user")
		}
	})
}

func BenchmarkKoanf_Unmarshal(b *testing.B) {
	var value struct {
		User string
	}
	err := k.Unmarshal("", &value)
	require.NoError(b, err)
	assert.Equal(b, os.Getenv("USER"), value.User)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = k.Unmarshal("", &value)
		}
	})
}

var k *koanf.Koanf

func init() {
	k = koanf.New(".")
	_ = k.Load(env.Provider("", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(s), "_", ".")
	}), nil)
}
