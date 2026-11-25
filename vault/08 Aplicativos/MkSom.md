---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - mksom
  - som
fontes:
  - "apps/README.md"
  - "apps/mksom/CONTRATO.md"
  - "system/mk/apps-links"
  - "migrations/system/20261205-mksom.sh"
  - "docs/limites.md"
---
# MkSom

O equalizador da saída e o teste de microfone (`mksom`), no molde de [[Apps do Mk (motor em Go, janela em QML)]]
(decisão [[D17 Apps próprios em Go e QML]]). O código mora em `apps/mksom`; o
`CONTRATO.md` de lá diz o que o motor responde. A janela flutua ([[Regras de janela]]).

## Equalizador

- Dez bandas fixas (31 Hz a 16 kHz): prateleira nas pontas, campânula no meio; ganho de −12 a +12 dB, e um pré-ganho para não estourar.
- É um **sink virtual do PipeWire** (`mksom.eq`, "Mk Equalizador"), montado com o módulo `filter-chain` e os filtros `builtin`. Nada de EasyEffects nem de plugin LADSPA/LV2.
- O motor escreve `~/.config/mk/mksom/eq.conf` e a unidade de usuário `mksom-eq.service` (`pipewire -c eq.conf`). Ligar torna o sink a saída padrão e lembra qual era; desligar devolve.
- Mexer num controle com ele ligado **não reinicia nada**: vai um `pw-cli set-param` por banda mudada.
- O app nunca escreve em `~/.config/pipewire` nem em `~/.config/wireplumber`.

## Microfone

- As entradas, a padrão, volume e mudo (`wpctl`).
- Nível ao vivo: o motor lê o PCM do `pw-record` e manda pico e RMS em dBFS.
- "Gravar e ouvir": cinco segundos em `XDG_RUNTIME_DIR`, tocados de volta e apagados.
- Uma frase de diagnóstico (muda, baixa demais, saturando).

## De onde veio

A ideia veio das ferramentas de equalizador e de microfone do Ryoku. Só a ideia: o código é do Mk.

Limites: [[Limitações conhecidas]] e `docs/limites.md`.
