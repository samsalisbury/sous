package cli

import (
	"flag"
	"fmt"
	"os"
)

// Invoke begins invoking the CLI starting with the base command.
func Invoke(base Command, args, env []string) Result {
	return invoke(base, args, env, []func(*flag.FlagSet){})
}

// invoke invokes this command, resolving subcommands and flags, and forwarding
// flags from lower commands to higher ones. This means that flags defined on
// the base command are also defined by default all its nested subcommands,
// which is usually a nicer user experience than having to remember strictly
// which subcommand a flag is applicable to. The ff parameter deals with these
// flags.
func invoke(base Command, args, env []string, ff []func(*flag.FlagSet)) Result {
	name := args[0]
	args = args[1:]
	// Add and parse flags for this command.
	if command, ok := base.(AddsFlags); ok {
		// add these flags to the agglomeration
		ff = append(ff, command.AddFlags)
		// make a flag.FlagSet named for this command.
		fs := flag.NewFlagSet(name, flag.ContinueOnError)
		command.AddFlags(fs)
		// add forwarded flags to the flagset, note that it will panic if
		// multiple flags with the same name are added.
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
	if command, ok := base.(HasSubcommand); ok && len(args) != 0 {
		subcommandName := args[0]
		args = args[1:]
		subcommand, err := command.Subcommand(subcommandName)
		if err != nil {
			if result, ok := err.(Result); ok {
				return result
			}
			return UnknownError{Err: err}
		}
		if subcommand != nil {
			// The point of no return, the subcommand will handle everything
			// else from here...
			return invoke(subcommand, args, env, ff)
		}
	}
	// If the command can itself be executed, do that now.
	if command, ok := base.(CanExecute); ok {
		result := command.Execute()
		if err, isErr := result.(error); isErr {
			fmt.Fprintln(os.Stderr, err)
		}
		if tipper, isTipper := result.(Tipper); isTipper {
			tip := tipper.UserTip()
			if tip != "" {
				fmt.Fprintln(os.Stderr, "TIP:", tip)
			}
		}
		if success, isSuccess := result.(SuccessResult); isSuccess {
			if len(success.Data) != 0 {
				os.Stdout.Write(success.Data)
			}
		}
		os.Exit(result.ExitCode())
	}
	// If we get here, this command is not configured correctly and cannot run.
	m := fmt.Sprintf("command %q is not correctly configured", name)
	return InternalError{Message: m}
}
