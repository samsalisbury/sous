package hy

import (
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// ReadContext is context collected during a read opration.
type ReadContext struct {
	// Targets is the collected targets for this read context.
	targets FileTargets
	// Reader reads data from path.
	Reader FileReader
	// Parent is the parent read context.
	Parent *ReadContext
	// PathName is the name of this section of the path.
	PathName string
	// Prefix is the path prefix.
	Prefix string
}

// NewReadContext returns a new read context.
func NewReadContext(prefix string, targets FileTargets, reader FileReader) ReadContext {
	return ReadContext{Prefix: prefix, targets: targets, Reader: reader}
}

// Push creates a derivative node context.
func (c ReadContext) Push(pathName string) ReadContext {
	return ReadContext{
		targets:  c.targets,
		Reader:   c.Reader,
		Parent:   &c,
		PathName: pathName,
		Prefix:   c.Prefix,
	}
}

// List lists files in the current directory.
// TODO: This is horrible, need a tree file structure for targets.
func (c ReadContext) List() []string {
	set := map[string]struct{}{}
	trim := c.Path() + "/"
	for _, path := range c.targets.Paths() {
		if !strings.HasPrefix(path, trim) {
			continue
		}
		p := strings.TrimPrefix(path, trim)
		set[p] = struct{}{}
	}
	l := make([]string, len(set))
	i := 0
	for pathName := range set {
		l[i] = pathName
		i++
	}
	sort.Strings(l)
	return l
}

func (c ReadContext) Read(v interface{}) error {
	if !c.Exists() {
		return nil
	}
	return errors.Wrapf(c.Reader.ReadFile(c.Prefix, c.Path(), v), "reading %q", c.Path())
}

// Exists checks that a file exists at the current path.
func (c ReadContext) Exists() bool {
	_, ok := c.targets.Snapshot()[c.Path()]
	if ok {
		return true
	}
	return len(c.List()) != 0
}

// Path returns the path of this context.
func (c ReadContext) Path() string {
	if c.Parent == nil {
		return c.PathName
	}
	return path.Join(c.Parent.Path(), c.PathName)
}
