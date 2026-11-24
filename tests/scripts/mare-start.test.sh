#!/usr/bin/env bash
# shell/mare/scripts/mare-start: a escala lida do ajustes.json, como o qs é
# lançado e o --restart (fechar, iniciar, devolver o teclado à janela). Roda
# numa cópia dentro de $SBX, com qs, hyprctl e sleep por stub.
. "$(dirname "$0")/../lib.sh"

# A disposição de uma instalação: <raiz>/ajustes.json e <raiz>/mare/scripts.
raiz=$SBX/mk
shell=$raiz/mare
start_sh=$shell/scripts/mare-start
log=$XDG_CACHE_HOME/mk/mare.log
mkdir -p "$shell/scripts"
cp "$REPO/shell/mare/scripts/mare-start" "$start_sh"

# qs: anota a escala que recebeu e escreve nas duas saídas (vão para o log).
stub qs 'echo "escala=${QT_SCALE_FACTOR:-}" >>"$SBX/qs.env"; echo "saída do qs"; echo "erro do qs" >&2'
stub hyprctl '[[ "$*" != "activewindow -j" ]] || cat "$SBX/janela.json" 2>/dev/null'
# As esperas do --restart (0,4 s, 1,5 s e 2 s) não precisam correr de verdade.
stub sleep
dorme() { command -p sleep "$1"; }

run() { bash "$start_sh" "$@"; }
escala() { tail -n 1 "$SBX/qs.env"; }
ajustes() { printf '%s' "$1" >"$raiz/ajustes.json"; }
zera() {
  rm -rf "$raiz/ajustes.json" "$SBX/qs.env" "$SBX/janela.json" "${XDG_CACHE_HOME:?}/mk"
  calls_reset
}
# Espera o pedaço do --restart que fica em segundo plano terminar.
assenta() {
  for _ in $(seq 1 50); do
    [[ $(grep -c '^sleep ' "$SBX/calls") -ge 3 && $(grep -c '^hyprctl activewindow' "$SBX/calls") -ge 2 ]] && break
    dorme 0.1
  done
  dorme 0.2
}

t "sem ajustes.json: inicia o Maré na escala de fábrica (0.85)"
zera
ok run
eq "escala=0.85" "$(escala)"

t "inicia o qs apontando para a pasta do shell (a de cima de scripts/)"
called "qs -p $shell"
eq 1 "$(grep -c '^qs ' "$SBX/calls")"

t "sem --restart: não fecha nada, não espera e não mexe no foco"
not_called "qs kill"
not_called "hyprctl"
not_called "sleep"

t "a saída e os erros do qs vão para ~/.cache/mk/mare.log, não para quem chamou"
zera
out=$(run 2>&1)
eq "" "$out"
has "$(<"$log")" "saída do qs"
has "$(<"$log")" "erro do qs"

t "ui.scale do ajustes.json vira a escala do Qt"
for v in 1 1.25 0.7 2; do
  zera
  ajustes "{\"ui.scale\": $v, \"power.sleep\": \"suspender\"}"
  run
  eq "escala=$v" "$(escala)" "ui.scale $v"
done

t "ui.scale guardado como texto (\"1.1\") também vale"
zera
ajustes '{"ui.scale": "1.1"}'
run
eq "escala=1.1" "$(escala)"

t "ajustes.json sem ui.scale, ou com null: escala de fábrica"
zera
ajustes '{"power.sleep": "nada"}'
run
eq "escala=0.85" "$(escala)"
ajustes '{"ui.scale": null}'
run
eq "escala=0.85" "$(escala)"

t "ajustes.json ilegível ou vazio: escala de fábrica, e o Maré inicia assim mesmo"
zera
ajustes '{"ui.scale": 1.5'
ok run
eq "escala=0.85" "$(escala)"
ajustes ''
ok run
eq "escala=0.85" "$(escala)"
called "qs -p $shell"

t "ui.scale que não é um número (texto, negativo, lista, comando) cai na escala de fábrica"
for v in '"grande"' '-1' '[1.5]' '{"x":1}' 'true' '"1.5; rm -rf x"' '"$(id)"' '"1,25"' '"1.2.3"' '"."' '".5."'; do
  zera
  ajustes "{\"ui.scale\": $v}"
  run
  eq "escala=0.85" "$(escala)" "ui.scale $v"
done

t "--restart: fecha o Maré que está rodando antes de iniciar o novo"
zera
echo '{"address":"0x55d0c0ffee","class":"kitty"}' >"$SBX/janela.json"
ok run --restart
assenta
called "qs kill -p $shell"
eq "kill -p" "$(grep '^qs ' "$SBX/calls" | head -1 | cut -d' ' -f2,3)"
eq "qs -p $shell" "$(grep '^qs ' "$SBX/calls" | tail -1)"
eq 2 "$(grep -c '^qs ' "$SBX/calls")"

t "--restart: dá um tempo (0,4 s) entre fechar e iniciar"
called "sleep 0.4"

t "--restart: depois de renascer, devolve o teclado à janela que estava em uso, duas vezes"
called "sleep 1.5"
called "sleep 2"
eq 2 "$(grep -c '^hyprctl activewindow -j$' "$SBX/calls")"
eq 2 "$(grep -cF 'hyprctl dispatch hl.dsp.focus({ window = "address:0x55d0c0ffee" })' "$SBX/calls")"

t "--restart: usa a escala do ajustes.json como o início comum"
zera
ajustes '{"ui.scale": 1.25}'
echo '{}' >"$SBX/janela.json"
run --restart
assenta
eq "escala=1.25" "$(escala)"

t "--restart: sem janela em foco, não pede foco a ninguém"
eq 2 "$(grep -c '^hyprctl activewindow -j$' "$SBX/calls")"
not_called "hyprctl dispatch"

t "--restart: resposta do hyprctl que não traz um endereço 0x... não vira comando"
zera
echo '{"address":"janela\" }) os.execute(\"x\") --"}' >"$SBX/janela.json"
run --restart
assenta
not_called "hyprctl dispatch"
echo 'isto não é JSON' >"$SBX/janela.json"
calls_reset
run --restart 2>/dev/null
assenta
not_called "hyprctl dispatch"

t "outro argumento qualquer inicia normalmente, sem fechar nada"
zera
ok run --bobagem
called "qs -p $shell"
not_called "qs kill"
not_called "hyprctl"

t "pasta do shell com espaço no caminho"
zera
mkdir -p "$SBX/com espaco/mare/scripts"
cp "$start_sh" "$SBX/com espaco/mare/scripts/mare-start"
printf '{"ui.scale": 1.5}' >"$SBX/com espaco/ajustes.json"
ok bash "$SBX/com espaco/mare/scripts/mare-start"
eq "escala=1.5" "$(escala)"
called "qs -p $SBX/com espaco/mare"
