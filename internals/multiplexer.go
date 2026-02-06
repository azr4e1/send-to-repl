package internals

import (
	"errors"
	"io"
	"log"
	"sync"
)

type TransformFunc func(data []byte) []byte

var DefaultTransformFunc = func(data []byte) []byte { return data }

type MultiPlexer struct {
	// All FDs the pipe must read from
	inputs     []io.Reader
	outputs    []*syncWriter
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	Logger     *log.Logger
	Transform  TransformFunc
	ErrHandler ErrHandler
}

func NewMultiPlexer(inputs []io.Reader, output []io.Writer) *MultiPlexer {
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
		Logger:     DiscardLogger,
		Transform:  DefaultTransformFunc,
		ErrHandler: DefaultErrHandler,
	}

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
			mp.Logger.Printf("read from input")
			err := mp.broadcast(buf[:n])
			if err != nil {
				return err
			}
			mp.Logger.Printf("written to output")
		}
	}
}

func (mp *MultiPlexer) Listen() {
	var wg sync.WaitGroup
	for _, i := range mp.inputs {
		input := i
		wg.Go(func() {
			err := mp.pipe(input)
			if err != nil {
				mp.ErrHandler(err)
			}
		})
	}
	mp.Logger.Printf("launched all goroutines")
	mp.Logger.Printf("listening")

	wg.Wait()
	err := mp.pipeWriter.Close()
	if err != nil {
		mp.ErrHandler(err)
	}
	mp.Logger.Printf("all streams closed")
}

func (mp *MultiPlexer) Read(p []byte) (int, error) {
	mp.Logger.Print("reading from buffer")
	return mp.pipeReader.Read(p)
}

func (mp *MultiPlexer) Close() error {
	return mp.pipeWriter.Close()
}
