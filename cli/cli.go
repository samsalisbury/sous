package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type (
	// CLI encapsulates a command line interface, and knows how to execute commands
	// and their subcommands and parse flags.
	CLI struct {
		// OutWriter will be sent the output of the CLI. This is typically set to
		// os.Stdout
		OutWriter,
		// ErrWriter will be sent all log messages from the CLI. This is typically
		// set to os.Stderr
		ErrWriter io.Writer
		// Env is a map of environment variable names to their values.
		Env map[string]string
		// Hooks allow you to perform pre and post processing on Commands at various
		// points in their lifecycle.
		Hooks Hooks
	}
	// Hooks is a collection of command hooks.
	Hooks struct {
		// PreExecute is run on a command before it executes.
		PreExecute func(Command) error
	}
)

// Invoke begins invoking the CLI starting with the base command.
func (c *CLI) Invoke(base Command, args []string) {
	result := c.invoke(base, args, []func(*flag.FlagSet){})
	if success, ok := result.(SuccessResult); ok {
		c.handleSuccessResult(success)
	}
	if result == nil {
		result = InternalErrorf(nil, "nil result returned from %T", base)
	}
	if err, ok := result.(ErrorResult); ok {
		c.handleErrorResult(err)
	}

	os.Exit(result.ExitCode())
}

func (c *CLI) PreExecute(command Command) error {
	if c.Hooks.PreExecute == nil {
		return nil
	}
	return c.Hooks.PreExecute(command)
}

func (c *CLI) Info(v ...interface{}) {
	fmt.Fprintln(c.ErrWriter, v...)
}

func (c *CLI) handleSuccessResult(s SuccessResult) {
	if len(s.Data) != 0 {
		c.OutWriter.Write(s.Data)
	}
}

func (c *CLI) handleErrorResult(e ErrorResult) {
	fmt.Fprintln(c.ErrWriter, e.Error())
	if tip := e.UserTip(); len(tip) != 0 {
		fmt.Fprintln(c.ErrWriter, tip)
	}
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
		// add these flags to the agglomeration
		ff = append(ff, command.AddFlags)
		// make a flag.FlagSet named for this command.
		fs := flag.NewFlagSet(name, flag.ContinueOnError)
		// add own and forwarded flags to the flagset, note that it will panic
		// if multiple flags with the same name are added.
		for _, addFlags := range ff {
			addFlags(fs)
		}
		// parse the entire flagset for this command
		if err := fs.Parse(args); err != nil {
			return UsageError{Message: err.Error()}
		}
		args = fs.Args()
	}
	// If this command has subcommands, first try to descend into one of them.
	if command, ok := base.(HasSubcommands); ok && len(args) != 0 {
		subcommandName := args[0]
		args = args[1:]
		subcommands := command.Subcommands()
		if subcommand, ok := subcommands[subcommandName]; ok {
			return c.invoke(subcommand, args, ff)
		}
	}
	// If the command can itself be executed, do that now.
	if command, ok := base.(CanExecute); ok {
		c.PreExecute(base)
		return command.Execute(args)
	}
	// If we get here, this command is not configured correctly and cannot run.
	m := fmt.Sprintf("command %q cannot execute and has no subcommands", name)
	return InternalError{Message: m}
}
