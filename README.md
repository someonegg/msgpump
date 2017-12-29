msgpump
======

msgpump provides a message-pump facility with golang.

The message-pump is a facility which will receive and dispatch messages
continuously after startup. It will also send messages continuously and
asynchronously after startup.

The message is defined as variable-length byte array, they are distinguished
by the message-type (a string).

Documentation
-------------

- [API Reference](http://godoc.org/github.com/someonegg/msgpump)

Installation
------------

Install msgpump using the "go get" command:

    go get github.com/someonegg/msgpump

The Go distribution is msgpump's only dependency.
