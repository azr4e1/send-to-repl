package internals

import (
	"io"
	"log"
)

type MultiPlexer struct {
	// All FDs the pipe must read from
	Inputs []io.Reader
	// Channel that collects data from Readers to buffer
	bufChan chan []byte
	// Channel to respond to read request: sends n bytes of buffer
	writeChan chan []byte
	// Buffered data
	buffer        []byte
	dataRequested int
	logger        *log.Logger
}

func NewMultiPlexer(inputs []io.Reader, logger *log.Logger) *MultiPlexer {
	funnelReader := &MultiPlexer{
		Inputs:    inputs,
		bufChan:   make(chan []byte),
		writeChan: make(chan []byte),
		buffer:    []byte{},
	}

	go funnelReader.listen()
	logger.Printf("listening")

	return funnelReader
}

func (mp *MultiPlexer) readInput(fd io.Reader) error {
	buf := make([]byte, BufSize)
	for {
		n, err := fd.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if n > 0 {
			mp.bufChan <- buf[:n]
		}
		mp.logger.Printf("read from input")
	}
}

func (mp *MultiPlexer) listen() {
	for _, i := range mp.Inputs {
		input := i
		go func() {
			err := mp.readInput(input)
			if err != nil {
				mp.logger.Println(err)
			}
		}()
	}
	mp.logger.Printf("launched all goroutines")

	for {
		data := <-mp.bufChan
		mp.buffer = append(mp.buffer, data...)
		if n := mp.dataRequested; n > 0 {
			readN := min(n, len(mp.buffer))
			data := mp.buffer[:readN]
			mp.buffer = mp.buffer[readN:]
			mp.writeChan <- data
			mp.dataRequested = 0
		}
		mp.logger.Printf("multiplexer loop")
	}
}

func (mp *MultiPlexer) Read(buf []byte) (int, error) {
	bufLen := len(buf)
	mp.dataRequested = bufLen
	data := <-mp.writeChan
	n := len(data)

	for i := 0; i < n; i++ {
		buf[i] = data[i]
	}

	mp.logger.Printf("read action")
	return n, nil
}
