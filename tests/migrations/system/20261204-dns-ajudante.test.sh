#!/usr/bin/env bash
# migrations/system/20261204-dns-ajudante.sh: o ajudante do DNS dos Ajustes
# (system/mk/mk-dns) copiado para /usr/lib/mk, do root e executável.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261204-dns-ajudante.sh

dst=$R/usr/lib/mk/mk-dns

t "não pede pacote: o NetworkManager e o polkit já vêm na base"
eq "" "$(sed -n 's/^# pacotes: //p' "$MIG")"
ok grep -qx networkmanager "$REPO/system/packages/base.packages"

t "copia o ajudante para /usr/lib/mk, com modo 755"
ok mig
ok cmp -s "$REPO/system/mk/mk-dns" "$dst"
eq 755 "$(stat -c %a "$dst")"
eq "" "$(cat "$ERR")"

t "não troca o DNS de ninguém: nenhum arquivo em /etc, nenhum comando do NetworkManager"
absent "$R/etc"
not_called "nmcli"
not_called "systemctl"

de_novo

t "um ajudante antigo ou mexido é trocado pelo do Mk instalado"
echo "#!/bin/bash" >"$dst"
chmod 777 "$dst"
ok mig
ok cmp -s "$REPO/system/mk/mk-dns" "$dst"
eq 755 "$(stat -c %a "$dst")"

t "um link no lugar do ajudante não é seguido"
echo "alvo" >"$R/alvo"
ln -sf "$R/alvo" "$dst"
ok mig
eq "alvo" "$(cat "$R/alvo")"
ok test ! -L "$dst"
ok cmp -s "$REPO/system/mk/mk-dns" "$dst"

t "um Mk instalado sem o ajudante (versão antiga em /usr/share/mk): avisa e sai com 0"
rm -f "$SHARE/system/mk/mk-dns" "$dst"
ok mig
has "$(cat "$ERR")" "aviso"
absent "$dst"
