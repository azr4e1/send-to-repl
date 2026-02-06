package internals

import (
	"fmt"
	"io"
	"log"
	"sync"
)

var DiscardLogger = NewLogger(io.Discard, "")
var DefaultErrHandler = func(err error) {}

type ErrHandler func(error)

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (sw *syncWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

func NewSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{
		mu: sync.Mutex{},
		w:  w,
	}
}

func NewLogger(w io.Writer, name string) *log.Logger {
	sw := NewSyncWriter(w)
	// prefix construction
	sub := "%s"
	if name != "" {
		sub += ":"
	}
	sub += " "
	logger := log.New(sw, fmt.Sprintf(sub, name), log.Ltime|log.Ldate|log.Lmsgprefix)

	return logger
}
