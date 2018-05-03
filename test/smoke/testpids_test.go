//+build smoke

package smoke

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

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
	if _, err := fmt.Fprintf(f, "%d\t%s\n", pid, psProc.Executable()); err != nil {
		t.Fatalf("could not write PID %d (exe %s) to file %q: %s",
			pid, psProc.Executable(), pidFile, err)
	}
}

func stopPIDs(t *testing.T) {
	pidMutex.Lock()
	defer pidMutex.Unlock()
	t.Helper()
	d, err := ioutil.ReadFile(pidFile)
	if err != nil {
		if isNotExist(err) {
			return
		}
		t.Fatalf("unable to read %q: %s", pidFile, err)
		return
	}
	pids := strings.Split(string(d), "\n")
	var failedPIDs []string
	for _, p := range pids {
		if len(p) == 0 {
			continue
		}
		parts := strings.Split(p, "\t")
		if len(parts) != 2 {
			t.Fatalf("%q corrupted: contains %s", pidFile, p)
		}
		tmp, executable := parts[0], parts[1]
		p = tmp
		pid, err := strconv.Atoi(p)
		if err != nil {
			t.Fatalf("%q corrupted: contains %q (not an int)", pidFile, p)
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			if err != os.ErrNotExist {
				t.Fatalf("cannot find proc %d: %s", pid, err)
			}
		}
		psProc, err := ps.FindProcess(pid)
		if err != nil {
			t.Fatalf("cannot inspect proc %d: %s", pid, err)
		}
		if psProc == nil {
			t.Logf("skipping cleanup of %d (already stopped)", pid)
			continue
		}

		if psProc.Executable() != executable {
			t.Logf("not killing process %s; it is %q not %q", p, psProc.Executable(), executable)
			continue
		}
		if err := proc.Kill(); err != nil {
			failedPIDs = append(failedPIDs, p)
			t.Errorf("failed to stop process %d: %s", pid, err)
		}
	}
	if len(failedPIDs) != 0 {
		err := ioutil.WriteFile(pidFile, []byte(strings.Join(failedPIDs, "\n")), 0777)
		if err != nil {
			t.Fatalf("Failed to track failed PIDs %s: %s", strings.Join(failedPIDs, ", "), err)
		}
	} else {
		os.Remove(pidFile)
	}
}
