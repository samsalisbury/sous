package shelltest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const (
	stateCapture = `
trap '(lasterr=$?; exec >&5; echo -n $lasterr; history 1)' ERR
`
	exitCapture = `
status=$?
env >&4
echo $status >&3
`
)

type (
	CaptiveShell struct {
		*exec.Cmd
		Stdin                                 io.WriteCloser
		stdout, stderr, scriptEnv, scriptErrs *liveStream
		doneRead                              *bufio.Scanner
		events                                chan int
	}

	liveStream struct {
		pipe io.Reader
		bufs []*bytes.Buffer
		sync.Mutex
	}
)

func newLiveStream(from io.Reader, events <-chan int) *liveStream {
	ls := &liveStream{
		pipe: from,
		bufs: []*bytes.Buffer{&bytes.Buffer{}},
	}

	go ls.reader(events)
	return ls
}

// NewShell creates a new CaptiveShell with an environment dictacted by env.
func NewShell(env map[string]string) (sh *CaptiveShell, err error) {
	sh = &CaptiveShell{}
	sh.Cmd = exec.Command("/bin/bash", "--norc", "-i")

	for k, v := range env {
		sh.Cmd.Env = append(sh.Cmd.Env, k+"="+v)
	}
	log.Printf("%v", sh.Cmd.Env)

	sh.events = make(chan int)
	sh.Stdin, err = sh.Cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	blended := &bytes.Buffer{}

	stdo, err := sh.Cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	sh.stdout = newLiveStream(stdo, sh.events)
	sh.stdout.addBuf(blended)

	stde, err := sh.Cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	sh.stderr = newLiveStream(stde, sh.events)
	sh.stderr.addBuf(blended)

	dr, doneWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	sh.doneRead = bufio.NewScanner(dr)

	envR, envW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	sh.scriptEnv = newLiveStream(envR, sh.events)

	errR, errW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	sh.scriptErrs = newLiveStream(errR, sh.events)

	sh.Cmd.ExtraFiles = []*os.File{doneWrite, envW, errW}

	sh.Cmd.Start()

	doneWrite.Close()
	envW.Close()
	errW.Close()

	return
}

func (sh *CaptiveShell) Run(script string) (Result, error) {
	st := stateCapture + script + exitCapture
	sh.Stdin.Write([]byte(st))
	exit, err := sh.readExitStatus()
	if err != nil {
		return Result{}, err
	}

	stdout := sh.stdout.consume(0)
	stderr := sh.stderr.consume(0)
	blended := sh.stdout.consume(1)

	errExits := sh.scriptErrs.consume(0)
	env := sh.scriptEnv.consume(0)

	return Result{
		Script:  script,
		Exit:    exit,
		Stdout:  stdout,
		Stderr:  stderr,
		Blended: blended,
		Errs:    errExits,
		Env:     env,
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
			ls.saveBytes(buf[0:count])
		case <-events:
			return
		}
	}
}

func (ls *liveStream) saveBytes(buf []byte) {
	ls.Lock()
	defer ls.Unlock()
	for _, b := range ls.bufs {
		b.Write(buf)
	}
}

func (ls *liveStream) addBuf(buf *bytes.Buffer) {
	ls.bufs = append(ls.bufs, buf)
}

func (ls *liveStream) consume(n int) string {
	ls.Lock()
	defer ls.Unlock()
	str := ls.bufs[n].String()
	ls.bufs[n].Reset()
	return str
}

func (sh *CaptiveShell) readExitStatus() (int, error) {
	if !sh.doneRead.Scan() {
		return -1, fmt.Errorf("Exit stream closed prematurely!\n%#v\n%s\n****\n%s************", sh, sh.stdout.consume(0), sh.stderr.consume(0))
	}

	return strconv.Atoi(string(bytes.TrimFunc(sh.doneRead.Bytes(), func(r rune) bool {
		return strings.Index(`0123456789`, string(r)) == -1
	})))
}
