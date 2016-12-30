package cmdr

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/opentable/sous/util/cmdr/style"
)

type (
	// CLI encapsulates a command line interface, and knows how to execute
	// commands and their subcommands and parse flags.
	CLI struct {
		// Root is the root command to execute.
		Root Command
		// Out is an *Output, defaults to a plain os.Stdout writer if left nil.
		Out,
		// Err is an *Output, defaults to a plain os.Stderr writer if left nil.
		Err *Output
		// Env is a map of environment variable names to their values.
		Env map[string]string
		// Hooks allow you to perform pre and post processing on Commands at
		// various points in their lifecycle.
		Hooks Hooks
		// HelpCommand tells the user how to get help. It should be a command
		// they can type in to get help.
		HelpCommand string
		// GlobalFlagSetFuncs allow global flags to be added to the CLI.
		GlobalFlagSetFuncs []func(*flag.FlagSet)
		// IndentString is the default indent to use for indenting command
		// output when Output.Indent() is called inside a command. If left
		// empty, defaults to DefaultIndentString.
		IndentString string
	}
	// Hooks is a collection of command hooks. If a hook returns a non-nil error
	// it cancels execution and the error is displayed to the user.
	Hooks struct {
		// Startup is run at the beginning of every invocation
		Startup func(*CLI) error
		// Parsed is run on a command when it's found on the command line
		Parsed func(Command) error
		// PreExecute is run on a command before it executes.
		PreExecute func(Command) error
		// PreFail is run just before a command exits with an error.
		// Is is passed the error, and must return the final ErrorResult.
		PreFail func(error) ErrorResult
	}

	// A PreparedExecution collects all the information needed to execute a command
	PreparedExecution struct {
		Cmd  Executor
		Args []string
	}
)

const (
	// DefaultIndentString is the default indent to use when writing
	// procedurally to the CLI outputs. It is set to two consecutive spaces,
	// matching default output from the flag package.
	DefaultIndentString = "  "
)

// init populates CLI with default values where none have been set already.
func (c *CLI) init() {
	if c.Out == nil {
		c.Out = NewOutput(os.Stdout)
	}
	if c.Err == nil {
		c.Err = NewOutput(os.Stderr)
	}
	indentString := DefaultIndentString
	if c.IndentString != "" {
		indentString = c.IndentString
	}
	c.Out.SetIndentStyle(indentString)
	c.Err.SetIndentStyle(indentString)
}

// Invoke begins invoking the CLI starting with the base command, and handles
// all command output. It then returns the result for further processing.
func (c *CLI) Invoke(args []string) Result {
	result := c.InvokeWithoutPrinting(args)
	c.OutputResult(result)
	return result
}

// IsSuccess checks if a Result is a success
func (c *CLI) IsSuccess(result Result) bool {
	_, ok := result.(SuccessResult)
	return ok
}

// OutputResult formats and outputs a cmdr.Result -
// handles tips when the result is an error, etc.
// returns true if the result was a success
func (c *CLI) OutputResult(result Result) {
	if success, ok := result.(SuccessResult); ok {
		c.handleSuccessResult(success)
	}
	if result == nil {
		result = InternalErrorf("nil result returned from %T", c.Root)
	}
	if err, ok := result.(ErrorResult); ok {
		c.handleErrorResult(err)
	}
}

// InvokeWithoutPrinting invokes the CLI without printing the results.
func (c *CLI) InvokeWithoutPrinting(args []string) Result {
	prepped, err := c.Prepare(args)
	if err != nil {
		return EnsureErrorResult(err)
	}
	return prepped.Cmd.Execute(prepped.Args)
}

// InvokeAndExit calls Invoke, and exits with the returned exit code.
func (c *CLI) InvokeAndExit(args []string) {
	os.Exit(c.Invoke(args).ExitCode())
}

// Prepare sets up a command for execution, resolving subcommands and flags, and forwarding
// flags from lower commands to higher ones. This means that flags defined on
// the base command are also defined by default all its nested subcommands,
// which is usually a nicer user experience than having to remember strictly
// which subcommand a flag is applicable to.
func (c *CLI) Prepare(args []string) (*PreparedExecution, error) {
	base, ff := c.Root, c.GlobalFlagSetFuncs
	if c.Hooks.Startup != nil {
		c.Hooks.Startup(c)
	}
	return c.prepare(base, args, ff)
}

// runHook runs the hook if it's not nil, and returns the hook's error.
func (c *CLI) runHook(hook func(Command) error, command Command) error {
	if hook == nil {
		return nil
	}
	return hook(command)
}

func (c *CLI) handleSuccessResult(s SuccessResult) {
	if len(s.Data) != 0 {
		// Hm.
		c.Out.Write(s.Data)
	}
}

func (c *CLI) handleErrorResult(e ErrorResult) {
	if c.Hooks.PreFail != nil {
		var underlyingErr error
		if ue, ok := e.(interface {
			UnderlyingError() error
		}); ok {
			underlyingErr = ue.UnderlyingError()
		}
		e = c.Hooks.PreFail(underlyingErr)
	}
	c.Err.Println(e)
	c.printTip(e.UserTip())
}

func (c *CLI) printTip(tip string) {
	if tip == "" {
		return
	}
	c.Err.PushStyle(style.Style{style.Blue, style.Bold})
	c.Err.Printf("Tip: ")
	c.Err.PopStyle()
	c.Err.Printfln(tip)
}

// SetVerbosity sets the verbosity.
func (c *CLI) SetVerbosity(v Verbosity) {
	c.Err.Verbosity = v
}

// ListSubcommands returns a slice of strings with the names of each subcommand
// as they need to be entered by the user, arranged alphabetically.
func (c *CLI) ListSubcommands(base Command) []string {
	hs, ok := base.(Subcommander)
	if !ok {
		return nil
	}
	subcommands := hs.Subcommands()
	list := make([]string, len(subcommands))
	i := 0
	for name := range subcommands {
		list[i] = name
		i++
	}
	sort.Strings(list)
	return list
}

func (c *CLI) prepare(cmd Command, cmdArgs []string, flagAddFuncs []func(*flag.FlagSet)) (*PreparedExecution, error) {
	if len(cmdArgs) == 0 {
		return nil, InternalErrorf("command %T received zero args", cmd)
	}
	cmdName := cmdArgs[0]
	cmdArgs = cmdArgs[1:]
	// Add and parse flags for this command.
	if cmdHasFlags, ok := cmd.(AddsFlags); ok {
		if flagAddFuncs == nil {
			flagAddFuncs = []func(*flag.FlagSet){}
		}
		// add these flags to the agglomeration
		flagAddFuncs = append(flagAddFuncs, cmdHasFlags.AddFlags)
	}
	// If this command has subcommands, first try to descend into one of them.
	if cmdHasSubcmd, ok := cmd.(Subcommander); ok && len(cmdArgs) != 0 {
		subcommandName := cmdArgs[0]
		subcommands := cmdHasSubcmd.Subcommands()
		if cmdHasSubCmd, ok := subcommands[subcommandName]; ok {
			if err := c.runHook(c.Hooks.Parsed, cmd); err != nil {
				return nil, EnsureErrorResult(err)
			}
			return c.prepare(cmdHasSubCmd, cmdArgs, flagAddFuncs)
		}
	}
	// If the command can itself be executed, do that now.
	if cmdCanExec, ok := cmd.(Executor); ok {
		c.init()
		// make a flag.FlagSet named for this command.
		fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
		// try to pipe normal flag output to /dev/null, don't fail if not though
		if devNull, err := os.Open(os.DevNull); err == nil {
			fs.SetOutput(devNull)
		}
		// add global flags
		for _, addFlags := range c.GlobalFlagSetFuncs {
			addFlags(fs)
		}
		// add own and forwarded flags to the flagset, note that it will panic
		// if multiple flags with the same name are added.
		for _, addFlags := range flagAddFuncs {
			addFlags(fs)
		}
		// parse the entire flagset for this command
		if err := fs.Parse(cmdArgs); err != nil {
			tip := fmt.Sprintf("for help, use `%s`", c.HelpCommand)
			if err == flag.ErrHelp {
				return nil, UsageErrorf(tip)
			}
			return nil, UsageErrorf(err.Error()).WithTip(tip)
		}
		// get the remaining args
		bottomCmdArgs := fs.Args()

		if err := c.runHook(c.Hooks.Parsed, cmd); err != nil {
			return nil, err
		}
		if err := c.runHook(c.Hooks.PreExecute, cmd); err != nil {
			return nil, err
		}
		return &PreparedExecution{Cmd: cmdCanExec, Args: bottomCmdArgs}, nil
	}
	// If we get here, this command is not configured correctly and cannot run.
	return nil, InternalErrorf("%q is not runnable and has no subcommands", cmdName)
}
