package cmd

import (
	"flag"
	"strings"
)

// parseFlagsAndArgs reorders arguments so flags precede positional arguments,
// enabling users to place flags before or after positional arguments.
func parseFlagsAndArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var flagArgs []string
	var posArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "-") {
			name := strings.TrimLeft(arg, "-")
			hasEqual := strings.Contains(name, "=")
			if hasEqual {
				parts := strings.SplitN(name, "=", 2)
				name = parts[0]
			}

			f := fs.Lookup(name)
			if f != nil {
				isBool := false
				if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
					isBool = true
				}

				if isBool || hasEqual {
					flagArgs = append(flagArgs, arg)
				} else if i+1 < len(args) {
					flagArgs = append(flagArgs, arg, args[i+1])
					i++
				} else {
					flagArgs = append(flagArgs, arg)
				}
			} else {
				flagArgs = append(flagArgs, arg)
			}
		} else {
			posArgs = append(posArgs, arg)
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

	return fs.Args(), nil
}
