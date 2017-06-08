package allfields

type (
	confNode interface {
		name() string
		confirm()
		selfConfirmed() bool
		confirmed() bool
		missed() []string
	}

	structNode struct {
		*confirmation
		kids  map[string]confNode
		needs []fieldNeed
	}

	confirmation struct {
		isConfirmed bool
		typeName    string
	}

	confTreeNode interface {
		confNode
		child(named string) confNode
		children() []confNode
	}
)

func (c *confirmation) name() string {
	return c.typeName
}

func (c *confirmation) confirm() {
	c.isConfirmed = true
}

func (c *confirmation) confirmed() bool {
	return c.isConfirmed
}

func (c *confirmation) selfConfirmed() bool {
	return c.isConfirmed
}

func (c *confirmation) missed() []string {
	if c.selfConfirmed() {
		return []string{}
	}
	return []string{""}
}

func newStructNode(name string) *structNode {
	return &structNode{
		kids: map[string]confNode{},
		confirmation: &confirmation{
			typeName: name,
		},
		needs: []fieldNeed{},
	}
}

func (sn *structNode) children() []confNode {
	ns := []confNode{}
	for _, k := range sn.kids {
		ns = append(ns, k)
	}
	return ns
}

func (sn *structNode) child(named string) confNode {
	return sn.kids[named]
}

func (sn *structNode) selfConfirmed() bool {
	return sn.confirmation.confirmed()
}

func (sn *structNode) missed() []string {
	missed := []string{}
	if !sn.selfConfirmed() {
		missed = append(missed, "")
	}
	for n, c := range sn.kids {
		for _, cm := range c.missed() {
			missed = append(missed, "."+n+cm)
		}
	}
	return missed
}

func (sn *structNode) confirmed() bool {
	if !sn.selfConfirmed() {
		return false
	}
	for _, c := range sn.children() {
		if !c.confirmed() {
			return false
		}
	}
	return true
}
