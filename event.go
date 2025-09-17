/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	headerID      = []byte("id:")
	headerData    = []byte("data:")
	headerEvent   = []byte("event:")
	headerRetry   = []byte("retry:")
	headerComment = []byte(":")
)

type EventParseConfig struct {
	EncodingBase64 bool
	Comments       bool
}

// Event holds all of the event source fields
type Event struct {
	timestamp time.Time
	ID        []byte
	Data      []byte
	Event     []byte
	Retry     []byte
	Comment   []byte
}

func ParseEvent(msg []byte, cfg EventParseConfig) (*Event, error) {
	var e Event
	var err error

	if len(msg) < 1 {
		return nil, errors.New("event message was empty")
	}

	// Normalize the crlf to lf to make it easier to split the lines.
	// Split the line by "\n" or "\r", per the spec.
	for _, line := range bytes.FieldsFunc(msg, func(r rune) bool { return r == '\n' || r == '\r' }) {
		switch {
		case bytes.HasPrefix(line, headerID):
			e.ID = append([]byte(nil), trimHeader(len(headerID), line)...)
		case bytes.HasPrefix(line, headerData):
			// The spec allows for multiple data fields per event, concatenated them with "\n".
			e.Data = append(e.Data[:], append(trimHeader(len(headerData), line), byte('\n'))...)
		// The spec says that a line that simply contains the string "data" should be treated as a data field with an empty body.
		case bytes.Equal(line, bytes.TrimSuffix(headerData, []byte(":"))):
			e.Data = append(e.Data, byte('\n'))
		case bytes.HasPrefix(line, headerEvent):
			e.Event = append([]byte(nil), trimHeader(len(headerEvent), line)...)
		case bytes.HasPrefix(line, headerRetry):
			e.Retry = append([]byte(nil), trimHeader(len(headerRetry), line)...)
		case cfg.Comments && bytes.HasPrefix(line, headerComment):
			e.Comment = append([]byte(nil), trimHeader(len(headerComment), line)...)
		default:
			// Ignore any garbage that doesn't match what we're looking for.
		}
	}

	// Trim the last "\n" per the spec.
	e.Data = bytes.TrimSuffix(e.Data, []byte("\n"))

	if cfg.EncodingBase64 {
		buf := make([]byte, base64.StdEncoding.DecodedLen(len(e.Data)))

		n, decodeErr := base64.StdEncoding.Decode(buf, e.Data)
		if decodeErr != nil {
			err = fmt.Errorf("failed to decode event message: %s", decodeErr)
		}
		e.Data = buf[:n]
	}
	return &e, err
}

func trimHeader(size int, data []byte) []byte {
	if data == nil || len(data) < size {
		return data
	}

	data = data[size:]
	// Remove optional leading whitespace
	if len(data) > 0 && data[0] == 32 {
		data = data[1:]
	}
	// Remove trailing new line
	if len(data) > 0 && data[len(data)-1] == 10 {
		data = data[:len(data)-1]
	}
	return data
}

type EventWriteConfig struct {
	SplitData bool
}

func (e *Event) Write(w io.Writer, cfg EventWriteConfig) (int, error) {
	var nWritten int
	writef := func(format string, a ...interface{}) error {
		n, err := fmt.Fprintf(w, format, a...)
		nWritten += n
		return err
	}

	if len(e.Data) > 0 {
		if err := writef("id: %s\n", e.ID); err != nil {
			return nWritten, err
		}

		if cfg.SplitData {
			sd := bytes.Split(e.Data, []byte("\n"))
			for i := range sd {
				if err := writef("data: %s\n", sd[i]); err != nil {
					return nWritten, err
				}
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		} else {
			if bytes.HasPrefix(e.Data, []byte(":")) {
				if err := writef("%s\n", e.Data); err != nil {
					return nWritten, err
				}
			} else {
				if err := writef("data: %s\n", e.Data); err != nil {
					return nWritten, err
				}
			}
		}

		if len(e.Event) > 0 {
			if err := writef("event: %s\n", e.Event); err != nil {
				return nWritten, err
			}
		}

		if len(e.Retry) > 0 {
			if err := writef("retry: %s\n", e.Retry); err != nil {
				return nWritten, err
			}
		}
	}

	if len(e.Comment) > 0 {
		if err := writef(": %s\n", e.Comment); err != nil {
			return nWritten, err
		}
	}

	return nWritten, nil
}

func (e *Event) hasContent() bool {
	return len(e.ID) > 0 || len(e.Data) > 0 || len(e.Event) > 0 || len(e.Retry) > 0
}

func (e *Event) hasComment() bool {
	return len(e.Comment) > 0
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

	split := func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// We have a full event payload to parse.
		if i, nlen := containsDoubleNewline(data); i >= 0 {
			return i + nlen, data[0:i], nil
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

// Returns a tuple containing the index of a double newline, and the number of bytes
// represented by that sequence. If no double newline is present, the first value
// will be negative.
func containsDoubleNewline(data []byte) (int, int) {
	// Search for each potentially valid sequence of newline characters
	crcr := bytes.Index(data, []byte("\r\r"))
	lflf := bytes.Index(data, []byte("\n\n"))
	crlflf := bytes.Index(data, []byte("\r\n\n"))
	lfcrlf := bytes.Index(data, []byte("\n\r\n"))
	crlfcrlf := bytes.Index(data, []byte("\r\n\r\n"))
	// Find the earliest position of a double newline combination
	minPos := minPosInt(crcr, minPosInt(lflf, minPosInt(crlflf, minPosInt(lfcrlf, crlfcrlf))))
	// Detemine the length of the sequence
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
