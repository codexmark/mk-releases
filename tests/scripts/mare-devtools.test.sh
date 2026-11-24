#!/usr/bin/env bash
# shell/mare/scripts/mare-devtools: onde procura o mise-tools.sh, os ramos de
# erro e o código de saída. Roda numa cópia dentro de $SBX: o mise-tools.sh
# de verdade nunca é executado.
. "$(dirname "$0")/../lib.sh"

# A mesma disposição do repositório: <raiz>/shell/mare/scripts/mare-devtools
# e <raiz>/scripts/user/mise-tools.sh.
raiz=$SBX/mk
dev_sh=$raiz/shell/mare/scripts/mare-devtools
share=$SBX/share
mkdir -p "$raiz/shell/mare/scripts" "$raiz/scripts/user" "$share/scripts/user"
cp "$REPO/shell/mare/scripts/mare-devtools" "$dev_sh"

# Um mise-tools.sh falso: anota quem rodou e sai com o código pedido.
falso() { # falso <arquivo> <marca> [saída]
  printf '#!/usr/bin/env bash\necho "instalando (%s)"\necho "%s" >>"%s/rodou"\nexit %s\n' "$2" "$2" "$SBX" "${3:-0}" >"$1"
}
# Sempre com a entrada fechada: sem terminal, não espera o Enter.
run() { MK_SHARE=$share bash "$dev_sh" </dev/null; }
# Um PATH só com o básico do sistema, os stubs e os guardas, sem os comandos
# dados: para o ramo em que o mise não está instalado.
path_sem() {
  local d c p
  d=$(mktemp -d "$SBX/sem.XXXXXX")
  mkdir "$d/s" "$d/g"
  for c in bash sh env cat rm mkdir dirname basename; do
    p=$(PATH=/usr/bin:/bin command -v "$c") && ln -s "$p" "$d/s/$c"
  done
  cp "$SBX"/guard/* "$d/g/"
  for c; do rm -f "$d/g/$c"; done
  echo "$SBX/bin:$d/g:$d/s"
}
zera() {
  rm -f "$SBX/rodou" "$share/scripts/user/mise-tools.sh" "$raiz/scripts/user/mise-tools.sh"
  calls_reset
}

t "no repositório, o mise-tools.sh está onde o script procura (ao lado de shell/)"
exists "$REPO/scripts/user/mise-tools.sh"
eq "$REPO/scripts/user/mise-tools.sh" "$(cd "$REPO/shell/mare/scripts/../../../scripts/user" && pwd)/mise-tools.sh"

t "sem o mise-tools.sh em lugar nenhum: avisa e sai com 1"
zera
out=$(run)
eq 1 $?
has "$out" "não achei o scripts/user/mise-tools.sh"
absent "$SBX/rodou"

t "instalado (MK_SHARE): roda o mise-tools.sh de lá e diz 'pronto'"
zera
falso "$share/scripts/user/mise-tools.sh" instalado
out=$(run)
eq 0 $?
eq "instalado" "$(<"$SBX/rodou")"
has "$out" "Mk · ferramentas de desenvolvimento (mise)"
has "$out" "instalando (instalado)"
has "$out" "pronto. Abra um terminal novo"

t "numa cópia do repositório, sem /usr/share/mk: roda o que está ao lado de shell/"
zera
falso "$raiz/scripts/user/mise-tools.sh" repositorio
out=$(run)
eq 0 $?
eq "repositorio" "$(<"$SBX/rodou")"
has "$out" "pronto."

t "com os dois, vale o instalado, e roda uma vez só"
zera
falso "$share/scripts/user/mise-tools.sh" instalado
falso "$raiz/scripts/user/mise-tools.sh" repositorio
run >/dev/null
eq "instalado" "$(<"$SBX/rodou")"

t "o mise-tools.sh falhando: diz o código, como tentar de novo, e sai com o mesmo código"
zera
falso "$share/scripts/user/mise-tools.sh" instalado 3
out=$(run)
eq 3 $?
has "$out" "parou com erro (saída 3). Para tentar de novo: mk ferramentas"
hasnt "$out" "pronto."

t "sem o mise instalado: diz como instalar, não roda nada e sai com 1"
zera
falso "$share/scripts/user/mise-tools.sh" instalado
out=$(PATH=$(path_sem mise) MK_SHARE=$share bash "$dev_sh" </dev/null)
eq 1 $?
has "$out" "falta o mise: sudo pacman -S --needed mise"
absent "$SBX/rodou"

t "não chama sudo, pacman nem o mise por conta própria"
zera
falso "$share/scripts/user/mise-tools.sh" instalado
run >/dev/null
eq "" "$(<"$SBX/calls")"

t "sem terminal na entrada, não fica esperando o Enter"
zera
falso "$share/scripts/user/mise-tools.sh" instalado
out=$(MK_SHARE=$share timeout 10 bash "$dev_sh" </dev/null)
eq 0 $?
hasnt "$out" "Enter fecha"

t "MK_SHARE com espaço no caminho"
zera
mkdir -p "$SBX/com espaco/scripts/user"
falso "$SBX/com espaco/scripts/user/mise-tools.sh" espaco
MK_SHARE="$SBX/com espaco" bash "$dev_sh" </dev/null >/dev/null
eq 0 $?
eq "espaco" "$(<"$SBX/rodou")"
