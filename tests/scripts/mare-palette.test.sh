#!/usr/bin/env bash
# shell/mare/scripts/mare-palette: a cor principal de uma imagem, com o
# matugen por stub.
. "$(dirname "$0")/../lib.sh"

pal_sh=$REPO/shell/mare/scripts/mare-palette
img=$SBX/papel.jpg
: >"$img"

# matugen: responde o que estiver em matugen.json; sem o arquivo, falha.
stub matugen 'cat "$SBX/matugen.json" 2>/dev/null || { echo "erro: imagem ilegível" >&2; exit 1; }'
echo '{"colors":{"primary":{"dark":{"color":"#ffb59d"},"light":{"color":"#8f4c38"}},"secondary":{"dark":{"color":"#e7bdb1"}}}}' >"$SBX/matugen.json"

run() { bash "$pal_sh" "$@"; }

t "imagem válida: imprime só a cor primary do tema escuro"
out=$(run "$img")
eq "#ffb59d" "$out"

t "pede ao matugen JSON em hex, tema escuro, preferindo saturação, sem aplicar nada (--dry-run)"
called "matugen image $img --json hex --dry-run -m dark --prefer saturation"

t "sem argumento: sai com 0, sem saída e sem chamar o matugen"
calls_reset
out=$(run)
eq 0 $?
eq "" "$out"
not_called "matugen"

t "arquivo que não existe, ou uma pasta: sai com 0 sem chamar o matugen"
out=$(run "$SBX/nao-existe.png")
eq 0 $?
eq "" "$out"
out=$(run "$SBX")
eq 0 $?
eq "" "$out"
not_called "matugen"

t "caminho com espaço, aspas e traço na frente vai inteiro num argumento só"
calls_reset
mkdir -p "$SBX/meus papéis"
hostil="$SBX/meus papéis/-foto \"nova\" \$(id).png"
: >"$hostil"
eq "#ffb59d" "$(run "$hostil")"
called "matugen image $hostil --json hex"

t "o matugen falhando (imagem ilegível): saída vazia, sem mensagem de erro"
rm -f "$SBX/matugen.json"
out=$(run "$img" 2>&1)
eq "" "$out"

t "resposta do matugen sem a cor esperada: saída vazia"
echo '{"colors":{"secondary":{"dark":{"color":"#e7bdb1"}}}}' >"$SBX/matugen.json"
eq "" "$(run "$img")"
echo '{"colors":{"primary":{"dark":{"color":null}}}}' >"$SBX/matugen.json"
eq "" "$(run "$img")"

t "resposta do matugen que não é JSON: saída vazia"
echo "matugen 3.0: opção desconhecida" >"$SBX/matugen.json"
eq "" "$(run "$img" 2>/dev/null)"
