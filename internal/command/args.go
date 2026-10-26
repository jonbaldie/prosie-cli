package command

import (
	"flag"
	"fmt"
	"strings"
)

// flagError keeps help requests successful and parser errors on standard error.
func FlagError(c *Environment, err error, help func(*Environment)) int {
	if err == flag.ErrHelp {
		help(c)
		return 0
	}
	if _, ok := err.(argumentCountError); ok {
		fmt.Fprintf(c.Err, "error: %v\n", err)
		return 1
	}
	fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
	return 1
}

// Args declares the positional limits and the usage for a command.
// Max is -1 for an unbounded number of arguments.
type Args struct {
	Min     int
	Max     int
	Usage   string
	Missing string
	// RedactUnexpected keeps possible credentials out of error output.
	RedactUnexpected bool
}

type argumentCountError string

func (e argumentCountError) Error() string { return string(e) }

// Validate checks the positional count after flags have been parsed.
func (a Args) Validate(args []string) error {
	if len(args) < a.Min {
		return argumentCountError(fmt.Sprintf("%s. Usage: %s", a.Missing, a.Usage))
	}
	if a.Max >= 0 && len(args) > a.Max {
		if a.RedactUnexpected {
			return argumentCountError(fmt.Sprintf("unexpected argument. Usage: %s", a.Usage))
		}
		return argumentCountError(fmt.Sprintf("unexpected argument %q. Usage: %s", args[a.Max], a.Usage))
	}
	return nil
}

// ParseFlagsAndArgs reorders arguments so flags precede positional arguments,
// enabling users to place flags before or after positional arguments.
func ParseFlagsAndArgs(fs *flag.FlagSet, args []string, limits Args) ([]string, error) {
	flagArgs, posArgs, valueMissing := splitArgs(fs, args)

	var reordered []string
	reordered = append(reordered, flagArgs...)
	// A flag without its value must stay last, so flag.FlagSet reports
	// "flag needs an argument". An injected "--" becomes its value.
	if len(posArgs) > 0 && !valueMissing {
		reordered = append(reordered, "--")
		reordered = append(reordered, posArgs...)
	}

	if err := fs.Parse(reordered); err != nil {
		return nil, err
	}

	if err := limits.Validate(posArgs); err != nil {
		return nil, err
	}
	return posArgs, nil
}

// splitArgs separates flags with their values from positional arguments.
// valueMissing is true when the last argument is a flag without its value.
func splitArgs(fs *flag.FlagSet, args []string) (flagArgs, posArgs []string, valueMissing bool) {
	count := len(args)
	for i := 0; i < count; i++ {
		arg := args[i]
		if arg == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			posArgs = append(posArgs, arg)
			continue
		}
		flagArgs = append(flagArgs, arg)
		if !flagNeedsValue(fs, arg) {
			continue
		}
		if i+1 == count {
			return flagArgs, posArgs, true
		}
		i++
		flagArgs = append(flagArgs, args[i])
	}

	return flagArgs, posArgs, false
}

func flagNeedsValue(fs *flag.FlagSet, arg string) bool {
	name := strings.TrimLeft(arg, "-")
	if name == "" || strings.Contains(name, "=") {
		return false
	}
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	boolean, ok := f.Value.(interface{ IsBoolFlag() bool })
	return !ok || !boolean.IsBoolFlag()
}
