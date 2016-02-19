package sse

import "fmt"

// Stream ...
type Stream struct {
	subscribers []*Subscriber
	register    chan *Subscriber
	unregister  chan *Subscriber
	event       chan []byte
	quit        chan bool
}

// NewStream returns a new stream
func NewStream(bufsize int) *Stream {
	return &Stream{
		subscribers: make([]*Subscriber, 0),
		register:    make(chan *Subscriber),
		unregister:  make(chan *Subscriber),
		event:       make(chan []byte, bufsize),
		quit:        make(chan bool),
	}
}

// NewSubscriber will create a new subscriber on a stream
func (str *Stream) NewSubscriber() *Subscriber {
	sub := &Subscriber{
		Quit:       str.unregister,
		Connection: make(chan []byte),
	}

	str.register <- sub
	return sub
}

// Publish sends an event to all subcribers of a stream
func (str *Stream) Publish(event []byte) {
	str.event <- event
}

func (str *Stream) run() {
	go func(str *Stream) {
		for {
			fmt.Println(len(str.subscribers))
			select {
			// Add new subscriber
			case subscriber := <-str.register:
				str.subscribers = append(str.subscribers, subscriber)

			// Remove closed subscriber
			case subscriber := <-str.unregister:
				i := str.getSubIndex(subscriber)
				if i != -1 {
					str.removeSubscriber(i)
				}

			// Publish event to subscribers
			case event := <-str.event:
				fmt.Println("got event!")
				for i := range str.subscribers {
					fmt.Printf("publishing to subscriber %d\n", i)
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
