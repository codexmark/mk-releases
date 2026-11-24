---
camada: "10 Atualização e ciclo de vida"
tags:
  - mk
  - atualização
  - lançamento
  - mk-release
fontes:
  - "tests/bin/mk-release.test.sh"
  - "bin/mk-release"
  - "docs/atualizacoes.md"
  - "docs/desenvolvimento.md"
  - "tests/README.md"
---

# mk-release

`bin/mk-release` publica o commit atual no repositório público de lançamentos. É o único caminho pelo qual código chega a um Mk instalado.

## Para quê

Transformar "o que está no meu repositório" em "o que todo Mk instalado vai receber", com algumas travas no caminho.

## Como funciona

```bash
bin/mk-release                 # lança o HEAD
bin/mk-release "o que mudou"   # com uma frase no lugar da lista de commits
```

Em ordem:

1. Recusa se houver mudança sem commit em arquivo rastreado.
2. Avisa (sem parar) se há commits que ainda não foram para o repositório privado.
3. Roda `bin/check-keybinds`, `bin/check-shell` e `tests/run` (todos os testes automáticos); se um falhar, não lança.
4. Clona ou atualiza o repositório público em `~/.cache/mk-release/mk-releases`. Se a pasta de trabalho sobrou de outro destino (o `origin` dela não é o repositório pedido agora), é apagada e clonada de novo.
5. Se o `VERSION` de lá já é o commit atual, diz "já está lançado" e sai com 0.
6. Troca tudo pelo conteúdo do `HEAD` (`git archive`: os arquivos rastreados, sem `.git`) e grava o `VERSION`.
7. Grava o `NOVIDADES`: a frase dada, ou os assuntos dos commits desde o lançamento anterior (no máximo 12 linhas; passando disso, 11 e um "… e mais N").
8. Faz um commit `Mk <data> (<commit>)` com as novidades no corpo e dá `git push` para o `main` do repositório público.

`MK_RELEASES` aponta para outro repositório; `MK_RELEASE_EMAIL` troca o e-mail de autor do commit público.

## Por quê assim

- **`git archive`, não um push do histórico**: o repositório público recebe um retrato por lançamento. O histórico de trabalho não vai junto; vão os **assuntos dos commits**, que viram o `NOVIDADES`.
- **E-mail "noreply" no commit público**: o endereço pessoal de quem lança não aparece.
- **`check-keybinds` antes de lançar**: `binds.lua` e `Keybinds.js` divergentes fariam os Ajustes mostrarem um atalho e o Hyprland executar outro.
- **`check-shell` antes de lançar**: "já foi lançado um `mk.bash` que derrubava todo terminal".
- **`tests/run` antes de lançar**: "os testes passam, ou não lança". É o portão que não se pula ([[D15 Teste primeiro e CI local]]).
- **Pasta de trabalho refeita quando o destino muda**: o `push` vai para o `origin` da pasta, "e tem de ser o repositório pedido agora"; com `MK_RELEASES` trocado, uma pasta antiga publicaria no lugar errado.
- **Não há "desfazer"**: o conserto é um commit novo e outro lançamento. Lançar duas vezes o mesmo commit não tem efeito, e um commit mais antigo que o último lançado não é aceito pelos Mk instalados.
- **Tudo o que está rastreado vai a público, inteiro**: nenhuma senha, chave, e-mail ou caminho pessoal pode estar num arquivo rastreado. Vale para a documentação (e para esta vault).
- **Arquivo novo sem `git add` não impede o lançamento**, mas também não vai nele: é o erro silencioso a conferir antes.

## Onde está no código

`bin/mk-release`; `docs/atualizacoes.md` (seção "Lançar"); a lista "Antes de lançar" em `docs/desenvolvimento.md`; `tests/bin/mk-release.test.sh`.

## Ligações

Depende de: [[Canal de lançamentos]], [[Checagens]], [[Testes e TDD]]. Alimenta: [[mk-update]], [[Atualizações na ilha e nos Ajustes]]. Decisão: [[D15 Teste primeiro e CI local]].
