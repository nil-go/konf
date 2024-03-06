// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package convert

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Converter struct {
	hooks   []hook
	tagName string
}

func New(opts ...Option) Converter {
	option := &options{}
	for _, opt := range opts {
		opt(option)
	}

	return Converter(*option)
}

func (c Converter) Convert(from, to any) error {
	toVal := reflect.ValueOf(to)
	if toVal.Kind() != reflect.Pointer {
		return errNotPointer
	}

	if !toVal.Elem().CanAddr() {
		return errNotAddressable
	}

	return c.convert("", from, toVal)
}

func (c Converter) convert(name string, from any, toVal reflect.Value) error { //nolint:cyclop
	if from == nil {
		return nil // Do nothing if from is nil.
	}

	fromVal := reflect.ValueOf(from)
	for fromVal.Kind() == reflect.Pointer {
		if fromVal.IsNil() {
			return nil // Do nothing if from is a nil.
		}
		fromVal = fromVal.Elem()
	}

	if toVal.Kind() == reflect.Pointer && !toVal.Elem().CanAddr() {
		return fmt.Errorf("could be a bug: %w", errNotAddressable)
	}

	for _, h := range c.hooks {
		if fromVal.Type().AssignableTo(h.fromType) && toVal.Type().AssignableTo(h.toType) {
			if err := h.hook(fromVal.Interface(), toVal.Interface()); !errors.Is(err, errors.ErrUnsupported) {
				return err
			}
		}
	}

	toVal = reflect.Indirect(toVal)
	switch {
	case toVal.Kind() == reflect.Bool:
		return c.convertBool(name, fromVal, toVal)
	case toVal.CanInt():
		return c.convertInt(name, fromVal, toVal)
	case toVal.CanUint():
		return c.convertUint(name, fromVal, toVal)
	case toVal.CanFloat():
		return c.convertFloat(name, fromVal, toVal)
	case toVal.CanComplex():
		return c.convertComplex(name, fromVal, toVal)
	case toVal.Kind() == reflect.Array:
		return c.convertArray(name, fromVal, toVal)
	case toVal.Kind() == reflect.Map:
		return c.convertMap(name, fromVal, toVal)
	case toVal.Kind() == reflect.Pointer:
		return c.convertPointer(name, fromVal, toVal)
	case toVal.Kind() == reflect.Slice:
		return c.convertSlice(name, fromVal, toVal)
	case toVal.Kind() == reflect.String:
		return c.convertString(name, fromVal, toVal)
	case toVal.Kind() == reflect.Struct:
		return c.convertStruct(name, fromVal, toVal)
	default:
		// If it reached here then it weren't able to convert it.
		return fmt.Errorf("%s: unsupported type: %s", name, toVal.Kind()) //nolint:goerr113
	}
}

func (c Converter) convertBool(name string, fromVal, toVal reflect.Value) error {
	switch {
	case fromVal.Kind() == reflect.Bool:
		toVal.SetBool(fromVal.Bool())
	case fromVal.CanInt():
		toVal.SetBool(fromVal.Int() != 0)
	case fromVal.CanUint():
		toVal.SetBool(fromVal.Uint() != 0)
	case fromVal.CanFloat():
		toVal.SetBool(fromVal.Float() != 0)
	case fromVal.CanComplex():
		toVal.SetBool(fromVal.Complex() != 0)
	case fromVal.Kind() == reflect.String:
		from := fromVal.String()
		if from == "" {
			toVal.SetBool(false)
		} else {
			b, err := strconv.ParseBool(from)
			if err != nil {
				return fmt.Errorf("cannot parse '%s' as bool: %w", name, err)
			}
			toVal.SetBool(b)
		}
	default:
		return fmt.Errorf( //nolint:goerr113
			"'%s' expected type '%s', got unconvertible type '%s', value: '%v'",
			name, toVal.Type(), fromVal.Type(), fromVal.Interface(),
		)
	}

	return nil
}

func (c Converter) convertInt(name string, fromVal, toVal reflect.Value) error { //nolint:cyclop
	switch {
	case fromVal.Kind() == reflect.Bool:
		if fromVal.Bool() {
			toVal.SetInt(1)
		} else {
			toVal.SetInt(0)
		}
	case fromVal.CanInt():
		toVal.SetInt(fromVal.Int())
	case fromVal.CanUint():
		toVal.SetInt(int64(fromVal.Uint()))
	case fromVal.CanFloat():
		toVal.SetInt(int64(fromVal.Float()))
	case fromVal.CanComplex():
		toVal.SetInt(int64(real(fromVal.Complex())))
	case fromVal.Kind() == reflect.String:
		from := fromVal.String()
		if from == "" {
			toVal.SetInt(0)
		} else {
			i, err := strconv.ParseInt(from, 0, toVal.Type().Bits())
			if err != nil {
				return fmt.Errorf("cannot parse '%s' as int: %w", name, err)
			}
			toVal.SetInt(i)
		}
	default:
		return fmt.Errorf( //nolint:goerr113
			"'%s' expected type '%s', got unconvertible type '%s', value: '%v'",
			name, toVal.Type(), fromVal.Type(), fromVal.Interface(),
		)
	}

	return nil
}

func (c Converter) convertUint(name string, fromVal, toVal reflect.Value) error { //nolint:cyclop
	switch {
	case fromVal.Kind() == reflect.Bool:
		if fromVal.Bool() {
			toVal.SetUint(1)
		} else {
			toVal.SetUint(0)
		}
	case fromVal.CanInt():
		i := fromVal.Int()
		if i < 0 {
			return fmt.Errorf("cannot parse '%s', %d overflows uint", name, i) //nolint:goerr113
		}
		toVal.SetUint(uint64(i))
	case fromVal.CanUint():
		toVal.SetUint(fromVal.Uint())
	case fromVal.CanFloat():
		f := fromVal.Float()
		if f < 0 {
			return fmt.Errorf("cannot parse '%s', %f overflows uint", name, f) //nolint:goerr113
		}
		toVal.SetUint(uint64(f))
	case fromVal.CanComplex():
		r := real(fromVal.Complex())
		if r < 0 {
			return fmt.Errorf("cannot parse '%s', %f overflows uint", name, r) //nolint:goerr113
		}
		toVal.SetUint(uint64(r))
	case fromVal.Kind() == reflect.String:
		from := fromVal.String()
		if from == "" {
			toVal.SetUint(0)
		} else {
			i, err := strconv.ParseUint(from, 0, toVal.Type().Bits())
			if err != nil {
				return fmt.Errorf("cannot parse '%s' as uint: %w", name, err)
			}
			toVal.SetUint(i)
		}
	default:
		return fmt.Errorf( //nolint:goerr113
			"'%s' expected type '%s', got unconvertible type '%s', value: '%v'",
			name, toVal.Type(), fromVal.Type(), fromVal.Interface(),
		)
	}

	return nil
}

func (c Converter) convertFloat(name string, fromVal, toVal reflect.Value) error { //nolint:cyclop
	switch {
	case fromVal.Kind() == reflect.Bool:
		if fromVal.Bool() {
			toVal.SetFloat(1)
		} else {
			toVal.SetFloat(0)
		}
	case fromVal.CanInt():
		toVal.SetFloat(float64(fromVal.Int()))
	case fromVal.CanUint():
		toVal.SetFloat(float64(fromVal.Uint()))
	case fromVal.CanFloat():
		toVal.SetFloat(fromVal.Float())
	case fromVal.CanComplex():
		toVal.SetFloat(real(fromVal.Complex()))
	case fromVal.Kind() == reflect.String:
		from := fromVal.String()
		if from == "" {
			toVal.SetFloat(0)
		} else {
			i, err := strconv.ParseFloat(from, toVal.Type().Bits())
			if err != nil {
				return fmt.Errorf("cannot parse '%s' as float: %w", name, err)
			}
			toVal.SetFloat(i)
		}
	default:
		return fmt.Errorf( //nolint:goerr113
			"'%s' expected type '%s', got unconvertible type '%s', value: '%v'",
			name, toVal.Type(), fromVal.Type(), fromVal.Interface(),
		)
	}

	return nil
}

func (c Converter) convertComplex(name string, fromVal, toVal reflect.Value) error { //nolint:cyclop
	switch {
	case fromVal.Kind() == reflect.Bool:
		if fromVal.Bool() {
			toVal.SetComplex(1)
		} else {
			toVal.SetComplex(0)
		}
	case fromVal.CanInt():
		toVal.SetComplex(complex(float64(fromVal.Int()), 0))
	case fromVal.CanUint():
		toVal.SetComplex(complex(float64(fromVal.Uint()), 0))
	case fromVal.CanFloat():
		toVal.SetComplex(complex(fromVal.Float(), 0))
	case fromVal.CanComplex():
		toVal.SetComplex(fromVal.Complex())
	case fromVal.Kind() == reflect.String:
		from := fromVal.String()
		if from == "" {
			toVal.SetComplex(0)
		} else {
			i, err := strconv.ParseComplex(from, toVal.Type().Bits())
			if err != nil {
				return fmt.Errorf("cannot parse '%s' as complex: %w", name, err)
			}
			toVal.SetComplex(i)
		}
	default:
		return fmt.Errorf( //nolint:goerr113
			"'%s' expected type '%s', got unconvertible type '%s', value: '%v'",
			name, toVal.Type(), fromVal.Type(), fromVal.Interface(),
		)
	}

	return nil
}

func (c Converter) convertArray(name string, fromVal, toVal reflect.Value) error {
	switch fromVal.Kind() {
	case reflect.Array, reflect.Slice:
		if fromVal.Len() > toVal.Len() {
			return fmt.Errorf( //nolint:goerr113
				"'%s': expected source data to have length less or equal to %d, got %d",
				name, toVal.Len(), fromVal.Len(),
			)
		}

		toVal.SetZero()
		errs := make([]error, 0, toVal.Len())
		for i := 0; i < fromVal.Len(); i++ {
			fromElem := fromVal.Index(i).Interface()
			toElemVal := toVal.Index(i)

			fieldName := name + "[" + strconv.Itoa(i) + "]"
			if err := c.convert(fieldName, fromElem, pointer(toElemVal)); err != nil {
				errs = append(errs, err)
			}
		}

		return errors.Join(errs...)
	case reflect.Map:
		// Empty maps turn into empty arrays
		if fromVal.Len() == 0 {
			toVal.SetZero()

			return nil
		}

		fallthrough
	default:
		// All other types it tries to convert to the array type
		// and "lift" it into it. i.e. a string becomes a string array.
		// Just re-try this function with data as a slice.
		return c.convertArray(name, reflect.ValueOf([]any{fromVal.Interface()}), toVal)
	}
}

func (c Converter) convertMap(name string, fromVal, toVal reflect.Value) error {
	switch fromVal.Kind() {
	case reflect.Map:
		if fromVal.IsNil() {
			toVal.SetZero()

			return nil
		}

		if toVal.IsNil() {
			toVal.Set(reflect.MakeMapWithSize(toVal.Type(), fromVal.Len()))
		} else {
			toVal.Clear()
		}

		toKeyType := toVal.Type().Key()
		toValueType := toVal.Type().Elem()
		errs := make([]error, 0, toVal.Len())
		for _, fromKey := range fromVal.MapKeys() {
			fieldName := name + "[" + fromKey.String() + "]"

			toKey := reflect.New(toKeyType)
			if err := c.convert(fieldName, fromKey.Interface(), toKey); err != nil {
				errs = append(errs, err)

				continue
			}

			toValue := reflect.New(toValueType)
			if err := c.convert(fieldName, fromVal.MapIndex(fromKey).Interface(), toValue); err != nil {
				errs = append(errs, err)

				continue
			}

			toVal.SetMapIndex(reflect.Indirect(toKey), reflect.Indirect(toValue))
		}

		return errors.Join(errs...)
	default:
		return fmt.Errorf("'%s' expected a map, got '%s'", name, fromVal.Kind()) //nolint:goerr113
	}
}

func (c Converter) convertPointer(name string, fromVal reflect.Value, toVal reflect.Value) error {
	switch fromVal.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		if fromVal.IsNil() {
			toVal.SetZero()

			return nil
		}
	default:
	}
	toVal.Set(reflect.New(toVal.Type().Elem()))

	return c.convert(name, fromVal.Interface(), reflect.Indirect(toVal.Elem()))
}

func (c Converter) convertSlice(name string, fromVal, toVal reflect.Value) error { //nolint:cyclop
	switch {
	case fromVal.Kind() == reflect.Array || fromVal.Kind() == reflect.Slice:
		if fromVal.Len() == 0 {
			toVal.SetZero()

			return nil // avoid extra heap allocation
		}

		toVal.Clear()
		if toVal.Len() < fromVal.Len() {
			toVal.Grow(fromVal.Len() - toVal.Len())
		}
		toVal.SetLen(fromVal.Len())
		toVal.SetCap(fromVal.Len())

		errs := make([]error, 0, toVal.Len())
		for i := 0; i < fromVal.Len(); i++ {
			fromElem := fromVal.Index(i).Interface()
			toElemVal := toVal.Index(i)

			fieldName := name + "[" + strconv.Itoa(i) + "]"
			if err := c.convert(fieldName, fromElem, pointer(toElemVal)); err != nil {
				errs = append(errs, err)
			}
		}

		return errors.Join(errs...)
	case fromVal.Kind() == reflect.String && toVal.Type().Elem().Kind() == reflect.Uint8:
		toVal.SetBytes([]byte(fromVal.String()))
	case fromVal.Kind() == reflect.Map:
		// Empty maps turn into empty arrays
		if fromVal.Len() == 0 {
			toVal.SetZero()

			return nil
		}

		fallthrough
	default:
		// All other types it tries to convert to the slice type
		// and "lift" it into it. i.e. a string becomes a string slice.
		// Just re-try this function with data as a slice.
		return c.convertSlice(name, reflect.ValueOf([]any{fromVal.Interface()}), toVal)
	}

	return nil
}

func (c Converter) convertString(name string, fromVal, toVal reflect.Value) error { //nolint:cyclop
	switch {
	case fromVal.Kind() == reflect.Bool:
		if fromVal.Bool() {
			toVal.SetString("1")
		} else {
			toVal.SetString("0")
		}
	case fromVal.CanInt():
		toVal.SetString(strconv.FormatInt(fromVal.Int(), 10))
	case fromVal.CanUint():
		toVal.SetString(strconv.FormatUint(fromVal.Uint(), 10))
	case fromVal.CanFloat():
		toVal.SetString(strconv.FormatFloat(fromVal.Float(), 'f', -1, 64))
	case fromVal.CanComplex():
		toVal.SetString(strconv.FormatComplex(fromVal.Complex(), 'f', -1, 128)) //nolint:gomnd
	case fromVal.Kind() == reflect.String:
		toVal.SetString(fromVal.String())
	case fromVal.Kind() == reflect.Array && fromVal.Type().Elem().Kind() == reflect.Uint8:
		bytes := make([]uint8, fromVal.Len()) //nolint:makezero
		reflect.Copy(reflect.ValueOf(bytes), fromVal)
		toVal.SetString(string(bytes))
	case fromVal.Kind() == reflect.Slice && fromVal.Type().Elem().Kind() == reflect.Uint8:
		toVal.SetString(string(fromVal.Bytes()))
	default:
		return fmt.Errorf( //nolint:goerr113
			"'%s' expected type '%s', got unconvertible type '%s', value: '%v'",
			name, toVal.Type(), fromVal.Type(), fromVal.Interface(),
		)
	}

	return nil
}

func (c Converter) convertStruct(name string, fromVal, toVal reflect.Value) error { //nolint:cyclop,funlen,gocognit
	switch fromVal.Kind() {
	case reflect.Map:
		if fromVal.Type().Key().Kind() != reflect.String {
			return fmt.Errorf( //nolint:goerr113
				"'%s' needs a map with string keys, has '%s' keys",
				name, fromVal.Type().Key().Kind(),
			)
		}

		fromKeys := fromVal.MapKeys()
		// This slice will keep track of all the structs it'll be decoding.
		// There can be more than one struct if there are embedded structs
		// that are squashed.
		structs := make([]reflect.Value, 0, 5) //nolint:gomnd
		structs = append(structs, toVal)

		var errs []error
		for len(structs) > 0 {
			structVal := structs[0]
			structs = structs[1:]

			structType := structVal.Type()
			for i := 0; i < structType.NumField(); i++ {
				fieldType := structType.Field(i)
				fieldVal := structVal.Field(i)
				if !fieldVal.CanSet() {
					// If it can't set the field, then it is unexported or something,
					// and it just continue onwards.
					continue
				}

				// It always parse the tags cause it's looking for other tags too
				fileName, tag, _ := strings.Cut(fieldType.Tag.Get(c.tagName), ",")
				if fileName == "" {
					fileName = fieldType.Name
				}
				if tag == "squash" {
					if fieldVal.Kind() != reflect.Struct {
						errs = append(errs, fmt.Errorf( //nolint:goerr113
							"%s: unsupported type for squash: %s",
							fieldType.Name, fieldVal.Kind(),
						))
					} else {
						structs = append(structs, fieldVal)
					}

					continue
				}

				elemVal := fromVal.MapIndex(reflect.ValueOf(fileName))
				if !elemVal.IsValid() {
					// Do a slower search by iterating over each key and
					// doing case-insensitive search.
					for _, fromKey := range fromKeys {
						if strings.EqualFold(fromKey.String(), fileName) {
							elemVal = fromVal.MapIndex(fromKey)

							break
						}
					}
				}
				if !elemVal.IsValid() {
					// There was no matching key in the map for the value in the struct.
					continue
				}

				if name != "" {
					fileName = name + "." + fileName
				}
				if err := c.convert(fileName, elemVal.Interface(), pointer(fieldVal)); err != nil {
					errs = append(errs, err)
				}
			}
		}

		return errors.Join(errs...)
	default:
		return fmt.Errorf("'%s' expected a map, got '%s'", name, fromVal.Kind()) //nolint:goerr113
	}
}

func pointer(val reflect.Value) reflect.Value {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}

		return val
	}

	return val.Addr()
}

var (
	errNotPointer     = errors.New("to must be a pointer")
	errNotAddressable = errors.New("to must be addressable (a pointer)")
)

type hook struct {
	fromType reflect.Type
	toType   reflect.Type
	hook     func(from, to any) error
}
