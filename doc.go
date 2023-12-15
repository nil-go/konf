// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

/*
Package konf provides a general-purpose configuration API and abstract interfaces
to back that API. Packages in the Go ecosystem can depend on this package,
while callers can load configuration from whatever source is appropriate.

It defines a type, [Config], which provides a method [Config.Unmarshal]
for loading configuration under the given path into the given object.

Each Config is associated with multiple [Loader](s),
Which load configuration from a source, such as file, environment variables etc.
There is a default Config accessible through top-level functions
(such as [Unmarshal] and [Get]) that call the corresponding Config methods.

Configuration is hierarchical, and the path is a sequence of keys that separated by delimiter.
The default delimiter is `.`, which makes configuration path like `parent.child.key`.

# Load Configuration
After creating a [Config], you can load configuration from multiple [Loader](s) using [Config.Load].
Each loader takes precedence over the loaders before it. As long as the configuration has been loaded,
it can be used in following code to get or unmarshal configuration, even for loading configuration
from another source. For example, it can read config file path from environment variables,
and then use the file path to load configuration from file system.

# Watch Changes

[Config.Watch] watches and updates configuration when it changes, which leads [Config.Unmarshal]
always returns latest configuration.

You may use [Config.OnChange] to register a callback if the value of any path have been changed.
It could push the change into application objects instead pulling the configuration periodically.
*/
package konf
