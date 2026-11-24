#!/usr/bin/env bash
# A tela de entrada nova: a marca "Mk" construída em movimento (eixos, guias,
# diagonais, letras e nós), no lugar do arco. O tema do sddm é trocado
# inteiro; o theme.conf.user da máquina (o teclado que o instalador gravou, ou
# um motion=nao posto à mão) fica como estava.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

themes=$R/usr/share/sddm/themes
novo=$share/system/mk/sddm/mk
# Sem o tema do Mk no sddm (a máquina usa outra entrada), nada a fazer.
[[ -d $themes/mk ]] || exit 0
if [[ ! -f $novo/Main.qml ]]; then
  echo "aviso: o tema novo da entrada não veio neste lançamento; a entrada fica como está" >&2
  exit 0
fi

# Copia ao lado e troca de uma vez: o sddm nunca vê o tema pela metade.
rm -rf "${themes:?}/mk.novo" "${themes:?}/mk.velho"
cp -r "$novo" "$themes/mk.novo"
if [[ -f $themes/mk/theme.conf.user ]]; then
  cp -p "$themes/mk/theme.conf.user" "$themes/mk.novo/theme.conf.user"
fi
mv "$themes/mk" "$themes/mk.velho"
mv "$themes/mk.novo" "$themes/mk"
rm -rf "${themes:?}/mk.velho"
