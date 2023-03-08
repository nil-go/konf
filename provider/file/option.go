// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package file

import (
	"encoding/json"
	"io/fs"
	"log"
)

// WithFS provides the fs.FS that config file is loaded from.
//
// The default file system is OS file system.
func WithFS(fs fs.FS) Option {
	return func(file *options) {
		file.fs = fs
	}
}

// WithUnmarshal provides the function that parses config file.
//
// The default function is json.Unmarshal.
func WithUnmarshal(unmarshal func([]byte, any) error) Option {
	return func(file *options) {
		file.unmarshal = unmarshal
	}
}

// IgnoreFileNotExit ignores the error if config file does not exist.
func IgnoreFileNotExit() Option {
	return func(file *options) {
		file.ignoreNotExist = true
	}
}

// WithLog provides the function that logs message.
//
// The default function [log.Print].
func WithLog(log func(...any)) Option {
	return func(file *options) {
		file.log = log
	}
}

// Option configures the given File.
type Option func(file *options)

type options File

func apply(path string, opts []Option) options {
	option := &options{
		path:      path,
		unmarshal: json.Unmarshal,
		log:       log.Print,
	}
	for _, opt := range opts {
		opt(option)
	}

	return *option
}
