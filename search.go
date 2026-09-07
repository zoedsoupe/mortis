package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sync/errgroup"
)

type source struct {
	name string
	url  string // remote git; vazio para caminhos locais
	dir  string // caminho local (arquivo ou pasta)
}

type match struct {
	file    string
	line    int
	content string
}

type result struct {
	name    string
	dir     string
	tmp     bool // dir é clone temporário
	matches []match
	err     error
}

func collect(sources []source, re *regexp.Regexp) []result {
	results := make([]result, len(sources))

	var g errgroup.Group
	for i, src := range sources {
		g.Go(func() error {
			results[i] = search(src, re)
			return nil
		})
	}
	_ = g.Wait()

	return results
}

func search(src source, re *regexp.Regexp) result {
	res := result{name: src.name}

	if src.url != "" {
		dir, err := os.MkdirTemp("", "mortis-*")
		if err != nil {
			res.err = err
			return res
		}
		res.dir, res.tmp = dir, true

		clone := exec.Command("git", "clone", "--depth", "1", src.url, dir)
		if out, err := clone.CombinedOutput(); err != nil {
			res.err = fmt.Errorf("clone: %w: %s", err, out)
			return res
		}

		res.matches, res.err = searchDir(dir, re)
		return res
	}

	info, err := os.Stat(src.dir)
	if err != nil {
		res.err = err
		return res
	}

	if info.IsDir() {
		res.dir = src.dir
		res.matches, res.err = searchDir(src.dir, re)
		return res
	}

	res.dir = filepath.Dir(src.dir)
	res.matches, res.err = searchFile(res.dir, src.dir, re)

	return res
}

func searchDir(dir string, re *regexp.Regexp) ([]match, error) {
	var matches []match

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		found, err := searchFile(dir, path, re)
		if err != nil {
			return err
		}
		matches = append(matches, found...)

		return nil
	})

	return matches, err
}

func searchFile(dir, path string, re *regexp.Regexp) ([]match, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if bytes.IndexByte(content, 0) >= 0 {
		return nil, nil
	}

	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return nil, err
	}

	var matches []match
	for i, line := range strings.Split(string(content), "\n") {
		if re.MatchString(line) {
			matches = append(matches, match{file: rel, line: i + 1, content: line})
		}
	}

	return matches, nil
}

func apply(res result, regexes []*regexp.Regexp, subs []substitution) error {
	seen := make(map[string]bool)

	for _, m := range res.matches {
		if seen[m.file] {
			continue
		}
		seen[m.file] = true

		path := filepath.Join(res.dir, m.file)

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%s: %w", m.file, err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", m.file, err)
		}

		replaced := replaceBytes(content, regexes, subs)
		if err := os.WriteFile(path, replaced, info.Mode()); err != nil {
			return fmt.Errorf("%s: %w", m.file, err)
		}
	}

	return nil
}
