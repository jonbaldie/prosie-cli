package command

import "strconv"

// JSONID converts a resource ID argument to an integer for JSON output, so
// that IDs keep the numeric type the API uses. A non-numeric ID stays a string.
func JSONID(id string) any {
	if intVal, err := strconv.Atoi(id); err == nil {
		return intVal
	}
	return id
}
