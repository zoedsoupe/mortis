package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type substitution struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// strList collects repeated flag occurrences, e.g. -from a -from b.
type strList []string

func (s *strList) String() string { return strings.Join(*s, ",") }

func (s *strList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// subsFromFlags pairs -from/-to flag values by index. With dryRun, -to may be omitted.
func subsFromFlags(froms, tos []string, dryRun bool) ([]substitution, error) {
	if len(froms) == 0 {
		if len(tos) > 0 {
			return nil, errors.New("flag -to requer -from")
		}
		return nil, nil
	}

	if len(tos) == 0 {
		if !dryRun {
			return nil, errors.New("flag -from requer -to (exceto com -dry-run)")
		}
	} else if len(tos) != len(froms) {
		return nil, fmt.Errorf("-from e -to precisam aparecer na mesma quantidade (%d vs %d)", len(froms), len(tos))
	}

	subs := make([]substitution, len(froms))
	for i, from := range froms {
		if err := validRegex(from); err != nil {
			return nil, err
		}
		subs[i].From = from
		if len(tos) > 0 {
			subs[i].To = tos[i]
		}
	}

	return subs, nil
}

type config struct {
	From  string         `json:"from"`
	To    string         `json:"to"`
	Repos []string       `json:"repos"`
	URLs  []string       `json:"urls"`
	Paths []string       `json:"paths"`
	Subs  []substitution `json:"substitutions"`
}

// normalize merges the from/to shorthand into Subs and cleans the source lists.
func (c *config) normalize() error {
	if c.From != "" || c.To != "" {
		c.Subs = append([]substitution{{From: c.From, To: c.To}}, c.Subs...)
		c.From, c.To = "", ""
	}

	repos, err := validateRepos(c.Repos)
	if err != nil {
		return err
	}
	c.Repos = repos
	c.URLs = cleanList(c.URLs)
	c.Paths = cleanList(c.Paths)

	return nil
}

func (c config) sources() ([]source, error) {
	var sources []source

	for _, r := range c.Repos {
		sources = append(sources, source{name: r, url: "https://github.com/" + r})
	}

	for _, u := range c.URLs {
		sources = append(sources, source{name: u, url: u})
	}

	for _, p := range c.Paths {
		dir, err := expandPath(p)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source{name: p, dir: dir})
	}

	return sources, nil
}

func loadConfig(path string) (config, error) {
	var cfg config

	if path == "" {
		return cfg, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("config inválida: %w", err)
	}

	if err := cfg.normalize(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func parseRepos(input string) ([]string, error) {
	repos, err := validateRepos(strings.Split(input, ","))
	if err != nil {
		return nil, err
	}

	if len(repos) == 0 {
		return nil, errors.New("nenhum repositório informado")
	}

	return repos, nil
}

func validateRepos(fields []string) ([]string, error) {
	var repos []string

	for _, r := range cleanList(fields) {
		if !strings.Contains(r, "/") {
			return nil, fmt.Errorf("repo %q inválido, formato esperado: dono/nome", r)
		}
		repos = append(repos, r)
	}

	return repos, nil
}

func parseList(input string) []string {
	return cleanList(strings.Split(input, ","))
}

func cleanList(fields []string) []string {
	seen := make(map[string]bool)
	var out []string

	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}

	return out
}

func expandPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
	}

	return path, nil
}
