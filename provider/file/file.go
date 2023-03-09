// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// File is a Provider that loads configuration from file.
type File struct {
	fs             fs.FS
	path           string
	unmarshal      func([]byte, any) error
	log            func(...any)
	ignoreNotExist bool
}

// New returns a File with the given path and Option(s).
func New(path string, opts ...Option) File {
	return File(apply(path, opts))
}

func (f File) Load() (map[string]any, error) {
	if f.fs == nil {
		bytes, err := os.ReadFile(f.path)
		if err != nil {
			return f.notExist(err)
		}

		return f.parse(bytes)
	}

	bytes, err := fs.ReadFile(f.fs, f.path)
	if err != nil {
		return f.notExist(err)
	}

	return f.parse(bytes)
}

func (f File) notExist(err error) (map[string]any, error) {
	if f.ignoreNotExist && os.IsNotExist(err) {
		f.log(fmt.Sprintf("Config file %s does not exist.", f.path))

		return make(map[string]any), nil
	}

	return nil, fmt.Errorf("read file: %w", err)
}

func (f File) parse(bytes []byte) (map[string]any, error) {
	var out map[string]any
	if err := f.unmarshal(bytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return out, nil
}

// Watch watches the file and triggers a callback when it changes.
// It blocks until ctx is done, or the service returns a non-retryable error.
//
//nolint:cyclop,funlen,gocognit,gocyclo
func (f File) Watch(ctx context.Context, watchFunc func(map[string]any)) error {
	// Resolve symlinks and save the original path so that changes to symlinks
	// can be detected.
	realPath, err := filepath.EvalSymlinks(f.path)
	if err != nil {
		return fmt.Errorf("eval symlike: %w", err)
	}
	realPath = filepath.Clean(realPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher for %s: %w", f.path, err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			f.log(fmt.Sprintf("Error when closing watcher for %s: %v", f.path, err))
		}
	}()

	// Although only a single file is being watched, fsnotify has to watch
	// the whole parent directory to pick up all events such as symlink changes.
	dir, _ := filepath.Split(f.path)
	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("watch dir %s: %w", dir, err)
	}

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

			evFile := filepath.Clean(event.Name)

			// Since the event is triggered on a directory, is this
			// one on the file being watched?
			if evFile != realPath && evFile != f.path {
				continue
			}

			// The file was removed.
			if event.Op&fsnotify.Remove != 0 {
				f.log(fmt.Sprintf("Config file %s has been removed.", f.path))
				watchFunc(nil)

				continue
			}

			// Resolve symlink to get the real path, in case the symlink's
			// target has changed.
			curPath, err := filepath.EvalSymlinks(f.path)
			if err != nil {
				f.log(fmt.Sprintf("Error when evaling symlink for %s: %v", f.path, err))

				continue
			}
			realPath = filepath.Clean(curPath)

			// Finally, we only care about create and write.
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Trigger event.
			values, err := f.Load()
			if err != nil {
				f.log(fmt.Sprintf("Error when loading configuration from %s: %v", f.path, err))

				continue
			}
			watchFunc(values)

		// There's an error.
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			f.log(fmt.Sprintf("Error when watching file %s: %v", f.path, err))

		case <-ctx.Done():
			return nil
		}
	}
}

func (f File) String() string {
	if f.fs == nil {
		return "os file:" + f.path
	}

	return "fs file:" + f.path
}
