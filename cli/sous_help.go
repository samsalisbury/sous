package cli

import "flag"

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
	if len(args) == 0 {
		sh.printMainSousHelp(out)
	} else if err := sh.printCommandHelp(out, args, "sous", sh.Sous); err != nil {
		return err
	}
	return Successf("\nsous version %s", sh.Sous.Version)
}

func (sh *SousHelp) printCommandHelp(out Output, args []string, name string, base HasSubcommands) ErrorResult {
	subcommandName := args[0]
	name = name + " " + subcommandName
	args = args[1:]
	commands := base.Subcommands()
	fullName := name
	c, ok := commands[subcommandName]
	if !ok {
		return UsageErrorf(nil, "command %q does not exist", name)
	}
	if len(args) != 0 {
		if hasSubCommands, ok := c.(HasSubcommands); ok {
			return sh.printCommandHelp(out, args, name, hasSubCommands)
		}
		return UsageErrorf(nil, "command %q does not exist", name)
	}
	help := c.Help()
	out.Println(help.Usage(fullName))
	out.Println()
	out.Println(help.Long)
	return nil
}

func (sh *SousHelp) printMainSousHelp(out Output) {
	sh.printSubcommands(out, "sous", sh.Sous.Subcommands())
	out.Println("\nglobal options:")
	sh.printOptions(out, "sous", sh.Sous)
}

func (sh *SousHelp) printSubcommands(out Output, name string, cs Commands) {
	out.Println(sh.Sous.Help().Usage(name))
	out.Println("\ncommands:")
	out.Indent()
	defer out.Outdent()
	out.Table(commandTable(cs))
}

func (sh *SousHelp) printOptions(out Output, name string, command Command) {
	addsFlags, ok := command.(AddsFlags)
	if !ok {
		return
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	addsFlags.AddFlags(fs)
	fs.SetOutput(out.Writer)
	fs.PrintDefaults()
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
