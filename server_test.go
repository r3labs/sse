package sse

import (
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func wait(ch chan []byte, duration time.Duration) ([]byte, error) {
	var err error
	var msg []byte

	select {
	case event := <-ch:
		msg = event
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
				So(s.streams["test"], ShouldNotBeNil)
			})
			Convey("It should be started", func() {
			})
		})

		Convey("When removing a stream", func() {
			s.CreateStream("test")
			s.RemoveStream("test")

			Convey("It should be removed", func() {
				So(s.streams["test"], ShouldBeNil)
			})
		})

		Convey("When publishing to a stream", func() {
			s.CreateStream("test")
			stream := s.getStream("test")
			sub := stream.addSubscriber()

			s.Publish("test", []byte("test"))
			Convey("It must be received by the subscribers", func() {
				msg, err := wait(sub.connection, time.Second*1)
				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "test")
			})

		})
	})

}
