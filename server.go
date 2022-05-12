/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"encoding/base64"
	"sync"
	"time"
)

// DefaultBufferSize size of the queue that holds the streams messages.
const DefaultBufferSize = 1024

// Server Is our main struct
type Server struct {
	Streams map[string]*Stream
	Headers map[string]string
	// Sets a ttl that prevents old events from being transmitted
	EventTTL time.Duration
	// Specifies the size of the message buffer for each stream
	BufferSize int
	mu         sync.Mutex
	// Encodes all data as base64
	EncodeBase64 bool
	// Splits an events data into multiple data: entries
	SplitData bool
	// Enables creation of a stream when a client connects
	AutoStream bool
	// Enables automatic replay for each new subscriber that connects
	AutoReplay bool

	// Specifies the function to run when client subscribe or un-subscribe
	OnSubscribe   func(streamID string, sub *Subscriber)
	OnUnsubscribe func(streamID string, sub *Subscriber)
}

// New will create a server and setup defaults
func New() *Server {
	return &Server{
		BufferSize: DefaultBufferSize,
		AutoStream: false,
		AutoReplay: true,
		Streams:    make(map[string]*Stream),
		Headers:    map[string]string{},
	}
}

// NewWithCallback will create a server and setup defaults with callback function
func NewWithCallback(onSubscribe, onUnsubscribe func(streamID string, sub *Subscriber)) *Server {
	return &Server{
		BufferSize:    DefaultBufferSize,
		AutoStream:    false,
		AutoReplay:    true,
		Streams:       make(map[string]*Stream),
		Headers:       map[string]string{},
		OnSubscribe:   onSubscribe,
		OnUnsubscribe: onUnsubscribe,
	}
}

// Close shuts down the server, closes all of the streams and connections
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id := range s.Streams {
		s.Streams[id].quit <- struct{}{}
		delete(s.Streams, id)
	}
}

// CreateStream will create a new stream and register it
func (s *Server) CreateStream(id string) *Stream {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Streams[id] != nil {
		return s.Streams[id]
	}

	str := newStream(id, s.BufferSize, s.AutoReplay, s.AutoStream, s.OnSubscribe, s.OnUnsubscribe)
	str.run()

	s.Streams[id] = str

	return str
}

// RemoveStream will remove a stream
func (s *Server) RemoveStream(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Streams[id] != nil {
		s.Streams[id].close()
		delete(s.Streams, id)
	}
}

// StreamExists checks whether a stream by a given id exists
func (s *Server) StreamExists(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.Streams[id] != nil
}

// Publish sends a mesage to every client in a streamID
func (s *Server) Publish(id string, event *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Streams[id] != nil {
		s.Streams[id].event <- s.process(event)
	}
}

func (s *Server) getStream(id string) *Stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Streams[id]
}

func (s *Server) process(event *Event) *Event {
	if s.EncodeBase64 {
		output := make([]byte, base64.StdEncoding.EncodedLen(len(event.Data)))
		base64.StdEncoding.Encode(output, event.Data)
		event.Data = output
	}
	return event
}
