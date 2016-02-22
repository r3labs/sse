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

	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	defer server.Close()

	Convey("Given a new http Handler", t, func() {
		Convey("When creating a new stream", func() {
			s.CreateStream("test")

			Convey("It should be stored", func() {
				So(s.getStream("test"), ShouldNotBeNil)
			})
			Convey("It should be started", func() {
			})
		})

		Convey("When removing a stream", func() {
			s.CreateStream("test")
			s.RemoveStream("test")

			Convey("It should be removed", func() {
				So(s.getStream("test"), ShouldBeNil)
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
