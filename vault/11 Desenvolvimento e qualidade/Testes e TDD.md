---
camada: "11 Desenvolvimento e qualidade"
tags:
  - mk
  - desenvolvimento
  - testes
  - tdd
fontes:
  - "tests/apps/mkpulso-go.test.sh"
  - "migrations/system/20261128-shellcheck.sh"
  - "tests/README.md"
  - "tests/run"
  - "tests/lib.sh"
  - "tests/lint.test.sh"
  - "tests/hooks/pre-push"
  - "tests/js/load.mjs"
  - "tests/js/qml-puro.mjs"
  - "tests/config/hl.lua"
  - "tests/installer/carrega.sh"
  - "tests/migrations/mig.sh"
  - "tests/migrations/regras.test.sh"
  - "tests/docs/docs.test.sh"
  - "tests/qml/estrutura.test.sh"
  - "bin/mk-release"
---

# Testes e TDD

O Mk tem uma suíte de testes automáticos em `tests/` (132 arquivos e cerca de 2800 casos na última contagem) que não precisa de root, de rede nem de sessão gráfica, e uma regra: **teste primeiro**.

## Para quê

Conferir a lógica do Mk inteiro (scripts, Ajustes, configuração, instalador, migrações, atualização) em segundos, a cada mudança, sem tocar na máquina.

## Como funciona

```bash
tests/run              # tudo
tests/run mare-sleep   # só os arquivos com esse trecho no caminho
tests/run -v scripts   # mostra cada caso
```

**A regra** (`tests/README.md`):

1. Escreva o caso em `tests/` dizendo o que se espera. Rode: ele falha.
2. Mude o código até passar.
3. Rode tudo antes do commit.

Um defeito achado em uso vira primeiro um caso que o reproduz, e só então a correção. Código novo sem teste não entra; código antigo que for tocado ganha o teste que faltava.

**Onde fica cada coisa**:

| Pasta | O que testa | Apoio |
|---|---|---|
| `tests/lint.test.sh` | sintaxe e lint de tudo: `bash -n`, shellcheck, QML, JavaScript, Lua, JSON, Python; bit de execução; `rm -rf` sempre com `${VAR:?}`; nenhum teste com `/usr/bin` no `PATH` | |
| `tests/js/` | o JavaScript do shell (`Calc`, `Emoji`, `Keybinds`, `SettingsSchema`, `Place`) e a lógica pura tirada de arquivos QML (`Monitors`, `Projects`, `Updates`, `BatteryWatch`, `NetworkPanel`, `LauncherPanel`, `CalendarPanel` e outros painéis e páginas), no node | `load.mjs`, `harness.mjs`, `qml-puro.mjs` |
| `tests/scripts/` | os 24 `shell/mare/scripts/mare-*` | |
| `tests/bin/` | `bin/*` (`mk-update`, `mk-release`, `mk-hibernate`, `try-session`…) | |
| `tests/installer/` | `installer/mk-install`: respostas, disco, sistema, boot, modo desassistido | `carrega.sh` |
| `tests/iso/` | `iso/build`, `iso/vm`, o perfil da ISO | |
| `tests/system/` | `system/` e `scripts/`: listas, `limine-default`, `os-release`, firewall, `mktunes-links` | |
| `tests/migrations/` | cada migração, em `system/` e `user/`, e as regras comuns a todas | `mig.sh`, `regras.test.sh` |
| `tests/config/` | o Lua do Hyprland, `mk.bash`, hypridle, hyprlock, ghostty, fastfetch, mimeapps | `hl.lua`, `harness.lua` |
| `tests/qml/` | a estrutura do shell que nenhum compilador confere | |
| `tests/apps/` | os apps de `apps/` (MkPulso, MkPacotes, MkUtil): os testes em Go de cada motor (`go test`, `gofmt`, `go vet` e a conferência de que o binário é o do código do repositório), a lógica e a estrutura das janelas, os lançadores | `*-exemplo/` (respostas gravadas) |
| `tests/docs/` | a documentação contra o código | |

Um arquivo de teste por arquivo testado, com o mesmo nome: `shell/mare/scripts/mare-sleep` → `tests/scripts/mare-sleep.test.sh`.

**Três tipos de arquivo**: `*.test.sh` (bash), `*.test.mjs` (node) e `*.test.lua` (lua). Todos imprimem `ok - …` / `not ok - …`; o `tests/run` soma e sai com 1 se algum reprovar.

**A caixa de areia** (`tests/lib.sh`):

- `$SBX` é uma pasta temporária só daquele arquivo; `HOME` e os `XDG_*` apontam para dentro dela; as variáveis da sessão gráfica são removidas;
- **guardas**: `sudo`, `systemctl`, `pacman`, `hyprctl`, `nmcli`, `mkfs.*`, `cryptsetup`, `efibootmgr`, `reboot` e dezenas de outros estão trocados por comandos que falham e reprovam o teste;
- **`stub <nome> [corpo]`** declara o que um desses comandos responde; cada chamada fica em `$SBX/calls` (`called`, `not_called`, `calls_reset`);
- **`path_sem <cmd>…`** dá um `PATH` em que o comando não existe, para o ramo "a ferramenta não está instalada";
- **`jeq`** compara dois JSON pelo valor (`60.00` = `60`); `eq`, `has`, `hasnt`, `like`, `ok`, `fails`, `exists`, `absent`, `skip` completam.

**Os apoios por linguagem**:

- `tests/js/load.mjs` carrega um `.js` do shell (biblioteca `.pragma library` do QML) como módulo do node;
- `tests/js/qml-puro.mjs` tira de um `.qml` o trecho entre os comentários `// --- puro: início ---` e `puro: fim` e o testa no node (é assim que o cálculo de modelines e o arranjador do `Monitors.qml` são conferidos; treze arquivos QML passaram a ter a lógica num bloco puro testado);
- `tests/config/hl.lua` é um `hl` falso: anota o que a configuração pediu ao Hyprland e **recusa uma função que o Mk não usa**;
- `tests/installer/carrega.sh` carrega o `mk-install` como biblioteca, sem chamar o `main`, com `$target`, `$log` e `$live_root` dentro de `$SBX`;
- `tests/migrations/mig.sh` monta um sistema falso (a raiz em `$R`, passada como `MK_ROOT`, e o Mk em `$SHARE`) e dá `mig`, `instalados`, `estado`, `de_novo` e `sem_mudanca`.

**Raízes trocáveis.** O código de produção aceita uma raiz por variável para poder ser testado: `MK_ROOT` (`mk-update`, `mare-updates`, migrações de sistema), `MK_SHARE`, `MK_HIB_ROOT` (`mk-hibernate`), `SYS`, `PROC` e `ETC` nos scripts `mare-*`, `live_root` no instalador. Sem a variável, os caminhos são os de sempre.

**Onde roda**: à mão; no `git push`, pelo gancho `tests/hooks/pre-push` (ligado uma vez por clone com `git config core.hooksPath tests/hooks`); e no `bin/mk-release`, antes de publicar.

## Por quê assim

- **Guardas em vez de confiança**: um teste que esquece de simular o `systemctl suspend` não suspende a máquina de quem testa; reprova.
- **Nunca montar um `PATH` com `/usr/bin`**: os guardas sairiam do caminho e o comando de verdade rodaria. O lint reprova um teste que faça isso; `path_sem` é a saída.
- **Raiz por variável**: "se o script ainda não tem essa variável, acrescente: é a emenda mínima que o deixa testável".
- **O `hl` falso recusa o que o Mk não usa**: um erro de digitação num nome da API vira falha de teste, não um erro no login.
- **O instalador nunca roda inteiro num teste**: ele formata discos. O arquivo só chama o `main` quando é executado; os testes carregam as funções e rodam uma de cada vez.
- **Cada migração tem três casos de base**: monta o "antes", roda e confere o "depois"; roda de novo e nada muda; num sistema sem nada a fazer, não quebra.
- **Docs contra o código** (`tests/docs`) e **estrutura do shell** (`tests/qml`): pegam o que nenhum compilador pega (um script sem linha no README, um painel registrado sem arquivo).
- **O shellcheck faz parte do kit**: está em `dev.packages`, e a migração `20261128-shellcheck.sh` o leva a quem já instalou. Sem ele, o lint de shell do `tests/run` é pulado.
- **Sem CI na nuvem**: "tudo roda aqui" ([[D15 Teste primeiro e CI local]]).
- **O que os testes não veem**: o desenho dos painéis, o foco do teclado, as animações, o Hyprland de verdade, a instalação num disco e o boot. Isso continua com o [[dev-shot]] e a [[VM de teste]].


## Onde está no código

`tests/README.md` (o guia), `tests/run`, `tests/lib.sh` (o cabeçalho lista as funções), `tests/hooks/pre-push`.

## Ligações

Usado por: [[Trabalhar no repositório]], [[mk-release]], [[Migrações]]. Complementado por: [[Checagens]], [[dev-shot]], [[VM de teste]]. Relacionado: [[Scripts mare]], [[Energia e suspensão]], [[D15 Teste primeiro e CI local]].
