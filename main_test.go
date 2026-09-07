package main

import (
	"fmt"
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
		write(t, dir, "bin.dat", []byte{'m', 'a', 't', 'h', 'e', 'u', 's', 0, 'x'})

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
		write(t, dir, "a.txt", []byte("carlos\ncarlos\n"))

		matches, err := searchDir(dir, regexp.MustCompile("(?i)carlos"))
		if err != nil {
			t.Fatal(err)
		}

		if len(matches) != 2 {
			t.Fatalf("got %v", matches)
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
