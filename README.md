msgpump
======

msgpump provides a message-pump facility with golang.

The message-pump will continuously receive, process and send messages after startup.

The message is defined as variable-length byte array, they are distinguished
by the message-type (a string).

The transport layer is customizable, there are already implementations over net.Conn
and websocket.Conn.

Documentation
-------------

- [API Reference](http://godoc.org/github.com/someonegg/msgpump)

Installation
------------

Install msgpump using the "go get" command:

    go get github.com/someonegg/msgpump

The Go distribution is msgpump's only dependency.
