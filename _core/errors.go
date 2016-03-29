package core

import "fmt"

type Severity string

const (
	ERROR   = Severity("ERROR")
	WARNING = Severity("WARNING")
)

type Error struct {
	error
	Severity Severity
}

func Errorf(format string, a ...interface{}) Error {
	return Error{fmt.Errorf(format, a...), ERROR}
}

// Warningf creates an Error with Severity = WARNING, this is mostly
// just for presenting messages about projects that could be better.
func Warningf(format string, a ...interface{}) Error {
	return Error{fmt.Errorf(format, a...), WARNING}
}

type ErrorCollection []Error

func (c *ErrorCollection) Add(e Error) {
	*c = append(*c, e)
}

func (c *ErrorCollection) AddErrorf(format string, a ...interface{}) {
	c.Add(Errorf(format, a...))
}

func (c *ErrorCollection) AddWarningf(format string, a ...interface{}) {
	c.Add(Warningf(format, a...))
}

func (c ErrorCollection) Filter(s Severity) ErrorCollection {
	errs := ErrorCollection{}
	for _, e := range c {
		if e.Severity == s {
			errs.Add(e)
		}
	}
	return errs
}

func (c ErrorCollection) Errors() ErrorCollection {
	return c.Filter(ERROR)
}

func (c ErrorCollection) Warnings() ErrorCollection {
	return c.Filter(WARNING)
}

func (c ErrorCollection) Strings() []string {
	s := make([]string, len(c))
	for i, e := range c {
		s[i] = e.Error()
	}
	return s
}
