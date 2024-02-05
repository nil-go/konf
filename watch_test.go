// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/provider/env"
)

func TestConfig_Watch(t *testing.T) {
	t.Parallel()

	config := konf.New()
	watcher := mapWatcher(make(chan map[string]any))
	err := config.Load(watcher)
	assert.NoError(t, err)

	var value string
	assert.NoError(t, config.Unmarshal("config", &value))
	assert.Equal(t, "string", value)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		assert.NoError(t, config.Watch(ctx))
	}()

	var newValue atomic.Value
	config.OnChange(func(config *konf.Config) {
		defer waitGroup.Done()

		var value string
		assert.NoError(t, config.Unmarshal("config", &value))
		newValue.Store(value)
	}, "config")
	watcher.change(map[string]any{"Config": "changed"})
	waitGroup.Wait()
	assert.Equal(t, "changed", newValue.Load())
}

func TestConfig_Watch_onchange_block(t *testing.T) {
	t.Parallel()

	buf := new(buffer)
	config := konf.New(konf.WithLogHandler(logHandler(buf)))
	watcher := mapWatcher(make(chan map[string]any))
	err := config.Load(watcher)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		waitGroup.Done()
		assert.NoError(t, config.Watch(ctx))
	}()
	waitGroup.Wait()

	config.OnChange(func(config *konf.Config) {
		time.Sleep(time.Minute)
	})
	watcher.change(map[string]any{"Config": "changed"})

	<-ctx.Done()
	time.Sleep(time.Millisecond)
	expected := "level=INFO msg=\"Configuration has been changed.\" loader=mapWatcher\n" +
		"level=WARN msg=\"Configuration has not been fully applied to onChanges due to timeout." +
		" Please check if the onChanges is blocking or takes too long to complete.\"\n"
	assert.Equal(t, expected, buf.String())
}

func TestConfig_Watch_without_loader(t *testing.T) {
	t.Parallel()

	config := konf.New()
	assert.NoError(t, config.Load(env.New()))
	assert.NoError(t, config.Watch(context.Background()))
}

func TestConfig_Watch_twice(t *testing.T) {
	t.Parallel()

	buf := new(buffer)
	config := konf.New(konf.WithLogHandler(logHandler(buf)))
	assert.NoError(t, config.Load(mapWatcher(make(chan map[string]any))))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		waitGroup.Done()
		assert.NoError(t, config.Watch(ctx))
	}()
	waitGroup.Wait()
	assert.NoError(t, config.Watch(ctx))
	expected := "level=WARN msg=\"Config has been watched, call Watch again has no effects.\"\n"
	assert.Equal(t, expected, buf.String())
}

func TestConfig_Watch_panic(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		call        func(*konf.Config)
		err         string
	}{
		{
			description: "watch",
			call: func(config *konf.Config) {
				_ = config.Watch(nil) //nolint:staticcheck
			},
			err: "cannot watch change with nil context",
		},
		{
			description: "onchange",
			call: func(config *konf.Config) {
				config.OnChange(nil)
			},
			err: "cannot register nil onChange",
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					assert.Equal(t, r.(string), testcase.err)
				}
			}()
			config := konf.New()
			assert.NoError(t, config.Load(mapWatcher(make(chan map[string]any))))
			testcase.call(config)
			t.Fail()
		})
	}
}

type mapWatcher chan map[string]any

func (m mapWatcher) Load() (map[string]any, error) {
	return map[string]any{"Config": "string"}, nil
}

func (m mapWatcher) Watch(ctx context.Context, fn func(map[string]any)) error {
	for {
		select {
		case values := <-m:
			fn(values)
		case <-ctx.Done():
			return nil
		}
	}
}

func (m mapWatcher) change(values map[string]any) {
	m <- values
}

func (m mapWatcher) String() string {
	return "mapWatcher"
}

func TestConfig_Watch_error(t *testing.T) {
	t.Parallel()

	config := konf.New()
	err := config.Load(errorWatcher{})
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	assert.EqualError(t, config.Watch(ctx), "watch configuration change: watch error")
}

type errorWatcher struct{}

func (errorWatcher) Load() (map[string]any, error) {
	return make(map[string]any), nil
}

func (errorWatcher) Watch(context.Context, func(map[string]any)) error {
	return errors.New("watch error")
}

func logHandler(buf *buffer) *slog.TextHandler {
	return slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == slog.TimeKey {
				return slog.Attr{}
			}

			return attr
		},
	})
}

type buffer struct {
	b bytes.Buffer
	m sync.RWMutex
}

func (b *buffer) Read(p []byte) (int, error) {
	b.m.RLock()
	defer b.m.RUnlock()

	return b.b.Read(p)
}

func (b *buffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p)
}

func (b *buffer) String() string {
	b.m.RLock()
	defer b.m.RUnlock()

	return b.b.String()
}

func TestConfig_error(t *testing.T) {
	t.Parallel()

	config := konf.New()
	err := config.Load(errorLoader{})
	assert.EqualError(t, err, "load configuration: load error")
}

type errorLoader struct{}

func (errorLoader) Load() (map[string]any, error) {
	return nil, errors.New("load error")
}
