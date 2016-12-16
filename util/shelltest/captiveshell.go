package shelltest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const (
	stateCapture = `
pwd
env | grep X=
pwd >3
env >4
`
	exitCapture = `
status=$?
echo $status >&3
`
)

type (
	CaptiveShell struct {
		*exec.Cmd
		Stdin          io.WriteCloser
		stdout, stderr *liveStream
		doneRead       *bufio.Scanner
		events         chan int
	}

	liveStream struct {
		pipe io.Reader
		buf  []byte
		sync.Mutex
	}
)

func newLiveStream(from io.Reader, events <-chan int) *liveStream {
	ls := &liveStream{
		pipe: from,
		buf:  []byte{},
	}

	go ls.reader(events)
	return ls
}

func NewShell() (sh *CaptiveShell, err error) {
	sh = &CaptiveShell{}
	sh.Cmd = exec.Command("/bin/sh")

	sh.events = make(chan int)
	sh.Stdin, err = sh.Cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdo, err := sh.Cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	sh.stdout = newLiveStream(stdo, sh.events)

	stde, err := sh.Cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	sh.stderr = newLiveStream(stde, sh.events)

	dr, doneWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	sh.doneRead = bufio.NewScanner(dr)
	sh.Cmd.ExtraFiles = []*os.File{doneWrite}

	sh.Cmd.Start()
	doneWrite.Close()

	return
}

func (sh *CaptiveShell) Run(script string) (Result, error) {
	st := script + exitCapture
	sh.Stdin.Write([]byte(st))
	exit, err := sh.readExitStatus()
	if err != nil {
		return Result{}, err
	}
	stdout := sh.stdout.consume()

	stderr := sh.stderr.consume()

	return Result{
		Script: script,
		Exit:   exit,
		Stdout: stdout,
		Stderr: stderr,
	}, nil
}

func (ls *liveStream) reader(events <-chan int) {
	buf := make([]byte, 1024)
	for {
		select {
		default:
			count, err := ls.pipe.Read(buf)
			if err != nil {
				return
			}
			ls.Lock()
			ls.buf = append(ls.buf, buf[0:count]...)
			ls.Unlock()
		case <-events:
			return
		}
	}
}

func (ls *liveStream) consume() string {
	ls.Lock()
	str := string(ls.buf)
	ls.buf = ls.buf[0:0]
	ls.Unlock()
	return str
}

func consumeStream(stream io.Reader) ([]byte, error) {
	got := []byte{}
	buf := make([]byte, 1024)
	for {
		count, err := stream.Read(buf)
		if err != nil {
			return []byte{}, err
		}
		if count == 0 {
			return got, nil
		}
		got = append(got, buf[0:count]...)
	}
}

func (sh *CaptiveShell) readExitStatus() (int, error) {
	if !sh.doneRead.Scan() {
		return -1, fmt.Errorf("Exit stream closed prematurely!")
	}

	return strconv.Atoi(string(bytes.TrimFunc(sh.doneRead.Bytes(), func(r rune) bool {
		return strings.Index(`0123456789`, string(r)) == -1
	})))
}
