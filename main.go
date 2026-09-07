package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"golang.org/x/sync/errgroup"
)

type config struct {
	dead  string
	alive string
	repos string
}

type match struct {
	file    string
	line    int
	content string
}

type repoResult struct {
	repo    string
	dir     string
	matches []match
	err     error
}

func main() {
	var cfg config

	if err := buildForm(&cfg).Run(); err != nil {
		log.Fatalln("deu ruim ao rodar o formulário")
	}

	repos, err := parseRepos(cfg.repos)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("clonando e buscando em %d repositórios...\n", len(repos))

	results := collect(repos, cfg.dead)

	fmt.Println("\nresultado:")
	for _, res := range results {
		defer os.RemoveAll(res.dir)

		if res.err != nil || len(res.matches) == 0 {
			continue
		}

		fmt.Printf("%s:\n", res.repo)
		for _, m := range res.matches {
			fmt.Printf("  %s:%d: %s\n", m.file, m.line, m.content)
		}
	}
}

func buildForm(cfg *config) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("nome morto/antigo").
				Validate(validRegex).
				Value(&cfg.dead).
				Description("golang regex"),
			huh.NewInput().
				Title("nome correto").
				Validate(required).
				Value(&cfg.alive),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("repositorios github").
				Description("(formato: dono/nome,dono/nome2)").
				Validate(validRepos).
				Value(&cfg.repos),
		),
	)
}

func required(str string) error {
	if len(str) < 1 {
		return errors.New("precisa preencher esse campo!")
	}

	return nil
}

func validRegex(str string) error {
	if err := required(str); err != nil {
		return err
	}

	if _, err := maybeRegex(str); err != nil {
		return fmt.Errorf("regex inválida: %w", err)
	}

	return nil
}

func validRepos(str string) error {
	if err := required(str); err != nil {
		return err
	}

	_, err := parseRepos(str)

	return err
}

func maybeRegex(str string) (*regexp.Regexp, error) {
	r, err := regexp.Compile(str)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func parseRepos(input string) ([]string, error) {
	seen := make(map[string]bool)
	var repos []string

	for r := range strings.SplitSeq(input, ",") {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}

		if !strings.Contains(r, "/") {
			return nil, fmt.Errorf("repo %q inválido, formato esperado: dono/nome", r)
		}

		if !seen[r] {
			seen[r] = true
			repos = append(repos, r)
		}
	}

	if len(repos) == 0 {
		return nil, errors.New("nenhum repositório informado")
	}

	return repos, nil
}

func collect(repos []string, pattern string) []repoResult {
	results := make([]repoResult, len(repos))

	var g errgroup.Group
	for i, repo := range repos {
		g.Go(func() error {
			results[i] = searchRepo(repo, pattern)
			return nil
		})
	}
	_ = g.Wait()

	return results
}

func searchRepo(repo, pattern string) repoResult {
	res := repoResult{repo: repo}

	dir, err := os.MkdirTemp("", "mortis-*")
	if err != nil {
		res.err = err
		return res
	}
	res.dir = dir

	fmt.Printf("[%s] clonando...\n", repo)
	clone := exec.Command("git", "clone", "--depth", "1", "https://github.com/"+repo, dir)
	if out, err := clone.CombinedOutput(); err != nil {
		res.err = fmt.Errorf("clone: %w: %s", err, out)
		fmt.Printf("[%s] erro no clone: %v\n", repo, res.err)
		return res
	}

	fmt.Printf("[%s] buscando %q...\n", repo, pattern)
	grep := exec.Command("git", "grep", "-n", "-E", pattern)
	grep.Dir = dir
	out, err := grep.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			fmt.Printf("[%s] nenhuma ocorrência\n", repo)
			return res
		}

		res.err = fmt.Errorf("git grep: %w", err)
		fmt.Printf("[%s] erro na busca: %v\n", repo, res.err)
		return res
	}

	res.matches, res.err = parseGrep(string(out))
	if res.err != nil {
		fmt.Printf("[%s] erro ao ler resultado: %v\n", repo, res.err)
		return res
	}

	fmt.Printf("[%s] %d ocorrências encontradas\n", repo, len(res.matches))

	return res
}

func parseGrep(output string) ([]match, error) {
	var matches []match

	for line := range strings.Lines(output) {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("linha de grep inesperada: %q", line)
		}

		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("linha de grep inesperada: %q", line)
		}

		matches = append(matches, match{file: parts[0], line: n, content: parts[2]})
	}

	return matches, nil
}
