package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestSearchDir(t *testing.T) {
	write := func(t *testing.T, dir, name string, content []byte) {
		t.Helper()

		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	re := regexp.MustCompile("carlos")

	t.Run("finds matches with line numbers", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "a.txt", []byte("first\nhello carlos\nlast carlos line\n"))

		matches, err := searchDir(dir, re)
		if err != nil {
			t.Fatal(err)
		}

		want := []match{
			{file: "a.txt", line: 2, content: "hello carlos"},
			{file: "a.txt", line: 3, content: "last carlos line"},
		}

		if fmt.Sprint(matches) != fmt.Sprint(want) {
			t.Fatalf("got %v", matches)
		}
	})

	t.Run("skips .git directory", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, ".git/config", []byte("carlos"))

		matches, err := searchDir(dir, re)
		if err != nil {
			t.Fatal(err)
		}

		if len(matches) != 0 {
			t.Fatalf("got %v", matches)
		}
	})

	t.Run("skips binary files", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "bin.dat", []byte{'c', 'a', 'r', 'l', 'o', 's', 0, 'x'})

		matches, err := searchDir(dir, re)
		if err != nil {
			t.Fatal(err)
		}

		if len(matches) != 0 {
			t.Fatalf("got %v", matches)
		}
	})

	t.Run("supports go-only regex syntax", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "a.txt", []byte("Carlos\ncarlos\n"))

		matches, err := searchDir(dir, regexp.MustCompile("(?i)carlos"))
		if err != nil {
			t.Fatal(err)
		}

		if len(matches) != 2 {
			t.Fatalf("got %v", matches)
		}
	})
}

func TestSearchLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello carlos\nuntouched\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := search(source{name: path, dir: path}, regexp.MustCompile("carlos"))
	if res.err != nil {
		t.Fatal(res.err)
	}

	if res.tmp {
		t.Fatal("local file must not be marked as temp")
	}

	want := []match{{file: "a.txt", line: 1, content: "hello carlos"}}
	if fmt.Sprint(res.matches) != fmt.Sprint(want) {
		t.Fatalf("got %v", res.matches)
	}

	if err := apply(res, compileOne(t, "carlos"), []substitution{{From: "carlos", To: "alice"}}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "hello alice\nuntouched\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApply(t *testing.T) {
	dir := t.TempDir()

	content := "hello carlos, bye carlos\nuntouched line\n"
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	res := result{
		dir: dir,
		matches: []match{
			{file: "a.txt", line: 1, content: "hello carlos, bye carlos"},
		},
	}

	subs := []substitution{{From: "carlos", To: "alice"}}
	if err := apply(res, compileOne(t, "carlos"), subs); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}

	want := "hello alice, bye alice\nuntouched line\n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	info, err := os.Stat(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode changed: %v", info.Mode().Perm())
	}
}

func TestApplyCaptureGroup(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("old_name\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := result{
		dir:     dir,
		matches: []match{{file: "a.txt", line: 1, content: "old_name"}},
	}

	subs := []substitution{{From: "old_(\\w+)", To: "new_$1"}}
	if err := apply(res, compileOne(t, "old_(\\w+)"), subs); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "new_name\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyMultiplePairs(t *testing.T) {
	dir := t.TempDir()

	content := "Copyright Carlos <carlos@x.com> github.com/carl0s_42\n"
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res := result{
		dir:     dir,
		matches: []match{{file: "a.txt", line: 1, content: content}},
	}

	subs := []substitution{
		{From: "Carlos", To: "Alice"},
		{From: "carlos@x.com", To: "alice@y.com"},
		{From: "carl0s_42", To: "alice_dev"},
	}

	regexes, err := compileSubs(subs)
	if err != nil {
		t.Fatal(err)
	}

	if err := apply(res, regexes, subs); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}

	want := "Copyright Alice <alice@y.com> github.com/alice_dev\n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
