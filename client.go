/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	backoff "gopkg.in/cenkalti/backoff.v1"
)

var (
	headerID    = []byte("id:")
	headerData  = []byte("data:")
	headerEvent = []byte("event:")
	headerRetry = []byte("retry:")
)

// ConnCallback defines a function to be called on a particular connection event
type ConnCallback func(c *Client)

// Client handles an incoming server stream
type Client struct {
	URL            string
	Connection     *http.Client
	Retry          time.Time
	subscribed     map[chan *Event]chan bool
	Headers        map[string]string
	EncodingBase64 bool
	EventID        string
	disconnectcb   ConnCallback
	mu             sync.Mutex
}

// NewClient creates a new client
func NewClient(url string) *Client {
	return &Client{
		URL:        url,
		Connection: &http.Client{},
		Headers:    make(map[string]string),
		subscribed: make(map[chan *Event]chan bool),
	}
}

// Subscribe to a data stream
func (c *Client) Subscribe(stream string, handler func(msg *Event)) error {
	operation := func() error {
		resp, err := c.request(stream)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		reader := NewEventStreamReader(resp.Body)

		for {
			// Read each new line and process the type of event
			event, err := reader.ReadEvent()
			if err != nil {
				if err == io.EOF {
					return nil
				}

				// run user specified disconnect function
				if c.disconnectcb != nil {
					c.disconnectcb(c)
				}

				return err
			}

			// If we get an error, ignore it.
			if msg, err := c.processEvent(event); err == nil {
				if len(msg.ID) > 0 {
					c.EventID = string(msg.ID)
				} else {
					msg.ID = []byte(c.EventID)
				}

				handler(msg)
			}
		}
	}
	return backoff.Retry(operation, backoff.NewExponentialBackOff())
}

// SubscribeChan sends all events to the provided channel
func (c *Client) SubscribeChan(stream string, ch chan *Event) error {
	var connected bool
	errch := make(chan error)
	c.mu.Lock()
	c.subscribed[ch] = make(chan bool)
	c.mu.Unlock()

	go func() {
		operation := func() error {
			resp, err := c.request(stream)
			if err != nil {
				c.cleanup(resp, ch)
				return err
			}

			if resp.StatusCode != 200 {
				c.cleanup(resp, ch)
				return errors.New("could not connect to stream")
			}

			if !connected {
				errch <- nil
				connected = true
			}

			reader := NewEventStreamReader(resp.Body)

			for {
				// Read each new line and process the type of event
				event, err := reader.ReadEvent()
				if err != nil {
					if err == io.EOF {
						c.cleanup(resp, ch)
						return nil
					}

					// run user specified disconnect function
					if c.disconnectcb != nil {
						c.disconnectcb(c)
					}

					return err
				}

				// If we get an error, ignore it.
				if msg, err := c.processEvent(event); err == nil {
					if len(msg.ID) > 0 {
						c.EventID = string(msg.ID)
					} else {
						msg.ID = []byte(c.EventID)
					}

					select {
					case <-c.subscribed[ch]:
						c.cleanup(resp, ch)
						return nil
					case ch <- msg:
						// message sent
					}
				}
			}
		}

		err := backoff.Retry(operation, backoff.NewExponentialBackOff())
		if err != nil && !connected {
			errch <- err
		}
	}()
	err := <-errch
	close(errch)

	return err
}

// SubscribeRaw to an sse endpoint
func (c *Client) SubscribeRaw(handler func(msg *Event)) error {
	return c.Subscribe("", handler)
}

// SubscribeChanRaw sends all events to the provided channel
func (c *Client) SubscribeChanRaw(ch chan *Event) error {
	return c.SubscribeChan("", ch)
}

// Unsubscribe unsubscribes a channel
func (c *Client) Unsubscribe(ch chan *Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscribed[ch] != nil {
		c.subscribed[ch] <- true
	}
}

// OnDisconnect specifies the function to run when the connection disconnects
func (c *Client) OnDisconnect(fn ConnCallback) {
	c.disconnectcb = fn
}

func (c *Client) request(stream string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.URL, nil)
	if err != nil {
		return nil, err
	}

	// Setup request, specify stream to connect to
	if stream != "" {
		query := req.URL.Query()
		query.Add("stream", stream)
		req.URL.RawQuery = query.Encode()
	}

	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")

	if c.EventID != "" {
		req.Header.Set("Last-Event-ID", c.EventID)
	}

	// Add user specified headers
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	return c.Connection.Do(req)
}

func (c *Client) processEvent(msg []byte) (event *Event, err error) {
	var e Event

	if len(msg) < 1 {
		return nil, errors.New("event message was empty")
	}

	// Normalize the crlf to lf to make it easier to split the lines.
	bytes.Replace(msg, []byte("\n\r"), []byte("\n"), -1)
	// Split the line by "\n" or "\r", per the spec.
	for _, line := range bytes.FieldsFunc(msg, func(r rune) bool { return r == '\n' || r == '\r' }) {
		switch {
		case bytes.HasPrefix(line, headerID):
			e.ID = trimHeader(len(headerID), line)
		case bytes.HasPrefix(line, headerData):
			// The spec allows for multiple data fields per event, concatenated them with "\n".
			e.Data = append(append(trimHeader(len(headerData), line), e.Data[:]...), byte('\n'))
		// The spec says that a line that simply contains the string "data" should be treated as a data field with an empty body.
		case bytes.Equal(line, bytes.TrimSuffix(headerData, []byte(":"))):
			e.Data = append(e.Data, byte('\n'))
		case bytes.HasPrefix(line, headerEvent):
			e.Event = trimHeader(len(headerEvent), line)
		case bytes.HasPrefix(line, headerRetry):
			e.Retry = trimHeader(len(headerRetry), line)
		default:
			// Ignore any garbage that doesn't match what we're looking for.
		}
	}

	// Trim the last "\n" per the spec.
	e.Data = bytes.TrimSuffix(e.Data, []byte("\n"))

	if c.EncodingBase64 {
		buf := make([]byte, base64.StdEncoding.DecodedLen(len(e.Data)))

		_, err := base64.StdEncoding.Decode(buf, e.Data)
		if err != nil {
			err = fmt.Errorf("failed to decode event message: %s", err)
		}
		e.Data = buf
	}
	return &e, err
}

func (c *Client) cleanup(resp *http.Response, ch chan *Event) {
	if resp != nil {
		resp.Body.Close()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscribed[ch] != nil {
		close(c.subscribed[ch])
		delete(c.subscribed, ch)
	}
}

func trimHeader(size int, data []byte) []byte {
	data = data[size:]
	// Remove optional leading whitespace
	if data[0] == 32 {
		data = data[1:]
	}
	// Remove trailing new line
	if len(data) > 0 && data[len(data)-1] == 10 {
		data = data[:len(data)-1]
	}
	return data
}
