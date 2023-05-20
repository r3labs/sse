/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"container/list"
	"strconv"
	"time"
)

// Events holds all of previous events
type Events []*Event

// EventLog holds the log of all of previous events
type EventLog struct {
	MaxEntries int
	log        *list.List
}

func newEventLog(maxEntries int) *EventLog {
	return &EventLog{
		MaxEntries: maxEntries,
		log:        list.New(),
	}
}

// Add event to log
func (e *EventLog) Add(ev *Event) {
	if !ev.hasContent() {
		return
	}

	ev.ID = []byte(e.currentIndex())
	ev.timestamp = time.Now()

	// if MaxEntries is set to greater than 0 (no limit) check entries
	if e.MaxEntries > 0 {
		// if we are at max entries limit
		// then remove the item at the back
		if e.log.Len() >= e.MaxEntries {
			e.log.Remove(e.log.Back())
		}
	}
	e.log.PushFront(ev)
}

// Clear events from log
func (e *EventLog) Clear() {
	e.log.Init()
}

// Replay events to a subscriber
func (e *EventLog) Replay(s *Subscriber) {
	for l := e.log.Back(); l != nil; l = l.Prev() {
		id, _ := strconv.Atoi(string(l.Value.(*Event).ID))
		if id >= s.eventid {
			s.connection <- l.Value.(*Event)
		}
	}
}

func (e *EventLog) currentIndex() string {
	return strconv.Itoa(e.log.Len())
}

func (e *EventLog) Len() int {
	return e.log.Len()
}
