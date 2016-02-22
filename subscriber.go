package sse

// Subscriber ...
type Subscriber struct {
	quit       chan *Subscriber
	connection chan []byte
}

// Close will let the stream know that the clients connection has terminated
func (s *Subscriber) close() {
	s.quit <- s
}
