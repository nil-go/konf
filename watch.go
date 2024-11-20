// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"
)

// Watch watches and updates configuration when it changes.
// It blocks until ctx is done, or the service returns an error.
// WARNING: All loaders passed in Load after calling Watch do not get watched.
//
// It only can be called once. Call after first has no effects.
func (c *Config) Watch(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
	c.nocopy.Check()

	if watched := c.watched.Swap(true); watched {
		c.log(ctx, slog.LevelWarn, "Config has been watched, call Watch more than once has no effects.")

		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// Start a goroutine to update the configuration while it has changes from watchers.
	onChangesChannel := make(chan []func(*Config), 1)
	defer close(onChangesChannel)

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		for {
			select {
			case <-ctx.Done():
				return

			case onChanges := <-onChangesChannel:
				c.providers.sync()
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

						tctx, tcancel := context.WithTimeout(ctx, time.Minute)
						defer tcancel()
						select {
						case <-done:
							c.log(ctx, slog.LevelDebug, "Configuration has been applied to onChanges.")
						case <-tctx.Done():
							if errors.Is(tctx.Err(), context.DeadlineExceeded) {
								c.log(
									ctx, slog.LevelWarn,
									"Configuration has not been fully applied to onChanges in one minute."+
										" Please check if the onChanges is blocking or takes too long to complete.",
								)
							}
						}
					}()
				}
			}
		}
	}()

	// Start a watching goroutine for each watcher registered.
	c.providers.traverse(func(provider *provider) {
		if watcher, ok := provider.loader.(Watcher); ok {
			waitGroup.Add(1)
			go func(ctx context.Context) {
				defer waitGroup.Done()

				onChange := func(values map[string]any) {
					c.transformKeys(values)
					oldValues := *provider.values.Swap(&values)
					onChangesChannel <- c.onChanges.get(
						func(path string) bool {
							return !reflect.DeepEqual(c.sub(oldValues, path), c.sub(values, path))
						},
					)

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
	})
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
	if onChange == nil {
		return // Do nothing is onchange is nil.
	}
	c.nocopy.Check()

	if !c.caseSensitive {
		for i := range paths {
			paths[i] = defaultKeyMap(paths[i])
		}
	}
	c.onChanges.register(onChange, paths)
}

type onChanges struct {
	subscribers map[string][]func(*Config)
	mutex       sync.RWMutex
}

func (o *onChanges) register(onChange func(*Config), paths []string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if len(paths) == 0 {
		paths = []string{""}
	}

	if o.subscribers == nil {
		o.subscribers = make(map[string][]func(*Config))
	}
	for _, path := range paths {
		o.subscribers[path] = append(o.subscribers[path], onChange)
	}
}

func (o *onChanges) get(filter func(string) bool) []func(*Config) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	var callbacks []func(*Config)
	for path, subscriber := range o.subscribers {
		if filter(path) {
			callbacks = append(callbacks, subscriber...)
		}
	}

	return callbacks
}
