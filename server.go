package replserver

import (
	"bufio"
	"io"
	"log"
	"os/exec"

	"github.com/google/shlex"
)

const BufSize = 4096

type Repl struct {
	Cmd          *exec.Cmd
	ReplStdin    io.WriteCloser
	ReplStdout   io.ReadCloser
	ReplStderr   io.ReadCloser
	DoneChan     chan bool
	NextLineChan chan string
	ErrChan      chan error
	BufSize      int
}

func NewRepl(command string) (*Repl, error) {
	tokens, err := shlex.Split(command)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(tokens[0], tokens[1:]...)

	// pipes
	replStdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	replStdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	replStderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// comms channels
	done := make(chan bool)
	nextLine := make(chan string)
	errChan := make(chan error, 4)

	repl := &Repl{
		Cmd:          cmd,
		ReplStdin:    replStdin,
		ReplStdout:   replStdout,
		ReplStderr:   replStderr,
		DoneChan:     done,
		NextLineChan: nextLine,
		ErrChan:      errChan,
		BufSize:      BufSize,
	}

	return repl, nil
}

// GetOutput reads from reader a bufsize amount until there is nothing to read
func getOutput(reader io.ReadCloser, writer io.WriteCloser, bufSize int) error {
	buf := make([]byte, bufSize)

	for {
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}
		if n > 0 {
			_, err = writer.Write(buf[:n])
			if err != nil {
				return err
			}
		}
	}
}

func (repl *Repl) SendReplStdOut(clientOutput io.WriteCloser) error {
	err := getOutput(repl.ReplStdout, clientOutput, repl.BufSize)

	return err
}

func (repl *Repl) SendReplStdErr(clientOutput io.WriteCloser) error {
	err := getOutput(repl.ReplStderr, clientOutput, repl.BufSize)

	return err
}

// WriteInput reads from main goroutine stdin, and sends lines
// to subprocess goroutine through channel
func (repl *Repl) SendToRepl(clientInput io.ReadCloser) error {
	scanner := bufio.NewScanner(clientInput)

	for scanner.Scan() {
		input := scanner.Text()
		repl.NextLineChan <- input
	}
	return scanner.Err()
}

// ProcessExit waits for subprocess to terminate and
// handles it gracefully
func (repl *Repl) ProcessExit() error {
	if err := repl.Cmd.Wait(); err != nil {
		log.Print(err)
	}
	repl.DoneChan <- true
	return nil
}

func (repl *Repl) Start(clientInput io.ReadCloser, clientOutput io.WriteCloser, clientErr io.WriteCloser) error {
	if err := repl.Cmd.Start(); err != nil {
		return err
	}

	// manage subprocess termination gracefully
	go func() {
		err := repl.ProcessExit()
		if err != nil {
			repl.ErrChan <- err
		}
	}()

	// redirect stdout
	go func() {
		err := repl.SendReplStdOut(clientOutput)
		if err != nil {
			if err == io.EOF {
				return
			}
			repl.ErrChan <- err
		}
	}()
	// redirect stderr
	go func() {
		err := repl.SendReplStdErr(clientErr)
		if err != nil {
			if err == io.EOF {
				return
			}
			repl.ErrChan <- err
		}
	}()
	// redirect stdin
	go func() {
		err := repl.SendToRepl(clientInput)
		if err != nil {
			repl.ErrChan <- err
		}
		// reached EOF, break
		repl.DoneChan <- true
	}()

	for {
		select {
		case <-repl.DoneChan:
			return nil
		case line := <-repl.NextLineChan:
			io.WriteString(repl.ReplStdin, line+"\n")
		case err := <-repl.ErrChan:
			return err
		}
	}
}
