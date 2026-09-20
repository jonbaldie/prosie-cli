package command

import "testing"

func TestJSONID(t *testing.T) {
	t.Run("numeric id becomes an integer", func(t *testing.T) {
		if got := JSONID("901"); got != 901 {
			t.Fatalf("expected 901, got %#v", got)
		}
	})

	t.Run("non-numeric id stays a string", func(t *testing.T) {
		if got := JSONID("abc"); got != "abc" {
			t.Fatalf("expected \"abc\", got %#v", got)
		}
	})
}
