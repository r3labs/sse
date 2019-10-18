/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"time"
)

// Event holds all of the event source fields
type Event struct {
	ID        []byte
	Data      []byte
	Event     []byte
	Retry     []byte
	timestamp time.Time
}

// EventStreamReader scans an io.Reader looking for EventStream messages.
type EventStreamReader struct {
	scanner *bufio.Scanner
	buffer  []byte
	idx     int
}

// NewEventStreamReader creates an instance of EventStreamReader.
func NewEventStreamReader(eventStream io.Reader) *EventStreamReader {
	scanner := bufio.NewScanner(eventStream)
	split := func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// We have a full event payload to parse.
		if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if i := bytes.Index(data, []byte("\r\r")); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
			return i + 1, data[0:i], nil
		}
		// If we're at EOF, we have all of the data.
		if atEOF {
			return len(data), data, nil
		}
		// Request more data.
		return 0, nil, nil
	}
	// Set the split function for the scanning operation.
	scanner.Split(split)

	return &EventStreamReader{
		scanner: scanner,
	}
}

// ReadEvent scans the EventStream for events.
func (e *EventStreamReader) ReadEvent() ([]byte, error) {
	if e.scanner.Scan() {
		event := e.scanner.Bytes()
		return event, nil
	}
	if err := e.scanner.Err(); err != nil {
		if err == context.Canceled {
			return nil, io.EOF
		}
		return nil, err
	}
	return nil, io.EOF
}
