# A minimalist configuration API for Go

![Go Version](https://img.shields.io/github/go-mod/go-version/nil-go/konf)
[![Go Reference](https://pkg.go.dev/badge/github.com/nil-go/konf.svg)](https://pkg.go.dev/github.com/nil-go/konf)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/nil-go/konf)](https://goreportcard.com/report/github.com/nil-go/konf)
[![Build](https://github.com/nil-go/konf/actions/workflows/test.yml/badge.svg)](https://github.com/nil-go/konf/actions/workflows/test.yml)
[![Coverage](https://codecov.io/gh/nil-go/konf/branch/main/graph/badge.svg)](https://codecov.io/gh/nil-go/konf)

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
        var config konf.Config

        // Load configuration from embed file system.
        if err := config.Load(fs.New(config, "config/config.json")); err != nil {
            // Handle error here.
        }
        // Load configuration from environment variables.
        if err := config.Load(env.New(env.WithPrefix("server"))); err != nil {
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

## Understand the configuration

While the configuration is loaded from multiple sources, static like environments or dynamic like AWS AppConfig,
it's hard to understand where a final value comes from. The `Config.Explain` method provides information
about how Config resolve each value from loaders for the given path. One example explanation is like:
```
config.nest has value [map] is loaded by map.
Here are other value(loader)s:
  - env(env)
```

Even more, the `Config.Explain` blurs sensitive information (e.g. password, secret, api keys).

## Configuration Providers

There are providers for the following configuration sources:

- [`env`](provider/env) loads configuration from environment variables.
- [`fs`](provider/fs) loads configuration from fs.FS.
- [`file`](provider/file) loads configuration from a file.
- [`flag`](provider/flag) loads configuration from flags.
- [`pflag`](provider/pflag) loads configuration from [spf13/pflag](https://github.com/spf13/pflag).
- [`appconfig`](provider/appconfig) loads configuration from [AWS AppConfig](https://aws.amazon.com/systems-manager/features/appconfig/).
- [`azappconfig`](provider/azappconfig) loads configuration from [Azure App Configuration](https://azure.microsoft.com/en-us/products/app-configuration).
- [`secretmanager`](provider/secretmanager) loads configuration from [GCP Secret Manager](https://cloud.google.com/security/products/secret-manager).

## Custom Configuration Providers

You can implement your own provider by implementing the `Loader` for static configuration loader (e.g [`fs`](provider/fs))
or both `Loader` and `Watcher` for dynamic configuration loader (e.g. [`appconfig`](provider/appconfig)).

## Inspiration

konf is inspired by [spf13/viper](https://github.com/spf13/viper) and
[knadh/koanf](https://github.com/knadh/koanf).
Thanks for authors of both awesome configuration libraries.
