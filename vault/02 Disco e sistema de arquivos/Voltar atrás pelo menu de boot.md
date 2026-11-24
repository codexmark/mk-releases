---
camada: "02 Disco e sistema de arquivos"
tags:
  - mk
  - disco
  - snapshots
  - boot
  - recuperação
fontes:
  - "docs/atualizacoes.md"
  - "installer/README.md"
  - "docs/limites.md"
---

# Voltar atrás pelo menu de boot

Quando uma atualização deixa o sistema sem iniciar, o caminho de volta é o submenu "Snapshots" do menu do limine: escolhe-se um snapshot anterior à atualização.

## Para quê

É a rede de segurança do sistema rolling: não é preciso um pendrive de resgate para desfazer um `pacman -Syu` ruim.

## Como funciona

1. Cada `pacman` cria um snapshot antes e outro depois ([[Snapper e snap-pac]]).
2. O `limine-snapper-sync` lista os snapshots no menu ([[Ganchos do limine]]), na ordem `BOOT_ORDER="*, *fallback, Snapshots"`.
3. Reinicia-se, abre-se "Snapshots" e escolhe-se um anterior à atualização.

O Mk **só configura** isso. Quem lista, inicia e restaura o snapshot é o `limine-snapper-sync`; não há comando do Mk para restaurar.

## Por quê assim

- O menu de boot é o único lugar que continua funcionando quando a sessão não abre; pôr a volta ali dispensa um ambiente de resgate.
- O `fstab` sem `subvolid` e o `rootflags=subvol=@` existem para a raiz continuar montando depois da troca ([[Btrfs e subvolumes]]).

> [!warning] O que não foi verificado
> Segundo `docs/atualizacoes.md`: iniciar por um snapshot, tornar a volta permanente e o comportamento com o kernel (que mora na ESP, fora do snapshot) **não foram verificados** neste projeto. `docs/limites.md` registra que o submenu aparece na VM; não que a restauração foi exercitada. Consulte a documentação do `limine-snapper-sync` antes de depender disso.

### O que a volta não desfaz

- A home: `~/.config/mk` e `~/.local/state/mk` ficam como estão.
- O kernel e o initramfs em `/boot`.
- **Só o Mk** para uma versão anterior: não existe. O `mk-update` sempre instala a ponta do canal e recusa data mais antiga; o conserto de um lançamento ruim é outro lançamento ([[Canal de lançamentos]]).

### Quando o submenu não existe

Se os ganchos do limine ficaram fora da ISO, o submenu só aparece depois que o `mk-update` os instala pelo `yay` ([[AUR na instalação]]).

## Onde está no código

- `docs/atualizacoes.md` (seção "Snapshots e voltar atrás")
- `installer/mk-install`: `configure_boot` (`/etc/default/limine`), `configure_snapper`

## Ligações

Depende de: [[Snapper e snap-pac]], [[Ganchos do limine]], [[Limine]]. Relacionado: [[Quando a atualização para]], [[Limitações conhecidas]].
