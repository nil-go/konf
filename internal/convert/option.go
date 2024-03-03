// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package convert

import (
	"errors"
	"reflect"
)

// WithTagName provides the tag name that reads for field names.
// The tag name is used when decoding configuration into structs.
//
// For example, with the default tag name `konf`, it would look for `konf` tags on struct fields.
func WithTagName(tagName string) Option {
	return func(options *options) {
		options.tagName = tagName
	}
}

func WithHookFunc[F, T any](hookFunc func(F, T) error) Option {
	return func(options *options) {
		if hookFunc == nil {
			return
		}

		options.hooks = append(options.hooks, hook{
			fromType: reflect.TypeOf((*F)(nil)).Elem(),
			toType:   reflect.TypeOf((*T)(nil)).Elem(),
			hook: func(f, t any) error {
				from, ok := f.(F)
				if !ok {
					return errors.ErrUnsupported
				}
				to, ok := t.(T)
				if !ok {
					return errors.ErrUnsupported
				}

				return hookFunc(from, to)
			},
		})
	}
}

func WithHook[F, T any](hookFunc func(F) (T, error)) Option {
	return WithHookFunc(func(f F, t *T) error {
		r, err := hookFunc(f)
		if err != nil {
			return err
		}
		*t = r

		return nil
	})
}

type (
	// Option configures a Config with specific options.
	Option  func(*options)
	options Converter
)
