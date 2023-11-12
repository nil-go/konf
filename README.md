# A minimal configuration API for Go

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

This decoupling allows application developers to write code in terms of `konf.Config`
while the configuration source(s) is managed "up stack" (e.g. in or near `main()`).
Application developers can then switch configuration sources(s) as necessary.

## Typical usage

Somewhere, early in an application's life, it will make a decision about which
configuration source(s) (implementation) it actually wants to use. Something like:

```
    //go:embed testdata
    var config embed.FS

    func main() {
        // Create the global Config that loads configuration
        // from embed file system and environment variables.
        cfg, err := konf.New(
            konf.WithLoader(
                file.New("config/config.json", file.WithFS(config)),
                env.New(env.WithPrefix("server")),
            ),
        )
        if err != nil {
            // Handle error here.
        }
        konf.SetGlobal(cfg)

        // ... other setup code ...
    }
```

Application also can watch the changes of configuration like:

```
    func main() {
        // ... setup global Config ...

        konf.Watch(func(){
            // Read configuration and reconfig application.
        })

        // ... other setup code ...
    }
```

Outside of this early setup, no other packages need to know about the choice of
configuration source(s). They read configuration in terms of functions in package `konf`:

```
    func (app *appObject) Run() {
        type serverConfig struct {
            Host string
            Port int
        }
        cfg := konf.Get[serverConfig]("server")

        // ... use cfg in app code ...
    }
```

## Inspiration

konf is inspired by [spf13/viper](https://github.com/spf13/viper) and
[knadh/koanf](https://github.com/knadh/koanf).
Thanks for authors of both awesome configuration libraries.

## Configuration Providers

There are providers for the following configuration sources:

- `env` loads configuration from environment variables.
- `file` loads configuration from a file.
- `flag` loads configuration from flags.
- `fs` loads configuration from fs.FS.
- `pflag` loads configuration from [spf13/pflag](https://github.com/spf13/pflag).

## Compatibility

konf ensures compatibility with the current supported versions of
the [Go language](https://golang.org/doc/devel/release#policy):

> Each major Go release is supported until there are two newer major releases.
> For example, Go 1.5 was supported until the Go 1.7 release,
> and Go 1.6 was supported until the Go 1.8 release.

For versions of Go that are no longer supported upstream, konf will stop ensuring
compatibility with these versions in the following manner:

- A minor release of konf will be made to add support for the new
  supported release of Go.
- The following minor release of konf will remove compatibility
  testing for the oldest (now archived upstream) version of Go. This, and
  future, releases of konf may include features only supported by
  the currently supported versions of Go.
