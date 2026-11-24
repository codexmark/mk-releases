---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - instalação
  - aur
fontes:
  - "iso/build"
  - "installer/mk-install"
  - "iso/README.md"
  - "system/packages/aur.packages"
---

# D10 AUR compilado na ISO

Os pacotes do AUR são compilados **quando a ISO é gerada** e viajam dentro dela, num repositório pacman local.

## Contexto

Alguns pacotes de que o Mk depende só existem no AUR: os ganchos do limine, o `yay`, uma fonte, o ble.sh. Compilar AUR exige `makepkg` (que não roda como root), ferramentas de build e rede, e pode falhar. O live da ISO roda como root, da memória.

## Escolha

- `iso/build` compila cada pacote de `aur.packages` como o usuário e monta o repositório `mk-aur`.
- O instalador soma esse repositório a um `pacman.conf` temporário e instala com `pacstrap`, como qualquer pacote.
- Uma falha de build tira o pacote da ISO, vai para o relatório e não para nada; o que faltou vira `aur-pendentes.txt`, que o `mk-update` oferece depois pelo `yay`.

## Alternativas

- **Compilar depois da instalação**, com o `yay`, no primeiro boot: era o desenho anterior (o cabeçalho de `aur.packages` o descrevia até ser corrigido). O sistema subiria sem os ganchos do boot.
- **Compilar no live, durante a instalação** *(não discutida no repositório)*: o live roda como root, e o `makepkg` não roda como root.
- **Um repositório binário público do Mk** *(não discutida no repositório)*.

## Consequência

- A instalação não compila nada.
- A qualidade da ISO depende da máquina que a gera: sem o `gradle`, os ganchos do limine não compilam e a ISO sai sem o submenu de snapshots. Daí o `--syncdeps`, o relatório e o reaproveitamento de pacotes já compilados.
- Os pacotes da ISO envelhecem; depois da instalação quem os atualiza é o `yay`, pelo `mk-update`.
- O repositório local é aceito sem assinatura (`SigLevel = Optional TrustAll`).
- Um pacote novo do AUR só chega a quem já instalou por uma migração que o ponha no `aur-pendentes.txt`.

## Onde está no código

`iso/build` (bloco "1. AUR"); `installer/mk-install` (`make_pacman_conf`, `install_aur`).

## Ligações

Afeta: [[AUR na instalação]], [[ISO]], [[Ganchos do limine]], [[Pacman e AUR]]. Índice: [[Registro de decisões]].
