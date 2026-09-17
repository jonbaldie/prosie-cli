package command

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmDeletion(t *testing.T) {
	t.Run("yes flag skips prompt", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		env := &Environment{
			In:  bytes.NewBuffer(nil),
			Out: out,
			Err: errOut,
		}

		confirmed := ConfirmDeletion(env, true, "book", 1)
		if !confirmed {
			t.Fatalf("expected confirmed to be true")
		}
		if out.Len() != 0 {
			t.Fatalf("expected stdout to be empty, got: %q", out.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("expected stderr to be empty, got: %q", errOut.String())
		}
	})

	t.Run("prompt writes to stderr and confirms with y", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		env := &Environment{
			In:  bytes.NewBufferString("y\n"),
			Out: out,
			Err: errOut,
		}

		confirmed := ConfirmDeletion(env, false, "book", 1)
		if !confirmed {
			t.Fatalf("expected confirmed to be true")
		}
		if out.Len() != 0 {
			t.Fatalf("expected stdout to be empty, got: %q", out.String())
		}
		expectedPrompt := "Are you sure you want to delete book 1? [y/N]: "
		if errOut.String() != expectedPrompt {
			t.Fatalf("expected stderr to be %q, got: %q", expectedPrompt, errOut.String())
		}
	})

	t.Run("prompt writes to stderr and confirms with yes", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		env := &Environment{
			In:  bytes.NewBufferString("yes\n"),
			Out: out,
			Err: errOut,
		}

		confirmed := ConfirmDeletion(env, false, "chapter", "abc")
		if !confirmed {
			t.Fatalf("expected confirmed to be true")
		}
		if out.Len() != 0 {
			t.Fatalf("expected stdout to be empty, got: %q", out.String())
		}
		expectedPrompt := "Are you sure you want to delete chapter abc? [y/N]: "
		if errOut.String() != expectedPrompt {
			t.Fatalf("expected stderr to be %q, got: %q", expectedPrompt, errOut.String())
		}
	})

	t.Run("prompt writes to stderr and cancels with n", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		env := &Environment{
			In:  bytes.NewBufferString("n\n"),
			Out: out,
			Err: errOut,
		}

		confirmed := ConfirmDeletion(env, false, "book", 1)
		if confirmed {
			t.Fatalf("expected confirmed to be false")
		}
		if out.Len() != 0 {
			t.Fatalf("expected stdout to be empty, got: %q", out.String())
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete book 1? [y/N]: ") {
			t.Fatalf("expected prompt in stderr, got: %q", errOut.String())
		}
		if !strings.Contains(errOut.String(), "Deletion cancelled.") {
			t.Fatalf("expected cancellation in stderr, got: %q", errOut.String())
		}
	})

	t.Run("prompt writes to stderr and cancels on empty input", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		env := &Environment{
			In:  bytes.NewBufferString("\n"),
			Out: out,
			Err: errOut,
		}

		confirmed := ConfirmDeletion(env, false, "book", 1)
		if confirmed {
			t.Fatalf("expected confirmed to be false")
		}
		if out.Len() != 0 {
			t.Fatalf("expected stdout to be empty, got: %q", out.String())
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete book 1? [y/N]: ") {
			t.Fatalf("expected prompt in stderr, got: %q", errOut.String())
		}
		if !strings.Contains(errOut.String(), "Deletion cancelled.") {
			t.Fatalf("expected cancellation in stderr, got: %q", errOut.String())
		}
	})

	t.Run("prompt writes to stderr and cancels on EOF", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		env := &Environment{
			In:  bytes.NewBuffer(nil),
			Out: out,
			Err: errOut,
		}

		confirmed := ConfirmDeletion(env, false, "book", 1)
		if confirmed {
			t.Fatalf("expected confirmed to be false")
		}
		if out.Len() != 0 {
			t.Fatalf("expected stdout to be empty, got: %q", out.String())
		}
		if !strings.Contains(errOut.String(), "Are you sure you want to delete book 1? [y/N]: ") {
			t.Fatalf("expected prompt in stderr, got: %q", errOut.String())
		}
		if !strings.Contains(errOut.String(), "Deletion cancelled.") {
			t.Fatalf("expected cancellation in stderr, got: %q", errOut.String())
		}
	})
}
