package konf

import (
	"context"
	"errors"
	"fmt"
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
func (c *Config) Watch(ctx context.Context) error { //nolint:cyclop,funlen,gocognit
	if ctx == nil {
		panic("cannot watch change with nil context")
	}

	if hasWatcher := slices.ContainsFunc(c.providers, func(provider *provider) bool {
		_, ok := provider.loader.(Watcher)

		return ok
	}); !hasWatcher {
		return nil
	}

	watched := true
	c.watchOnce.Do(func() {
		watched = false
	})
	if watched {
		c.logger.Warn("Config has been watched, call Watch again has no effects.")

		return nil
	}

	onChangesChannel := make(chan []func(*Config))
	defer close(onChangesChannel)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
				c.logger.DebugContext(ctx, "Configuration has been updated with change.")

				if len(onChanges) > 0 {
					func() {
						ctx, cancel = context.WithTimeout(ctx, time.Minute)
						defer cancel()

						done := make(chan struct{})
						go func() {
							defer close(done)

							for _, onChange := range onChanges {
								onChange(c)
							}
						}()

						select {
						case <-done:
							c.logger.DebugContext(ctx, "Configuration has been applied to onChanges.")
						case <-ctx.Done():
							if errors.Is(ctx.Err(), context.DeadlineExceeded) {
								c.logger.WarnContext(ctx, "Configuration has not been fully applied to onChanges due to timeout."+
									" Please check if the onChanges is blocking or takes too long to complete.")
							}
						}
					}()
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	errChan := make(chan error, len(c.providers))
	for _, provider := range c.providers {
		provider := provider

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
						c.onChangesMutex.RLock()
						defer c.onChangesMutex.RUnlock()

						var callbacks []func(*Config)
						for path, onChanges := range c.onChanges {
							keys := strings.Split(path, c.delimiter)
							if sub(oldValues, keys) != nil || sub(newValues, keys) != nil {
								callbacks = append(callbacks, onChanges...)
							}
						}

						return callbacks
					}
					onChangesChannel <- onChanges()

					c.logger.Info(
						"Configuration has been changed.",
						"loader", watcher,
					)
				}

				c.logger.DebugContext(ctx, "Watching configuration change.", "loader", watcher)
				if err := watcher.Watch(ctx, onChange); err != nil {
					errChan <- fmt.Errorf("watch configuration change: %w", err)
					cancel()
				}
			}()
		}
	}
	waitGroup.Wait()
	close(errChan)

	var err error
	for e := range errChan {
		err = errors.Join(e)
	}

	return err
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
func (c *Config) OnChange(onChange func(*Config), paths ...string) {
	if onChange == nil {
		panic("cannot register nil onChange")
	}

	c.onChangesMutex.Lock()
	defer c.onChangesMutex.Unlock()

	if len(paths) == 0 {
		paths = []string{""}
	}

	for _, path := range paths {
		path = strings.ToLower(path)
		c.onChanges[path] = append(c.onChanges[path], onChange)
	}
}
