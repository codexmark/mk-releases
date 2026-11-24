---
camada: "04 Sessão"
tags:
  - mk
  - sessão
  - login
  - sddm
fontes:
  - "tests/system/sddm-tema.test.sh"
  - "system/mk/sddm/mk/geometria.js"
  - "migrations/system/20261128-entrada-marca.sh"
  - "system/mk/sddm/mk/Main.qml"
  - "system/mk/sddm/mk/theme.conf"
  - "system/mk/sddm/mk/metadata.desktop"
  - "installer/mk-install"
  - "migrations/system/20261120-entrada-mk.sh"
  - "migrations/system/20261122-boot-entrada.sh"
---

# SDDM

O SDDM é a tela de login. O Mk traz um tema próprio em QML e deixa o usuário e a sessão "Mk" já escolhidos.

## Para quê

Pedir a senha e iniciar a sessão. É também onde se escolhe entre o Mk e outro desktop instalado na mesma máquina.

## Como funciona

- **Tema** (`system/mk/sddm/mk/`): `Main.qml` (Qt 6), `theme.conf`, `metadata.desktop`, `letra-m.png`, `letra-k.png`, `geometria.js`, `grain.png`. Desenha a folha de desenho técnico da [[Direção Folha]]: moldura, régua, a marca "Mk" construída em movimento (eixos, guias, diagonais, letras e nós, como no logo; `motion=nao` no `theme.conf.user` desliga), o relógio de anéis da tela de bloqueio à esquerda, quem entra e a senha (uma linha fina com cursor de cobre) e, no canto, o carimbo com sessão, teclado e energia.
- O instalador copia o tema para `/usr/share/sddm/themes/mk` e o escolhe em `/etc/sddm.conf.d/20-mk-theme.conf` (`[Theme] Current=mk`).
- **Sessão**: `/usr/share/wayland-sessions/mk.desktop`, com `Exec=/usr/bin/mk-session` ([[mk-session e uwsm]]).
- **Pré-seleção**: `/var/lib/sddm/state.conf` é gravado com `Session=…/mk.desktop` e `User=<usuário>`.
- **Com [[LUKS]]**: `/etc/sddm.conf.d/10-mk.conf` com `[Autologin]`.
- **Teclado e movimento**: `theme.conf.user` recebe `layout=<layout>` do instalador; a mesma chave de arquivo aceita `motion=nao`, que desliga o movimento da marca e dos anéis. (A chave `place` e o `mark.png` saíram: o tema não os usava.)
- **A marca construída**: os eixos, as guias, as diagonais, as letras e os nós aparecem nessa ordem, em pouco mais de dois segundos. Depois fica quase parada: um cursor de cobre anda pela cota com os segundos, um nó pisca de cada vez, cada tecla da senha acende o próximo e, enquanto a senha é conferida, uma guia varre a marca. Senha errada, o cobre do k apaga e volta.
- **O relógio de anéis**: os mesmos dois anéis da [[Tela de bloqueio]] (minutos e segundos girando em volta de um ponto fora da tela), com a hora grande e a data; ao abrir, chegam girando até a hora certa, e o minuto e o segundo exatos aparecem ao passar pela pílula. Quem entra e a senha ficam embaixo da marca; a sessão, o teclado e a energia, no alto.

## Por quê assim

- **Tema próprio**: boot, login e sessão falam a mesma língua visual. As fontes e cores são as do Maré, declaradas de novo no `Main.qml` (o tema do SDDM não importa o `Theme.qml`).
- **`theme.conf.user`**: a lista de layouts do próprio SDDM diz "us" mesmo com o X em outro layout; com um layout só, o tema mostra o do sistema.
- **Autologin com LUKS**: a senha já foi digitada no boot.
- **Troca do tema num Mk instalado**: por migração. A `20261122-boot-entrada.sh` copia para `mk.novo`, grava o layout e troca com `mv`, no padrão "troca de pasta de uma vez" ([[Convenções]]).
- **A entrada nova num Mk instalado**: a migração `20261128-entrada-marca.sh` troca o tema inteiro (copia ao lado e troca de uma vez) e mantém o `theme.conf.user` da máquina. Sem o tema do Mk no sddm, não faz nada.
- **As peças vêm do gerador do logo**: `make-logo.py` gera `letra-m.png`, `letra-k.png` e `geometria.js`; um teste (`tests/system/sddm-tema.test.sh`) confere que o tema se basta, só usa as cores do Mk, que a geometria bate com o gerador e que a conta dos anéis da entrada não diverge da do bloqueio.
- A escolha do SDDM em vez de outro gerenciador de login: **motivo não registrado**.

## Onde está no código

- `system/mk/sddm/mk/`
- `installer/mk-install`: `install_desktop`, `configure_services`

## Ligações

Depende de: [[Serviços systemd]], [[Locale, fuso e teclado]], [[Direção Folha]], [[Fontes]]. Inicia: [[mk-session e uwsm]]. Parte do [[Fluxo de boot]].
