package main

import (
	"errors"
	"log"
	"regexp"

	"charm.land/huh/v2"
)

var dead string
var alive string
var repos string

func main() {
	form := buildForm()

	if err := form.Run(); err != nil {
		log.Fatalln("deu ruim ao rodar o formulário")
	}
}

func buildForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("nome morto/antigo").
				Validate(required).
				Value(&dead).
				Description("can be a golang regex"),
			huh.NewInput().
				Title("nome correto").
				Validate(required).
				Value(&alive).
				Description("can be a golang regex"),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("repositorios github").
				Description("(formato: dono/nome,dono/nome2)").
				Validate(required).
				Value(&repos),
		),
	)
}

func required(str string) error {
	if len(str) < 1 {
		return errors.New("precisa preencher esse campo!")
	}

	return nil
}

func maybeRegex(str string) (*regexp.Regexp, string) {
	r, err := regexp.Compile(str)

	if err != nil {
		return nil, str
	}

	return r, ""
}
