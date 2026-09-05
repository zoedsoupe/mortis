# mortis 💀

cli mínima pra renomear, aposentar ou ressignificar seu morto (e/ou nome antigo) em vários lugares de uma vez. por enquanto só no github, mas devo ir atualizando novas fontes aos poucos.

você mudou seu nome e agora ele tá espalhado em uma penca de lugar pela internet (ou arquivos no computador)? `mortis` te ajuda procurar pelo seu antigo nome, achar ocorrências e deixa você revisar uma a uma caso queira removê-las/editá-las.

![gopher rebirthing](./mascot.png)

## como funciona

1. clona os repos que você passar
2. busca em paralelo pelo nome antigo (aceita regex)
3. abre um formulário interativo pra aprovar ou descartar cada ocorrência, com preview
4. aplica as substituições aprovadas e commita em cada repo
5. pergunta se você quer dar push

## instalação

```sh
go install github.com/zoedsoupe/mortis@latest
```

ou clone o repo e rode `go build`.

## uso

```sh
mortis -from "Nome Antigo" -to "Nome Novo" -repos dono/repo1,dono/repo2
```

| flag     | descrição                                          |
|----------|----------------------------------------------------|
| `-from`  | nome antigo / dead name (regex do go, RE2)         |
| `-to`    | nome novo (substituição literal)                   |
| `-repos` | repos `dono/repo`, separados por vírgula           |

exemplo com regex, pegando variações de capitalização e separador:

```sh
mortis -from '(?i)nome[ _-]?antigo' -to "Nome Novo" -repos zoedsoupe/taina,zoedsoupe/peri
```

> dica: use `\b` nas bordas (`\bana\b`) pra não casar substring sem querer.

## o que o mortis não faz

- não lida com issues, nem comentários em PRs (ainda)

## licença

WTFPL - faça o que quiser. veja [LICENSE](./LICENSE).
