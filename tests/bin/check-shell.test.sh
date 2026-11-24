#!/usr/bin/env bash
# bin/check-shell: o teste de fumaça do bash do Mk, com um mk.bash falso numa
# cópia do repositório em $SBX (o bash interativo roda num pseudo-terminal).
. "$(dirname "$0")/../lib.sh"

falso=$SBX/repo
mkdir -p "$falso"/{bin,config/bash}
cp "$REPO/bin/check-shell" "$falso/bin/check-shell"
confere() { bash "$falso/bin/check-shell" 2>&1; }

# Um PATH sem o script(1): só o que o check-shell usa antes de procurar por ele.
mkdir -p "$SBX/min"
for c in bash env dirname; do
  ln -s "$(PATH=/usr/bin:/bin command -v "$c")" "$SBX/min/$c"
done

t "sem o script (util-linux): diz que precisa dele e sai com 2"
echo ": # nada" >"$falso/config/bash/mk.bash"
out=$(PATH=$SBX/min confere)
eq 2 $?
has "$out" "check-shell: precisa do script"

command -v script >/dev/null || skip "falta o script (util-linux)"

t "mk.bash são: o bash sobe, responde e sai; saída 0"
cat >"$falso/config/bash/mk.bash" <<'RC'
# Um mk.bash que só anota com que TERM e MK_NO_FETCH foi carregado.
echo "$TERM fetch=${MK_NO_FETCH:-}" >>"$SBX/carregou"
alias ll='ls -l'
RC
out=$(confere)
eq 0 $?
has "$out" "bash do Mk: sobe e sai limpo"

t "carrega o mk.bash deste repositório nos dois terminais, sem a ficha do fastfetch"
eq "xterm-256color fetch=1
xterm-ghostty fetch=1" "$(cat "$SBX/carregou" 2>/dev/null)"

t "não deixa o arquivo temporário para trás"
eq 0 "$(find "$TMPDIR" -mindepth 1 -maxdepth 1 | wc -l)"

t "mk.bash que derruba o shell (exit): acusa nos dois terminais e sai com 1"
echo 'exit 3' >"$falso/config/bash/mk.bash"
out=$(confere)
eq 1 $?
has "$out" "o bash interativo caiu (TERM=xterm-256color"
has "$out" "o bash interativo caiu (TERM=xterm-ghostty"
hasnt "$out" "sobe e sai limpo"

t "mk.bash que só cai num dos terminais: acusa esse, e sai com 1"
echo '[[ $TERM == xterm-ghostty ]] && exit 3' >"$falso/config/bash/mk.bash"
out=$(confere)
eq 1 $?
hasnt "$out" "TERM=xterm-256color"
has "$out" "o bash interativo caiu (TERM=xterm-ghostty"

t "uma queda isolada (a máquina ocupada, o terminal que não respondeu a tempo) não reprova: tenta de novo"
# Visto num lançamento: com a máquina carregada o bash subiu, saiu com 0 e a
# resposta não veio. Na tentativa seguinte passou. Um mk.bash quebrado de
# verdade cai em todas.
rm -f "$SBX/carregou"
cat >"$falso/config/bash/mk.bash" <<'RC'
echo x >>"$SBX/carregou"
# Cai só na primeira vez em que é carregado.
[[ $(wc -l <"$SBX/carregou") -eq 1 ]] && exit 0
RC
out=$(confere)
eq 0 $?
has "$out" "bash do Mk: sobe e sai limpo"
ok test "$(wc -l <"$SBX/carregou")" -ge 3

t "mk.bash quebrado de verdade cai em todas as tentativas: sai com 1, e diz quantas foram"
echo 'exit 3' >"$falso/config/bash/mk.bash"
out=$(confere)
eq 1 $?
has "$out" "3 tentativas"

t "mk.bash com um comando que falha, mas não derruba o shell: passa"
echo 'false; comando-que-nao-existe 2>/dev/null; true' >"$falso/config/bash/mk.bash"
confere >/dev/null
eq 0 $?

t "mk.bash que troca a saída do shell por erro (trap no EXIT): sai com 1"
echo "trap 'exit 9' EXIT" >"$falso/config/bash/mk.bash"
confere >/dev/null
eq 1 $?

t "pode ser chamado de qualquer pasta"
echo ": # nada" >"$falso/config/bash/mk.bash"
(cd / && bash "$falso/bin/check-shell" >/dev/null 2>&1)
eq 0 $?
