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
	stream := s.getStream(streamID)

	if stream == nil {
		http.Error(w, "Stream not found!", http.StatusInternalServerError)
		return
	}

	// Create the stream subscriber
	sub := stream.addSubscriber()
	defer sub.close()

	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		sub.close()
	}()

	// Push events to client
	for {
		fmt.Fprintf(w, "data: %s\n\n", <-sub.connection)
		flusher.Flush()
	}
}
