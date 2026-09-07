package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRepos(t *testing.T) {
	t.Run("trims and splits", func(t *testing.T) {
		repos, err := parseRepos(" a/b , c/d ")
		if err != nil {
			t.Fatal(err)
		}

		if len(repos) != 2 || repos[0] != "a/b" || repos[1] != "c/d" {
			t.Fatalf("got %v", repos)
		}
	})

	t.Run("dedupes", func(t *testing.T) {
		repos, err := parseRepos("a/b,a/b")
		if err != nil {
			t.Fatal(err)
		}

		if len(repos) != 1 {
			t.Fatalf("got %v", repos)
		}
	})

	t.Run("rejects missing owner", func(t *testing.T) {
		if _, err := parseRepos("noslash"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects empty", func(t *testing.T) {
		if _, err := parseRepos(" , "); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParseGrep(t *testing.T) {
	t.Run("parses lines", func(t *testing.T) {
		matches, err := parseGrep("a.go:10:foo: bar\nb.go:2:x\n")
		if err != nil {
			t.Fatal(err)
		}

		if len(matches) != 2 {
			t.Fatalf("got %v", matches)
		}

		if matches[0] != (match{file: "a.go", line: 10, content: "foo: bar"}) {
			t.Fatalf("got %+v", matches[0])
		}
	})

	t.Run("rejects malformed line", func(t *testing.T) {
		if _, err := parseGrep("not-a-grep-line\n"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects bad line number", func(t *testing.T) {
		if _, err := parseGrep("a.go:x:content\n"); err == nil {
			t.Fatal("expected error")
		}
	})
}

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

func TestApply(t *testing.T) {
	dir := t.TempDir()

	content := "hello carlos, bye carlos\nuntouched line\n"
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	res := repoResult{
		dir: dir,
		matches: []match{
			{file: "a.txt", line: 1, content: "hello carlos, bye carlos"},
		},
	}

	re, err := maybeRegex("carlos")
	if err != nil {
		t.Fatal(err)
	}

	if err := apply(res, re, "zoey"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}

	want := "hello zoey, bye zoey\nuntouched line\n"
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

	res := repoResult{
		dir:     dir,
		matches: []match{{file: "a.txt", line: 1, content: "old_name"}},
	}

	re, err := maybeRegex("old_(\\w+)")
	if err != nil {
		t.Fatal(err)
	}

	if err := apply(res, re, "new_$1"); err != nil {
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
