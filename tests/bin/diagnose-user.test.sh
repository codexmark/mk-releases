#!/usr/bin/env bash
# bin/diagnose-user: junta o diagnóstico de um usuário sem alterar a home
# dele, com a saída numa raiz falsa (MK_DIAG_ROOT). id, getent, ps e chown são
# stubs.
. "$(dirname "$0")/../lib.sh"

R=$SBX/raiz
casa=$R/home/mare
saida=$R/tmp/mk-diagnostico
diag=$REPO/bin/diagnose-user

stub getent '[[ $1 == passwd ]] && grep "^$2:" "$SBX/passwd"'
stub id 'grep -q "^${*: -1}:" "$SBX/passwd"'
stub ps 'echo "    PID     ELAPSED CMD"; echo "   4242       01:02 qs -p /home/mare/.config/mk/shell"'
stub chown

# Uma home com o Mk, um registro do Maré, crashes e capturas de tela.
zera() {
  rm -rf "${R:?}"
  mkdir -p "$casa"/.config/mk/{shell,hypr} "$casa/.config/quickshell" "$casa/.cache/mk" \
    "$casa/.local/state/quickshell/by-id" "$casa/Pictures/Wallpapers" "$casa/Pictures/Screenshots"
  echo "mare:x:1001:1001::$casa:/bin/bash" >"$SBX/passwd"
  echo "import Quickshell" >"$casa/.config/mk/shell/shell.qml"
  echo "-- config" >"$casa/.config/mk/hypr/hyprland.lua"
  for i in $(seq 1 50); do echo "linha $i do mare.log"; done >"$casa/.cache/mk/mare.log"
  calls_reset
}
imagens() { find "$saida" -maxdepth 1 -type f \( -iname '*.png' -o -iname '*.jpg' \) -printf '%f\n' | sort; }
diagnostica() { MK_DIAG_ROOT=$R bash "$diag" "$@" 2>&1; }
arvore() { (cd "$casa" && find . | sort && find . -type f -exec md5sum {} + | sort && find . -printf '%p %T@\n' | sort); }

if ((EUID != 0)); then
  t "sem root (e fora da raiz de teste): pede o sudo e sai com 1"
  zera
  out=$(bash "$diag" mare 2>&1)
  eq 1 $?
  has "$out" "rode com sudo"
  eq "" "$(cat "$SBX/calls")"
fi

t "usuário que não existe: diz qual, sai com 2 e não apaga o diagnóstico anterior"
zera
mkdir -p "$saida"
echo "anterior" >"$saida/relatorio.txt"
out=$(diagnostica fulano)
eq 2 $?
has "$out" "usuário não existe: fulano"
eq anterior "$(cat "$saida/relatorio.txt")"

t "sem nome: também sai com 2"
diagnostica >/dev/null
eq 2 $?

t "o relatório tem a config do Maré, do Hyprland, o fim do registro e os processos"
zera
out=$(diagnostica mare)
eq 0 $?
rel=$(cat "$saida/relatorio.txt")
has "$rel" "== ~/.config/mk/shell (o Maré)"
has "$rel" "shell.qml"
has "$rel" "== ~/.config/mk/hypr"
has "$rel" "hyprland.lua"
has "$rel" "== ~/.config/quickshell"
has "$rel" "== processos de mare agora"
has "$rel" "qs -p /home/mare/.config/mk/shell"
called "ps -u mare -o pid,etime,cmd"
has "$out" "Pronto: $saida"

t "do mare.log, o relatório traz só as 40 últimas linhas; o arquivo inteiro vai junto"
has "$rel" "linha 50 do mare.log"
has "$rel" "linha 11 do mare.log"
hasnt "$rel" "linha 10 do mare.log"
ok cmp -s "$casa/.cache/mk/mare.log" "$saida/mare.log"

t "sem SUDO_USER, a saída fica com o root"
called "chown -R root: $saida"

t "com SUDO_USER, a saída fica legível por quem chamou o sudo"
calls_reset
SUDO_USER=dona diagnostica mare >/dev/null
called "chown -R dona: $saida"

t "não altera nada na home do usuário"
zera
antes=$(arvore)
diagnostica mare >/dev/null
eq "$antes" "$(arvore)"

t "rodar de novo dá a mesma saída, e o que era do diagnóstico anterior some"
primeira=$(cd "$saida" && find . -type f -exec md5sum {} + | sort)
echo "sobra" >"$saida/de-outra-vez.txt"
diagnostica mare >/dev/null
eq 0 $?
eq "$primeira" "$(cd "$saida" && find . -type f -exec md5sum {} + | sort)"
absent "$saida/de-outra-vez.txt"

t "crashes do Quickshell: copia os 4 relatórios de texto mais recentes, com a pasta no nome"
zera
for i in 1 2 3 4 5; do
  mkdir -p "$casa/.cache/quickshell/crashes/c$i"
  echo "crash $i" >"$casa/.cache/quickshell/crashes/c$i/info.txt"
  touch -d "$i hours ago" "$casa/.cache/quickshell/crashes/c$i/info.txt"
done
echo "binário" >"$casa/.cache/quickshell/crashes/c1/minidump.dmp"
diagnostica mare >/dev/null
eq "crash-c1-info.txt
crash-c2-info.txt
crash-c3-info.txt
crash-c4-info.txt" "$(cd "$saida" && printf '%s\n' crash-*)"
eq "crash 1" "$(cat "$saida/crash-c1-info.txt")"
has "$(cat "$saida/relatorio.txt")" "crashes/c5/info.txt"

t "capturas de tela: as 3 imagens mais recentes da home, das últimas 4 horas"
zera
for i in 1 2 3 4; do
  echo "captura $i" >"$casa/Pictures/Screenshots/captura$i.png"
  touch -d "$i minutes ago" "$casa/Pictures/Screenshots/captura$i.png"
done
echo "velha" >"$casa/Pictures/Screenshots/velha.png"
touch -d "5 hours ago" "$casa/Pictures/Screenshots/velha.png"
echo "foto" >"$casa/foto.JPG"
touch -d "10 minutes ago" "$casa/foto.JPG"
diagnostica mare >/dev/null
eq "captura1.png
captura2.png
captura3.png" "$(imagens)"

t "capturas de tela: papéis de parede, .config, .cache e .local não contam"
zera
echo "papel" >"$casa/Pictures/Wallpapers/fundo.png"
echo "icone" >"$casa/.config/mk/icone.png"
echo "miniatura" >"$casa/.cache/mk/miniatura.png"
mkdir -p "$casa/.local/share"
echo "dado" >"$casa/.local/share/dado.png"
echo "captura" >"$casa/Pictures/tela.png"
diagnostica mare >/dev/null
eq "tela.png" "$(imagens)"

t "home sem nada do Mk: ainda assim termina com 0 e deixa o relatório dizendo o que faltou"
zera
rm -rf "${casa:?}"
mkdir -p "$casa"
out=$(diagnostica mare)
eq 0 $?
exists "$saida/relatorio.txt"
has "$(cat "$saida/relatorio.txt")" "== ~/.config/mk/shell (o Maré)"
absent "$saida/mare.log"
eq "relatorio.txt" "$(ls "$saida")"
