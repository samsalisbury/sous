package test_with_docker

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
)

type (
	command struct {
		itself         *exec.Cmd
		err            error
		stdout, stderr string
	}
)

func buildCommand(cmdName string, args ...string) command {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var c command
	c.itself = exec.Command(cmdName, args...)
	c.itself.Stdout = &stdout
	c.itself.Stderr = &stderr
	return c
}

func startCommand(cmdName string, args ...string) command {
	c := buildCommand(cmdName, args...)

	c.start()
	return c
}

func runCommand(cmdName string, args ...string) command {
	c := buildCommand(cmdName, args...)

	c.run()
	return c
}

func (c *command) start() error {
	c.err = c.itself.Start()
	return c.err
}

func (c *command) wait() error {
	c.err = c.itself.Wait()
	c.bufferStreams()
	return c.err
}

func (c *command) bufferStreams() {
	c.stdout = c.itself.Stdout.(*bytes.Buffer).String()
	c.stderr = c.itself.Stderr.(*bytes.Buffer).String()
}

func (c *command) run() error {
	c.start()
	if c.err != nil {
		c.bufferStreams()
		return c.err
	}
	c.wait()

	return c.err
}

func (c *command) interrupt() {
	c.itself.Process.Signal(syscall.SIGTERM)
	c.wait()
}

func (c *command) String() string {
	if c.err == nil {
		return fmt.Sprintf("%v ok", (*c.itself).Args)
	} else {
		return fmt.Sprintf("%v %v\nout: %serr: %s", (*c.itself).Args, c.err, c.stdout, c.stderr)
	}
}
