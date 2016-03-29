package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type File struct {
	From, Maintainer string
	Instructions     Instructions
}

func (f File) String() string {
	return fmt.Sprintf("FROM %s\nMAINTAINER %s\n%s",
		f.From, f.Maintainer, f.Instructions)
}

type Instructions []Instruction

func (is Instructions) String() string {
	b := &bytes.Buffer{}
	for _, i := range is {
		fmt.Fprintf(b, "%s\n", i)
	}
	return b.String()
}

func (is *Instructions) Add(name string, args Args) {
	*is = append(*is, Instruction{name, args})
}

type Instruction struct {
	Name string
	Args Args
}

func (i Instruction) String() string {
	return fmt.Sprintf("%s %s", i.Name, i.Args)
}

type Args interface {
	String() string
}

func (f *File) ADD(target string, files ...string) {
	f.Instructions.Add("ADD", ArrayArgs(append(files, target)))
}

func (f *File) COPY(target string, files ...string) {
	f.Instructions.Add("COPY", ArrayArgs(append(files, target)))
}

func (f *File) RUN(args ...string) {
	f.Instructions.Add("RUN", SpaceSeparatedArgs(args))
}

func (f *File) LABEL(m map[string]string) {
	f.Instructions.Add("LABEL", KeyValueArgs(m))
}

func (f *File) ENV(m map[string]string) {
	f.Instructions.Add("ENV", KeyValueArgs(m))
}

func (f *File) CMD(args ...string) {
	f.Instructions.Add("CMD", ArrayArgs(args))
}

func (f *File) ENTRYPOINT(args ...string) {
	f.Instructions.Add("ENTRYPOINT", ArrayArgs(args))
}

func (f *File) WORKDIR(path string) {
	f.Instructions.Add("WORKDIR", SingleArg(path))
}

func (f *File) USER(username string) {
	f.Instructions.Add("USER", SingleArg(username))
}

type SingleArg string

func (s SingleArg) String() string {
	return string(s)
}

type KeyValueArgs map[string]string

func (kv KeyValueArgs) String() string {
	items := kv.Flatten()
	return lines(items)
}

func (kv KeyValueArgs) Flatten() []string {
	out := make([]string, len(kv))
	for k, v := range kv {
		out = append(out, fmt.Sprintf("%s=%s", quote(k), quote(v)))
	}
	return out
}

func quote(s string) string {
	if !strings.ContainsAny(s, `' "`) {
		return s
	}
	return fmt.Sprintf("%q", s)
}

type ArrayArgs []string

func (a ArrayArgs) String() string {
	b, err := json.Marshal(a)
	if err != nil {
		panic("unable to marshal array: " + err.Error())
	}
	return string(b)
}

type SpaceSeparatedArgs []string

func (s SpaceSeparatedArgs) String() string {
	return strings.Join(s, " ")
}

func lines(s []string) string {
	out := &bytes.Buffer{}
	lineStart := " \\\n\t"
	for _, line := range s {
		out.WriteString(lineStart + line)
	}
	return out.String()
}

type ArrayArg []string
