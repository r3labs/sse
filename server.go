package sse

import "sync"

// DefaultBufferSize size of the queue that holds the streams messages.
const DefaultBufferSize = 1024

// Server ...
type Server struct {
	BufferSize int
	streams    map[string]*Stream
	mu         sync.Mutex
}

// New will create a server and setup defaults
func New() *Server {
	return &Server{
		BufferSize: DefaultBufferSize,
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

	s.streams[id] = str

	return str
}

// RemoveStream will remove a stream
func (s *Server) RemoveStream(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.streams[id].close()
	delete(s.streams, id)
}

// Publish sends a mesage to every client in a streamID// Publish sends an event to all subcribers of a stream
func (s *Server) Publish(id string, event []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[id].event <- event
}

func (s *Server) getStream(id string) *Stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streams[id]
}
