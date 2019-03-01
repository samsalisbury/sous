package testagents

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	ps "github.com/mitchellh/go-ps"
)

// ProcMan manages processes via a pidfile that also contains the name of the
// process for a bit of extra safety when killing things.
// It's far from perfect, but quite useful nonetheless.
//
type ProcMan interface {
	WritePID(t *testing.T, pid int)
	StopPIDs() error
}

// procMan is the default implementation of ProcMan.
type procMan struct {
	ProcManOpts
	sync.Mutex
}

// DefaultPIDFile is the default PIDFile if not specified in ProcManOpts.
const DefaultPIDFile = ".procman-pids"

// ProcManOpts optionally configures a ProcMan.
type ProcManOpts struct {
	// PIDFile is the file this ProcMan stores state in.
	// Defaults to DefaultPIDFile if left empty.
	PIDFile string
	// LogFunc is called to emit realtime logs.
	// Defaults to log.Printf if left nil.
	LogFunc func(string, ...interface{})
	// Verbose can be set to true to emit more logs to LogFunc.
	Verbose bool
}

// NewProcMan returns a new ProcMan.
// Your tests should create a ProcMan using this func and share it with each Bin
// or Service you create. Call StopPIDs to stop any such processes that have not
// already stopped.
func NewProcMan(opts ProcManOpts) ProcMan {
	if opts.PIDFile == "" {
		opts.PIDFile = DefaultPIDFile
	}
	if opts.LogFunc == nil {
		opts.LogFunc = log.Printf
	}
	return &procMan{ProcManOpts: opts}
}

// DefaultProcMan returns a ProcMan configured with the defaults.
func DefaultProcMan() ProcMan {
	return NewProcMan(ProcManOpts{})
}

// writePID adds a PID to the pid file.
func (pm *procMan) WritePID(t *testing.T, pid int) {
	pm.Lock()
	defer pm.Unlock()
	psProc, err := ps.FindProcess(pid)
	if err != nil {
		t.Fatalf("cannot inspect proc %d: %s", pid, err)
	}
	if psProc == nil {
		time.Sleep(time.Second)
		psProc, err := ps.FindProcess(pid)
		if err != nil {
			t.Fatalf("cannot inspect proc %d: %s", pid, err)
		}
		if psProc == nil {
			t.Logf("Warning! Possibly orphaned PID: %d", pid)
			return
		}
	}

	pidFile := pm.PIDFile

	var f *os.File
	if s, err := os.Stat(pidFile); err != nil {
		if !isNotExist(err) {
			t.Fatalf("could not stat %q: %s", pidFile, err)
			return
		}
		if s != nil && s.IsDir() {
			t.Fatalf("cannot write to file %q: it's a directory", pidFile)
		}
		f, err = os.Create(pidFile)
		defer closeFiles(f)
		if err != nil {
			t.Fatalf("could not create %q: %s", pidFile, err)
		}
	}
	if f == nil {
		var err error
		f, err = os.OpenFile(pidFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			t.Fatalf("could not open %q: %s", pidFile, err)
			return
		}
		defer closeFiles(f)
	}
	if psProc == nil {
		t.Logf("Warning! Unable to write PID %d to pidfile: psProc became nil all of a sudden", pid)
		return

	}
	if _, err := fmt.Fprintf(f, "%d\t%s\n", pid, psProc.Executable()); err != nil {
		t.Fatalf("could not write PID %d (exe %s) to file %q: %s",
			pid, psProc.Executable(), pidFile, err)
	}
}

// stopPIDs stops all pids in the pidfile.
func (pm *procMan) StopPIDs() error {
	pm.Lock()
	defer pm.Unlock()
	pidFile := pm.PIDFile
	d, err := ioutil.ReadFile(pidFile)
	if err != nil {
		if isNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to read %q: %s", pidFile, err)
	}
	pids := strings.Split(string(d), "\n")
	var failedPIDs []string
	for _, p := range pids {
		if len(p) == 0 {
			continue
		}
		parts := strings.Split(p, "\t")
		if len(parts) != 2 {
			return fmt.Errorf("%q corrupted: contains %s", pidFile, p)
		}
		tmp, executable := parts[0], parts[1]
		p = tmp
		pid, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("%q corrupted: contains %q (not an int)", pidFile, p)
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			if err != os.ErrNotExist {
				return fmt.Errorf("cannot find proc %d: %s", pid, err)
			}
		}
		psProc, err := ps.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("cannot inspect proc %d: %s", pid, err)
		}
		if psProc == nil {
			if pm.Verbose {
				pm.LogFunc("skipping cleanup of %d (already stopped)", pid)
			}
			continue
		}

		if psProc.Executable() != executable {
			pm.LogFunc("not killing process %s; it is %q not %q", p, psProc.Executable(), executable)
			continue
		}
		if err := proc.Kill(); err != nil {
			failedPIDs = append(failedPIDs, p)
			pm.LogFunc("failed to stop process %d: %s", pid, err)
		}
	}
	if len(failedPIDs) != 0 {
		err := ioutil.WriteFile(pidFile, []byte(strings.Join(failedPIDs, "\n")), 0777)
		if err != nil {
			return fmt.Errorf("failed to track failed PIDs %s: %s", strings.Join(failedPIDs, ", "), err)
		}
		return nil
	}
	return os.Remove(pidFile)
}
