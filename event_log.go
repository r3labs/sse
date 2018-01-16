/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import "strconv"

// EventLog holds all of previous events
type EventLog []*Event

// Add event to eventlog
func (e *EventLog) Add(ev *Event) {
	ev.ID = []byte(e.currentindex())
	(*e) = append((*e), ev)
}

// Clear events from eventlog
func (e *EventLog) Clear() {
	*e = nil
}

// Replay events to a subscriber
func (e *EventLog) Replay(s *Subscriber) {
	for i := 0; i < len((*e)); i++ {
		if string((*e)[i].ID) >= s.eventid {
			s.connection <- (*e)[i]
		}
	}
}

func (e *EventLog) currentindex() string {
	return strconv.Itoa(len((*e)))
}
