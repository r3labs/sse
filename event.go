package sse

// Event holds all of the event source fields
type Event struct {
	ID    []byte
	Data  []byte
	Event []byte
	Error []byte
}
