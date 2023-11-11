// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"bytes"
	"context"
	"log"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	require.NoError(t, err)
	konf.SetGlobal(cfg)

	var v string
	require.NoError(t, konf.Unmarshal("config", &v))
	require.Equal(t, "string", v)
}

func TestGet(t *testing.T) {
	t.Parallel()

	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	require.NoError(t, err)
	konf.SetGlobal(cfg)

	require.Equal(t, "string", konf.Get[string]("config"))
}

func TestGet_error(t *testing.T) {
	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	require.NoError(t, err)
	konf.SetGlobal(cfg)

	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	log.SetFlags(0)

	require.False(t, konf.Get[bool]("config"))
	expected := "ERROR Could not read config, return empty value instead." +
		" error=\"[konf] decode: cannot parse '' as bool: strconv.ParseBool: parsing \\\"string\\\": invalid syntax\"" +
		" path=config type=bool\n"
	require.Equal(t, expected, buf.String())
}

func TestWatch(t *testing.T) {
	watcher := mapWatcher(make(chan map[string]any))
	config, err := konf.New(konf.WithLoader(watcher))
	require.NoError(t, err)
	konf.SetGlobal(config)

	cfg := konf.Get[string]("config")
	require.Equal(t, "string", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		err := konf.Watch(ctx, func() {
			defer waitGroup.Done()

			cfg = konf.Get[string]("config")
		})
		require.NoError(t, err)
	}()

	watcher.change(map[string]any{"config": "changed"})
	waitGroup.Wait()

	require.Equal(t, "changed", cfg)
}
