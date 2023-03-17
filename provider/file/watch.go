// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build darwin || dragonfly || freebsd || openbsd || linux || netbsd || solaris || windows

package file

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch watches the file and triggers a callback when it changes.
// It blocks until ctx is done, or the service returns a non-retryable error.
//
//nolint:cyclop,funlen,gocognit
func (f File) Watch(ctx context.Context, watchFunc func(map[string]any)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher for %s: %w", f.path, err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Printf("Error when closing watcher for %s: %v", f.path, err)
		}
	}()

	// Although only a single file is being watched, fsnotify has to watch
	// the whole parent directory to pick up all events such as symlink changes.
	dir, _ := filepath.Split(f.path)
	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("watch dir %s: %w", dir, err)
	}

	// Resolve symlinks and save the original path so that changes to symlinks
	// can be detected.
	realPath, err := filepath.EvalSymlinks(f.path)
	if err != nil {
		return fmt.Errorf("eval symlike: %w", err)
	}
	realPath = filepath.Clean(realPath)

	var (
		lastEvent     string
		lastEventTime time.Time
	)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Use a simple timer to buffer events as certain events fire
			// multiple times on some platforms.
			if event.String() == lastEvent && time.Since(lastEventTime) < 5*time.Millisecond {
				continue
			}
			lastEvent = event.String()
			lastEventTime = time.Now()

			// Since the event is triggered on a directory, is this
			// one on the file being watched?
			evFile := filepath.Clean(event.Name)
			if evFile != realPath && evFile != f.path {
				continue
			}

			switch {
			case event.Has(fsnotify.Remove):
				log.Printf("Config file %s has been removed.", f.path)
				watchFunc(nil)
			case event.Has(fsnotify.Create) || event.Has(fsnotify.Write):
				values, err := f.Load()
				if err != nil {
					log.Printf("Error when reloading configuration from %s: %v", f.path, err)

					continue
				}
				watchFunc(values)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			log.Printf("Error when watching file %s: %v", f.path, err)

		case <-ctx.Done():
			return nil
		}
	}
}
