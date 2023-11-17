// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

/*
Package konf defines a general-purpose configuration API and abstract interfaces
to back that API. Packages in the Go ecosystem can depend on this package,
while callers can load configuration from whatever source is appropriate.

It defines a type, [Config], which provides a method [Config.Unmarshal]
for loading configuration under the given path into the given object.

Each Config is associated with multiple [Loader],
Which loads configuration from a source, such as file, environment variables etc.
There is a default Config accessible through top-level functions
(such as [Unmarshal] and [Get]) that call the corresponding Config methods.
*/
package konf
