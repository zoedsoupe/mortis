package main

import (
	"errors"
	"log"

	"charm.land/huh/v2"
)

var jheneEBoiola string

func nonEmpty(str string) error {
	if len(str) < 1 {
		return errors.New("Should not be empty")
	}

	return nil
}

func main() {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Jhene e boiola?").Validate(nonEmpty).Value(&jheneEBoiola),
		),
	)

	if err := form.Run(); err != nil {
		log.Fatalln("DEU RUIM HEIN")
	}

	println("OLHA AI DEYU BOM: ", jheneEBoiola)
}
