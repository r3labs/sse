package sse

// Subscriber ...
type Subscriber struct {
	Quit       chan *Subscriber
	Connection chan []byte
}
