package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func maybeRegex(str string) (*regexp.Regexp, error) {
	r, err := regexp.Compile(str)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func compileSubs(subs []substitution) ([]*regexp.Regexp, error) {
	if len(subs) == 0 {
		return nil, errors.New("nenhuma substituição informada")
	}

	regexes := make([]*regexp.Regexp, len(subs))
	for i, s := range subs {
		re, err := maybeRegex(s.From)
		if err != nil {
			return nil, fmt.Errorf("substituição %d (%q): regex inválida: %w", i, s.From, err)
		}
		regexes[i] = re
	}

	return regexes, nil
}

func unionPattern(subs []substitution) string {
	parts := make([]string, len(subs))
	for i, s := range subs {
		parts[i] = "(" + s.From + ")"
	}

	return strings.Join(parts, "|")
}

func replaceBytes(content []byte, regexes []*regexp.Regexp, subs []substitution) []byte {
	for i, re := range regexes {
		content = re.ReplaceAll(content, []byte(subs[i].To))
	}

	return content
}

func replaceString(str string, regexes []*regexp.Regexp, subs []substitution) string {
	return string(replaceBytes([]byte(str), regexes, subs))
}

// maskLine esconde o nome antigo, mantendo o contexto da linha visível.
func maskLine(line string, union *regexp.Regexp) string {
	return union.ReplaceAllString(line, red.Render("[nome antigo]"))
}
