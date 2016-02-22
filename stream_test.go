package sse

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

// Tests are accessing subscriber in a non-threadsafe way.
// Maybe fix this in the future so we can test with -race enabled

func TestStream(t *testing.T) {
	Convey("Given a new stream", t, func() {
		// New Stream
		s := newStream(1024)
		s.run()

		Convey("When adding a subscriber", func() {
			sub := s.addSubscriber()

			Convey("It should be stored", func() {
				So(len(s.subscribers), ShouldEqual, 1)
			})
			Convey("It should receive messages", func() {
				s.event <- []byte("test")
				msg, err := wait(sub.connection, time.Second*1)

				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "test")
			})
		})

		Convey("When removing a subscriber", func() {
			s.addSubscriber()
			s.removeSubscriber(0)
			Convey("It should be removed from the list of subscribers", func() {
				So(len(s.subscribers), ShouldEqual, 0)
			})
		})

		Convey("When closing a subscriber down gracefully", func() {
			sub := s.addSubscriber()
			sub.close()
			time.Sleep(time.Millisecond * 100)
			Convey("It should be removed from the list of subscribers", func() {
				So(len(s.subscribers), ShouldEqual, 0)
			})
		})

		Convey("When adding multiple subscribers", func() {
			var subs []*Subscriber
			for i := 0; i < 10; i++ {
				subs = append(subs, s.addSubscriber())
			}

			// Wait for all subscribers to be added
			time.Sleep(time.Millisecond * 100)

			Convey("They should all receive messages", func() {
				s.event <- []byte("test")
				for _, sub := range subs {
					msg, err := wait(sub.connection, time.Second*1)
					So(err, ShouldBeNil)
					So(string(msg), ShouldEqual, "test")
				}
			})

			Convey("They should all shutdown gracefully when the stream is closed", func() {
				s.close()

				// Wait for all subscribers to close
				time.Sleep(time.Millisecond * 100)

				So(len(s.subscribers), ShouldEqual, 0)
			})

		})
	})

}
