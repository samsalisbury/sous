package hy

// WriteTarget represents an output target, typically a file.
type WriteTarget interface {
	// Path is the path where this target is stored.
	Path() string
	// Data is the go value to be stored.
	Data() interface{}
}

// ReadTarget represents an input target, typically a file.
type ReadTarget interface {
	// Path is the path where this target is stored.
	Path() string
}
