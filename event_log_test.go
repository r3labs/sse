/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventLog(t *testing.T) {
	ev := newEventLog(0)
	testEvent := &Event{Data: []byte("test")}

	ev.Add(testEvent)
	ev.Clear()

	assert.Equal(t, 0, ev.Len())

	ev.Add(testEvent)
	ev.Add(testEvent)

	assert.Equal(t, 2, ev.Len())
}

func TestEventLogMaxEntries(t *testing.T) {
	ev := newEventLog(2)
	testEvent := &Event{Data: []byte("test")}

	ev.Add(testEvent)
	ev.Add(testEvent)
	ev.Add(testEvent)

	assert.Equal(t, 2, ev.Len())
}
