package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

// started from http://stackoverflow.com/questions/17796043/golang-embedding-text-file-into-compiled-executable

// Reads all .txt files in the current folder
// and encodes them as strings literals in textfiles.go

const header = `// This file was automatically generated based on the contents of *.tmpl
// If you need to update this file, change the contents of those files
// (or add new ones) and run 'go generate'

`

func main() {
	out, err := os.Create("templates.go")
	if err != nil {
		log.Fatal(err)
	}

	fs, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	out.Write([]byte(header))
	out.Write([]byte("package sous \n\nconst (\n"))
	for _, f := range fs {
		if strings.HasSuffix(f.Name(), ".tmpl") {
			out.Write([]byte(strings.TrimSuffix(f.Name(), ".tmpl") + "Tmpl = \""))
			f, err := os.Open(f.Name())
			if err != nil {
				log.Fatal(err)
			}
			r := newEscaper(f)

			io.Copy(out, r)
			out.Write([]byte("\"\n\n"))
		}
	}
	out.Write([]byte(")\n"))
}

type escaper struct {
	r   io.Reader
	old []byte
}

var doublesRE = regexp.MustCompile(`"`)
var newlsRE = regexp.MustCompile("(?m)\n")

func (e *escaper) Read(p []byte) (n int, err error) {
	new := make([]byte, len(p)-len(e.old))
	c, err := e.r.Read(new)
	new = append(e.old, new[0:c]...)

	i, n := 0, 0
	for ; i < len(new) && n < len(p); i, n = i+1, n+1 {
		switch new[i] {
		default:
			p[n] = new[i]
		case '"', '\\':
			p[n] = '\\'
			n++
			p[n] = new[i]
		case '\n':
			p[n] = '\\'
			n++
			p[n] = 'n'
		}
	}
	if len(p) < i {
		e.old = new[len(new)-(len(p)-i):]
	} else {
		e.old = new[0:0]
	}

	log.Print(i, "/", n, "\n", len(e.old), ":", string(e.old), "\n", len(p), ":", string(p), "\n\n**************************\n\n")

	return
}

func newEscaper(r io.Reader) *escaper {
	return &escaper{r, make([]byte, 0)}
}
