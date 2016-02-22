package sse

import (
	"bufio"
	"bytes"
	"net/http"
)

var (
	headerID    = []byte("id:")
	headerData  = []byte("data:")
	headerEvent = []byte("event:")
	headerError = []byte("error:")
)

// Client handles an incoming server stream
type Client struct {
	URL        string
	Connection *http.Client
}

// NewClient creates a new client
func NewClient(url string) *Client {
	return &Client{
		URL:        url,
		Connection: &http.Client{},
	}
}

// Subscribe to a data stream
func (c *Client) Subscribe(stream string, handler func(msg []byte)) error {
	resp, err := c.request(stream)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	for {
		// Read each new line and process the type of event
		line, err := reader.ReadBytes('\n')
		msg := processEvent(line)
		if err != nil {
			return err
		}

		// If we have data, pass it to the handler
		if msg.Data != nil {
			handler(msg.Data)
		}
	}
}

func (c *Client) request(stream string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.URL, nil)
	if err != nil {
		return nil, err
	}

	// Setup request, specify stream to connect to
	req.URL.Query().Add("stream", stream)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.Connection.Do(req)
	return resp, nil
}

func processEvent(msg []byte) *Event {
	e := Event{}
	switch h := msg[:6]; {
	case bytes.Contains(h, headerID):
		e.ID = trimHeader(len(headerID), msg)
	case bytes.Contains(h, headerData):
		e.Data = trimHeader(len(headerData), msg)
	case bytes.Contains(h, headerEvent):
		e.Event = trimHeader(len(headerEvent), msg)
	case bytes.Contains(h, headerError):
		e.Error = trimHeader(len(headerError), msg)
	}
	return &e
}

func trimHeader(size int, data []byte) []byte {
	return []byte{}
}
