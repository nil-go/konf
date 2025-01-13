// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package convert

import (
	"errors"
	"reflect"
)

func WithTagName(tagName string) Option {
	return func(options *options) {
		options.tagName = tagName
	}
}

func WithKeyMapper(keyMap func(string) string) Option {
	return func(options *options) {
		options.keyMap = keyMap
	}
}

func WithHook[F, T any, FN func(F) (T, error) | func(F, T) error](hook FN) Option {
	switch hookFunc := any(hook).(type) {
	case func(F) (T, error):
		return withHookFunc(func(f F, t *T) error {
			r, err := hookFunc(f)
			if err != nil {
				return err
			}
			*t = r

			return nil
		})
	case func(F, T) error:
		return withHookFunc[F, T](hookFunc)
	default:
		return func(*options) {}
	}
}

func withHookFunc[F, T any](hookFunc func(F, T) error) Option {
	return func(options *options) {
		if hookFunc == nil {
			return
		}

		options.hooks = append(options.hooks, hook{
			fromType: reflect.TypeFor[F](),
			toType:   reflect.TypeFor[T](),
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

type (
	// Option configures a Config with specific options.
	Option  func(*options)
	options Converter
)
