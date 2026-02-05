package internals

import (
	"io"
	"log"
	"os"
	"sync"
)

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (sw *syncWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

func NewSyncWriter(filename string) (*syncWriter, error) {
	w, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return &syncWriter{
		mu: sync.Mutex{},
		w:  w,
	}, nil
}

func NewLogger(sw *syncWriter, prefix string) *log.Logger {
	logger := log.New(sw, prefix, log.Ltime|log.Ldate|log.Lmsgprefix)

	return logger
}
