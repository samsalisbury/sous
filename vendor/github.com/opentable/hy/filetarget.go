package hy

import "github.com/pkg/errors"

// FileTarget represents a target file to be written.
type FileTarget struct {
	FilePath string
	Value    interface{}
}

// Path returns FilePath.
func (ft FileTarget) Path() string { return ft.FilePath }

// Data returns the Value.
func (ft FileTarget) Data() interface{} { return ft.Value }

// FileTargets is a map of file targets.
type FileTargets struct {
	m map[string]*FileTarget
}

// NewFileTargets creates a new FileTargets.
func NewFileTargets(targets ...*FileTarget) (FileTargets, error) {
	fts := FileTargets{m: make(map[string]*FileTarget, len(targets))}
	return fts.add(targets)
}

// MakeFileTargets creates a new FileTargets with a starting capacity.
func MakeFileTargets(capacity int) FileTargets {
	return FileTargets{m: make(map[string]*FileTarget, capacity)}
}

func (fts FileTargets) add(targets []*FileTarget) (FileTargets, error) {
	if fts.m == nil {
		fts.m = make(map[string]*FileTarget, len(targets))
	}
	for _, t := range targets {
		if _, ok := fts.m[t.FilePath]; ok {
			return fts, errors.Errorf("duplicate file target %q", t.FilePath)
		}
		fts.m[t.FilePath] = t
	}
	return fts, nil
}

// Len returns the length.
func (fts FileTargets) Len() int { return len(fts.m) }

// Paths returns the paths of all file targets.
func (fts FileTargets) Paths() []string {
	paths := make([]string, len(fts.m))
	i := 0
	for k := range fts.m {
		paths[i] = k
		i++
	}
	return paths
}

// Snapshot returns the map this FileTargets represents.
func (fts FileTargets) Snapshot() map[string]*FileTarget {
	// TODO: Make this a copy.
	return fts.m
}

// AddAll adds the contents of another FileTargets to this one.
// Returns an error if any of them share a path.
func (fts FileTargets) AddAll(other FileTargets) error {
	for _, t := range other.m {
		if err := fts.Add(t); err != nil {
			return err
		}
	}
	return nil
}

// Add adds any number of file targets to this one.
// Returns an error if any of them share a path.
func (fts FileTargets) Add(targets ...*FileTarget) error {
	_, err := fts.add(targets)
	return err
}
