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
	fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
	return 1
}

// ParseFlagsAndArgs reorders arguments so flags precede positional arguments,
// enabling users to place flags before or after positional arguments.
func ParseFlagsAndArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var flagArgs []string
	var posArgs []string

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
		if flagNeedsValue(fs, arg) && i+1 < count {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}

	var reordered []string
	reordered = append(reordered, flagArgs...)
	if len(posArgs) > 0 {
		reordered = append(reordered, "--")
		reordered = append(reordered, posArgs...)
	}

	if err := fs.Parse(reordered); err != nil {
		return nil, err
	}

	return posArgs, nil
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
