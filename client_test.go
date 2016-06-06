/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

var url string

func setup() {
	// New Server
	s := New()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.HTTPHandler)
	server := httptest.NewServer(mux)
	url = server.URL + "/events"

	s.CreateStream("test")

	// Send continuous string of events to the client
	go func(s *Server) {
		for {
			s.Publish("test", []byte("ping"))
			time.Sleep(time.Millisecond * 500)
		}
	}(s)
}

func TestClient(t *testing.T) {
	setup()
	Convey("Given a new Client", t, func() {
		c := NewClient(url)

		Convey("When connecting to a new stream", func() {
			Convey("It should receive events ", func() {
				events := make(chan []byte)
				var cErr error
				go func(cErr error) {
					cErr = c.Subscribe("test", func(msg *Event) {
						if msg.Data != nil {
							events <- msg.Data
							return
						}
					})
				}(cErr)

				for i := 0; i < 5; i++ {
					msg, err := wait(events, time.Second*1)
					So(err, ShouldBeNil)
					So(string(msg), ShouldEqual, "ping")
				}
				So(cErr, ShouldBeNil)
			})

		})
	})
}
