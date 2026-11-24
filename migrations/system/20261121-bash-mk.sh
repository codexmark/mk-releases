#!/usr/bin/env bash
# O bash novo do Mk: o ble.sh (do AUR, já compilado no pendrive ou pelo yay
# depois) e o 7zip; e o seu.bash no skel dos usuários novos. O 7zip quem
# instala é o mk-update, junto com a atualização do sistema.
# pacotes: 7zip
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)
skel=$R/etc/skel

[[ -e $skel/.config/mk/seu.bash ]] || install -Dm644 "$share/config/bash/seu.bash.exemplo" "$skel/.config/mk/seu.bash"
rm -rf "$skel/.config/mk/bash"
cp -r "$share/config/bash" "$skel/.config/mk/bash"
touch "$skel/.config/mk/bash/.instalado-pelo-mk"
# O ble.sh vem do AUR: o mk-update instala depois das migrações (é um pacote
# do AUR que ainda não está instalado). É o 0.4 (blesh-git), o que tem as
# cores do Mk; quem tem o 0.3 troca na 20261122-boot-entrada.
if ! pacman -Qq blesh >/dev/null 2>&1 && ! pacman -Qq blesh-git >/dev/null 2>&1 &&
  ! grep -qxE 'blesh(-git)?' "$share/aur-pendentes.txt" 2>/dev/null; then
  echo blesh-git >>"$share/aur-pendentes.txt"
fi
