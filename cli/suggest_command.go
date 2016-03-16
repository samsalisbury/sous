package cli

import "fmt"

type SuggestCommand struct {
	EnteredCommand string
}

func (sc *SuggestCommand) Help() string { return "" }

func (sc *SuggestCommand) Execute() error {
	return UsageError{
		Message: fmt.Sprintf("command %s not recognised", sc.EnteredCommand),
		Tip:     "try sous help for a list of available commands",
	}
}
