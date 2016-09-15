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

func TestCommand(t *testing.T) {
	testValue := "test-stdout"
	sh := &Sh{}
	echoCmd := sh.Cmd("echo", testValue)
	output, err := echoCmd.Stdout()
	if err != nil {
		t.Error(err)
	}
	if output != testValue {
		t.Errorf("%s != %s\n", output, testValue)
	}

	falsePath := "/bin/false"
	falseCmd := sh.Cmd(falsePath)
	_, err = falseCmd.Stdout()
	if err == nil {
		t.Errorf("Expected error executing %s not seen.", falsePath)
	}

	testValue = "test-stderr"
	errPath := "../../bin/test-stderr"
	errCmd := sh.Cmd(errPath)
	output, err = errCmd.Stderr()
	if err != nil {
		t.Error(err)
	}
	if output != testValue {
		t.Errorf("%s != %s\n", output, testValue)
	}

	expected := []string{"first", "second", "third"}
	linePath := "../../bin/test-lines"
	lineCmd := sh.Cmd(linePath)
	lines, err := lineCmd.Lines()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expected, lines) {
		t.Errorf("Failed to correctly parse output of %s.\n", linePath)
	}

	failStatus := -1
	nonexistPath := "/this/file/does/not/exist"
	nonexistCmd := sh.Cmd(nonexistPath)
	status, err := nonexistCmd.ExitCode()
	if status != failStatus {
		t.Errorf("Attempted execution of %s returned %d.\n", nonexistPath, failStatus)
	}

	failCmd := sh.Cmd("../..//bin/false") // Macs don't have /bin/true
	err = failCmd.Fail()
	if err != nil {
		t.Errorf("Fail() should have returned nil from a command that exited with an error status. Returned %#v", err)
	}

	successCmd := sh.Cmd("../../bin/true") // Or, apparently /bin/false
	err = successCmd.Fail()
	if err == nil {
		t.Error("Fail() should return an error from a command that exited with a successful status.")
	}

	tableCmd := sh.Cmd("../../bin/test-table")
	table, err := tableCmd.Table()
	if !(reflect.DeepEqual([]string{"one", "two", "three"}, table[0]) &&
		reflect.DeepEqual([]string{"four", "five", "six"}, table[1])) {
		t.Errorf("Failed to parse tabular output of %s.\n", tableCmd)
	}
}
