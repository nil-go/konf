// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"encoding"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/nil-go/konf/internal"
	"github.com/nil-go/konf/internal/convert"
	"github.com/nil-go/konf/internal/credential"
)

// Config reads configuration from appropriate sources.
//
// To create a new Config, call [New].
type Config struct {
	nocopy internal.NoCopy[Config]

	// Options.
	caseSensitive bool
	delimiter     string
	logger        *slog.Logger
	onStatus      func(loader Loader, changed bool, err error)
	converter     convert.Converter

	// Loaded configuration.
	values    *internal.Store
	providers []provider

	// For watching changes.
	onChanges      map[string][]func(*Config)
	onChangesMutex sync.RWMutex
	watched        atomic.Bool
}

// New creates a new Config with the given Option(s).
func New(opts ...Option) *Config {
	option := &options{}
	for _, opt := range opts {
		opt(option)
	}

	if len(option.hooks) == 0 {
		option.hooks = defaultHooks
	}
	if option.tagName == "" {
		option.hooks = append(option.hooks, defaultTagName)
	} else {
		option.hooks = append(option.hooks, convert.WithTagName(option.tagName))
	}
	option.hooks = append(option.hooks, convert.WithKeyMapper(option.keyMap()))
	option.converter = convert.New(option.hooks...)

	return &(option.Config)
}

// Load loads configuration from the given loader.
// Each loader takes precedence over the loaders before it.
//
// This method can be called multiple times but it is not concurrency-safe.
func (c *Config) Load(loader Loader) error {
	c.nocopy.Check()

	if loader == nil {
		return nil
	}
	if c.values == nil {
		c.values = internal.NewStore(make(map[string]any), c.delim(), c.keyMap())
	}

	if statuser, ok := loader.(Statuser); ok {
		statuser.Status(func(changed bool, err error) {
			if err != nil {
				c.log(context.Background(),
					slog.LevelWarn,
					"Error when loading configuration.",
					slog.Any("loader", loader),
					slog.Any("error", err),
				)
			}
			if c.onStatus != nil {
				c.onStatus(loader, changed, err)
			}
		})
	}

	values, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	prd := provider{
		loader: loader,
		values: internal.NewStore(values, c.delim(), c.keyMap()),
	}
	c.providers = append(c.providers, prd)
	c.values.Merge(prd.values)

	return nil
}

// Unmarshal reads configuration under the given path from the Config
// and decodes it into the given object pointed to by target.
// The path is case-insensitive unless konf.WithCaseSensitive is set.
func (c *Config) Unmarshal(path string, target any) error {
	if c == nil {
		return nil
	}
	if c.values == nil {
		c.values = internal.NewStore(make(map[string]any), c.delim(), c.keyMap())
	}

	c.nocopy.Check()

	converter := c.converter
	if reflect.ValueOf(converter).IsZero() {
		converter = defaultConverter
	}

	if err := converter.Convert(c.values.Sub(path), target); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

func (c *Config) log(ctx context.Context, level slog.Level, message string, attrs ...slog.Attr) {
	logger := c.logger
	if c.logger == nil {
		logger = slog.Default()
	}
	logger.LogAttrs(ctx, level, message, attrs...)
}

func (c *Config) keyMap() func(string) string {
	if c.caseSensitive {
		return nil
	}

	return toLower
}

func toLower(s string) string {
	return strings.Map(unicode.ToLower, s)
}

func (c *Config) delim() string {
	if c.delimiter == "" {
		return "."
	}

	return c.delimiter
}

// Explain provides information about how Config resolve each value
// from loaders for the given path. It blur sensitive information.
// The path is case-insensitive unless konf.WithCaseSensitive is set.
func (c *Config) Explain(path string) string {
	if c == nil {
		return path + " has no configuration.\n\n"
	}

	c.nocopy.Check()

	explanation := &strings.Builder{}
	c.explain(explanation, path, c.values.Sub(path))

	return explanation.String()
}

func (c *Config) explain(explanation *strings.Builder, path string, value any) {
	if values, ok := value.(map[string]any); ok {
		for key, val := range values {
			newPath := path
			if newPath != "" {
				newPath += c.delim()
			}
			newPath += key
			c.explain(explanation, newPath, val)
		}

		return
	}

	type loaderValue struct {
		loader Loader
		value  any
	}
	var loaders []loaderValue
	for _, provider := range c.providers {
		if v := provider.values.Sub(path); v != nil {
			loaders = append(loaders, loaderValue{provider.loader, v})
		}
	}
	slices.Reverse(loaders)

	if len(loaders) == 0 {
		explanation.WriteString(path)
		explanation.WriteString(" has no configuration.\n\n")

		return
	}
	explanation.WriteString(path)
	explanation.WriteString(" has value[")
	explanation.WriteString(credential.Blur(path, loaders[0].value))
	explanation.WriteString("] that is loaded by loader[")
	explanation.WriteString(fmt.Sprintf("%v", loaders[0].loader))
	explanation.WriteString("].\n")
	if len(loaders) > 1 {
		explanation.WriteString("Here are other value(loader)s:\n")
		for _, loader := range loaders[1:] {
			explanation.WriteString("  - ")
			explanation.WriteString(credential.Blur(path, loader.value))
			explanation.WriteString("(")
			explanation.WriteString(fmt.Sprintf("%v", loader.loader))
			explanation.WriteString(")\n")
		}
	}
	explanation.WriteString("\n")
}

type provider struct {
	loader Loader
	values *internal.Store
}

//nolint:gochecknoglobals
var (
	defaultTagName = convert.WithTagName("konf")
	defaultKeyMap  = convert.WithKeyMapper(toLower)
	defaultHooks   = []convert.Option{
		convert.WithHook[string, time.Duration](time.ParseDuration),
		convert.WithHook[string, []string](func(f string) ([]string, error) {
			return strings.Split(f, ","), nil
		}),
		convert.WithHook[string, encoding.TextUnmarshaler](func(f string, t encoding.TextUnmarshaler) error {
			return t.UnmarshalText(internal.String2ByteSlice(f))
		}),
	}
	defaultConverter = convert.New(
		append(defaultHooks, defaultTagName, defaultKeyMap)...,
	)
)
