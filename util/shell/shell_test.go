package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	sh, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if sh.Cwd == wd {
		t.Logf("Shell returned by Default() is using the current directory, %s\n", wd)
	} else {
		t.Fatalf("Shell returned by Default() is using directory [%s], expected [%s]\n",
			wd, sh.Cwd)
	}
}

func TestDefaultInDir(t *testing.T) {
	testDir := "/tmp"
	sh, err := DefaultInDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	epochString := fmt.Sprintf("%d", time.Now().Unix())
	fileName := "soustest-" + epochString
	filePath := filepath.Join(testDir, fileName)
	// it is important to use fileName here instead of filePath as the argument,
	// as the test determines if testDir is the current working directory.
	cmd := sh.Cmd("touch", fileName)
	defer os.Remove(filePath)
	err = cmd.Succeed()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filePath); err == nil {
		t.Logf("Tested creation of file %s in %s. Shell is using correct directory.\n",
			fileName, testDir)
	} else {
		t.Fatalf("Tried to create file %s in %s. File not found.\n", fileName, testDir)
	}
}

func TestCommandStdout(t *testing.T) {
	testValue := "test-stdout"
	sh := &Sh{}
	echoCmd := sh.Cmd("echo", testValue)
	output, err := echoCmd.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	if output == testValue {
		t.Logf("%s ==  %s\n", output, testValue)
	} else {
		t.Fatalf("%s != %s\n", output, testValue)
	}
}

func TestCommandStdoutExpectsError(t *testing.T) {
	sh := &Sh{}
	falsePath := "false"
	falseCmd := sh.Cmd(falsePath)
	_, err := falseCmd.Stdout()
	if err == nil {
		t.Fatalf("Expected error executing %s not seen.", falsePath)
	} else {
		t.Logf("%s correctly created an error condition.\n", falsePath)
	}
}

func TestCommandStderr(t *testing.T) {
	sh := &Sh{}
	testValue := "test-stderr"
	errPath := "testdata/test-stderr"
	errCmd := sh.Cmd(errPath)
	output, err := errCmd.Stderr()
	if err != nil {
		t.Fatal(err)
	}
	if output == testValue {
		t.Logf("%s == %s\n", output, testValue)
	} else {
		t.Fatalf("%s != %s\n", output, testValue)
	}
}

func TestCommandLines(t *testing.T) {
	sh := &Sh{}
	expected := []string{"first", "second", "third"}
	linePath := "testdata/test-lines"
	lineCmd := sh.Cmd(linePath)
	lines, err := lineCmd.Lines()
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(expected, lines) {
		t.Logf("Successfully parsed output of %s.\n", linePath)
	} else {
		t.Fatalf("Failed to correctly parse output of %s.\n", linePath)
	}
}

func TestCommandExitCode(t *testing.T) {
	sh := &Sh{}
	failStatus := -1
	nonexistPath := "/this/file/does/not/exist"
	nonexistCmd := sh.Cmd(nonexistPath)
	status, err := nonexistCmd.ExitCode()
	if err == nil {
		t.Fatalf("Attempt to execute %s should have returned an error.\n", nonexistPath)
	}
	if status == failStatus {
		t.Logf("Attempted execution of %s returned %d, as expected.\n",
			nonexistPath, failStatus)
	}
}

func TestCommandExpectsError(t *testing.T) {
	sh := &Sh{}
	failCmd := sh.Cmd("false")
	err := failCmd.Fail()
	if err == nil {
		t.Log("Fail() correctly returns nil on a command that exited with an error status.")
	} else {
		t.Fatal("Fail() should have returned nil from a command that exited with an error status.")
	}
}

func TestCommandFail(t *testing.T) {
	sh := &Sh{}
	successCmd := sh.Cmd("true")
	err := successCmd.Fail()
	if err != nil {
		t.Log("Fail() correctly returns an error on a command that exited with a successful status.")
	} else {
		t.Fatal("Fail() should return an error from a command that exited with a successful status.")
	}
}

func TestCommandTable(t *testing.T) {
	sh := &Sh{}
	tableCmd := sh.Cmd("testdata/test-table")
	table, err := tableCmd.Table()
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual([]string{"one", "two", "three"}, table[0]) &&
		reflect.DeepEqual([]string{"four", "five", "six"}, table[1]) {
		t.Logf("Successfully parsed tabular output of %s.\n", tableCmd)
	} else {
		t.Fatalf("Failed to parse tabular output of %s.\n", tableCmd)
	}
}
