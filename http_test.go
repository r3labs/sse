package sse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHTTP(t *testing.T) {
	// New Server
	s := New()
	s.CreateStream("test")

	go func(s *Server) {
		for {
			s.Publish("test", []byte("ping"))
			time.Sleep(time.Second * 1)
		}
	}(s)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.HTTPHandler)

	server := httptest.NewServer(mux)
	defer server.Close()
	fmt.Println(server.URL)

	time.Sleep(time.Minute * 2)

	Convey("Given a new http Handler", t, func() {
		s.CreateStream("test")

		Convey("When creating a new stream", func() {
			c := NewClient(server.URL + "/events")

			Convey("It should publish events to its subscribers", func() {
				events := make(chan []byte)
				go c.Subscribe("test", func(msg []byte) {
					fmt.Println(string(msg))
					events <- msg
				})

				time.Sleep(time.Millisecond * 100)

				s.Publish("test", []byte("test"))

				msg, err := wait(events, time.Millisecond*500)
				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "test")
			})

		})
	})

}
