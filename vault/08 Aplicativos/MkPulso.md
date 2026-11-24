---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - mkpulso
  - sistema
  - processos
fontes:
  - "apps/mkpulso/CONTRATO.md"
  - "apps/README.md"
  - "apps/mkpulso/bin/mkpulso"
  - "apps/mkpulso/qml/App.qml"
  - "system/mk/mkpulso-links"
  - "migrations/system/20261129-mkpulso.sh"
  - "shell/mare/SystemPanel.qml"
  - "tests/apps/mkpulso-go.test.sh"
  - "docs/limites.md"
---

# MkPulso

O MkPulso é o monitor de sistema do Mk: processos agrupados por app e por serviço, desempenho, a ficha da máquina, o que sobe no login e os serviços do systemd. Entrou no lugar do `btop` num terminal.

## Para quê

Responder "o que está pesando?" pelo nome do app, não por uma lista de centenas de processos, e agir sobre isso (encerrar, pausar, mudar a prioridade, parar um serviço) sem derrubar a própria sessão.

## Como funciona

A janela tem cinco áreas; `mkpulso <área>` abre direto numa (`processos`, `desempenho`, `sistema`, `inicio`, `servicos`).

- **Processos, agrupados por identidade.** O grupo vem do **cgroup** do processo:
  - `app-<nome>-*.scope` e `app-<nome>@*.service` viram `app:<nome>`;
  - uma unidade vira `servico:<unidade>`, com escopo `usuario` ou `sistema`;
  - o que não está em unidade nenhuma (uma sessão de terminal) agrupa pelo programa do topo;
  - threads do kernel e o resto ficam em `sistema:kernel` e `sistema:outros`.

  Um grupo só é do tipo "app" se tem janela ou atalho de menu. A chave de um grupo é `id` + `escopo`: a mesma unidade pode existir no sistema e na sessão.
- **Energia** de cada grupo, em seis degraus ("nenhuma" a "muito alta"): impacto = CPU do grupo ÷ núcleos + GPU ÷ 2 + disco em MiB/s (até 10).
- **Médias** do último minuto e dos últimos três, guardadas enquanto o `watch` roda.
- **Desempenho**: gráficos de CPU, memória, disco, rede, GPU e temperatura.
- **Sistema**: a ficha da máquina, inclusive a saúde dos discos quando dá para saber sem root.
- **Inicialização**: os atalhos de autostart (XDG), os serviços de usuário habilitados e, só para leitura, o que o `autostart.lua` do Mk sobe. Um atalho do sistema é desligado do jeito do padrão XDG: uma cópia em `~/.config/autostart` com `Hidden=true`.
- **Serviços**: unidades `.service`, `.socket` e `.timer` do sistema e do usuário, com estado, ações e as últimas linhas do journal.

Subcomandos do coletor: `watch`, `sistema`, `sinal`, `grupo`, `prioridade`, `servicos`, `servico`, `registro`, `inicio`.

**Recusas de segurança** (código 2):

- sinal: o pid 1, o próprio coletor, a sessão em que ele roda (o Hyprland e o Maré, a menos de `--mesmo-assim`) e processos de outro usuário;
- grupo: `sistema:*` é sempre recusado; uma recusa em qualquer processo aborta antes do primeiro sinal;
- serviço: parar ou desabilitar o que derruba a sessão ou a máquina (`sddm.service`, `dbus*.service`, `systemd-*`, `user@*.service`; `NetworkManager.service` só com `--mesmo-assim`), e parar ou reiniciar uma unidade que contém a sessão ou o próprio coletor;
- o nome de uma unidade só aceita um conjunto restrito de caracteres e tem de terminar em `.service`, `.socket` ou `.timer`; o resto é erro, sem chegar ao `systemctl`.

## Por quê assim

- **No lugar do btop**: o painel Sistema do Maré abre o MkPulso (`Hypr.exec("mkpulso")`). O `btop` continua em `dev.packages`.
- **Agrupar pelo cgroup**: o contrato fixa que o grupo vem "do cgroup do processo", isto é, da unidade do systemd em que ele roda; os muitos processos de um app aparecem como uma linha só.
- **Um grupo de serviço é encerrado pelo systemd** (`systemctl stop` para encerrar, `systemctl kill --signal=…` para os outros sinais), não por sinais avulsos a cada processo.
- **Unidades de sistema passam pelo polkit**: o coletor nunca pede senha nem roda como root; o `systemctl` chama o polkit e o Maré mostra o pedido.
- **Baixar o nice pede privilégio**: o coletor tenta e, recusado, devolve a frase; não escala sozinho.
- **Não pede pacote novo**: usa o Quickshell e o `jq`, que o Mk já tem. A migração `20261129-mkpulso.sh` só cria os links.
- **Coube em tela pequena e flutua**: flutuando em 1220x780 numa tela menor, o alto ficava atrás da ilha; o tamanho inicial é o dele ou o que couber. Lado a lado com outras janelas virava uma coluna estreita e a tabela ficava cortada.

> [!warning] Pouco visto
> Encerrar um processo e as recusas (processo 1, SDDM) foram exercitados na VM. As ações de serviço pela janela só rodaram contra dublês ou pela linha de comando ([[Limitações conhecidas]]).

## Onde está no código

`apps/mkpulso/` (`src/`, `qml/`, `bin/mkpulso`, `CONTRATO.md`); a regra de grupos está no cabeçalho de `src/internal/grupos/grupos.go`, e a de energia em `src/internal/grupos/energia.go`. `system/mk/mkpulso-links`; `shell/mare/SystemPanel.qml`.

## Ligações

Molde: [[Apps do Mk (motor em Go, janela em QML)]]. Depende de: [[Serviços systemd]], [[mk-session e uwsm]], [[Polkit]]. Aberto por: [[Painéis]] (Sistema), [[Busca e calculadora]]. Relacionado: [[Autostart]], [[Regras de janela]].
