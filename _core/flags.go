package core

import (
	"flag"
	"fmt"
	"reflect"
	"time"
)

type Flags struct {
	*flag.FlagSet
	flags map[string]Flag
}

func NewFlags() *Flags {
	return &Flags{flag.NewFlagSet("", flag.ExitOnError), map[string]Flag{}}
}

type Flag struct {
	DefaultValue interface{}
	CreateFlag   func(out interface{})
}

func (f *Flag) Bind(v interface{}) {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Ptr || t.Elem() != reflect.TypeOf(f.DefaultValue) {
		panic(fmt.Sprintf("Bind: received %T; want *%T", v, f.DefaultValue))
	}
	f.CreateFlag(v)
}

func (fs *Flags) Bind(name string, v interface{}) {
	f, ok := fs.flags[name]
	if !ok {
		panic(fmt.Sprintf("flag %s not defined", name))
	}
	f.Bind(v)
}

func (fs *Flags) AddFlag(name, usage string, value interface{}) {
	f := Flag{DefaultValue: value}
	switch v := value.(type) {
	default:
		panic(fmt.Sprintf("AddFlag does not support %T values", value))
	case string:
		f.CreateFlag = func(out interface{}) { o := out.(*string); fs.StringVar(o, name, v, usage) }
	case bool:
		f.CreateFlag = func(out interface{}) { o := out.(*bool); fs.BoolVar(o, name, v, usage) }
	case int:
		f.CreateFlag = func(out interface{}) { o := out.(*int); fs.IntVar(o, name, v, usage) }
	case time.Duration:
		f.CreateFlag = func(out interface{}) { o := out.(*time.Duration); fs.DurationVar(o, name, v, usage) }
	}
	fs.flags[name] = f
}

func _init() {
	f := NewFlags()
	// Universal (almost) flags
	f.AddFlag("rebuild", "force rebuild of the target", false)
	f.AddFlag("rebuild-all", "force rebuild of the target and all dependencies", false)

	// Contracts flags
	f.AddFlag("timeout", "per-contract timeout", 5*time.Second)
	f.AddFlag("timeout-all", "total contract run timeout", 5*time.Second)
	f.AddFlag("parallelism", "how many contracts to run at once", 1)
	f.AddFlag("image", "run contracts against a specific docker image", "")

	var rebuild, rebuildAll bool
	var timeout time.Duration
	var parallelism int
	f.Bind("rebuild", &rebuild)
	f.Bind("rebuild-all", &rebuildAll)
	f.Bind("timeout", &timeout)
	f.Bind("parallelism", &parallelism)

	f.Parse([]string{"-timeout", "7m", "-rebuild", "-parallelism", "3", "some", "args"})
	f.PrintDefaults()
	fmt.Printf("Program name: %s; Parallelism: %d, Duration: %s; Rebuild: %v; Remaining args: %+v",
		f.Arg(0), parallelism, timeout, rebuild, f.Args()[1:])
}
