package sous

// OwnerSet collects the names of the owners of a deployment.
type OwnerSet map[string]struct{}

// Add adds an owner to an ownerset.
func (os OwnerSet) Add(owner string) {
	os[owner] = struct{}{}
}

// Remove removes an owner from an ownerset.
func (os OwnerSet) Remove(owner string) {
	delete(os, owner)
}

// Equal returns true if two ownersets contain the same owner names.
func (os OwnerSet) Equal(o OwnerSet) bool {
	if len(os) != len(o) {
		return false
	}
	for ownr := range os {
		if _, has := o[ownr]; !has {
			return false
		}
	}

	return true
}
