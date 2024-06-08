/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"fmt"
	"strconv"
	"time"
)

// EventLog holds all of previous events
type EventLog []*Event

// Add event to eventlog
func (e *EventLog) Add(ev *Event) {
	if !ev.hasContent() {
		return
	}

	ev.ID = []byte(e.currentindex())
	ev.timestamp = time.Now()
	*e = append(*e, ev)
}

// Clear events from eventlog
func (e *EventLog) Clear() {
	*e = nil
}

// Replay events to a subscriber
func (e *EventLog) Replay(s *Subscriber) {
	// NOTE: we need to add proper logging capacity to the SSE server
	// check that eventlog still exists
	if e == nil {
		// Log the error or handle it accordingly
		fmt.Println("EventLog is nil")
		return
	}
	// iterate events and send to subscriber
	for i := 0; i < len(*e); i++ {
		// check that event exists
		if (*e)[i] == nil {
			// Log the error or handle it accordingly
			fmt.Printf("Event at index %d is nil\n", i)
			continue
		}
		// get the event id
		id, err := strconv.Atoi(string((*e)[i].ID))
		if err != nil {
			// Log the error or handle it accordingly
			fmt.Println("Error converting event ID to int")
			continue
		}
		// check if event id is greater than subscriber event id
		if id >= s.eventid {
			s.connection <- (*e)[i]
		}
	}
}

func (e *EventLog) currentindex() string {
	return strconv.Itoa(len(*e))
}
