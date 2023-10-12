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
	timestamp time.Time
	ID        []byte
	Data      []byte
	Event     []byte
	Retry     []byte
	Comment   []byte
}

func (e *Event) hasContent() bool {
	return len(e.ID) > 0 || len(e.Data) > 0 || len(e.Event) > 0 || len(e.Retry) > 0
}

// EventStreamReader scans an io.Reader looking for EventStream messages.
type EventStreamReader struct {
	scanner *bufio.Scanner
}

// NewEventStreamReader creates an instance of EventStreamReader.
func NewEventStreamReader(eventStream io.Reader, maxBufferSize int) *EventStreamReader {
	scanner := bufio.NewScanner(eventStream)
	initBufferSize := minPosInt(4096, maxBufferSize)
	scanner.Buffer(make([]byte, initBufferSize), maxBufferSize)

	// this ensures we don't keep checking data we've already scanned within one Split
	newDataIndex := 0
	split := func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// We have a full event payload to parse.
		if i, nlen := containsDoubleNewline(data, newDataIndex); i >= 0 {
			newDataIndex = 0 // reset for next token
			return i + nlen, data[0:i], nil
		}
		newDataIndex = len(data) // we've already scanned the entire data up to this point
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

// Returns a tuple containing the index of a double newline, and the number of bytes
// represented by that sequence. If no double newline is present, the first value
// will be negative.
func containsDoubleNewline(data []byte, newDataIndex int) (int, int) {
	// Look back of the length of the longest search string to start from the end of the data that
	// we looked at already minus the smallest length that could contain the search string from
	// backtracking. this prevents n^2 lookups if data repeatedly does not contain the search
	// string and slowly grows, especially to a large size.
	lookBackStart := max(0, newDataIndex-3) // len(cr lf cr lf) - 1
	lookBack := data[lookBackStart:]

	// Search for each potentially valid sequence of newline characters
	crcr := bytes.Index(lookBack, []byte("\r\r"))
	if crcr >= 0 {
		crcr += lookBackStart
	}
	lflf := bytes.Index(lookBack, []byte("\n\n"))
	if lflf >= 0 {
		lflf += lookBackStart
	}
	crlflf := bytes.Index(lookBack, []byte("\r\n\n"))
	if crlflf >= 0 {
		crlflf += lookBackStart
	}
	lfcrlf := bytes.Index(lookBack, []byte("\n\r\n"))
	if lfcrlf >= 0 {
		lfcrlf += lookBackStart
	}
	crlfcrlf := bytes.Index(lookBack, []byte("\r\n\r\n"))
	if crlfcrlf >= 0 {
		crlfcrlf += lookBackStart
	}
	// Find the earliest position of a double newline combination
	minPos := minPosInt(crcr, minPosInt(lflf, minPosInt(crlflf, minPosInt(lfcrlf, crlfcrlf))))
	// Determine the length of the sequence
	nlen := 2
	if minPos == crlfcrlf {
		nlen = 4
	} else if minPos == crlflf || minPos == lfcrlf {
		nlen = 3
	}
	return minPos, nlen
}

// Returns the minimum non-negative value out of the two values. If both
// are negative, a negative value is returned.
func minPosInt(a, b int) int {
	if a < 0 {
		return b
	}
	if b < 0 {
		return a
	}
	if a > b {
		return b
	}
	return a
}

// max returns the max integer between the two inputs
// TODO remove when min supported go version is 1.21, as max is now a built in function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
