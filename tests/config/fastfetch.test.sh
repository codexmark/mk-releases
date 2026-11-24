#!/usr/bin/env bash
# config/fastfetch/config.jsonc: a ficha do terminal. JSON válido (fora os
# comentários), a marca no lugar e os rótulos em português.
. "$(dirname "$0")/../lib.sh"

dir=$REPO/config/fastfetch
command -v jq >/dev/null || skip "sem jq"
# Os comentários do jsonc são linhas inteiras começando por //.
json=$(grep -v '^[[:space:]]*//' "$dir/config.jsonc")

t "é JSON válido depois de tirar os comentários"
ok jq -e . <<<"$json" >/dev/null

t "os comentários são só linhas inteiras (nenhum no fim de uma linha de dados)"
eq "" "$(grep -v '^[[:space:]]*//' "$dir/config.jsonc" | grep -nE '[,{}\]"][[:space:]]*//[^"]*$' || true)"

t "sem imagem: a ficha é só texto, encostada à esquerda"
eq "none" "$(jq -r .logo.type <<<"$json")"
hasnt "$json" "kitty-direct"

t "a linha do Mk abre com duas setas de cobre, não com quadrados"
linha=$(jq -r '.modules[] | objects | select(.type == "command") | .format' <<<"$json" | grep 'MK · ')
has "$linha" $'\e[38;2;206;94;50m❯❯\e[0m'
hasnt "$json" "■"

t "três seções, cada uma com dois traços de cobre, o título e a régua"
titulos=$(jq -r '.modules[] | objects | select(.type == "custom") | .format' <<<"$json" | grep -c $'^\e\[38;2;206;94;50m── ──\e\[0m \e\[1;38;2;205;196;186m[A-ZÁÃÇ]* *\e\[0m \e\[38;2;58;54;50m─*\e\[0m$')
eq 3 "$titulos"

t "as réguas terminam todas na mesma coluna"
larguras=$(jq -r '.modules[] | objects | select(.type == "custom") | .format' <<<"$json" | grep '── ──' |
  sed $'s/\e\\[[0-9;]*m//g' | awk '{ print length($0) }' | sort -u | wc -l)
eq 1 "$larguras"

t "a máquina diz a frequência, o tipo da placa e o sistema de arquivos; os pacotes vêm por gerenciador"
has "$(jq -r '.modules[] | objects | select(.type == "cpu") | .format' <<<"$json")" "{freq-max}"
has "$(jq -r '.modules[] | objects | select(.type == "gpu") | .format' <<<"$json")" "{type}"
has "$(jq -r '.modules[] | objects | select(.type == "disk") | .format' <<<"$json")" "{filesystem}"
eq "null" "$(jq -r '.modules[] | objects | select(.type == "packages") | .format' <<<"$json")"

t "as cores são as do Mk: osso, os cinzas dele e o cobre"
while read -r c; do
  case $c in "205;196;186" | "176;169;160" | "149;143;135" | "128;122;115" | "113;107;105" | "58;54;50" | "206;94;50") ;; *) _fail "cor fora da paleta: $c" ;; esac
done < <(grep -oE '38;2;[0-9]+;[0-9]+;[0-9]+' <<<"$json" | sed 's/^38;2;//' | sort -u)

t "cada módulo é um nome ou um objeto com type"
eq "" "$(jq -r '.modules[] | if type == "string" then empty elif (type == "object" and (.type | type) == "string") then empty else "ruim: \(.)" end' <<<"$json")"

t "os rótulos estão em português e não se repetem"
chaves=$(jq -r '.modules[] | objects | .key // empty | select(. != " ")' <<<"$json")
eq "" "$(sort <<<"$chaves" | uniq -d)"
for k in CPU VÍDEO MEMÓRIA DISCO SISTEMA KERNEL JANELAS PACOTES LIGADO TERMINAL; do
  ok grep -qx "$k" <<<"$chaves"
done

t "os comandos da ficha não saem para a rede nem mexem no sistema"
cmds=$(jq -r '.modules[] | objects | select(.type == "command") | .text' <<<"$json")
eq 2 "$(wc -l <<<"$cmds")"
eq "" "$(grep -E '\b(curl|wget|sudo|rm|systemctl|ssh)\b' <<<"$cmds" || true)"

t "é a config que o mk.bash passa ao fastfetch"
has "$(cat "$REPO/config/bash/mk.bash")" 'fastfetch --config "$mk_cfg/fastfetch/config.jsonc"'

t "a pasta só tem a config: a imagem da marca saiu com o desenho antigo"
eq "config.jsonc" "$(find "$dir" -type f -printf '%f\n' | sort | tr '\n' ' ' | sed 's/ $//')"
hasnt "$(<"$REPO/system/mk/art/make-logo.py")" "config/fastfetch/mark.png"
