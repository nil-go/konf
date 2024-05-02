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
	"sync"
	"time"

	"github.com/nil-go/konf/internal"
)

// Watch watches and updates configuration when it changes.
// It blocks until ctx is done, or the service returns an error.
// WARNING: All loaders passed in Load after calling Watch do not get watched.
//
// It only can be called once. Call after first has no effects.
func (c *Config) Watch(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
	c.nocopy.Check()

	if hasWatcher := slices.ContainsFunc(c.providers, func(provider provider) bool {
		_, ok := provider.loader.(Watcher)

		return ok
	}); !hasWatcher {
		return nil
	}

	if watched := c.watched.Swap(true); watched {
		c.log(ctx, slog.LevelWarn, "Config has been watched, call Watch again has no effects.")

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
				values := internal.NewStore(make(map[string]any), c.delim(), c.keyMap())
				for _, w := range c.providers {
					values.Merge(w.values)
				}
				c.values = values
				c.log(ctx, slog.LevelDebug, "Configuration has been updated with change.")

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
							c.log(ctx, slog.LevelDebug, "Configuration has been applied to onChanges.")
						case <-ctx.Done():
							if errors.Is(ctx.Err(), context.DeadlineExceeded) {
								c.log(
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
			go func(ctx context.Context) {
				defer waitGroup.Done()

				onChange := func(values map[string]any) {
					oldValues := provider.values
					newValues := internal.NewStore(values, c.delim(), c.keyMap())
					provider.values = newValues

					// Find the onChanges should be triggered.
					onChanges := func() []func(*Config) {
						c.onChangesMutex.RLock()
						defer c.onChangesMutex.RUnlock()

						var callbacks []func(*Config)
						for path, onChanges := range c.onChanges {
							oldVal := oldValues.Sub(path)
							newVal := newValues.Sub(path)
							if !reflect.DeepEqual(oldVal, newVal) {
								callbacks = append(callbacks, onChanges...)
							}
						}

						return callbacks
					}
					onChangesChannel <- onChanges()

					c.log(ctx, slog.LevelInfo,
						"Configuration has been changed.",
						slog.Any("loader", watcher),
					)
				}

				c.log(ctx, slog.LevelDebug, "Watching configuration change.", slog.Any("loader", watcher))
				if err := watcher.Watch(ctx, onChange); err != nil {
					cancel(fmt.Errorf("watch configuration change on %v: %w", watcher, err))
				}
			}(ctx)
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
// The paths are case-insensitive unless konf.WithCaseSensitive is set.
//
// The register function must be non-blocking and usually completes instantly.
// If it requires a long time to complete, it should be executed in a separate goroutine.
//
// This method is concurrency-safe.
func (c *Config) OnChange(onChange func(*Config), paths ...string) {
	c.nocopy.Check()

	if onChange == nil {
		return // Do nothing is onchange is nil.
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	c.onChangesMutex.Lock()
	defer c.onChangesMutex.Unlock()

	if c.onChanges == nil {
		c.onChanges = make(map[string][]func(*Config))
	}
	for _, path := range paths {
		c.onChanges[path] = append(c.onChanges[path], onChange)
	}
}
