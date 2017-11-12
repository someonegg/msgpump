package msgpump

import (
	"bytes"
	"testing"
)

func TestMessageDump(test *testing.T) {
	rw := &mockMRW{}

	dump := &bytes.Buffer{}

	md := &MessageDump{
		RW:   rw,
		Dump: dump,
	}

	t, m, err := md.ReadMessage()
	if err != nil {
		test.Fatal(err)
	}

	err = md.WriteMessage(t, m)
	if err != nil {
		test.Fatal(err)
	}

	if dump.Len() != 22 {
		test.Fatal("dump format")
	}
}

func TestMessageDumpFilter(test *testing.T) {
	rw := &mockMRW{}

	dump := &bytes.Buffer{}

	md := &MessageDump{
		RW:     rw,
		Dump:   dump,
		Filter: func(t string, m Message, read bool) bool { return !read },
	}

	t, m, err := md.ReadMessage()
	if err != nil {
		test.Fatal(err)
	}

	err = md.WriteMessage(t, m)
	if err != nil {
		test.Fatal(err)
	}

	if dump.Len() != 11 {
		test.Fatal("dump filter")
	}
}
