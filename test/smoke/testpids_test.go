//+build smoke

package smoke

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

var pidMutex sync.Mutex

func writePID(t *testing.T, pid int) {
	pidMutex.Lock()
	defer pidMutex.Unlock()
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
		defer closeFile(t, f)
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
		defer closeFile(t, f)
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

func stopPIDs() error {
	pidMutex.Lock()
	defer pidMutex.Unlock()
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
			log.Printf("skipping cleanup of %d (already stopped)", pid)
			continue
		}

		if psProc.Executable() != executable {
			log.Printf("not killing process %s; it is %q not %q", p, psProc.Executable(), executable)
			continue
		}
		if err := proc.Kill(); err != nil {
			failedPIDs = append(failedPIDs, p)
			log.Printf("failed to stop process %d: %s", pid, err)
		}
	}
	if len(failedPIDs) != 0 {
		err := ioutil.WriteFile(pidFile, []byte(strings.Join(failedPIDs, "\n")), 0777)
		if err != nil {
			return fmt.Errorf("Failed to track failed PIDs %s: %s", strings.Join(failedPIDs, ", "), err)
		}
		return nil
	}
	return os.Remove(pidFile)
}
