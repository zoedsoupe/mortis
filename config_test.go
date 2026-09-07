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
				{"from": "Carlos", "to": "Alice"},
				{"from": "carl0s_42", "to": "alice_dev"}
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

		if len(cfg.Subs) != 2 || cfg.Subs[0].From != "Carlos" {
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

	t.Run("merges from/to shorthand into substitutions", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		json := `{
			"from": "Carlos",
			"to": "Alice",
			"paths": ["~/notas", "./arquivo.txt"],
			"substitutions": [{"from": "carl0s_42", "to": "alice_dev"}]
		}`
		if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := loadConfig(path)
		if err != nil {
			t.Fatal(err)
		}

		if len(cfg.Subs) != 2 || cfg.Subs[0].From != "Carlos" || cfg.Subs[0].To != "Alice" {
			t.Fatalf("got %+v", cfg.Subs)
		}

		if len(cfg.Paths) != 2 || cfg.Paths[0] != "~/notas" {
			t.Fatalf("got %v", cfg.Paths)
		}
	})
}

func TestSources(t *testing.T) {
	cfg := config{
		Repos: []string{"a/b"},
		URLs:  []string{"https://codeberg.org/c/d.git"},
		Paths: []string{"./notas"},
	}

	sources, err := cfg.sources()
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 3 {
		t.Fatalf("got %v", sources)
	}

	if sources[0].url != "https://github.com/a/b" {
		t.Fatalf("got %q", sources[0].url)
	}

	if sources[1].url != "https://codeberg.org/c/d.git" {
		t.Fatalf("got %q", sources[1].url)
	}

	if sources[2].dir != "./notas" || sources[2].url != "" {
		t.Fatalf("got %+v", sources[2])
	}
}

func TestSubsFromFlags(t *testing.T) {
	t.Run("pairs by index", func(t *testing.T) {
		subs, err := subsFromFlags([]string{"Carlos", "carl0s_42"}, []string{"Alice", "alice_dev"}, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(subs) != 2 || subs[1].From != "carl0s_42" || subs[1].To != "alice_dev" {
			t.Fatalf("got %+v", subs)
		}
	})

	t.Run("from alone requires dry-run", func(t *testing.T) {
		if _, err := subsFromFlags([]string{"Carlos"}, nil, false); err == nil {
			t.Fatal("expected error")
		}

		subs, err := subsFromFlags([]string{"Carlos"}, nil, true)
		if err != nil {
			t.Fatal(err)
		}

		if len(subs) != 1 || subs[0].To != "" {
			t.Fatalf("got %+v", subs)
		}
	})

	t.Run("rejects mismatched counts", func(t *testing.T) {
		if _, err := subsFromFlags([]string{"a", "b"}, []string{"x"}, false); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("to alone errors", func(t *testing.T) {
		if _, err := subsFromFlags(nil, []string{"x"}, false); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty returns nil", func(t *testing.T) {
		subs, err := subsFromFlags(nil, nil, false)
		if err != nil || subs != nil {
			t.Fatalf("got %+v, %v", subs, err)
		}
	})
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	got, err := expandPath("~/notas")
	if err != nil {
		t.Fatal(err)
	}

	if got != filepath.Join(home, "notas") {
		t.Fatalf("got %q", got)
	}

	got, err = expandPath("./notas")
	if err != nil {
		t.Fatal(err)
	}

	if got != "./notas" {
		t.Fatalf("got %q", got)
	}
}
