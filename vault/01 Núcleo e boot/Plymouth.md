---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - plymouth
  - aparência
fontes:
  - "system/mk/plymouth/mk/mk.plymouth"
  - "system/mk/plymouth/mk/mk.script"
  - "installer/mk-install"
  - "migrations/system/20261120-boot-mk.sh"
  - "migrations/system/20261121-logo-plano.sh"
---

# Plymouth

O Plymouth mostra o boot com a cara do Mk: o logo no centro, sobre o preto quente, e uma linha fina onde uma cópia em cobre avança com o progresso.

## Para quê

Esconder as mensagens do kernel atrás de uma tela coerente com o resto do sistema e, com disco criptografado, pedir a senha no mesmo desenho.

## Como funciona

- O tema é `system/mk/plymouth/mk/`: `mk.plymouth` (módulo `script`), `mk.script` e as imagens `logo.png`, `track.png`, `fill.png`, `bullet.png`.
- O `mk.script` pinta o fundo de `#0b0b0b`, centraliza o logo (26% da largura da tela, no máximo 720 px), desenha a trilha em osso e estica o `fill.png` conforme o `progress` do boot. O logo e a trilha entram num *fade*.
- Com [[LUKS]], `display_password_callback` escreve "senha do disco" e desenha um ponto por caractere digitado, no lugar da linha de progresso.
- Mensagens (por exemplo, verificação de disco) aparecem pequenas, no rodapé.
- O instalador copia a pasta para `/usr/share/plymouth/themes/mk` e grava `/etc/plymouth/plymouthd.conf` com `Theme=mk` e `ShowDelay=0`.
- O hook `plymouth` está no [[Initramfs]] e `quiet splash` na [[Kernel e linha de comando|linha do kernel]].

## Por quê assim

- O logo do boot (`logo.png`) é gerado pelo `make-logo.py`, o mesmo que gera a marca do resto do sistema: uma fonte só para todas as cópias ([[Arte e geradores]]).
- O tema mora dentro do initramfs. Trocar o tema num Mk instalado pede uma [[Migrações|migração]] que copia a pasta e roda `mkinitcpio -P`.

> [!warning] Não testado
> O pedido de senha do LUKS nunca foi visto funcionando: a instalação com disco criptografado não foi exercitada ([[Limitações conhecidas]]).

## Onde está no código

- `system/mk/plymouth/mk/`
- `installer/mk-install`: `install_desktop` (bloco "Boot (plymouth) e entrada (sddm)")

## Ligações

Depende de: [[Initramfs]], [[Direção Folha]], [[Arte e geradores]]. Parte do [[Fluxo de boot]].
