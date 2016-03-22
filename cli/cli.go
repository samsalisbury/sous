package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

type (
	// CLI encapsulates a command line interface, and knows how to execute
	// commands and their subcommands and parse flags.
	CLI struct {
		// OutWriter will be sent the output of the CLI. This is typically set
		// to os.Stdout
		OutWriter,
		// ErrWriter will be sent all log messages from the CLI. This is
		// typically set to os.Stderr
		ErrWriter io.Writer
		// Out is an Output wired up to OutWriter
		Out,
		// Err is an Output wired up to ErrWriter
		Err Output
		// Env is a map of environment variable names to their values.
		Env map[string]string
		// Hooks allow you to perform pre and post processing on Commands at
		// various points in their lifecycle.
		Hooks Hooks
		// HelpCommand tells the user how to get help.
		HelpCommand string
		// IndentString is the default indent to use for command output.
		// Defaults to two spaces, if left empty.
		IndentString string
	}
	// Hooks is a collection of command hooks. If a hook returns a non-nil error
	// it cancels execution and the error is displayed to the user.
	Hooks struct {
		// PreExecute is run on a command before it executes.
		PreExecute func(Command) error
	}
)

const (
	defaultIndentString = "  "
)

// Invoke begins invoking the CLI starting with the base command, and handles
// all command output. It then returns the result for further processing.
func (c *CLI) Invoke(base Command, args []string) Result {
	result := c.invoke(base, args, nil)
	if success, ok := result.(SuccessResult); ok {
		c.handleSuccessResult(success)
	}
	if result == nil {
		result = InternalErrorf(nil, "nil result returned from %T", base)
	}
	if err, ok := result.(ErrorResult); ok {
		c.handleErrorResult(err)
	}
	return result
}

// InvokeAndExit calls Invoke, and exits with the returned exit code.
func (c *CLI) InvokeAndExit(base Command, args []string) {
	os.Exit(c.Invoke(base, args).ExitCode())
}

func (c *CLI) Info(v ...interface{}) {
	if c.ErrWriter == nil {
		return
	}
	fmt.Fprintln(c.ErrWriter, v...)
}

func (c *CLI) handleSuccessResult(s SuccessResult) {
	if len(s.Data) != 0 && c.OutWriter != nil {
		c.OutWriter.Write(s.Data)
	}
}

// runHook runs the hook if it's not nil, and returns the hook's error.
func (c *CLI) runHook(hook func(Command) error, command Command) error {
	if hook == nil {
		return nil
	}
	return hook(command)
}

func (c *CLI) handleErrorResult(e ErrorResult) {
	if c.ErrWriter == nil {
		return
	}
	fmt.Fprintln(c.ErrWriter, e.Error())
	if tip := e.UserTip(); len(tip) != 0 {
		fmt.Fprintln(c.ErrWriter, tip)
	}
}

// ListSubcommands returns a slice of strings with the names of each subcommand
// as they need to be entered by the user, arranged alphabetically.
func (c *CLI) ListSubcommands(base Command) []string {
	hs, ok := base.(HasSubcommands)
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

// invoke invokes this command, resolving subcommands and flags, and forwarding
// flags from lower commands to higher ones. This means that flags defined on
// the base command are also defined by default all its nested subcommands,
// which is usually a nicer user experience than having to remember strictly
// which subcommand a flag is applicable to. The ff parameter deals with these
// flags.
func (c *CLI) invoke(base Command, args []string, ff []func(*flag.FlagSet)) Result {
	if len(args) == 0 {
		return InternalErrorf(nil, "command %T received zero args", base)
	}
	name := args[0]
	args = args[1:]
	// Add and parse flags for this command.
	if command, ok := base.(AddsFlags); ok {
		if ff == nil {
			ff = []func(*flag.FlagSet){}
		}
		// add these flags to the agglomeration
		ff = append(ff, command.AddFlags)
		// make a flag.FlagSet named for this command.
		fs := flag.NewFlagSet(name, flag.ContinueOnError)
		devNull, err := os.Open(os.DevNull)
		if err != nil {
			return OSError{Err: err}
		}
		fs.SetOutput(devNull)
		// add own and forwarded flags to the flagset, note that it will panic
		// if multiple flags with the same name are added.
		for _, addFlags := range ff {
			addFlags(fs)
		}
		// parse the entire flagset for this command
		if err := fs.Parse(args); err != nil {
			if err == flag.ErrHelp {
				return UsageErrorf(nil, "for help, use `%s`", c.HelpCommand)
			}
			return UsageError{Message: err.Error()}
		}
		// get the remaining args
		args = fs.Args()
	}
	// If this command has subcommands, first try to descend into one of them.
	if command, ok := base.(HasSubcommands); ok && len(args) != 0 {
		subcommandName := args[0]
		subcommands := command.Subcommands()
		if subcommand, ok := subcommands[subcommandName]; ok {
			return c.invoke(subcommand, args, ff)
		}
	}
	// If the command can itself be executed, do that now.
	if command, ok := base.(CanExecute); ok {
		if err := c.runHook(c.Hooks.PreExecute, base); err != nil {
			return EnsureErrorResult(err)
		}
		out := Output{Writer: c.OutWriter}
		errout := Output{Writer: c.ErrWriter}
		indentString := c.IndentString
		if indentString == "" {
			indentString = defaultIndentString
		}
		out.SetIndentStyle(indentString)
		errout.SetIndentStyle(indentString)
		return command.Execute(args, out, errout)
	}
	// If we get here, this command is not configured correctly and cannot run.
	m := fmt.Sprintf("command %q cannot execute and has no subcommands", name)
	return InternalError{Message: m}
}
