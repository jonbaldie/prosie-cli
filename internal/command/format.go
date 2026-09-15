package command

import "time"

func FormatTimestamp(raw string) string {
	if raw == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000000Z", raw)
	}
	if err != nil {
		return raw
	}
	return t.Format("2006-01-02 15:04")
}
