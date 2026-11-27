package client

import "testing"

func TestConversationDisplayTitle(t *testing.T) {
	cases := []struct {
		name string
		c    Conversation
		want string
	}{
		{"titled", Conversation{ID: 3, Title: strptr("My Chat")}, "My Chat"},
		{"nil title", Conversation{ID: 3}, "Conversation 3"},
		{"empty title", Conversation{ID: 3, Title: strptr("")}, "Conversation 3"},
	}
	for _, tc := range cases {
		if got := tc.c.DisplayTitle(); got != tc.want {
			t.Fatalf("%s: DisplayTitle() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestContinueParamsBody(t *testing.T) {
	body := ContinueParams{Persist: true, Instruction: "Go", Length: "long", WordTarget: 500, LineLimit: 42}.body()
	want := map[string]any{
		"persist": true, "instruction": "Go", "length": "long", "word_target": 500, "line_limit": 42,
	}
	if len(body) != len(want) {
		t.Fatalf("body has %d keys, want %d: %v", len(body), len(want), body)
	}
	for k, v := range want {
		if body[k] != v {
			t.Fatalf("body[%q] = %v, want %v", k, body[k], v)
		}
	}
	empty := ContinueParams{}.body()
	if len(empty) != 1 {
		t.Fatalf("empty body should have only persist, got %v", empty)
	}
}
