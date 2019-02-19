msgpump
======

msgpump provides a message-pump facility with golang.

The message-pump will continuously receive, process and send messages after startup.

The message-peer implements a synchronous request response model over message-pump, it
considers the client and server peers, allowing each to send requests to the other concurrently.

The transport layer is customizable, there are already implementations over net.Conn
and websocket.Conn.

Documentation
-------------

- [API Reference](http://godoc.org/github.com/someonegg/msgpump/msgpeer)

Installation
------------

Install msgpump using the "go get" command:

    go get github.com/someonegg/msgpump

The Go distribution is msgpump's only dependency.
