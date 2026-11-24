#!/usr/bin/env bash
# shell/mare/scripts/mare-cava: a configuração escrita para o cava e como ele
# é lançado (pkill do anterior, setpriv --pdeathsig), tudo por stub.
. "$(dirname "$0")/../lib.sh"

cava_sh=$REPO/shell/mare/scripts/mare-cava
conf=$XDG_RUNTIME_DIR/mare-cava.conf

# Um PATH só com o básico do sistema, os stubs e os guardas: para o ramo em
# que o cava não está instalado.
path_sem() {
  local d c p
  d=$(mktemp -d "$SBX/sem.XXXXXX")
  mkdir "$d/s" "$d/g"
  for c in bash sh env cat rm mv mkdir dirname basename; do
    p=$(PATH=/usr/bin:/bin command -v "$c") && ln -s "$p" "$d/s/$c"
  done
  cp "$SBX"/guard/* "$d/g/"
  for c; do rm -f "$d/g/$c"; done
  echo "$SBX/bin:$d/g:$d/s"
}
# shellcheck disable=SC2120  # os argumentos chegam por `ok run 12`
run() { bash "$cava_sh" "$@"; }
# O valor de uma chave numa seção do arquivo do cava.
chave() { # chave <seção> <nome>
  awk -v s="[$1]" -v k="$2" '
    /^\[/ { dentro = ($0 == s) }
    dentro && $1 == k && $2 == "=" { sub(/^[^=]*= /, ""); print }' "$conf"
}

t "sem o cava instalado: sai com 127, sem escrever configuração nem matar nada"
stub pkill
stub setpriv
PATH=$(path_sem cava) bash "$cava_sh" 2>/dev/null
eq 127 $?
absent "$conf"
eq "" "$(<"$SBX/calls")"

stub cava 'echo "12;40;7;"; exit "$(cat "$SBX/cava.rc" 2>/dev/null || echo 0)"'
# O setpriv falso só passa a vez ao comando, como o de verdade.
stub setpriv 'shift 2; exec "$@"'

t "sem argumento: 28 faixas, ouvindo o que sai nos alto-falantes"
calls_reset
ok run >/dev/null
eq 28 "$(chave general bars)"
eq pipewire "$(chave input method)"
eq auto "$(chave input source)"

t "a saída do cava é uma linha por quadro, em texto, de 0 a 100, separada por ';'"
eq raw "$(chave output method)"
eq /dev/stdout "$(chave output raw_target)"
eq ascii "$(chave output data_format)"
eq 100 "$(chave output ascii_max_range)"
eq 59 "$(chave output bar_delimiter)" "59 = ;"
eq 10 "$(chave output frame_delimiter)" "10 = quebra de linha"
eq mono "$(chave output channels)"

t "o arquivo tem as quatro seções e nada sobrando de fora delas"
eq "[general] [input] [output] [smoothing]" "$(grep '^\[' "$conf" | tr '\n' ' ' | sed 's/ $//')"
eq 0 "$(grep -vcE '^(\[[a-z]+\]|[a-z_]+ = [A-Za-z0-9/]+|)$' "$conf")"

t "a configuração fica na pasta da sessão (XDG_RUNTIME_DIR)"
exists "$XDG_RUNTIME_DIR/mare-cava.conf"

t "o cava roda com essa configuração, sob setpriv --pdeathsig (morre junto com o Maré)"
called "setpriv --pdeathsig TERM cava -p $conf"
called "cava -p $conf"

t "antes de lançar, mata o cava de um Maré anterior (só o desta configuração)"
called "pkill -f cava -p $conf"
eq "pkill setpriv cava" "$(awk '{ printf "%s%s", (NR > 1 ? " " : ""), $1 }' "$SBX/calls")"

t "o que o cava imprime sai direto, e a saída dele é a do script"
eq "12;40;7;" "$(run)"
echo 3 >"$SBX/cava.rc"
run >/dev/null
eq 3 $?
rm -f "$SBX/cava.rc"

t "sem cava anterior para matar (pkill sai com 1), segue normalmente"
calls_reset
stub pkill 'exit 1'
ok run >/dev/null
called "cava -p $conf"
stub pkill

t "o argumento escolhe o número de faixas"
ok run 12 >/dev/null
eq 12 "$(chave general bars)"
ok run 64 >/dev/null
eq 64 "$(chave general bars)"

t "MARE_CAVA_SOURCE escolhe de onde ouvir"
MARE_CAVA_SOURCE=alsa_output.usb-fone.monitor run >/dev/null
eq alsa_output.usb-fone.monitor "$(chave input source)"

t "cada chamada reescreve a configuração inteira (não acumula)"
ok run 9 >/dev/null
eq 1 "$(grep -c '^bars = ' "$conf")"
eq 9 "$(chave general bars)"

t "faixas que não são um número caem no padrão, sem injetar linhas na configuração"
for hostil in "muitas" "-5" "1.5" "" $'8\n[output]\nraw_target = /tmp/invadiu' '8; rm -rf x' '$(id)'; do
  ok run "$hostil" >/dev/null
  eq 28 "$(chave general bars)" "faixas: $hostil"
  eq /dev/stdout "$(chave output raw_target)" "faixas: $hostil"
  eq 1 "$(grep -c '^raw_target' "$conf")" "faixas: $hostil"
done
