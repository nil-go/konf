// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package benchmark_test

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkViper_Get(b *testing.B) {
	assert.Equal(b, os.Getenv("USER"), viper.Get("user"))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = viper.Get("user")
		}
	})
}

func BenchmarkViper_Unmarshal(b *testing.B) {
	var value struct {
		User string
	}
	err := viper.Unmarshal(&value)
	require.NoError(b, err)
	assert.Equal(b, os.Getenv("USER"), value.User)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = viper.Unmarshal(&value)
		}
	})
}

func init() {
	_ = viper.BindEnv("USER")
}
