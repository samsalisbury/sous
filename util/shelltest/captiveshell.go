package shelltest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	headerMarker = `# END SCRIPT HEADER`
	footerMarker = `# BEGIN SCRIPT FOOTER`

	stateCapture = `
trap '(lasterr=$?; exec >&5; echo -n $lasterr; history 1)' ERR
` + headerMarker
	exitCapture = footerMarker + `
status=$?
env | sort >&4
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
	// XXX: there's room here for a "stateful stdin" - using DEBUG traps and PS1
	// hacks to keep track of the shell's state, and streaming the stdin in more
	// like a human would - e.g. normal characters very quickly, but with delays
	// after "Enter" (and complete pauses if the "enter" triggers a DEBUG trap).
	// The purpose would be to avoid weird test-level hacks like </dev/null for
	// SSH.
	//
	// One practical upshot is that it wouldn't be 100% either, since it'd have
	// to rely on both timing (i.e. that the shell is likely to trigger the DEBUG
	// trap quickly, but it might not) and fragile environment variables
	// (effectively, we'd need to echo $PS1 every DEBUG and bail out if it'd been
	// changed.)
	//
	// (Or... echo $PS1 and then watch for some version of that on stderr instead
	// of waiting for a PS1 eval hack to write to a stream. Of course, if PS1
	// isn't a simple string (because there's an eval hack...) we can't just
	// match it... so we'd wind up spinning off a shell to evaluate PS1 to find
	// out what it prints... granted that it doesn't screw anything up...
	//
	// ...which is why, for the moment, I'm sticking with </dev/null on SSH.

	liveStream struct {
		debugDir string
		pipe     io.Reader
		bufs     []*bytes.Buffer
		debugs   []io.Writer
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
	sh.Cmd = exec.Command("bash", "--norc", "-i")

	for k, v := range env {
		sh.Cmd.Env = append(sh.Cmd.Env, k+"="+v)
	}

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

// Run runs a script in the context of the shell, waits for it to complete and
// returns the Result of running it.
func (sh *CaptiveShell) Run(script string) (Result, error) {
	st := stateCapture + "\n" + script + "\n" + exitCapture
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

	hre := regexp.MustCompile(`(?ms).*` + headerMarker)
	fre := regexp.MustCompile(`(?ms)` + footerMarker + `.*`)

	stdout = hre.ReplaceAllString(fre.ReplaceAllString(stdout, ""), "")
	blended = hre.ReplaceAllString(fre.ReplaceAllString(blended, ""), "")
	stderr = hre.ReplaceAllString(fre.ReplaceAllString(stderr, ""), "")

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

func (ls *liveStream) debugTo(name, prefix string) {
	dir, _ := ioutil.TempDir("", prefix)
	ls.debugDir = dir
	os.MkdirAll(ls.debugDir, os.ModePerm)
	ls.debugs = make([]io.Writer, len(ls.bufs))
	log.Printf("Writing low-level debug output for %s's buffers into %q.", name, ls.debugDir)
	for n := range ls.bufs {
		path := filepath.Join(ls.debugDir, fmt.Sprint(n))
		ls.debugs[n], _ = os.Create(path)
	}
}

func (ls *liveStream) saveBytes(buf []byte) {
	ls.Lock()
	defer ls.Unlock()
	for n, b := range ls.bufs {
		b.Write(buf)

		if ls.debugDir != "" {
			_, err := ls.debugs[n].Write(buf)
			if err != nil {
				log.Printf("Error while writing bytes: %v", err)
				log.Printf("Was trying to write: %s", string(buf))
			}
		}
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
		return -1, fmt.Errorf("Exit stream closed prematurely!\n%#v\n%#v\n%s\n****\n%s************", sh, sh.Cmd, sh.stdout.consume(0), sh.stderr.consume(0))
	}

	return strconv.Atoi(string(bytes.TrimFunc(sh.doneRead.Bytes(), func(r rune) bool {
		return strings.Index(`0123456789`, string(r)) == -1
	})))

}
