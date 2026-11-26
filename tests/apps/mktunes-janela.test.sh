#!/usr/bin/env bash
# apps/mktunes: o lançador (bin/mktunes) e a pasta do app. A janela (qml/) é a
# mesma da versão em Kotlin e se confere na tela; aqui: o lançador sobe o
# mktunesd quando ninguém responde no socket, troca um daemon de outra versão,
# encerra o daemon antigo (o da JVM), abre uma janela só e avisa na tela
# quando falta algo; a pasta não tem mais a JVM nem pede Python.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mktunes
lancador=$app/bin/mktunes

# O mundo do lançador: uma cópia de mentira da pasta do app, com um mktunesd
# que responde o que os arquivos em $casa/daemon mandam e anota o que pediram.
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml" "$casa/daemon"
: >"$casa/qml/shell.qml"
cat >"$casa/bin/mktunesd" <<DUBLE
#!/usr/bin/env bash
printf 'mktunesd %s\n' "\$*" >>"$SBX/calls"
d="$casa/daemon"
case "\${1:-}" in
  --versao) cat "\$d/versao" ;;
  --ping) [[ -e \$d/viva ]] || exit 1; cat "\$d/viva" ;;
  --encerra) rm -f "\$d/viva" ;;
  "") printf 'LAUNCHER=%s\n' "\${MKTUNES_LAUNCHER:-}" >>"$SBX/calls" ;;
esac
DUBLE
chmod +x "$casa/bin/mktunesd"
echo "0.2.0" >"$casa/daemon/versao"
# Um /proc de mentira, vazio: o de verdade nunca é olhado.
proc=$SBX/proc
mkdir -p "$proc"
export MKTUNES_HOME=$casa MKTUNES_PROC=$proc

stub mpv
stub notify-send
stub setsid 'shift; "$@"'
stub qs 'case "$1" in list) cat "'"$SBX"'/qs-lista" 2>/dev/null ;; esac'
stub hyprctl 'case "$1" in clients) cat "'"$SBX"'/clientes" 2>/dev/null ;; esac'

roda() {
  calls_reset
  rc=0
  out=$("$lancador" "$@" 2>&1) || rc=$?
}

t "sem daemon no socket: sobe o mktunesd (dizendo por onde a ilha abre a janela) e abre a janela"
roda
eq 0 "$rc" "$out"
called "mktunesd --ping"
called "setsid -f $casa/bin/mktunesd"
called "LAUNCHER=$lancador"
called "qs -p $casa/qml"
exists "$XDG_CACHE_HOME/mktunes/mktunesd.log"
not_called "--encerra"

t "o que vem depois do nome vai para a janela"
roda --um argumento
called "qs -p $casa/qml --um argumento"

t "com o daemon desta versão no ar: não sobe outro"
echo "0.2.0" >"$casa/daemon/viva"
roda
eq 0 "$rc" "$out"
not_called "setsid"
not_called "--encerra"
called "qs -p $casa/qml"

t "com um daemon de outra versão no ar (o Mk foi atualizado): ele é encerrado e o novo sobe"
echo "0.1.9" >"$casa/daemon/viva"
roda
eq 0 "$rc" "$out"
called "mktunesd --encerra"
called "setsid -f $casa/bin/mktunesd"
eq "mktunesd --ping|mktunesd --versao|mktunesd --encerra|mktunesd --ping|setsid -f $casa/bin/mktunesd" \
  "$(grep -E '^(mktunesd --|setsid)' "$SBX/calls" | paste -sd'|')"

t "um daemon de outra versão que não sai fica: a janela abre com ele, sem um segundo daemon"
cat >"$casa/bin/mktunesd.teimoso" <<DUBLE
#!/usr/bin/env bash
printf 'mktunesd %s\n' "\$*" >>"$SBX/calls"
case "\${1:-}" in --versao) echo 0.2.0 ;; --ping) echo 0.1.9 ;; --encerra) exit 1 ;; esac
DUBLE
chmod +x "$casa/bin/mktunesd.teimoso"
mv "$casa/bin/mktunesd" "$casa/bin/mktunesd.bom"
cp "$casa/bin/mktunesd.teimoso" "$casa/bin/mktunesd"
roda
eq 0 "$rc" "$out"
called "mktunesd --encerra"
not_called "setsid"
called "qs -p $casa/qml"
mv "$casa/bin/mktunesd.bom" "$casa/bin/mktunesd"
rm -f "$casa/daemon/viva"

t "o daemon antigo (o da JVM) que ficou no ar depois da atualização é encerrado; os outros processos não"
sleep 300 &
antigo=$!
sleep 300 &
outro=$!
mkdir -p "$proc/$antigo" "$proc/$outro" "$proc/self"
printf 'java\0-cp\0/usr/share/mk/apps/mktunes/daemon/lib/*\0dev.mk.mktunes.desktop.MainKt\0' >"$proc/$antigo/cmdline"
# Citar a classe no meio de outro argumento (um grep, um editor) não é ser o daemon.
printf 'grep\0-r\0x dev.mk.mktunes.desktop.MainKt y\0' >"$proc/$outro/cmdline"
roda
eq 0 "$rc" "$out"
fails kill -0 "$antigo"
ok kill -0 "$outro"
kill "$outro" 2>/dev/null
wait 2>/dev/null
called "setsid -f $casa/bin/mktunesd"
rm -rf "${proc:?}"/*

t "o lançador acha o daemon antigo sem pgrep nem pkill, e não cita o nome do programa da JVM"
src=$(<"$lancador")
hasnt "$src" "pgrep"
hasnt "$src" "pkill"
has "$src" 'classe_antiga=dev.mk.mktunes.desktop.MainKt'
has "$src" '[[ -O $dir && -r $dir/cmdline ]] || continue'
has "$src" '"${MKTUNES_PROC:-/proc}"/[0-9]*'

t "uma janela só: se já há uma aberta, ela é focada e nenhuma outra abre"
printf 'Instance abc:\n  Config path: %s/qml/shell.qml\n' "$casa" >"$SBX/qs-lista"
echo '[{"address":"0x55d0aa","title":"Outra"},{"address":"0x55d0bb","title":"MkTunes"}]' >"$SBX/clientes"
echo "0.2.0" >"$casa/daemon/viva"
roda
eq 0 "$rc" "$out"
called 'hyprctl dispatch hl.dsp.focus({ window = "address:0x55d0bb" })'
not_called "qs -p"

t "a janela de outro app com o mesmo título de aba não conta: sem a nossa na lista do qs, abre"
printf 'Instance abc:\n  Config path: /outro/app/qml/shell.qml\n' >"$SBX/qs-lista"
roda
called "qs -p $casa/qml"
not_called "hyprctl dispatch"

t "a lista diz que há janela mas o Hyprland não a acha (ou o endereço vem estranho): abre"
printf 'Instance abc:\n  Config path: %s/qml/shell.qml\n' "$casa" >"$SBX/qs-lista"
echo '[{"address":"$(rm -rf x)","title":"MkTunes"}]' >"$SBX/clientes"
roda
called "qs -p $casa/qml"
not_called "hyprctl dispatch"
rm -f "$SBX/qs-lista" "$SBX/clientes"

t "sem mpv: avisa na tela com o pacote e sai com 1, sem subir nada"
calls_reset
rc=0
out=$(PATH=$(path_sem mpv) "$lancador" 2>&1) || rc=$?
eq 1 "$rc"
has "$out" "O MkTunes precisa do mpv"
has "$out" "sudo pacman -S mpv"
called "notify-send -a MkTunes"
not_called "mktunesd"
not_called "qs -p"

t "sem Quickshell ou sem o binário do daemon: avisa e sai com 1"
rc=0
out=$(PATH=$(path_sem qs) "$lancador" 2>&1) || rc=$?
eq 1 "$rc"
has "$out" "do Quickshell"
mv "$casa/bin/mktunesd" "$casa/bin/mktunesd.fora"
roda
eq 1 "$rc"
has "$out" "do daemon (bin/mktunesd)"
not_called "qs -p"
mv "$casa/bin/mktunesd.fora" "$casa/bin/mktunesd"

t "a pasta do app: o daemon é um binário só, com o contrato, o pack e a assinatura ao lado"
for f in bin/mktunes bin/mktunesd bin/mktunesd.de pack assinatura CONTRATO.md LICENSE NOTICE.md VERSION src/go.mod src/main.go \
  qml/shell.qml qml/Daemon.qml share/mktunes.desktop share/mktunes.svg; do
  exists "$app/$f"
done
for f in bin/mktunes pack assinatura bin/mktunesd; do
  ok test -x "$app/$f"
done
like "$(<"$app/VERSION")" '^0\.[0-9]+\.[0-9]+ go 20[0-9]{2}-[0-9]{2}-[0-9]{2}$'

t "a pasta daemon/ (os jars e o script da JVM) não existe mais"
absent "$app/daemon"
eq "" "$(find "$app" -name '*.jar' -o -name '*.class' -o -name '*.kt' | head -5)"

t "nenhum arquivo do app cita java nem python"
# (a LICENSE é o texto da GPL; os binários não são texto)
eq "" "$(grep -rIlwiE 'java|python3?|jre|jdk|openjdk' "$app" --exclude=LICENSE --exclude=mktunesd || true)"
eq "" "$(grep -rlE '^#!.*(python|java)' "$app" 2>/dev/null || true)"

t "bash: o lançador, o pack e a assinatura têm sintaxe válida (e passam no shellcheck, se houver)"
for f in bin/mktunes pack assinatura; do
  ok bash -n "$app/$f"
done
if command -v shellcheck >/dev/null; then
  out=$(shellcheck -S warning -f gcc "$app/bin/mktunes" "$app/pack" "$app/assinatura" 2>&1) || _fail "$out"
fi

t "a janela fala com o socket do daemon e não mudou com o porte: nenhum pedido novo"
ok grep -qF '+ "/mktunes/mktunes.sock"' "$app/qml/Daemon.qml"
ok grep -qF 'title: "MkTunes"' "$app/qml/shell.qml"
# O título é o que o lançador procura para focar a janela.
has "$src" 'select(.title == "MkTunes")'

t "o atalho de menu abre o lançador, com o ícone do app"
desktop=$(<"$app/share/mktunes.desktop")
has "$desktop" "Exec=mktunes"
has "$desktop" "Icon=mktunes"

t "o NOTICE diz de onde o código vem, que o NewPipe saiu e quais serviços são usados"
aviso=$(<"$app/NOTICE.md")
for trecho in "Copyright (C) 2026 codexmark" "SPDX-License-Identifier: GPL-3.0-or-later" "porte do daemon em Kotlin" "Ryotunes" \
  "Não usa mais o NewPipe" "LRCLIB" "radio-browser.info" "InnerTube"; do
  has "$aviso" "$trecho"
done
# O endereço do áudio vem do cliente InnerTube próprio: nenhum extrator de fora.
like "$aviso" 'endereço do áudio também vem desse cliente'
ok grep -qF "GNU GENERAL PUBLIC LICENSE" "$app/LICENSE"

t "o Início pede a página de novo quando o daemon volta (trocado numa atualização, com a janela aberta)"
inicio=$(<"$app/qml/HomePage.qml")
has "$inicio" 'function onConnectedChanged() { if (daemon.connected && page.visible) daemon.send({ cmd: "home" }); }'
