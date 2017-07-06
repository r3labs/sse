/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func wait(ch chan *Event, duration time.Duration) ([]byte, error) {
	var err error
	var msg []byte

	select {
	case event := <-ch:
		msg = event.Data
	case <-time.After(duration):
		err = errors.New("timeout")
	}
	return msg, err
}

func TestServer(t *testing.T) {
	// New Server
	s := New()

	Convey("Given a new server", t, func() {
		Convey("When creating a new stream", func() {
			s.CreateStream("test")

			Convey("It should be stored", func() {
				So(s.getStream("test"), ShouldNotBeNil)
			})
			Convey("It should be started", func() {
			})
		})

		Convey("When removing a stream that exists", func() {
			s.CreateStream("test")
			s.RemoveStream("test")

			Convey("It should be removed", func() {
				So(s.getStream("test"), ShouldBeNil)
			})
		})

		Convey("When removing a stream that doesn't exist", func() {
			Convey("It should not panic", func() {
				So(func() { s.RemoveStream("test") }, ShouldNotPanic)
			})
		})

		Convey("When publishing to a stream that exists", func() {
			s.CreateStream("test")
			stream := s.getStream("test")
			sub := stream.addSubscriber("0")

			s.Publish("test", &Event{Data: []byte("test")})
			Convey("It must be received by the subscribers", func() {
				msg, err := wait(sub.connection, time.Second*1)
				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "test")
			})

		})

		Convey("When publishing to a stream that doesnt exist", func() {
			s.Publish("test", &Event{Data: []byte("test")})
			Convey("It must not cause an error", func() {
				So(func() { s.Publish("test", &Event{Data: []byte("test")}) }, ShouldNotPanic)
			})

		})
	})

}
