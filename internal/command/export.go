package command

import (
	"fmt"
	"os"
)

func WriteExportFile(c *Environment, label string, id any, output string, data []byte, jsonOutput bool) int {
	if err := os.WriteFile(output, data, 0644); err != nil {
		fmt.Fprintf(c.Err, "error writing to file: %v\n", err)
		return 1
	}
	if jsonOutput {
		_ = WriteJSON(c, map[string]any{"id": id, "output": output, "bytes": len(data)})
		return 0
	}
	fmt.Fprintf(c.Out, "Exported %s to %s (%d bytes).\n", label, output, len(data))
	return 0
}

func ExportProse(c *Environment, label string, id any, output string, data []byte, jsonOutput bool) int {
	if output != "" {
		return WriteExportFile(c, label, id, output, data, jsonOutput)
	}
	if jsonOutput {
		_ = WriteJSON(c, map[string]any{"id": id, "content": string(data), "bytes": len(data)})
		return 0
	}
	_, _ = c.Out.Write(data)
	return 0
}
