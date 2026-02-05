package internals

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
	logger       *log.Logger
}

func NewRepl(command string, logger *log.Logger) (*Repl, error) {
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
	errChan := make(chan error)

	repl := &Repl{
		Cmd:          cmd,
		ReplStdin:    replStdin,
		ReplStdout:   replStdout,
		ReplStderr:   replStderr,
		DoneChan:     done,
		NextLineChan: nextLine,
		ErrChan:      errChan,
		BufSize:      BufSize,
		logger:       logger,
	}

	return repl, nil
}

// GetOutput reads from reader a bufsize amount until there is nothing to read
func (repl *Repl) getOutput(name string, reader io.Reader, writer io.Writer) error {
	buf := make([]byte, repl.BufSize)

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
		repl.logger.Printf("%s just processed\n", name)
	}
}

// Read REPL Output and send it to client
func (repl *Repl) SendReplStdOut(clientInput io.Writer) error {
	err := repl.getOutput("stdout", repl.ReplStdout, clientInput)

	return err
}

// Read REPL Error and send it to client
func (repl *Repl) SendReplStdErr(clientInput io.Writer) error {
	err := repl.getOutput("stderr", repl.ReplStderr, clientInput)

	return err
}

// SendToRepl reads from client stdout, and sends lines
// to repl stdin through channel
func (repl *Repl) SendToRepl(clientOutput io.Reader) error {
	scanner := bufio.NewScanner(clientOutput)

	for scanner.Scan() {
		input := scanner.Text()
		repl.NextLineChan <- input
		repl.logger.Printf("%s just scanned a line\n", "stdin")
	}
	return scanner.Err()
}

// ProcessExit waits for REPL to terminate and
// handles it gracefully
func (repl *Repl) ProcessExit() error {
	if err := repl.Cmd.Wait(); err != nil {
		log.Print(err)
	}
	repl.logger.Printf("process just terminated")
	repl.DoneChan <- true
	return nil
}

func (repl *Repl) Run(clientOutput io.Reader, clientInput io.Writer, clientErr io.Writer) error {
	// Start REPL
	if err := repl.Cmd.Start(); err != nil {
		return err
	}
	repl.logger.Printf("process started")

	// manage subprocess termination gracefully
	go func() {
		err := repl.ProcessExit()
		if err != nil {
			repl.ErrChan <- err
		}
	}()
	repl.logger.Printf("launched termination handling")

	// redirect stdout
	go func() {
		err := repl.SendReplStdOut(clientInput)
		if err != nil {
			if err == io.EOF {
				return
			}
			repl.ErrChan <- err
		}
	}()
	repl.logger.Printf("launched stdout redirection")

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
	repl.logger.Printf("launched stderr redirection")

	// redirect stdin
	go func() {
		err := repl.SendToRepl(clientOutput)
		if err != nil {
			repl.ErrChan <- err
		}
		// reached EOF, break
		repl.DoneChan <- true
	}()
	repl.logger.Printf("launched stdin redirection")

	for {
		select {
		case <-repl.DoneChan:
			return nil
		case line := <-repl.NextLineChan:
			io.WriteString(repl.ReplStdin, line+"\n")
		case err := <-repl.ErrChan:
			return err
		}
		repl.logger.Println("select loop")
	}
}
