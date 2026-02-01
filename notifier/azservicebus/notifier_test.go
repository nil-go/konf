// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package azservicebus_test

import (
	"context"
	"testing"

	"github.com/nil-go/konf/notifier/azservicebus"
	"github.com/nil-go/konf/notifier/azservicebus/internal/assert"
)

func TestNotifier_nil(t *testing.T) {
	t.Parallel()

	var n *azservicebus.Notifier
	n.Register(nil) // no panic
	err := n.Start(context.Background())
	assert.EqualError(t, err, "nil Notifier")
}

func TestNotifier(t *testing.T) {
	t.Parallel()

	t.Skip("Could not fake service bus for testing. See https://github.com/Azure/azure-sdk-for-go/issues/22364")
}
