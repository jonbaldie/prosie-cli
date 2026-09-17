package command

import (
	"bufio"
	"fmt"
	"strings"
)

func ConfirmDeletion(c *Environment, yes bool, resource string, id any) bool {
	if yes {
		return true
	}
	fmt.Fprintf(c.Err, "Are you sure you want to delete %s %v? [y/N]: ", resource, id)
	scanner := bufio.NewScanner(c.In)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "y" || answer == "yes" {
			return true
		}
	}
	fmt.Fprintln(c.Err, "Deletion cancelled.")
	return false
}
