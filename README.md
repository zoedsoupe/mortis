# mortis 💀

![gopher rebirthing](./mascot.png)

cli mínima pra renomear, aposentar ou ressignificar seu morto (e/ou nome antigo) em vários lugares de uma vez: repositórios github, qualquer remote git e arquivos/pastas locais.

você mudou seu nome e agora ele tá espalhado em uma penca de lugar pela internet (ou arquivos no computador)? `mortis` te ajuda procurar pelo seu antigo nome, achar ocorrências e deixa você revisar uma a uma caso queira removê-las/editá-las.

## como funciona

1. junta as fontes que você passar (clonando repos quando preciso)
2. busca em paralelo pelo nome antigo (aceita regex)
3. mostra as ocorrências com o nome antigo escondido por padrão 💜
4. abre um formulário interativo pra aprovar ou descartar cada fonte, com preview
5. aplica as substituições aprovadas

## instalação

```sh
go install github.com/zoedsoupe/mortis@latest
```

ou clone o repo e rode `go build`.

## uso

```sh
mortis -from "Nome Antigo" -to "Nome Novo" -repos dono/repo1,dono/repo2
```

| flag       | descrição                                                |
|------------|----------------------------------------------------------|
| `-from`    | nome antigo (regex do go, RE2)                           |
| `-to`      | nome novo (substituição literal)                         |
| `-repos`   | repos github `dono/repo`, separados por vírgula          |
| `-urls`    | repos git por URL (gitlab, codeberg...), por vírgula     |
| `-paths`   | arquivos ou pastas locais, separados por vírgula         |
| `-dry-run` | só busca e mostra as ocorrências, sem alterar nada       |
| `-reveal`  | mostra o nome antigo nas prévias (escondido por padrão)  |
| `-config`  | arquivo de configuração JSON                             |

exemplo com regex, pegando variações de capitalização e separador:

```sh
mortis -from '(?i)nome[ _-]?antigo' -to "Nome Novo" -repos zoedsoupe/taina,zoedsoupe/peri
```

> dica: use `\b` nas bordas (`\bana\b`) pra não casar substring sem querer.

buscando em arquivos locais (`~` funciona):

```sh
mortis -from "Nome Antigo" -to "Nome Novo" -paths "~/notas,./diario.txt"
```

### nome antigo escondido por padrão

ver o nome antigo repetido na tela pode ser desconfortável (principalmente pra pessoas trans), então o mortis mascara ele nas prévias como `[nome antigo]`, mantendo o contexto da linha visível. se preferir ver o texto original, rode com `-reveal`.

### arquivo de configuração

pra não repetir flags, dá pra usar um JSON com `-config`:

```json
{
  "from": "Nome Antigo",
  "to": "Nome Novo",
  "repos": ["dono/repo1", "dono/repo2"],
  "urls": ["https://codeberg.org/dono/repo.git"],
  "paths": ["~/notas"]
}
```

`from`/`to` no topo valem como uma substituição. pra mais de uma, use `substitutions`:

```json
{
  "repos": ["dono/repo"],
  "substitutions": [
    {"from": "Nome Antigo", "to": "Nome Novo"},
    {"from": "login_antigo", "to": "login_novo"}
  ]
}
```

flags passadas na linha de comando sobrescrevem o que estiver no arquivo.

## o que o mortis não faz

- não lida com issues, nem comentários em PRs (ainda)
- não commita nem dá push nos repos (ainda)

## licença

WTFPL - faça o que quiser. veja [LICENSE](./LICENSE).
