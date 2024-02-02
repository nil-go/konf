// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package pflag_test

import (
	"testing"

	"github.com/spf13/pflag"

	kflag "github.com/nil-go/konf/provider/pflag"
	"github.com/nil-go/konf/provider/pflag/internal/assert"
)

func BenchmarkNew(b *testing.B) {
	set := &pflag.FlagSet{}
	set.String("k", "v", "")
	b.ResetTimer()

	var loader kflag.PFlag
	for i := 0; i < b.N; i++ {
		loader = kflag.New(konf{}, kflag.WithFlagSet(set))
	}
	b.StopTimer()

	values, err := loader.Load()
	assert.NoError(b, err)
	assert.Equal(b, "v", values["k"])
}

func BenchmarkLoad(b *testing.B) {
	set := &pflag.FlagSet{}
	set.String("k", "v", "")
	loader := kflag.New(konf{}, kflag.WithFlagSet(set))
	b.ResetTimer()

	var (
		values map[string]any
		err    error
	)
	for i := 0; i < b.N; i++ {
		values, err = loader.Load()
	}
	b.StopTimer()

	assert.NoError(b, err)
	assert.Equal(b, "v", values["k"])
}

type konf struct {
	exists bool
}

func (k konf) Exists([]string) bool {
	return k.exists
}
