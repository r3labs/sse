# SSE - Server Side Events Client/Server Library for Go

## Synopsis

SSE is a client/server implementation for Server Side Events for Golang. Right now functionality is basic, but there is support for multiple separate streams with many clients connected to each. The client is able to specify the stream it wants to connect to by the URL parameter _stream_.

## Build status

* Master: [![CircleCI  Master](https://circleci.com/gh/r3labs/sse/tree/master.svg?style=svg)](https://circleci.com/gh/r3labs/sse/tree/master)

## Quick start

To install:

```sh
$ go get github.com/r3labs/sse
```

To Test:

```sh
$ make deps
$ make test
```

#### Example Server

There are two parts of the server. It is comprised of the message scheduler and a http handler function.
The messaging system is started when running:

```go
func main() {
	server := sse.New()
}
```

To add a stream to this handler:

```go
func main() {
	server := sse.New()
	server.CreateStream("messages")
}
```

This creates a new stream inside of the scheduler. Seeing as there are no consumers, publishing a message to this channel will do nothing.
Clients can connect to this stream once the http handler is started by specifying _stream_ as a url parameter, like so:

```
http://server/events?stream=messages
```


In order to start the http server:

```go
func main() {
	server := sse.New()

	// Create a new Mux and set the handler
	mux := http.NewServeMux()
	mux.HandleFunc("/events", server.HTTPHandler)

	http.ListenAndServe(":8080", mux)
}
```

To publish messages to a stream:

```go
func main() {
	server := sse.New()

	// Publish a payload to the stream
	server.Publish("messages", &sse.Event{
		Data: []byte("ping"),
	})
}
```

Please note there must be a stream with the name you specify and there must be subscribers to that stream


#### Example Client

The client exposes a way to connect to an SSE server. The client can also handle multiple events under the same url.

To create a new client:

```go
func main() {
	client := sse.NewClient("http://server/events")
}
```

To subscribe to an event stream, please use the Subscribe function. This accepts the name of the stream and a handler function:

```go
func main() {
	client := sse.NewClient("http://server/events")

	client.Subscribe("messages", func(msg *sse.Event) {
		// Got some data!
		fmt.Println(msg.Data)
	})
}
```

Please note that this function will block the current thread. You can run this function in a go routine.

If you wish to have events sent to a channel, you can use SubscribeChan:

```go
func main() {
	events := make(chan *sse.Event)

	client := sse.NewClient("http://server/events")
	client.SubscribeChan("messages", events)
}
```

#### HTTP client parameters

To add additional parameters to the http client, such as disabling ssl verification for self signed certs, you can override the http client or update its options:

```go
func main() {
	client := sse.NewClient("http://server/events")
	client.Connection.Transport =  &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}
```

#### URL query parameters

To set custom query parameters on the client or disable the stream parameter altogether:

```go
func main() {
	client := sse.NewClient("http://server/events?search=example")

	client.SubscribeRaw(func(msg *sse.Event) {
		// Got some data!
		fmt.Println(msg.Data)
	})
}
```


## Contributing

Please read through our
[contributing guidelines](CONTRIBUTING.md).
Included are directions for opening issues, coding standards, and notes on
development.

Moreover, if your pull request contains patches or features, you must include
relevant unit tests.

## Versioning

For transparency into our release cycle and in striving to maintain backward
compatibility, this project is maintained under [the Semantic Versioning guidelines](http://semver.org/).

## Copyright and License

Code and documentation copyright since 2015 r3labs.io authors.

Code released under
[the Mozilla Public License Version 2.0](LICENSE).
