#!/usr/bin/env bash
# config/ghostty/config: o terminal na direção Folha. Formato, paleta
# completa e as fontes que o Mk instala.
. "$(dirname "$0")/../lib.sh"

conf=$REPO/config/ghostty/config
corpo=$(grep -v '^#' "$conf" | grep -v '^[[:space:]]*$')
valor() { sed -n "s/^$1 = //p" <<<"$corpo"; }

t "cada linha é 'chave = valor'"
eq "" "$(grep -vE '^[a-z-]+ = [^[:space:]].*$' <<<"$corpo" || true)"

t "só as fontes repetem (a lista de reserva); nenhuma outra chave aparece duas vezes"
eq "font-family" "$(cut -d' ' -f1 <<<"$corpo" | grep -v '^palette$' | sort | uniq -d)"

t "a paleta tem as 16 cores, de 0 a 15, uma vez cada, em hexadecimal"
eq "$(seq 0 15)" "$(valor palette | cut -d= -f1 | sort -n)"
while read -r cor; do
  like "$cor" '^[0-9]+=#[0-9a-f]{6}$' "cor"
done < <(valor palette)

t "as cores Folha: fundo quase preto, texto em osso, cursor de cobre"
eq "#0f0e0d" "$(valor background)"
eq "#cdc4ba" "$(valor foreground)"
eq "#ce5e32" "$(valor cursor-color)"
eq "1=#ce5e32" "$(valor palette | grep '^1=')" "o vermelho é o cobre"

t "as fontes, na ordem, vêm nas listas de pacotes do Mk"
eq $'SpaceMono Nerd Font\nGeistMono Nerd Font\nJetBrainsMono Nerd Font' "$(valor font-family)"
pkgs=$(sed -e 's/#.*//' "$REPO"/system/packages/{base,desktop,dev}.packages)
for p in ttf-space-mono-nerd otf-geist-mono-nerd ttf-jetbrains-mono-nerd ghostty; do
  ok grep -qx "$p" <<<"$pkgs"
done

t "sem decoração própria (a moldura é do Hyprland) e sem pergunta ao fechar"
eq "false" "$(valor window-decoration)"
eq "false" "$(valor confirm-close-surface)"

t "é o arquivo que o terminal de emergência do Hyprland carrega"
has "$(cat "$REPO/config/hypr/mare/paths.lua")" 'ghostty = root .. "ghostty/config"'
has "$(cat "$REPO/config/hypr/mare/binds.lua")" "--config-file='?\" .. paths.ghostty"
