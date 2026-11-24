---
camada: "05 Compositor"
tags:
  - mk
  - compositor
  - monitores
  - recuperação
fontes:
  - "config/hypr/hyprland.lua"
  - "shell/mare/scripts/mare-monitors"
---

# Recuperação de monitores

Se a máquina desligar ou o Hyprland cair **durante** a contagem de um teste de monitores, sobra um arquivo `monitores.lua.anterior`. No login seguinte, o `hyprland.lua` vê a sobra e devolve a configuração de antes.

## Para quê

É a terceira e última rede de segurança da página Monitores: cobre a queda de energia, quando nem o vigia teve tempo de agir.

## Como funciona

No `hyprland.lua`, antes de carregar os arquivos gerados:

1. Abre `<raiz>/monitores.lua.anterior`. Se não existe, nada a fazer.
2. Lê a primeira linha (a marca) e o resto (o conteúdo de antes do teste).
3. Compara a marca com `-- mk:teste <HYPRLAND_INSTANCE_SIGNATURE>` **desta** sessão.
   - **Igual**: o teste ainda está correndo nesta sessão; o `mare-monitors` cuida dele. Não mexe.
   - **Diferente**: é sobra de outra sessão. Se o conteúdo anterior é vazio (antes não havia arquivo), apaga o `monitores.lua`; senão regrava o `monitores.lua` com o conteúdo anterior. Em seguida apaga o `.anterior`.

Só então o `monitores.lua` é carregado ([[Ordem de carga]]).

## Por quê assim

As três camadas de proteção, da mais comum à mais rara:

| Situação | Quem devolve |
|---|---|
| ninguém confirmou em 15 s | o vigia, pela contagem |
| o Hyprland caiu com a config nova, ou a máquina foi desligada/reiniciada | o vigia: `alive` falha, ou o `trap` de TERM/HUP/INT devolve antes de sair |
| queda de energia: nem o vigia rodou | o `hyprland.lua`, no próximo login |

A marca da sessão existe porque o `hyprland.lua` também é recarregado **durante** o teste (o próprio `trial` dá `hyprctl reload`). Sem ela, o reload desfaria o teste que acabou de começar.

O caso "antes não havia arquivo" é tratado à parte para a recuperação não deixar um `monitores.lua` vazio.

## Onde está no código

- `config/hypr/hyprland.lua` (o bloco `do … end` depois dos `require`)
- `shell/mare/scripts/mare-monitors`: `restore`, `watch` e o `trial`

## Ligações

Depende de: [[Monitores]], [[Ordem de carga]]. Decisão: [[D13 Monitores por arquivo com vigia]].
