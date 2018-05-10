package main

import (
	"github.com/opentable/sous/util/allfields"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

func main() {
	log := *(logging.SilentLogSet().Child("tester").(*logging.LogSet))
	ast := allfields.ParseDir("lib/")
	tree := allfields.ExtractTree(ast, "Deployment")
	messages.ReportLogFieldsMessage("Dump", logging.ExtraDebug1Level, log, allfields.ConfirmTree(tree, ast, "Diff"))
}
