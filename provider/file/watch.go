// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (f *File) Status(onStatus func(bool, error)) {
	f.onStatus = onStatus
}

//nolint:cyclop,funlen
func (f *File) Watch(ctx context.Context, onChange func(map[string]any)) (err error) { //nolint:gocognit,nonamedreturns
	if f == nil {
		return errNil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher for %s: %w", f.path, err)
	}
	defer func() {
		e := watcher.Close()
		if e != nil {
			err = errors.Join(err, e)
		}
	}()

	// Although only a single file is being watched, fsnotify has to watch
	// the whole parent directory to pick up all events such as symlink changes.
	dir, _ := filepath.Split(f.path)
	e := watcher.Add(dir)
	if e != nil {
		return fmt.Errorf("watch dir %s: %w", dir, e)
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
		case event := <-watcher.Events:
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
				if f.onStatus != nil {
					f.onStatus(true, nil)
				}
				onChange(nil)
			case event.Has(fsnotify.Create) || event.Has(fsnotify.Write):
				values, err := f.Load()
				if f.onStatus != nil {
					f.onStatus(true, err)
				}
				onChange(values)
			}

		case err := <-watcher.Errors:
			if f.onStatus != nil {
				f.onStatus(false, err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
