package sse

// Stream ...
type Stream struct {
	subscribers []*Subscriber
	register    chan *Subscriber
	deregister  chan *Subscriber
	event       chan []byte
	quit        chan bool
}

// StreamRegistration ...
type StreamRegistration struct {
	id     string
	stream *Stream
}

// newStream returns a new stream
func newStream(bufsize int) *Stream {
	return &Stream{
		subscribers: make([]*Subscriber, 0),
		register:    make(chan *Subscriber),
		deregister:  make(chan *Subscriber),
		event:       make(chan []byte, bufsize),
		quit:        make(chan bool),
	}
}

// addSubscriber will create a new subscriber on a stream
func (str *Stream) addSubscriber() *Subscriber {
	sub := &Subscriber{
		quit:       str.deregister,
		Connection: make(chan []byte, 64),
	}

	str.register <- sub
	return sub
}

func (str *Stream) run() {
	go func(str *Stream) {
		for {
			select {
			// Add new subscriber
			case subscriber := <-str.register:
				str.subscribers = append(str.subscribers, subscriber)

			// Remove closed subscriber
			case subscriber := <-str.deregister:
				i := str.getSubIndex(subscriber)
				if i != -1 {
					str.removeSubscriber(i)
				}

			// Publish event to subscribers
			case event := <-str.event:
				for i := range str.subscribers {
					str.subscribers[i].Connection <- event
				}

			// Shutdown if the server closes
			case <-str.quit:
				// remove connections
				for i := range str.subscribers {
					str.removeSubscriber(i)
				}
				return
			}
		}
	}(str)
}

func (str *Stream) close() {
	str.quit <- true
}

func (str *Stream) getSubIndex(sub *Subscriber) int {
	for i := range str.subscribers {
		if str.subscribers[i] == sub {
			return i
		}
	}
	return -1
}

func (str *Stream) removeSubscriber(i int) {
	close(str.subscribers[i].Connection)
	str.subscribers = append(str.subscribers[:i], str.subscribers[i+1:]...)
}
