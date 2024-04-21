// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package pubsub_test

import (
	"context"
	"testing"

	kpubsub "github.com/nil-go/konf/notifier/pubsub"
	"github.com/nil-go/konf/notifier/pubsub/internal/assert"
)

func TestNotifier_nil(t *testing.T) {
	t.Parallel()

	var n *kpubsub.Notifier
	n.Register(nil) // no panic
	err := n.Start(context.Background())
	assert.EqualError(t, err, "nil Notifier")
}
