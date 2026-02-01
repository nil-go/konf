// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:dupword
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

# Field Tags

When decoding to a struct, konf will use the field name by default to perform the mapping.
For example, if a struct has a field "Username" then konf will look for a key
in the source value of "username" (case insensitive).

	type User struct {
	    Username string
	}

You can change the behavior of konf by using struct tags.
The default struct tag that konf looks for is "konf"
but you can customize it using DecoderConfig.

# Renaming Fields

To rename the key that konf looks for, use the "konf"
tag and set a value directly. For example, to change the "username" example
above to "user":

	type User struct {
	    Username string `konf:"user"`
	}

# Embedded Structs and Squashing

Embedded structs are treated as if they're another field with that name.
By default, the two structs below are equivalent when decoding with konf:

	type Person struct {
	    Name string
	}

	type Friend struct {
	    Person
	}

	type Friend struct {
	    Person Person
	}

This would require an input that looks like below:

	map[string]interface{}{
	    "person": map[string]interface{}{"name": "alice"},
	}

If your "person" value is NOT nested, then you can append ",squash" to
your tag value and konf will treat it as if the embedded struct
were part of the struct directly. Example:

	type Friend struct {
	    Person `konf:",squash"`
	}

Now the following input would be accepted:

	map[string]interface{}{
	    "name": "alice",
	}

# Unexported fields

Since unexported (private) struct fields cannot be set outside the package
where they are defined, the decoder will simply skip them.

For this output type definition:

	type Exported struct {
	    private string // this unexported field will be skipped
	    Public string
	}

Using this map as input:

	map[string]interface{}{
	    "private": "I will be ignored",
	    "Public":  "I made it through!",
	}

The following struct will be decoded:

	type Exported struct {
	    private: "" // field is left with an empty string (zero value)
	    Public: "I made it through!"
	}
*/
package konf
