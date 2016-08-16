package hy

// NodeSet is a set of Node pointers indexed by ID.
type NodeSet struct {
	nodes map[NodeID]*Node
}

// NewNodeSet creates a new node set.
func NewNodeSet() NodeSet {
	return NodeSet{nodes: map[NodeID]*Node{}}
}

// Register tries to register a node ID. If the ID is not yet registered, it
// returns a new node pointer and true. Otherwise it returns the already
// registered node pointer and false.
func (ns NodeSet) Register(id NodeID) (*Node, bool) {
	n, ok := ns.nodes[id]
	if ok {
		return n, false
	}
	n = new(Node)
	ns.nodes[id] = n
	return n, true
}
