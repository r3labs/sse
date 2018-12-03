/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var url string
var srv *Server
var server *httptest.Server

func setup(empty bool) {
	// New Server
	srv = New()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", srv.HTTPHandler)
	server = httptest.NewServer(mux)
	url = server.URL + "/events"

	srv.CreateStream("test")

	// Send continuous string of events to the client
	go func(s *Server) {
		for {
			if empty {
				s.Publish("test", &Event{Data: []byte("\n")})
			} else {
				s.Publish("test", &Event{Data: []byte("ping")})
			}
			time.Sleep(time.Millisecond * 50)
		}
	}(srv)
}

func cleanup() {
	server.CloseClientConnections()
	server.Close()
	srv.Close()
}

func TestClientSubscribe(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(url)

	events := make(chan *Event)
	var cErr error
	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if msg.Data != nil {
				events <- msg
				return
			}
		})
	}()

	for i := 0; i < 5; i++ {
		msg, err := wait(events, time.Second*1)
		require.Nil(t, err)
		assert.Equal(t, []byte(`ping`), msg)
	}

	assert.Nil(t, cErr)
}

func TestClientChanSubscribeEmptyMessage(t *testing.T) {
	setup(true)
	defer cleanup()

	c := NewClient(url)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	for i := 0; i < 5; i++ {
		_, err := waitEvent(events, time.Second)
		require.Nil(t, err)
	}
}

func TestClientChanSubscribe(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(url)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	for i := 0; i < 5; i++ {
		msg, merr := wait(events, time.Second*1)
		if msg == nil {
			i--
			continue
		}
		assert.Nil(t, merr)
		assert.Equal(t, []byte(`ping`), msg)
	}
	c.Unsubscribe(events)
}

func TestClientOnDisconnect(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(url)

	called := make(chan bool)
	c.OnDisconnect(func(client *Client) {
		called <- true
	})

	go c.Subscribe("test", func(msg *Event) {})

	time.Sleep(time.Second)
	server.CloseClientConnections()

	assert.True(t, <-called)
}

func TestClientChanReconnect(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(url)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	for i := 0; i < 10; i++ {
		if i == 5 {
			// kill connection
			server.CloseClientConnections()
		}
		msg, merr := wait(events, time.Second*1)
		if msg == nil {
			i--
			continue
		}
		assert.Nil(t, merr)
		assert.Equal(t, []byte(`ping`), msg)
	}
	c.Unsubscribe(events)
}

func TestClientUnsubscribe(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(url)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	time.Sleep(time.Millisecond * 500)

	go c.Unsubscribe(events)
	go c.Unsubscribe(events)
}
