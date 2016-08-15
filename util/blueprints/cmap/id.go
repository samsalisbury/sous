package cmap

// ID returns the Key identifying this Value.
// Note: this is in a separate file so that it doesn't get copied.
func (v Value) ID() CMKey {
	return CMKey(string(v))
}
