/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"strconv"
	"time"
)

// Events holds all of previous events
type Events []*Event

// EventLog holds the log of all of previous events
type EventLog struct {
	MaxEntries int
	Log        Events
}

func newEventLog(maxEntries int) *EventLog {
	return &EventLog{
		MaxEntries: maxEntries,
		Log:        make(Events, 0),
	}
}

// Add event to eventlog
func (e *EventLog) Add(ev *Event) {
	if !ev.hasContent() {
		return
	}

	ev.ID = []byte(e.currentindex())
	ev.timestamp = time.Now()

	// if MaxEntries is set to greater than 0 (no limit) check entries
	if e.MaxEntries > 0 {
		// ifa we are at max entries limit
		// then reset the first log item and then pop it
		if len(e.Log) >= e.MaxEntries {
			e.Log[0] = nil
			e.Log = e.Log[1:]
		}
	}

	e.Log = append(e.Log, ev)
}

// Clear events from eventlog
func (e *EventLog) Clear() {
	e.Log = nil
}

// Replay events to a subscriber
func (e *EventLog) Replay(s *Subscriber) {
	for i := 0; i < len(e.Log); i++ {
		id, _ := strconv.Atoi(string((e.Log)[i].ID))
		if id >= s.eventid {
			s.connection <- (e.Log)[i]
		}
	}
}

func (e *EventLog) currentindex() string {
	return strconv.Itoa(len(e.Log))
}
