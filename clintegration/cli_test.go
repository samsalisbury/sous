package clintegration

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"testing"
)

const stateCapture = `
pwd
env | grep X=
pwd >3
env >4
`

type CaptiveShell struct {
}

func (sh *CaptiveShell) Run(script string) error {
}

func ShellScript(script string) (shCmd *exec.Cmd, pwd, env string) {
	pwdRead, pwdWrite, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	envRead, envWrite, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	shCmd = exec.Command("/bin/sh")

	shCmd.Stdin = bytes.NewBufferString(script + stateCapture)
	shCmd.Stdout = &bytes.Buffer{}
	shCmd.Stderr = &bytes.Buffer{}
	shCmd.ExtraFiles = []*os.File{pwdWrite, envWrite}
	shCmd.Run()
	log.Printf("%#v", shCmd)
	log.Printf("%#v", shCmd.ProcessState)
	envWrite.Close()
	pwdWrite.Close()

	pwdB, err := ioutil.ReadAll(pwdRead)
	if err != nil {
		panic(err)
	}
	envB, err := ioutil.ReadAll(envRead)
	if err != nil {
		panic(err)
	}

	return shCmd, string(pwdB), string(envB)
}

func SousCLIIntegrationTest(t *testing.T) {

}
