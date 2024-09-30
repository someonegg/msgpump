// Copyright 2023 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpump

type Message []byte // single part

func (m Message) Size() int {
	return len(m)
}

type MPMessage [][]byte // multiple parts

func (m MPMessage) Size() int {
	l := 0
	for _, p := range m {
		l += len(p)
	}
	return l
}

type message struct {
	mS Message // or
	mM MPMessage
}

func (m message) Size() int {
	return m.mS.Size() + m.mM.Size()
}

type MessageReader interface {
	ReadMessage() (m Message, err error)
}

type MessageWriter interface {
	WriteMessage(m Message) error
	WriteMessageMP(m MPMessage) error
}

type MessageReadWriter interface {
	MessageReader
	MessageWriter
}

type StopNotifier interface {
	OnStop()
}
