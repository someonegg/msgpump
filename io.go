// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpump

type Message []byte

type MessageReader interface {
	ReadMessage() (t string, m Message, err error)
}

type MessageWriter interface {
	WriteMessage(t string, m Message) error
}

type MessageReadWriter interface {
	MessageReader
	MessageWriter
}

type Flusher interface {
	Flush() error
}

type StopNotifier interface {
	OnStop()
}

type StopNotifierFunc func()

func (f StopNotifierFunc) OnStop() {
	f()
}
