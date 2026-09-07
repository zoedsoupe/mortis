package main

import (
	"os"
	"path/filepath"
	"regexp"
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

func TestLoadConfig(t *testing.T) {
	t.Run("empty path returns zero config", func(t *testing.T) {
		cfg, err := loadConfig("")
		if err != nil {
			t.Fatal(err)
		}

		if len(cfg.Repos) != 0 || len(cfg.Subs) != 0 {
			t.Fatalf("got %+v", cfg)
		}
	})

	t.Run("parses json and dedupes repos", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		json := `{
			"repos": ["a/b", "a/b", "c/d"],
			"substitutions": [
				{"dead": "carlos", "alive": "Zoey"},
				{"dead": "carl0s_42", "alive": "zoedsoupe"}
			]
		}`
		if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := loadConfig(path)
		if err != nil {
			t.Fatal(err)
		}

		if len(cfg.Repos) != 2 {
			t.Fatalf("got %v", cfg.Repos)
		}

		if len(cfg.Subs) != 2 || cfg.Subs[0].Dead != "carlos" {
			t.Fatalf("got %+v", cfg.Subs)
		}
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}

		if _, err := loadConfig(path); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCompileSubs(t *testing.T) {
	t.Run("rejects empty", func(t *testing.T) {
		if _, err := compileSubs(nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects invalid regex", func(t *testing.T) {
		if _, err := compileSubs([]substitution{{Dead: "(["}}); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestUnionPattern(t *testing.T) {
	subs := []substitution{{Dead: "a+"}, {Dead: "b|c"}}

	got := unionPattern(subs)
	want := "(a+)|(b|c)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReplaceOrder(t *testing.T) {
	subs := []substitution{
		{Dead: "foo", Alive: "bar"},
		{Dead: "bar", Alive: "baz"},
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

func compileOne(t *testing.T, dead string) []*regexp.Regexp {
	t.Helper()

	regexes, err := compileSubs([]substitution{{Dead: dead}})
	if err != nil {
		t.Fatal(err)
	}

	return regexes
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

	subs := []substitution{{Dead: "carlos", Alive: "zoey"}}
	if err := apply(res, compileOne(t, "carlos"), subs); err != nil {
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

	subs := []substitution{{Dead: "old_(\\w+)", Alive: "new_$1"}}
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

	content := "Copyright carlos <carlos@x.com> github.com/carl0s_42\n"
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res := repoResult{
		dir:     dir,
		matches: []match{{file: "a.txt", line: 1, content: content}},
	}

	subs := []substitution{
		{Dead: "carlos", Alive: "Zoey"},
		{Dead: "carlos@x.com", Alive: "zoey@y.com"},
		{Dead: "carl0s_42", Alive: "zoedsoupe"},
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

	want := "Copyright Zoey <zoey@y.com> github.com/zoedsoupe\n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
