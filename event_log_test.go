/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEventLog(t *testing.T) {
	Convey("Given a new eventlog", t, func() {
		// New EventLog
		ev := make(EventLog, 0)
		testEvent := &Event{Data: []byte("test")}

		Convey("When clearing the eventlog", func() {
			ev.Add(testEvent)
			ev.Clear()

			Convey("It should be empty", func() {
				So(len(ev), ShouldEqual, 0)
			})

			Convey("It should be able to be populated again", func() {
				ev.Add(testEvent)
				ev.Add(testEvent)
				So(len(ev), ShouldEqual, 2)
			})
		})
	})
}
