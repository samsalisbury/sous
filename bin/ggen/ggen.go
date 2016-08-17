package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var blueprints = "github.com/opentable/sous/util/blueprints"

type spec struct {
	Pkg, Typ, File string
}

func main() {
	a := os.Args[1:]
	if len(a) < 2 {
		usage()
	}
	in, err := parseSpec(a[0])
	if err != nil {
		usage(err)
	}
	out, err := parseSpec(a[1])
	if err != nil {
		usage(err)
	}

	a = a[2:]
	m := make(map[string]string, len(a))
	for _, p := range a {
		pair := strings.Split(p, ":")
		if len(pair) != 2 {
			usage(fmt.Sprintf("%q not a valid mapping, want OrigName:NewName", p))
		}
		m[pair[0]] = pair[1]
	}

	goinlineArgs := []string{
		"-package=" + path.Join(blueprints, in.Pkg),
		"--target-package-name=" + out.Pkg,
		"--target-dir=.",
		"-w",
	}
	for in, out := range m {
		goinlineArgs = append(goinlineArgs, fmt.Sprintf("%s->%s", in, out))
	}

	do("goinline", goinlineArgs...)
	do("mv", in.File, out.File)
	do("sed", "-i", ".bak", fmt.Sprintf("s/%s/%s/g", in.Typ, out.Typ), out.File)
	for k, v := range m {
		do("sed", "-i", ".bak", fmt.Sprintf("s/%s/%s/g", k, v), out.File)
		do("rm", out.File+".bak")
	}
	do("goimports", "-w", out.File)
	ext := filepath.Ext(out.File)
	finalFileName := strings.TrimSuffix(out.File, ext) + "_generated" + ext
	os.Rename(out.File, finalFileName)
}

func do(name string, args ...string) {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		fmt.Sprintln(err)
		os.Exit(1)
	}
}

func parseSpec(s string) (*spec, error) {
	err := fmt.Errorf(
		"got spec %q; expected format: %q", s, "pkg.TypeName(filename.go)")
	c := strings.Split(strings.TrimSuffix(s, ")"), "(")
	if len(c) != 2 {
		return nil, err
	}
	filename := c[1]
	if !strings.HasSuffix(filename, ".go") {
		return nil, err
	}
	c = strings.Split(c[0], ".")
	if len(c) != 2 {
		return nil, err
	}
	pkg := c[0]
	typ := c[1]
	return &spec{pkg, typ, filename}, nil
}

func usage(a ...interface{}) {
	if len(a) != 0 {
		fmt.Println(a...)
	}
	fmt.Println("usage: gen <input-spec> <output-spec> [<orig-type>:<new-type>]...")
	os.Exit(1)
}
