// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpump

import (
	"fmt"
	"io"
)

// MessageDump is a debugging helper, it implements the MessageReadWriter
// interface and provides message dump function.
//
// The dump format is:
//	 R|W:Type:MessageSize\nMessage\n\n
type MessageDump struct {
	RW   MessageReadWriter
	Dump io.Writer

	// Filter can be nil. If nil, dump all messages.
	Filter func(t string, m Message, read bool) bool
}

func (d *MessageDump) needDump(t string, m Message, read bool) bool {
	if d.Filter != nil {
		return d.Filter(t, m, read)
	}
	return true
}

func (d *MessageDump) ReadMessage() (t string, m Message, err error) {
	t, m, err = d.RW.ReadMessage()
	if err != nil {
		return
	}

	if !d.needDump(t, m, true) {
		return
	}

	fmt.Fprintf(d.Dump, "R:%v:%v\n", t, len(m))
	d.Dump.Write(m)
	fmt.Fprintf(d.Dump, "\n\n")

	return
}

func (d *MessageDump) WriteMessage(t string, m Message) (err error) {
	err = d.RW.WriteMessage(t, m)
	if err != nil {
		return
	}

	if !d.needDump(t, m, false) {
		return
	}

	fmt.Fprintf(d.Dump, "W:%v:%v\n", t, len(m))
	d.Dump.Write(m)
	fmt.Fprintf(d.Dump, "\n\n")

	return
}
