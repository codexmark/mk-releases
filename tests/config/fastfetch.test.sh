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

t "a marca: imagem pelo protocolo do kitty, no caminho onde o Mk a instala, e o arquivo existe"
eq "kitty-direct" "$(jq -r .logo.type <<<"$json")"
has "$json" '"source": "~/.config/mk/fastfetch/mark.png"'
exists "$dir/mark.png"
eq "PNG" "$(head -c 4 "$dir/mark.png" | tail -c 3)"

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
