---
camada: "11 Desenvolvimento e qualidade"
tags:
  - mk
  - desenvolvimento
  - convenções
fontes:
  - "tests/lint.test.sh"
  - "docs/desenvolvimento.md"
  - "bin/mk-update"
  - "shell/mare/scripts/mare-cava"
  - "shell/mare/Settings.qml"
  - "bin/try-session"
  - "iso/build"
---

# Convenções

As regras que o código do Mk segue de fato. A lista vem de `docs/desenvolvimento.md`, que avisa: "só o que o código de hoje segue de fato".

## Para quê

Para código novo parecer com o que já existe, e para quem lê saber em que pode confiar.

## Como funciona

- **Português.** Comentários, mensagens, nomes de migração e assuntos de commit. Todo script abre com um cabeçalho que diz o que faz e como se usa; o `--help` de alguns imprime o próprio cabeçalho.
- **Idempotência.** Scripts e migrações podem rodar duas vezes com o mesmo resultado (`--needed`, `[[ -e … ]] ||`, `grep -q … ||`).
- **`rm -rf` com `"${VAR:?}"`.** Se a variável estiver vazia, o comando falha em vez de apagar outra coisa. Deixou de ser só recomendação: o lint (`tests/lint.test.sh`) reprova um `rm -rf` com variável sem a guarda.
- **Só se apaga o que o Mk criou.** A [[Marca instalado-pelo-mk]].
- **Root só onde precisa.** O `try-session` imprime os comandos com `sudo` em vez de rodar; o `iso/build` recusa rodar como root e só chama `sudo` no `mkarchiso`; nenhum script do Maré usa sudo.
- **Troca de pasta de uma vez.** Copia a pasta nova ao lado (`.novo`) e troca com `mv`, para ninguém ver a pasta vazia no meio.
- **`pkill -f` com padrão que não casa com quem chama.** Dois casos: `mare-cava` (`cava -p <config>`) e `Settings.qml` (`pkill -x -f "hypridle -c …"`). Onde dá, o código prefere o nome exato (`pgrep -x`) ou a sessão do processo (`pkill -s`).
- **Segredo fica em `$XDG_RUNTIME_DIR`, só para o dono.** O QR do Wi-Fi (`umask 077`, apagado ao fechar o painel) e o arquivo que o `mare-vpn` importa.
- **Papéis de parede no repositório são só os do Mk**, desenhados por código.
- **Dois lugares para a mesma lista, um script que confere.** Os atalhos.
- **O que muda fora de `~/.config/mk` vai no instalador e numa migração.** Sempre os dois.
- **Nada de dados pessoais em arquivo rastreado**: o `mk-release` publica tudo, inteiro.
- **Caminhos descobertos, não fixos**: o `paths.lua` e o `Settings.dir` acham a instalação pelo próprio arquivo; as migrações leem por `${MK_SHARE:-/usr/share/mk}`.
- **Teste primeiro**, e raiz do sistema por variável (`MK_ROOT`, `SYS`, `PROC`, `ETC`, `live_root`) no que precisa ser testado ([[Testes e TDD]], [[D15 Teste primeiro e CI local]]).

## Por quê assim

- **Idempotência** é o que torna "rode de novo" a resposta para quase toda falha do instalador e do atualizador ([[Quando a atualização para]]).
- **A guarda do `rm -rf`**: um `rm -rf "$tmp/x"` com `tmp` vazio apaga `/x`.
- **A troca de pasta de uma vez**: o Hyprland e o Maré vigiam os próprios arquivos; uma pasta vazia no meio da troca virava "cannot open hyprland.lua".
- **O `pkill -f`**: sem cuidado, o padrão casa com a linha de comando do próprio shell que roda o `pkill`, e o script se mata.
- **Português**: a regra é declarada sem justificativa (motivo não registrado). Vale também para mensagens de erro, nomes de arquivos gerados (`ajustes.json`, `atalhos.lua`, `monitores.lua`) e os verbos do comando `mk`.
- **Honestidade da lista**: o documento de origem só lista o que é seguido e aponta onde ainda não é.

> [!note] Convenções do shell
> As regras específicas do QML (tokens do `Theme` e do `Motion`, `Hypr.exec`, `Mare.ignoreZone`, scripts chamados com `bash` + `shellPath`) estão em [[Armadilhas do Maré]] e [[Singletons]].

## Onde está no código

`docs/desenvolvimento.md` (seção "Convenções"), com os exemplos apontados por arquivo.

## Ligações

Aplicado em: [[mk-update]], [[Migrações]], [[Instalador passo a passo]], [[Scripts mare]], [[Marca instalado-pelo-mk]]. Conferido por: [[Checagens]], [[Testes e TDD]].
