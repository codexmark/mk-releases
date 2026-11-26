#!/usr/bin/env bash
# system/packages/*: o formato das listas de pacotes e o que uma promete à
# outra (sem repetição, nada do AUR nas listas dos repositórios). Não consulta
# a rede nem o pacman: só os arquivos.
. "$(dirname "$0")/../lib.sh"

export PKG_DIR=$REPO/system/packages
# shellcheck source=/dev/null
source "$REPO/bin/lib-packages.sh"
repos=(base desktop dev apps)

t "as seis listas existem e nenhuma está vazia"
for l in "${repos[@]}" aur; do
  exists "$PKG_DIR/$l.packages"
  ok test "$(read_list "$PKG_DIR/$l.packages" | wc -l)" -ge 1
done
exists "$PKG_DIR/hardware.packages"
eq "apps.packages aur.packages base.packages desktop.packages dev.packages hardware.packages" \
  "$(cd "$PKG_DIR" && echo *)"

t "formato: um pacote por linha (nome válido do pacman); o resto é comentário ou linha vazia"
for l in "${repos[@]}" aur; do
  n=0
  while IFS= read -r linha; do
    n=$((n + 1))
    [[ $linha =~ ^[[:space:]]*(#|$) ]] && continue
    [[ $linha =~ ^[a-z0-9][a-z0-9@._+-]*$ ]] || _fail "$l.packages:$n: '$linha'"
  done <"$PKG_DIR/$l.packages"
done

t "formato: nenhum comentário na mesma linha de um pacote, nem espaço sobrando"
for l in "${repos[@]}" aur hardware; do
  eq "" "$(grep -n -e '^[^#].*#' -e '[[:space:]]$' -e '^[[:space:]]\+[^[:space:]]' "$PKG_DIR/$l.packages")" "$l.packages"
done

t "formato: toda lista começa com um comentário dizendo para que serve e termina com quebra de linha"
for l in "${repos[@]}" aur hardware; do
  like "$(head -1 "$PKG_DIR/$l.packages")" '^# .+' "$l.packages"
  eq "" "$(tail -c 1 "$PKG_DIR/$l.packages")" "$l.packages termina sem quebra de linha"
done

t "read_list: tira comentários e linhas vazias e devolve só nomes"
printf '%s\n' '# cabeçalho' '' 'um' '  # recuado' 'dois  ' 'três # no fim' >"$SBX/x.packages"
eq "um
dois
três" "$(read_list "$SBX/x.packages")"

t "nenhum pacote repetido dentro da mesma lista"
for l in "${repos[@]}" aur; do
  eq "" "$(read_list "$PKG_DIR/$l.packages" | sort | uniq -d)" "$l.packages"
done

t "nenhum pacote repetido entre base, desktop e dev (as três entram sempre, juntas)"
eq "" "$(for l in base desktop dev; do read_list "$PKG_DIR/$l.packages"; done | sort | uniq -d)"

t "apps só repete de outra lista o que o desktop não pode dispensar (mpv, ffmpeg e yt-dlp, do MkTunes)"
eq "ffmpeg
mpv
yt-dlp" "$(comm -12 <(read_list "$PKG_DIR/apps.packages" | sort) \
  <(for l in base desktop dev; do read_list "$PKG_DIR/$l.packages"; done | sort))"
has "$(<"$PKG_DIR/desktop.packages")" "mpv, ffmpeg e yt-dlp também estão em"

t "nenhum pacote do AUR nas listas dos repositórios"
for l in "${repos[@]}"; do
  eq "" "$(comm -12 <(read_list "$PKG_DIR/$l.packages" | sort) <(read_list "$PKG_DIR/aur.packages" | sort))" "$l.packages"
done
eq "" "$(comm -12 <(read_hardware amd intel nvidia vm | sort) <(read_list "$PKG_DIR/aur.packages" | sort))" "hardware.packages"

t "nenhum nome com cara de AUR (-bin, -git) nas listas dos repositórios"
for l in "${repos[@]}"; do
  eq "" "$(read_list "$PKG_DIR/$l.packages" | grep -e '-bin$' -e '-git$' || true)" "$l.packages"
done
eq "" "$(read_hardware amd intel nvidia vm | grep -e '-bin$' -e '-git$' || true)"

t "aur: traz o que o instalador e o README contam com ele (yay e os ganchos do limine)"
aur=$(read_list "$PKG_DIR/aur.packages")
for p in yay-bin limine-mkinitcpio-hook limine-snapper-sync ufw-docker; do ok grep -qx "$p" <<<"$aur"; done

t "aur: o helper (yay) vem primeiro, para compilar o resto"
eq yay-bin "$(head -1 <<<"$aur")"

t "base: traz o que o instalador configura (boot, snapshots, rede, firewall, impressão)"
base=$(read_list "$PKG_DIR/base.packages")
for p in base linux linux-firmware mkinitcpio sudo btrfs-progs cryptsetup dosfstools efibootmgr limine plymouth \
  snapper snap-pac networkmanager avahi nss-mdns cups ufw bluez power-profiles-daemon jq; do
  ok grep -qx "$p" <<<"$base"
done

t "desktop e dev: trazem o que as unidades e os scripts do instalador esperam"
resto=$(read_list "$PKG_DIR/desktop.packages" && read_list "$PKG_DIR/dev.packages")
for p in hyprland quickshell sddm uwsm gcr-4 docker plocate neovim git mise gum; do ok grep -qx "$p" <<<"$resto"; done

t "hardware: as seções são exatamente os perfis que o instalador aceita"
eq "amd intel nvidia vm" "$(sed -n 's/^\[\(.*\)\]$/\1/p' "$PKG_DIR/hardware.packages" | tr '\n' ' ' | sed 's/ $//')"
has "$(<"$REPO/installer/mk-install")" '^(amd|intel|nvidia|vm)$'

t "hardware: formato [seção] seguida de pacotes; nada antes da primeira seção"
secao=""
n=0
while IFS= read -r linha; do
  n=$((n + 1))
  [[ $linha =~ ^[[:space:]]*(#|$) ]] && continue
  if [[ $linha =~ ^\[[a-z]+\]$ ]]; then
    secao=$linha
  elif [[ $linha =~ ^[a-z0-9][a-z0-9@._+-]*$ ]]; then
    [[ -n $secao ]] || _fail "hardware.packages:$n: pacote '$linha' fora de seção"
  else
    _fail "hardware.packages:$n: '$linha'"
  fi
done <"$PKG_DIR/hardware.packages"

t "read_hardware: cada perfil devolve os pacotes da própria seção, e só eles"
eq "amd-ucode
vulkan-radeon" "$(read_hardware amd)"
has "$(read_hardware intel)" "intel-ucode"
hasnt "$(read_hardware intel)" "amd-ucode"
has "$(read_hardware nvidia)" "nvidia-utils"
eq "vulkan-swrast" "$(read_hardware vm)"
for s in amd intel nvidia vm; do ok test -n "$(read_hardware "$s")"; done

t "read_hardware: vários perfis somam; nenhum ou um desconhecido não devolve nada"
eq "$(read_hardware amd; read_hardware nvidia)" "$(read_hardware amd nvidia)"
eq "" "$(read_hardware)"
eq "" "$(read_hardware radeon)"

t "hardware: nenhum pacote repetido entre seções, nem com as outras listas"
eq "" "$(read_hardware amd intel nvidia vm | sort | uniq -d)"
eq "" "$(comm -12 <(read_hardware amd intel nvidia vm | sort) \
  <(for l in "${repos[@]}"; do read_list "$PKG_DIR/$l.packages"; done | sort -u))"

t "o LibreOffice em português fica por conta do instalador, não da lista"
fails grep -qx libreoffice-fresh-pt-br <(read_list "$PKG_DIR/apps.packages")
ok grep -qx libreoffice-fresh <(read_list "$PKG_DIR/apps.packages")
