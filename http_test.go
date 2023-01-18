/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPStreamHandler(t *testing.T) {
	s := New()
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	c := NewClient(server.URL + "/events")

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

	// Wait for subscriber to be registered and message to be published
	time.Sleep(time.Millisecond * 200)
	require.Nil(t, cErr)
	s.Publish("test", &Event{Data: []byte("test")})

	msg, err := wait(events, time.Millisecond*500)
	require.Nil(t, err)
	assert.Equal(t, []byte(`test`), msg)
}

func TestHTTPStreamHandlerExistingEvents(t *testing.T) {
	s := New()
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	s.Publish("test", &Event{Data: []byte("test 1")})
	s.Publish("test", &Event{Data: []byte("test 2")})
	s.Publish("test", &Event{Data: []byte("test 3")})

	time.Sleep(time.Millisecond * 100)

	c := NewClient(server.URL + "/events")

	events := make(chan *Event)
	var cErr error
	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if len(msg.Data) > 0 {
				events <- msg
			}
		})
	}()

	require.Nil(t, cErr)

	for i := 1; i <= 3; i++ {
		msg, err := wait(events, time.Millisecond*500)
		require.Nil(t, err)
		assert.Equal(t, []byte("test "+strconv.Itoa(i)), msg)
	}
}

func TestHTTPStreamHandlerEventID(t *testing.T) {
	s := New()
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	s.Publish("test", &Event{Data: []byte("test 1")})
	s.Publish("test", &Event{Data: []byte("test 2")})
	s.Publish("test", &Event{Data: []byte("test 3")})

	time.Sleep(time.Millisecond * 100)

	c := NewClient(server.URL + "/events")
	c.LastEventID.Store([]byte("2"))

	events := make(chan *Event)
	var cErr error
	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if len(msg.Data) > 0 {
				events <- msg
			}
		})
	}()

	require.Nil(t, cErr)

	msg, err := wait(events, time.Millisecond*500)
	require.Nil(t, err)
	assert.Equal(t, []byte("test 3"), msg)
}

func TestHTTPStreamHandlerEventTTL(t *testing.T) {
	s := New()
	defer s.Close()

	s.EventTTL = time.Second * 1

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	s.Publish("test", &Event{Data: []byte("test 1")})
	s.Publish("test", &Event{Data: []byte("test 2")})
	time.Sleep(time.Second * 2)
	s.Publish("test", &Event{Data: []byte("test 3")})

	time.Sleep(time.Millisecond * 100)

	c := NewClient(server.URL + "/events")

	events := make(chan *Event)
	var cErr error
	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if len(msg.Data) > 0 {
				events <- msg
			}
		})
	}()

	require.Nil(t, cErr)

	msg, err := wait(events, time.Millisecond*500)
	require.Nil(t, err)
	assert.Equal(t, []byte("test 3"), msg)
}

func TestHTTPStreamHandlerHeaderFlushIfNoEvents(t *testing.T) {
	s := New()
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	c := NewClient(server.URL + "/events")

	subscribed := make(chan struct{})
	events := make(chan *Event)
	go func() {
		assert.NoError(t, c.SubscribeChan("test", events))
		subscribed <- struct{}{}
	}()

	select {
	case <-subscribed:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Subscribe should returned in 100 milliseconds")
	}
}

func TestHTTPStreamHandlerAutoStream(t *testing.T) {
	t.Parallel()

	sseServer := New()
	defer sseServer.Close()

	sseServer.AutoReplay = false

	sseServer.AutoStream = true

	mux := http.NewServeMux()
	mux.HandleFunc("/events", sseServer.ServeHTTP)
	server := httptest.NewServer(mux)

	c := NewClient(server.URL + "/events")

	events := make(chan *Event)

	cErr := make(chan error)

	go func() {
		cErr <- c.SubscribeChan("test", events)
	}()

	require.Nil(t, <-cErr)

	sseServer.Publish("test", &Event{Data: []byte("test")})

	msg, err := wait(events, 1*time.Second)

	require.Nil(t, err)

	assert.Equal(t, []byte(`test`), msg)

	c.Unsubscribe(events)

	_, _ = wait(events, 1*time.Second)

	assert.Equal(t, (*Stream)(nil), sseServer.getStream("test"))
}
