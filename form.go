package main

import (
	"errors"
	"strings"

	"charm.land/huh/v2"
)

func fillConfig(cfg *config) error {
	var groups []*huh.Group
	var reposInput, pathsInput, fromInput, toInput string

	noSources := len(cfg.Repos) == 0 && len(cfg.URLs) == 0 && len(cfg.Paths) == 0
	if noSources {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title("repositórios github (opcional)").
				Description("(formato: dono/nome,dono/nome2)").
				Validate(optionalRepos).
				Value(&reposInput),
			huh.NewInput().
				Title("arquivos/pastas locais (opcional)").
				Description("(separados por vírgula)").
				Value(&pathsInput),
		))
	}

	if len(cfg.Subs) == 0 {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title("nome antigo").
				Validate(validRegex).
				Value(&fromInput).
				Description("golang regex"),
			huh.NewInput().
				Title("nome novo").
				Validate(required).
				Value(&toInput),
		))
	}

	if len(groups) == 0 {
		return nil
	}

	if err := huh.NewForm(groups...).Run(); err != nil {
		return err
	}

	if noSources {
		var repos []string
		if strings.TrimSpace(reposInput) != "" {
			var err error
			repos, err = parseRepos(reposInput)
			if err != nil {
				return err
			}
		}

		paths := parseList(pathsInput)

		if len(repos) == 0 && len(paths) == 0 {
			return errors.New("informe ao menos um repositório ou caminho local")
		}

		cfg.Repos = repos
		cfg.Paths = paths
	}

	if len(cfg.Subs) == 0 {
		cfg.Subs = []substitution{{From: fromInput, To: toInput}}
	}

	return nil
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
		return err
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

func optionalRepos(str string) error {
	if strings.TrimSpace(str) == "" {
		return nil
	}

	return validRepos(str)
}
