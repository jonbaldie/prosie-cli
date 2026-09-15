package command

import "fmt"

func Dispatch(c *Environment, args []string, commands map[string]Handler, help func(*Environment), unknown string) int {
	if len(args) == 0 {
		help(c)
		return 0
	}
	switch args[0] {
	case "help", "--help", "-h":
		help(c)
		return 0
	}
	if execute, ok := commands[args[0]]; ok {
		return execute(c, args[1:])
	}
	fmt.Fprintf(c.Err, unknown, args[0])
	return 1
}

// Handler is one command family entry point.
type Handler func(*Environment, []string) int
