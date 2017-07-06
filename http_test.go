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

	. "github.com/smartystreets/goconvey/convey"
)

func TestHTTP(t *testing.T) {
	// New Server
	s := New()
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.HTTPHandler)

	Convey("Given a new http Handler", t, func() {
		server := httptest.NewServer(mux)

		Convey("When creating a new stream", func() {
			s.CreateStream("test")

			Convey("It should publish events to its subscriber", func() {
				c := NewClient(server.URL + "/events")

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

		Convey("When creating a new stream with existing events", func() {
			s.CreateStream("test2")
			defer s.RemoveStream("test2")

			s.Publish("test2", &Event{Data: []byte("test 1")})
			s.Publish("test2", &Event{Data: []byte("test 2")})
			s.Publish("test2", &Event{Data: []byte("test 3")})
			time.Sleep(time.Millisecond * 100)

			Convey("And eventid is not specified", func() {
				Convey("It should publish all previous events to its subscriber", func() {
					var cErr error

					c := NewClient(server.URL + "/events")

					events := make(chan *Event)

					go func(cErr error) {
						cErr = c.Subscribe("test2", func(msg *Event) {
							if len(msg.Data) > 0 {
								events <- msg
							}
						})
					}(cErr)

					So(cErr, ShouldBeNil)

					for i := 1; i <= 3; i++ {
						msg, err := wait(events, time.Millisecond*500)
						So(err, ShouldBeNil)
						So(string(msg), ShouldEqual, "test "+strconv.Itoa(i))
					}
				})
			})

			Convey("And eventid is specified", func() {
				Convey("It should publish all events after eventid to its subscriber", func() {
					var cErr error

					c := NewClient(server.URL + "/events")
					c.EventID = "2"

					events := make(chan *Event)

					go func(cErr error) {
						cErr = c.Subscribe("test2", func(msg *Event) {
							if len(msg.Data) > 0 {
								events <- msg
							}
						})
					}(cErr)

					So(cErr, ShouldBeNil)

					for i := 3; i <= 3; i++ {
						msg, err := wait(events, time.Millisecond*500)
						So(err, ShouldBeNil)
						So(string(msg), ShouldEqual, "test "+strconv.Itoa(i))
					}
				})
			})
		})

	})
}
