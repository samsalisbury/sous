package main

import (
	"github.com/opentable/sous/util/allfields"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

func main() {
	ast := allfields.ParseDir("lib/")
	tree := allfields.ExtractTree(ast, "Deployment")
	messages.ReportLogFieldsMessage("Dump", logging.ExtraDebug1Level, logging.Log, allfields.ConfirmTree(tree, ast, "Diff"))
}
