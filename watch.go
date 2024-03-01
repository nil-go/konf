// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nil-go/konf/internal/maps"
)

// Watch watches and updates configuration when it changes.
// It blocks until ctx is done, or the service returns an error.
// WARNING: All loaders passed in Load after calling Watch do not get watched.
//
// It only can be called once. Call after first has no effects.
func (c *Config) Watch(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
	if hasWatcher := slices.ContainsFunc(c.providers, func(provider provider) bool {
		_, ok := provider.loader.(Watcher)

		return ok
	}); !hasWatcher {
		return nil
	}

	logger := c.logger
	if logger == nil {
		logger = slog.Default()
	}

	if watched := c.watched.Swap(true); watched {
		logger.LogAttrs(ctx, slog.LevelWarn, "Config has been watched, call Watch again has no effects.")

		return nil
	}

	onChangesChannel := make(chan []func(*Config), 1)
	defer close(onChangesChannel)
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		for {
			select {
			case onChanges := <-onChangesChannel:
				values := make(map[string]any)
				for _, w := range c.providers {
					maps.Merge(values, w.values)
				}
				c.values = values
				logger.LogAttrs(ctx, slog.LevelDebug, "Configuration has been updated with change.")

				if len(onChanges) > 0 {
					func() {
						done := make(chan struct{})
						go func() {
							defer close(done)

							for _, onChange := range onChanges {
								onChange(c)
							}
						}()

						var tcancel context.CancelFunc
						ctx, tcancel = context.WithTimeout(ctx, time.Minute)
						defer tcancel()
						select {
						case <-done:
							logger.LogAttrs(ctx, slog.LevelDebug, "Configuration has been applied to onChanges.")
						case <-ctx.Done():
							if errors.Is(ctx.Err(), context.DeadlineExceeded) {
								logger.LogAttrs(
									ctx, slog.LevelWarn,
									"Configuration has not been fully applied to onChanges due to timeout."+
										" Please check if the onChanges is blocking or takes too long to complete.",
								)
							}
						}
					}()
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	for i := range c.providers {
		provider := &c.providers[i] // Use pointer for later modification.

		if watcher, ok := provider.loader.(Watcher); ok {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()

				onChange := func(values map[string]any) {
					// Merged to empty map to convert to lower case.
					newValues := make(map[string]any)
					maps.Merge(newValues, values)

					oldValues := provider.values
					provider.values = newValues

					// Find the onChanges should be triggered.
					onChanges := func() []func(*Config) {
						var callbacks []func(*Config)
						c.onChange.walk(func(path string, onChanges []func(*Config)) {
							keys := c.split(path)
							oldVal := maps.Sub(oldValues, keys)
							newVal := maps.Sub(newValues, keys)
							if !reflect.DeepEqual(oldVal, newVal) {
								callbacks = append(callbacks, onChanges...)
							}
						})

						return callbacks
					}
					onChangesChannel <- onChanges()

					logger.LogAttrs(
						context.Background(),
						slog.LevelInfo,
						"Configuration has been changed.",
						slog.Any("loader", watcher),
					)
				}

				logger.LogAttrs(ctx, slog.LevelDebug, "Watching configuration change.", slog.Any("loader", watcher))
				if err := watcher.Watch(ctx, onChange); err != nil {
					cancel(fmt.Errorf("watch configuration change on %v: %w", watcher, err))
				}
			}()
		}
	}
	waitGroup.Wait()

	if err := context.Cause(ctx); err != nil && !errors.Is(err, ctx.Err()) {
		return err //nolint:wrapcheck
	}

	return nil
}

// OnChange registers a callback function that is executed
// when the value of any given path in the Config changes.
// It requires Config.Watch has been called first.
// The paths are case-insensitive.
//
// The register function must be non-blocking and usually completes instantly.
// If it requires a long time to complete, it should be executed in a separate goroutine.
//
// This method is concurrency-safe.
func (c *Config) OnChange(onChange func(*Config), paths ...string) {
	if onChange == nil {
		return // Do nothing is onchange is nil.
	}

	if len(paths) == 0 {
		paths = []string{""}
	}
	c.onChange.register(onChange, paths)
}

type onChange struct {
	onChanges      map[string][]func(*Config)
	onChangesMutex sync.RWMutex
}

func (c *onChange) register(onChange func(*Config), paths []string) {
	if c.onChanges == nil {
		c.onChanges = make(map[string][]func(*Config))
	}

	c.onChangesMutex.Lock()
	defer c.onChangesMutex.Unlock()

	for _, path := range paths {
		path = strings.ToLower(path)
		c.onChanges[path] = append(c.onChanges[path], onChange)
	}
}

func (c *onChange) walk(fn func(path string, onChanges []func(*Config))) {
	c.onChangesMutex.RLock()
	defer c.onChangesMutex.RUnlock()

	for path, onChanges := range c.onChanges {
		fn(path, onChanges)
	}
}
