// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"bytes"
	"context"
	"log"
	"sync"
	"testing"

	"github.com/ktong/konf"
	"github.com/ktong/konf/internal/assert"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	assert.NoError(t, err)
	konf.SetGlobal(cfg)

	var v string
	assert.NoError(t, konf.Unmarshal("config", &v))
	assert.Equal(t, "string", v)
}

func TestGet(t *testing.T) {
	t.Parallel()

	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	assert.NoError(t, err)
	konf.SetGlobal(cfg)

	assert.Equal(t, "string", konf.Get[string]("config"))
}

func TestGet_error(t *testing.T) {
	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	assert.NoError(t, err)
	konf.SetGlobal(cfg)

	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	log.SetFlags(0)

	assert.True(t, !konf.Get[bool]("config"))
	expected := "ERROR Could not read config, return empty value instead." +
		" error=\"[konf] decode: cannot parse '' as bool: strconv.ParseBool: parsing \\\"string\\\": invalid syntax\"" +
		" path=config type=bool\n"
	assert.Equal(t, expected, buf.String())
}

func TestWatch(t *testing.T) {
	watcher := mapWatcher(make(chan map[string]any))
	config, err := konf.New(konf.WithLoader(watcher))
	assert.NoError(t, err)
	konf.SetGlobal(config)

	cfg := konf.Get[string]("config")
	assert.Equal(t, "string", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		err := konf.Watch(ctx, func() {
			defer waitGroup.Done()

			cfg = konf.Get[string]("config")
		})
		assert.NoError(t, err)
	}()

	watcher.change(map[string]any{"config": "changed"})
	waitGroup.Wait()

	assert.Equal(t, "changed", cfg)
}
