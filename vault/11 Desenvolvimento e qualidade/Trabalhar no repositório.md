---
camada: "11 Desenvolvimento e qualidade"
tags:
  - mk
  - desenvolvimento
  - fluxo
fontes:
  - "docs/desenvolvimento.md"
  - "README.md"
  - "tests/README.md"
---

# Trabalhar no repositório

Como se muda o Mk sem quebrar a própria sessão: um ciclo em degraus, do mais barato ao mais caro, subindo só quando o anterior passou. O degrau zero é um teste que falha.

## Para quê

O Mk é o desktop de quem o desenvolve. Um erro no QML ou na config do Hyprland pode deixar a pessoa sem tela; o ciclo existe para o erro aparecer antes disso.

## Como funciona

| Degrau | O que roda | O que cobre |
|---|---|---|
| 0. Teste primeiro | um caso novo em `tests/`, que falha; depois `tests/run` | a lógica de scripts, Ajustes, configuração, instalador, migrações e atualização, com o sistema trocado por dublês ([[Testes e TDD]]) |
| 1. Editar e conferir | `bin/check-keybinds`, `bin/check-packages`, `bin/check-shell` | atalhos, listas, o bash do Mk ([[Checagens]]) |
| 2. Maré aninhado | `bin/dev-shot <saida.png> "<ipc>"…` | "o QML que eu escrevi carrega?" ([[dev-shot]]) |
| 3. A sessão de verdade | `bin/try-session install` ou `sudo bin/new-user <nome>` | atalhos, regras, autostart, uso real ([[Experimentar sem instalar]]) |
| 4. Uma VM | `iso/build`, `iso/vm iso`, `iso/vm` | ISO, instalador, boot, snapshots, login, migrações, atualização ([[VM de teste]]) |
| 5. Lançar | `bin/mk-release` | chega a todo Mk instalado ([[mk-release]]) |

Uma vez por clone, liga-se o gancho que roda os testes antes de cada `git push`:

```bash
git config core.hooksPath tests/hooks
```

Onde mexer, por assunto, está em [[Mapa do repositório]].

A regra que atravessa todos os degraus: **o que muda fora de `~/.config/mk` vai no instalador e numa migração. Sempre os dois** ([[Migrações]]).

## Por quê assim

- **Degraus por custo**: os testes rodam em segundos e não tocam na máquina; o `dev-shot` leva segundos e não toca na sessão; uma VM leva uns 15 minutos de instalação.
- **Teste primeiro**: "toda mudança de comportamento começa por um caso em `tests/` que falha, e só então o código" ([[D15 Teste primeiro e CI local]]).
- **Os degraus 2 a 4 continuam necessários**: os testes não veem o desenho dos painéis, o foco do teclado, o Hyprland de verdade, a instalação num disco nem o boot.
- **Não existe "Hyprland aninhado com a config inteira"** no repositório: o `dev-shot` sobe com uma config mínima e fecha tudo ao terminar.
- **Cuidado ao subir um aninhado à mão**: o `autostart.lua` iniciaria um `hypridle` que suspende a máquina de verdade ([[Ociosidade e bloqueio]]).
- **Rodar `new-user` de novo com o mesmo nome** só atualiza a config dele: é o jeito de levar uma mudança ao usuário de teste.
- **O repositório de trabalho é privado**; o que se lança vai para outro, público ([[Canal de lançamentos]]).

## Onde está no código

`docs/desenvolvimento.md` (seção "O ciclo de trabalho", que abre com "0. Teste primeiro"); `tests/README.md`.

## Ligações

Usa: [[Testes e TDD]], [[Checagens]], [[dev-shot]], [[Experimentar sem instalar]], [[VM de teste]], [[mk-release]]. Regras em: [[Convenções]].
