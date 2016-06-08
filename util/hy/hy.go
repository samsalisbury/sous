/*
Package hy enables marshaling and unmarshaling tagged structs as filesystem
trees of YAML files.

For example, the following program...

    type Thing struct {
    	Config  map[string]string `hy:"config.yaml"`
    	Widgets map[string]Widget `hy:"widgets/"`
    }

    type Widget struct{ Colour string }

    func main() {
    	t := Thing{
    		Config: map[string]string{"some-key": "some-value"},
    		Widgets: map[string]Widget{
    			"blue":   {"Blue"},
    			"orange": {"Orange"},
    		},
    	}
    	hy.Marshal("some_dir", &t)
	}

would produce a filesystem hierarchy that looks like this:

    some_dir/
    ├── config.yaml
    └── widgets
        ├── blue.yaml
        └── orange.yaml

With file contents you would expect:

    $ cat some_dir/config.yaml
    some-key: some-value
	$ cat some_dir/widgets/blue.yaml
	Colour: Blue
	$ cat some_dir/widgets/orange.yaml
	Colour: Orange

*/
package hy

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
)

// Debug is a global flag, set it to true to print debug messages when
// marshaling and unmarshaling using the log package.
var Debug = false

func debugf(format string, a ...interface{}) {
	if !Debug {
		return
	}
	_, fn, ln, ok := runtime.Caller(1)
	if ok {
		fn := filepath.Base(fn)
		log.Printf("%s:%d %s", fn, ln, fmt.Sprintf(format, a...))
	} else {
		log.Printf(format, a...)
	}
}

func debug(a ...interface{}) {
	if !Debug {
		return
	}
	_, fn, ln, ok := runtime.Caller(1)
	if ok {
		fn := filepath.Base(fn)
		log.Printf("%s:%d %s", fn, ln, fmt.Sprintln(a...))
	} else {
		log.Println(a...)
	}
}
