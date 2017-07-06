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

func TestHTTP(t *testing.T) {
	// New Server
	s := New()
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.HTTPHandler)

	server := httptest.NewServer(mux)

	Convey("Given a new http Handler", t, func() {
		s.CreateStream("test")

		Convey("When creating a new stream", func() {
			c := NewClient(server.URL + "/events")

			Convey("It should publish events to its subscriber", func() {
				events := make(chan *Event)
				var cErr error
				go func(cErr error) {
					cErr = c.Subscribe("test", func(msg *Event) {
						if msg.Data != nil {
							events <- msg
							return
						}
					})
				}(cErr)

				// Wait for subscriber to be registered and message to be published
				time.Sleep(time.Millisecond * 200)
				So(cErr, ShouldBeNil)
				s.Publish("test", &Event{Data: []byte("test")})

				msg, err := wait(events, time.Millisecond*500)
				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "test")
			})

		})
	})
}
