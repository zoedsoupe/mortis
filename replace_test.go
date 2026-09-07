package main

import (
	"regexp"
	"testing"
)

func TestMaybeRegex(t *testing.T) {
	t.Run("compiles valid regex", func(t *testing.T) {
		r, err := maybeRegex("fo+")
		if err != nil {
			t.Fatal(err)
		}

		if r == nil {
			t.Fatal("expected compiled regex")
		}
	})

	t.Run("rejects invalid regex", func(t *testing.T) {
		if _, err := maybeRegex("(["); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestMaskLine(t *testing.T) {
	union := regexp.MustCompile("(carlos)|(carl0s_42)")

	got := maskLine("hi carlos aka carl0s_42", union)
	want := "hi " + red.Render("[nome antigo]") + " aka " + red.Render("[nome antigo]")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCompileSubs(t *testing.T) {
	t.Run("rejects empty", func(t *testing.T) {
		if _, err := compileSubs(nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects invalid regex", func(t *testing.T) {
		if _, err := compileSubs([]substitution{{From: "(["}}); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestUnionPattern(t *testing.T) {
	subs := []substitution{{From: "a+"}, {From: "b|c"}}

	got := unionPattern(subs)
	want := "(a+)|(b|c)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReplaceOrder(t *testing.T) {
	subs := []substitution{
		{From: "foo", To: "bar"},
		{From: "bar", To: "baz"},
	}

	regexes, err := compileSubs(subs)
	if err != nil {
		t.Fatal(err)
	}

	got := replaceString("foo", regexes, subs)
	if got != "baz" {
		t.Fatalf("got %q, pairs must apply in order", got)
	}
}

func compileOne(t *testing.T, from string) []*regexp.Regexp {
	t.Helper()

	regexes, err := compileSubs([]substitution{{From: from}})
	if err != nil {
		t.Fatal(err)
	}

	return regexes
}
