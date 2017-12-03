package mail

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/mail"
)

// This file implements access to mbox files.
// The mbox source file is expected to be a random access
// SectionReader, which will be indexed at first access.
//
// The implementation assumes quoting of lines starting with "From "
// aka the "mboxrd" format.

type Mailbox struct {
	r    io.ReaderAt
	msgs []mboxMsg
}

type mboxMsg struct {
	offset  int
	length  int
	subject string
}

func Open(r io.ReaderAt) (Mailbox, error) {
	// 1GB should be enough for everybody :)
	rd := io.NewSectionReader(r, 0, 1<<30)
	s := bufio.NewScanner(rd)
	s.Split(scanMessage)

	offset := 0
	var msgs []mboxMsg
	for s.Scan() {
		data := s.Bytes()
		msg := mboxMsg{offset: offset, length: len(data)}
		m, err := mail.ReadMessage(bytes.NewReader(data))
		if err == nil {
			msg.subject = m.Header.Get("Subject")
		}
		msgs = append(msgs, msg)
		offset += len(data)
	}
	if err := s.Err(); err != nil {
		return Mailbox{}, err
	}
	box := Mailbox{
		r:    r,
		msgs: msgs,
	}
	return box, nil
}

func scanMessage(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) == 0 && atEOF {
		return 0, nil, nil
	}
	if !bytes.HasPrefix(data, []byte("From ")) {
		return 0, nil, fmt.Errorf("invalid mbox file: first line %30q", data)
	}
	end := bytes.Index(data, []byte("\n\nFrom "))
	if end >= 0 {
		return end + 2, data[:end+2], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func (m *Mailbox) Message(idx int) (*mail.Message, error) {
	msg := m.msgs[idx]
	r := io.NewSectionReader(m.r, msg.offset, msg.length)
	return mail.ReadMessage(r)
}