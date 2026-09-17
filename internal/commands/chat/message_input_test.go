package chatcmd

import "testing"

func TestMessageInput_ContinuingConversation_MultiWordMessage(t *testing.T) {
	_, message, err := messageInput([]string{"hello", "world"}, "12", "send")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if message != "hello world" {
		t.Fatalf("message = %q, want %q", message, "hello world")
	}
}

func TestMessageInput_ContinuingConversation_SingleWordMessage(t *testing.T) {
	_, message, err := messageInput([]string{"hello"}, "12", "send")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if message != "hello" {
		t.Fatalf("message = %q, want %q", message, "hello")
	}
}

func TestMessageInput_NewConversation_MultiWordMessage(t *testing.T) {
	bookID, message, err := messageInput([]string{"42", "hello", "world"}, "", "send")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bookID != "42" {
		t.Fatalf("bookID = %q, want %q", bookID, "42")
	}
	if message != "hello world" {
		t.Fatalf("message = %q, want %q", message, "hello world")
	}
}
