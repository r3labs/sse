/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"encoding/base64"
	"sync"
)

// DefaultBufferSize size of the queue that holds the streams messages.
const DefaultBufferSize = 1024

// Server Is our main struct
type Server struct {
	// Specifies the size of the message buffer for each stream
	BufferSize int
	// Enables creation of a stream when a client connects
	AutoStream   bool
	EncodeBase64 bool
	streams      map[string]*Stream
	mu           sync.Mutex
}

// New will create a server and setup defaults
func New() *Server {
	return &Server{
		BufferSize: DefaultBufferSize,
		AutoStream: false,
		streams:    make(map[string]*Stream),
	}
}

// Close shuts down the server, closes all of the streams and connections
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id := range s.streams {
		s.streams[id].quit <- true
		delete(s.streams, id)
	}
}

// CreateStream will create a new stream and register it
func (s *Server) CreateStream(id string) *Stream {
	str := newStream(s.BufferSize)
	str.run()

	// Register new stream
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.streams[id] != nil {
		return s.streams[id]
	}

	s.streams[id] = str

	return str
}

// RemoveStream will remove a stream
func (s *Server) RemoveStream(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.streams[id] != nil {
		s.streams[id].close()
		delete(s.streams, id)
	}
}

// StreamExists checks whether a stream by a given id exists
func (s *Server) StreamExists(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.streams[id] != nil
}

// Publish sends a mesage to every client in a streamID
func (s *Server) Publish(id string, event *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.streams[id] != nil {
		s.streams[id].event <- s.process(event)
	}
}

func (s *Server) getStream(id string) *Stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streams[id]
}

func (s *Server) process(event *Event) *Event {
	if s.EncodeBase64 {
		output := make([]byte, base64.StdEncoding.EncodedLen(len(event.Data)))
		base64.StdEncoding.Encode(output, event.Data)
		event.Data = output
	}
	return event
}
