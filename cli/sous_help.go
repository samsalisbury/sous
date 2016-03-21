package cli

type SousHelp struct {
	Sous *Sous
}

const sousHelpHelp = `
show help information

args: [command]

sous help shows help information for sous itself, as well as all its subcommands
for detailed help with any command, use 'sous help <command>'
`

func (sh *SousHelp) Help() *Help { return ParseHelp(sousHelpHelp) }

func (sh *SousHelp) Execute(args []string, out, _ Output) Result {
	commands := sh.Sous.Subcommands()
	if len(args) == 0 {
		out.Println(sh.Sous.Help().Usage(""), "\n")
		out.Println("commands:")
		out.Indent()
		out.Table(commandTable(commands))
		out.Outdent()
		out.Println()
		return Successf("sous version %s", sh.Sous.Version)
	}
	name := args[0]
	c, ok := commands[name]
	if !ok {
		return UsageErrorf(nil, "command %s does not exist, try `sous help`",
			name)
	}
	return commandHelp(out, c)
}

func commandHelp(out Output, c Command) Result {
	return Success()
}

func commandTable(cs Commands) [][]string {
	t := make([][]string, len(cs))
	for i, name := range cs.SortedKeys() {
		t[i] = make([]string, 2)
		t[i][0] = name
		t[i][1] = cs[name].Help().Short
	}
	return t
}
