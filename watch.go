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
// It panics if ctx is nil.
func (c Config) Watch(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
	if ctx == nil {
		panic("cannot watch change with nil context")
	}

	if hasWatcher := slices.ContainsFunc(c.values.providers, func(provider provider) bool {
		_, ok := provider.loader.(Watcher)

		return ok
	}); !hasWatcher {
		return nil
	}

	if watched := c.values.watched.Swap(true); watched {
		c.logger.LogAttrs(ctx, slog.LevelWarn, "Config has been watched, call Watch again has no effects.")

		return nil
	}

	onChangesChannel := make(chan []func(Config), 1)
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
				for _, w := range c.values.providers {
					maps.Merge(values, w.values)
				}
				c.values.values = values
				c.logger.LogAttrs(ctx, slog.LevelDebug, "Configuration has been updated with change.")

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
							c.logger.LogAttrs(ctx, slog.LevelDebug, "Configuration has been applied to onChanges.")
						case <-ctx.Done():
							if errors.Is(ctx.Err(), context.DeadlineExceeded) {
								c.logger.LogAttrs(
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

	for i := range c.values.providers {
		provider := &c.values.providers[i] // Use pointer for later modification.

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
					onChanges := func() []func(Config) {
						c.values.onChangesMutex.RLock()
						defer c.values.onChangesMutex.RUnlock()

						var callbacks []func(Config)
						for path, onChanges := range c.values.onChanges {
							keys := strings.Split(path, c.delimiter)
							oldVal := maps.Sub(oldValues, keys)
							newVal := maps.Sub(newValues, keys)
							if !reflect.DeepEqual(oldVal, newVal) {
								callbacks = append(callbacks, onChanges...)
							}
						}

						return callbacks
					}
					onChangesChannel <- onChanges()

					c.logger.LogAttrs(
						context.Background(),
						slog.LevelInfo,
						"Configuration has been changed.",
						slog.Any("loader", watcher),
					)
				}

				c.logger.LogAttrs(ctx, slog.LevelDebug, "Watching configuration change.", slog.Any("loader", watcher))
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
// The onChange function must be non-blocking and usually completes instantly.
// If it requires a long time to complete, it should be executed in a separate goroutine.
//
// This method is concurrency-safe.
// It panics if onChange is nil.
func (c Config) OnChange(onChange func(Config), paths ...string) {
	if onChange == nil {
		panic("cannot register nil onChange")
	}

	c.values.onChangesMutex.Lock()
	defer c.values.onChangesMutex.Unlock()

	if len(paths) == 0 {
		paths = []string{""}
	}

	for _, path := range paths {
		path = strings.ToLower(path)
		c.values.onChanges[path] = append(c.values.onChanges[path], onChange)
	}
}
