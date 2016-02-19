package sse

// Subscriber ...
type Subscriber struct {
	Quit       chan *Subscriber
	Connection chan []byte
}

// Close will let the stream know that the clients connection has terminated
func (s *Subscriber) Close() {
	s.Quit <- s
}
