// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"context"
	"testing"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	var config konf.Config
	err := config.Load(mapLoader{"config": "string"})
	assert.NoError(t, err)
	konf.SetDefault(&config)

	var v string
	assert.NoError(t, konf.Unmarshal("config", &v))
	assert.Equal(t, "string", v)
}

func TestGet(t *testing.T) {
	t.Parallel()

	var config konf.Config
	err := config.Load(mapLoader{"config": "string"})
	assert.NoError(t, err)
	konf.SetDefault(&config)

	assert.Equal(t, "string", konf.Get[string]("config"))
}

func TestGet_error(t *testing.T) {
	buf := &buffer{}
	config := konf.New(konf.WithLogHandler(logHandler(buf)))
	err := config.Load(mapLoader{"config": "string"})
	assert.NoError(t, err)
	konf.SetDefault(config)

	assert.True(t, !konf.Get[bool]("config"))
	expected := `level=WARN msg="Could not read config, return empty value instead."` +
		` path=config type=bool` +
		` error="decode: cannot parse '' as bool: strconv.ParseBool: parsing \"string\": invalid syntax"` +
		"\n"
	assert.Equal(t, expected, buf.String())
}

func TestOnChange(t *testing.T) {
	var config konf.Config
	watcher := stringWatcher{key: "Config", value: make(chan string)}
	err := config.Load(watcher)
	assert.NoError(t, err)
	konf.SetDefault(&config)

	var value string
	assert.NoError(t, config.Unmarshal("config", &value))
	assert.Equal(t, "", value)

	stopped := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		<-stopped
	}()
	go func() {
		defer close(stopped)

		assert.NoError(t, config.Watch(ctx))
	}()

	newValue := make(chan string)
	konf.OnChange(func() {
		var value string
		assert.NoError(t, konf.Unmarshal("config", &value))
		newValue <- value
	}, "config")
	watcher.change("changed")
	assert.Equal(t, "changed", <-newValue)
}
