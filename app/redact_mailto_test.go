package app

import (
	"slices"
	"testing"
)

func TestRedactMailtoArg(t *testing.T) {
	args := []string{"--composer", "--account", "a1", "--mailto", "mailto:x@example.com?body=secret"}
	got := redactMailtoArg(args)
	want := []string{"--composer", "--account", "a1", "--mailto", "[redacted]"}
	if !slices.Equal(got, want) {
		t.Errorf("redactMailtoArg = %q, want %q", got, want)
	}
	if args[4] != "mailto:x@example.com?body=secret" {
		t.Error("redactMailtoArg modified its input")
	}
	if got := redactMailtoArg([]string{"--mailto"}); !slices.Equal(got, []string{"--mailto"}) {
		t.Errorf("trailing flag: got %q", got)
	}
}
