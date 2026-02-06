package internals

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path"
)

type TempNPipe struct {
	TempDir string
	Name    string
	fd      *os.File
}

func MkTempFifo(name string) (*TempNPipe, error) {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("%s-*", name))
	if err != nil {
		return nil, err
	}
	npipePath := path.Join(tempDir, fmt.Sprintf("%s-namedpipe", name))
	err = unix.Mkfifo(npipePath, 0666)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(npipePath, os.O_RDWR, 0)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	return &TempNPipe{
		TempDir: tempDir,
		Name:    npipePath,
		fd:      fd,
	}, nil
}

func (tnp *TempNPipe) Read(p []byte) (int, error) {
	return tnp.fd.Read(p)
}

func (tnp *TempNPipe) Write(p []byte) (int, error) {
	return tnp.fd.Write(p)
}

func (tnp *TempNPipe) Close() error {
	err := tnp.fd.Close()
	if err != nil {
		return err
	}
	err = os.RemoveAll(tnp.TempDir)
	return err
}
