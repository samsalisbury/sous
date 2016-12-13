package clintegration

import (
	"log"
	"testing"
)

func TestShAssumptions(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	cmd, pwd, env := ShellScript(`
	cd /tmp
	X=7
	export CYGNUS=blackhole
	echo $X
	`)

	log.Print(cmd.Stdout)
	log.Print(cmd.Stderr)
	log.Print(pwd)
	log.Print(env)
	log.Printf("%#v", cmd)
	log.Printf("%#v", cmd.Env)
	log.Printf("%#v", cmd.Dir)
	log.Printf("%#v", cmd.Process)
	log.Printf("%#v", cmd.ProcessState)
}
