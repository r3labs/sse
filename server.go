package sse

import (
	"fmt"
	"net/http"
)

const (
	// DefaultBufferSize size of the queue that holds the streams messages.
	DefaultBufferSize = 1024
)

// Server ...
type Server struct {
	BufferSize       int
	DefaultStream    bool
	streams          map[string]*Stream
	registerStream   chan StreamRegistration
	deregisterStream chan string
	quit             chan bool
}

// NewServer will create a server and setup defaults
func NewServer() *Server {
	return &Server{
		BufferSize:       DefaultBufferSize,
		DefaultStream:    false,
		streams:          make(map[string]*Stream),
		registerStream:   make(chan StreamRegistration),
		deregisterStream: make(chan string),
		quit:             make(chan bool),
	}
}

// Start the server's internal messaging
func (s *Server) Start() {
	go func(s *Server) {
		for {
			select {
			// Add new streams
			case reg := <-s.registerStream:
				s.streams[reg.id] = reg.stream
				s.streams[reg.id].run()

			// Remove old streams
			case id := <-s.deregisterStream:
				s.streams[id].close()
				delete(s.streams, id)

			// Close all streams
			case <-s.quit:
				for id := range s.streams {
					s.streams[id].quit <- true
					delete(s.streams, id)
				}
				return
			}
		}
	}(s)
}

// Close shuts down the server, closes all of the streams and connections
func (s *Server) Close() {
	s.quit <- true
}

// CreateStream will add a new stream
func (s *Server) CreateStream(id string) *Stream {
	sr := StreamRegistration{
		id:     id,
		stream: NewStream(s.BufferSize),
	}
	s.registerStream <- sr

	return sr.stream
}

// RemoveStream will remove a stream
func (s *Server) RemoveStream(id string) {
	s.deregisterStream <- id
}

// GetStream returns a stream entity based on its id
func (s *Server) GetStream(id string) *Stream {
	// Hashmap is unsafe, might cause race if register/deregister
	// is run at the same time
	return s.streams[id]
}

// HTTPHandler serves new connections with events for a given stream ...
func (s *Server) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	flusher, err := w.(http.Flusher)
	if !err {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get the StreamID from
	streamID := r.URL.Query().Get("streamID")

	stream := s.GetStream(streamID)
	if stream == nil {
		http.Error(w, "Stream not found!", http.StatusInternalServerError)
		return
	}

	sub := stream.NewSubscriber()
	defer sub.Close()

	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		sub.Close()
	}()
	for {
		// Write to the ResponseWriter
		// Server Sent Events compatible
		fmt.Fprintf(w, "data: %s\n\n", <-sub.Connection)
		// Flush the data immediatly instead of buffering it for later.
		flusher.Flush()
	}
}
