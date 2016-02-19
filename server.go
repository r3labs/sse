package sse

const (
	// DefaultBufferSize size of the queue that holds the streams messages.
	DefaultBufferSize = 1024
)

// Server ...
type Server struct {
	BufferSize       int
	streams          map[string]*Stream
	registerStream   chan StreamRegistration
	unregisterStream chan string
	quit             chan bool
}

// StreamRegistration ...
type StreamRegistration struct {
	id     string
	stream *Stream
}

// NewServer will create a server and setup defaults
func NewServer() *Server {
	return &Server{
		BufferSize:       DefaultBufferSize,
		streams:          make(map[string]*Stream),
		registerStream:   make(chan StreamRegistration),
		unregisterStream: make(chan string),
		quit:             make(chan bool),
	}
}

// Run starts the server listening ...
func (s *Server) Run() {
	go func(s *Server) {
		for {
			select {
			case reg := <-s.registerStream:
				s.streams[reg.id] = reg.stream
				s.streams[reg.id].run()

			case id := <-s.unregisterStream:
				s.streams[id].close()
				delete(s.streams, id)

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
	s.unregisterStream <- id
}

// GetStream returns a stream entity based on its id
func (s *Server) GetStream(id string) *Stream {
	return s.streams[id]
}
