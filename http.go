/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package sse

import (
	"fmt"
	"net/http"
)

// HTTPHandler serves new connections with events for a given stream ...
func (s *Server) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	flusher, err := w.(http.Flusher)
	if !err {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get the StreamID from the URL
	streamID := r.URL.Query().Get("stream")
	if streamID == "" {
		http.Error(w, "Please specify a stream!", http.StatusInternalServerError)
		return
	}

	stream := s.getStream(streamID)

	if stream == nil && !s.AutoStream {
		http.Error(w, "Stream not found!", http.StatusInternalServerError)
		return
	} else if stream == nil && s.AutoStream {
		stream = s.CreateStream(streamID)
	}

	eventid := r.Header.Get("Last-Event-ID")
	if eventid == "" {
		eventid = "0"
	}

	// Create the stream subscriber
	sub := stream.addSubscriber(eventid)
	defer sub.close()

	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		sub.close()
	}()

	// Push events to client
	for {
		select {
		case ev, ok := <-sub.connection:
			if !ok {
				return
			}

			// If the data buffer is an empty string abort.
			if len(ev.Data) == 0 {
				break
			}

			fmt.Fprintf(w, "id: %s\n", ev.ID)
			fmt.Fprintf(w, "data: %s\n", ev.Data)
			if len(ev.Event) > 0 {
				fmt.Fprintf(w, "event: %s\n", ev.Event)
			}
			if len(ev.Retry) > 0 {
				fmt.Fprintf(w, "retry: %s\n", ev.Retry)
			}
			fmt.Fprint(w, "\n")
			flusher.Flush()
		}
	}
}
