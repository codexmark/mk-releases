---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - lacunas
fontes:
  - "installer/mk-install"
  - "system/packages/base.packages"
  - "system/packages/desktop.packages"
  - "config/hypr/mare/look.lua"
  - "config/hypr/mare/rules.lua"
---

# Motivos não registrados

As escolhas do Mk cujo **porquê não está escrito** em lugar nenhum do repositório: nem em comentário, nem na documentação, nem em mensagem de commit. Esta vault descreve o que cada uma faz e não inventa a razão.

## Para quê

Para quem lê não tomar uma dedução por decisão documentada, e para quem mantém saber o que vale registrar.

## Como funciona

| Escolha | O que o código faz | Nota |
|---|---|---|
| limine como bootloader (e não GRUB ou systemd-boot) | usa o submenu de snapshots do `limine-snapper-sync` | [[Limine]] |
| btrfs com `@`, `@home`, `@log`, `@pkg` | separa o que entra no snapshot do que não entra | [[Btrfs e subvolumes]] |
| `noatime,compress=zstd` | opções fixas de montagem | [[Btrfs e subvolumes]] |
| ESP de 2 GiB em `/boot` | kernel e entradas de snapshot ficam na ESP | [[Particionamento]] |
| mínimo de 24 GiB de disco | validação fixa | [[Particionamento]] |
| snapper sem timeline e com limite de 5 | snapshots só por transação do pacman | [[Snapper e snap-pac]] |
| sem swap e sem zram | o instalador não cria nenhum | [[Swap e hibernação]] |
| um kernel só (`linux`), sem `linux-lts` | uma entrada de kernel | [[Kernel e linha de comando]] |
| hook `systemd` no initramfs | `sd-vconsole`, `sd-encrypt`, `rd.luks.name=` | [[Initramfs]] |
| senha do disco igual à do usuário | uma pergunta a menos | [[LUKS]] |
| SDDM como tela de login | tema próprio em QML | [[SDDM]] |
| `uwsm` para iniciar a sessão | "mesmo esquema do Omarchy" (sem dizer o que traz) | [[mk-session e uwsm]] |
| `iwd` instalado junto com o NetworkManager | o backend do NetworkManager não é configurado | [[Rede]] |
| `ufw-docker install` fora do instalador | fica para um script rodado depois | [[Firewall]] |
| `en_US.UTF-8` sempre gerado | além do idioma escolhido | [[Locale, fuso e teclado]] |
| fuso de fábrica | valor fixo no instalador | [[Locale, fuso e teclado]] |
| root bloqueado, sudo com senha | `passwd -l root`, `%wheel ALL=(ALL:ALL) ALL` | [[Usuários e sudo]] |
| layout `dwindle` | `general.layout` | [[Hyprland em Lua]] |
| `suppress_event = "maximize"` para todas as janelas | regra `no-maximize-requests` | [[Regras de janela]] |
| bash (e não zsh ou fish) | a camada do terminal é sobre o bash | [[Terminal e bash do Mk]] |
| português em tudo | regra declarada sem justificativa | [[Convenções]] |
| os Ajustes como folha, não painel da ilha | janela *Overlay* própria | [[Ajustes]] |
| sem a entrada de acessibilidade por voz na ISO | o build a remove do menu UEFI | [[ISO]] |
| `check-packages` fora do `mk-release` | só `check-keybinds` e `check-shell` barram | [[Checagens]] |
| um "hibernar" antigo vira "não dormir" | `sleepMode` | [[Swap e hibernação]] |

## Por quê assim

- **A origem está registrada; o motivo, não.** O `README.md` diz que "a estrutura segue o modelo do Ryoku" e que o conteúdo "junta a base do Ryoku com o kit de desenvolvimento do Omarchy". Os comentários apontam a mesma origem: `base.packages` (onde estão btrfs, snapper e limine) é "o base.packages do Ryoku, sem o que é específico do shell dele"; o `uwsm` é "o mesmo esquema do Omarchy"; o firewall segue "o mesmo princípio" do Omarchy. Isso explica **de onde** as escolhas vieram, não **por que** foram mantidas.
- **O que se pode afirmar** é sempre o efeito, que está no código. O efeito está descrito na nota de cada linha.
- **Não é uma lista de defeitos.** São lacunas de registro. A maior parte é o padrão sensato de um desktop Arch; falta só alguém ter escrito a razão.

> [!note] Para quem mantém
> Cada linha desta tabela é um candidato a um comentário de uma frase no código. Quando o motivo for escrito, a linha sai daqui e, se for uma decisão de peso, ganha uma nota em [[Registro de decisões]].

## Onde está no código

Os arquivos de cada linha estão nas notas ligadas. Os mais frequentes: `installer/mk-install` (`partition`, `configure_boot`, `configure_snapper`, `create_user`) e as listas de `system/packages/`.

## Ligações

[[Registro de decisões]] · [[Limitações conhecidas]] · [[Como ler esta vault]]
