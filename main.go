package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

var (
	red   = lipgloss.NewStyle().Foreground(lipgloss.Red)
	green = lipgloss.NewStyle().Foreground(lipgloss.Green)
)

func main() {
	configPath := flag.String("config", "", "arquivo de configuração JSON")
	dryRun := flag.Bool("dry-run", false, "apenas busca, sem aplicar alterações")
	reveal := flag.Bool("reveal", false, "mostra o nome antigo nas prévias (escondido por padrão)")
	from := flag.String("from", "", "regex do nome antigo")
	to := flag.String("to", "", "nome novo")
	reposFlag := flag.String("repos", "", "repositórios github (formato: dono/nome,dono/nome2)")
	urlsFlag := flag.String("urls", "", "repositórios git por URL, separados por vírgula")
	pathsFlag := flag.String("paths", "", "arquivos ou pastas locais, separados por vírgula")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalln(err)
	}

	if *reposFlag != "" {
		repos, err := parseRepos(*reposFlag)
		if err != nil {
			log.Fatalln(err)
		}

		cfg.Repos = repos
	}

	if *urlsFlag != "" {
		cfg.URLs = parseList(*urlsFlag)
	}

	if *pathsFlag != "" {
		cfg.Paths = parseList(*pathsFlag)
	}

	if *from != "" || *to != "" {
		if *from == "" {
			log.Fatalln("flag -to requer -from")
		}

		if *to == "" && !*dryRun {
			log.Fatalln("flag -from requer -to (exceto com -dry-run)")
		}

		if err := validRegex(*from); err != nil {
			log.Fatalln(err)
		}

		cfg.Subs = []substitution{{From: *from, To: *to}}
	}

	if err := fillConfig(&cfg); err != nil {
		log.Fatalln("deu ruim ao rodar o formulário")
	}

	if err := cfg.normalize(); err != nil {
		log.Fatalln(err)
	}

	regexes, err := compileSubs(cfg.Subs)
	if err != nil {
		log.Fatalln(err)
	}

	union, err := maybeRegex(unionPattern(cfg.Subs))
	if err != nil {
		log.Fatalln(err)
	}

	sources, err := cfg.sources()
	if err != nil {
		log.Fatalln(err)
	}

	if len(sources) == 0 {
		log.Fatalln("nenhuma fonte informada (repos, urls ou paths)")
	}

	fmt.Printf("buscando em %d fontes...\n", len(sources))

	results := collect(sources, union)

	warned := false
	for _, res := range results {
		if res.tmp {
			defer os.RemoveAll(res.dir)
		}

		if res.err != nil {
			fmt.Printf("[%s] erro: %v\n", res.name, res.err)
			continue
		}

		if len(res.matches) == 0 {
			fmt.Printf("[%s] nenhuma ocorrência\n", res.name)
			continue
		}

		if !warned {
			fmt.Println("\n💜 aviso: as prévias escondem o nome antigo por padrão (use -reveal pra ver). vai no seu tempo.")
			warned = true
		}

		fmt.Printf("\n%s (%d ocorrências):\n", res.name, len(res.matches))
		for _, m := range res.matches {
			old := m.content
			if !*reveal {
				old = maskLine(old, union)
			}

			if *dryRun {
				fmt.Printf("  %s:%d: %s\n", m.file, m.line, old)
				continue
			}

			if *reveal {
				old = red.Render(old)
			}

			fmt.Printf("  %s:%d\n  - %s\n  + %s\n", m.file, m.line, old, green.Render(replaceString(m.content, regexes, cfg.Subs)))
		}

		if *dryRun {
			continue
		}

		var approve bool
		confirm := huh.NewConfirm().
			Title(fmt.Sprintf("aplicar alterações em %s?", res.name)).
			Value(&approve)
		if err := confirm.Run(); err != nil {
			log.Fatalln("deu ruim ao rodar o formulário")
		}

		if !approve {
			fmt.Printf("[%s] pulado\n", res.name)
			continue
		}

		if err := apply(res, regexes, cfg.Subs); err != nil {
			fmt.Printf("[%s] erro ao aplicar: %v\n", res.name, err)
			continue
		}

		fmt.Printf("[%s] alterações aplicadas\n", res.name)
	}
}
