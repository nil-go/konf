# A minimalist configuration API for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/ktong/konf.svg)](https://pkg.go.dev/github.com/ktong/konf)
[![Build](https://github.com/ktong/konf/actions/workflows/test.yml/badge.svg)](https://github.com/ktong/konf/actions/workflows/test.yml)
[![Coverage](https://codecov.io/gh/ktong/konf/branch/main/graph/badge.svg)](https://codecov.io/gh/ktong/konf)

konf offers an(other) opinion on how Go programs can read configuration without
becoming coupled to a particular configuration source. It contains two APIs with two
different sets of users.

The `Config` type is intended for application authors. It provides a relatively
small API which can be used everywhere you want to read configuration.
It defers the actual configuration loading to the `Loader` interface.

The `Loader` and `Watcher` interface is intended for configuration source library implementers.
They are pure interfaces which can be implemented to provide the actual configuration.

This decoupling allows application developers to write code in terms of `*konf.Config`
while the configuration source(s) is managed "up stack" (e.g. in or near `main()`).
Application developers can then switch configuration sources(s) as necessary.

## Usage

Somewhere, early in an application's life, it will make a decision about which
configuration source(s) (implementation) it actually wants to use. Something like:

```
    //go:embed config
    var config embed.FS

    func main() {
        // Create the Config.
        config := konf.New()

        // Load configuration from embed file system and environment variables.
        if err := config.Load(
            fs.New(config, "config/config.json"),
            env.New(env.WithPrefix("server")),
        ); err != nil {
            // Handle error here.
        }

        // Watch the changes of configuration.
        go func() {
          if err := config.Watch(ctx); err != nil {
            // Handle error here.
          }
        }

        konf.SetDefault(config)

        // ... other setup code ...
    }
```

Outside of this early setup, no other packages need to know about the choice of
configuration source(s). They read configuration in terms of functions in package `konf`:

```
    func (app *appObject) Run() {
        // Read the server configuration.
        type serverConfig struct {
            Host string
            Port int
        }
        cfg := konf.Get[serverConfig]("server")

        // Register callbacks while server configuration changes.
        konf.OnChange(func() {
          // Reconfig the application object.
        }, "server")

        // ... use cfg in app code ...
    }
```

## Inspiration

konf is inspired by [spf13/viper](https://github.com/spf13/viper) and
[knadh/koanf](https://github.com/knadh/koanf).
Thanks for authors of both awesome configuration libraries.

## Configuration Providers

There are providers for the following configuration sources:

- [`env`](provider/env) loads configuration from environment variables.
- [`file`](provider/file) loads configuration from a file.
- [`flag`](provider/flag) loads configuration from flags.
- [`fs`](provider/fs) loads configuration from fs.FS.
- [`pflag`](provider/pflag) loads configuration from [spf13/pflag](https://github.com/spf13/pflag).
