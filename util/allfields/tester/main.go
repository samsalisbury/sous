package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/util/allfields"
)

func main() {
	ast := allfields.ParseDir("lib/")
	tree := allfields.ExtractTree(ast, "Deployment")
	spew.Dump(tree)
	spew.Dump(allfields.ConfirmTree(tree, ast, "Diff"))
}
