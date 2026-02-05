package internals

import (
	"errors"
	"io"
	"log"
	"sync"
)

type MultiPlexer struct {
	// All FDs the pipe must read from
	inputs     []io.Reader
	outputs    []*syncWriter
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	logger     *log.Logger
	Transform  func(data []byte) []byte
}

func NewMultiPlexer(inputs []io.Reader, output []io.Writer, logger *log.Logger) *MultiPlexer {
	syncOutputs := []*syncWriter{}
	for _, w := range output {
		syncOutputs = append(syncOutputs, NewSyncWriter(w))
	}

	pipeReader, pipeWriter := io.Pipe()
	multiPlexer := &MultiPlexer{
		inputs:     inputs,
		outputs:    syncOutputs,
		pipeReader: pipeReader,
		pipeWriter: pipeWriter,
		logger:     logger,
		Transform:  func(data []byte) []byte { return data },
	}

	go multiPlexer.listen()

	return multiPlexer
}

func (mp *MultiPlexer) broadcast(p []byte) error {
	errSlice := []error{}
	var err error
	_, err = mp.pipeWriter.Write(p)
	errSlice = append(errSlice, err)
	for _, w := range mp.outputs {
		_, err = w.Write(mp.Transform(p))
		errSlice = append(errSlice, err)
	}

	return errors.Join(errSlice...)
}

func (mp *MultiPlexer) pipe(fd io.Reader) error {
	buf := make([]byte, BufSize)
	for {
		n, err := fd.Read(buf)
		if err != nil {
			return err
		}

		if n > 0 {
			mp.logger.Printf("read from input")
			err := mp.broadcast(buf[:n])
			if err != nil {
				return err
			}
			mp.logger.Printf("written to output")
		}
	}
}

func (mp *MultiPlexer) listen() {
	var wg sync.WaitGroup
	for _, i := range mp.inputs {
		input := i
		wg.Add(1)
		go func() {
			err := mp.pipe(input)
			if err != nil {
				mp.logger.Println(err)
			}
			wg.Done()
		}()
	}
	mp.logger.Printf("launched all goroutines")
	mp.logger.Printf("listening")

	wg.Wait()
	err := mp.pipeWriter.Close()
	if err != nil {
		mp.logger.Print(err)
	}
	mp.logger.Printf("all streams closed")
}

func (mp *MultiPlexer) Read(p []byte) (int, error) {
	mp.logger.Print("reading from buffer")
	return mp.pipeReader.Read(p)
}
