#!/usr/bin/env bash
# shell/mare/scripts/mare-default-apps: a lista de papéis e apps (list), os
# tipos de cada papel (mimes), a troca do padrão (set), o terminal
# (xdg-terminals.list) e o atalho que abre o padrão (open), com .desktop de
# mentira e xdg-mime e gio falsos.
. "$(dirname "$0")/../lib.sh"

apps_sh=$REPO/shell/mare/scripts/mare-default-apps
sysd=$SBX/sys-data/applications
home=$XDG_DATA_HOME/applications
padroes=$SBX/padroes
mkdir -p "$sysd" "$home" "$XDG_CONFIG_HOME"

# Um PATH só com os stubs e o que o script usa: os terminais e o
# xdg-terminal-exec instalados nesta máquina não podem aparecer na lista.
bash_bin=$(command -v bash)
mkdir -p "$SBX/limpo"
for c in jq find sort awk mktemp head rm mkdir grep chmod mv cat sed cut; do
  ln -s "$(command -v "$c")" "$SBX/limpo/$c"
done

# O xdg-mime de mentira guarda os padrões em $padroes ("tipo=app.desktop").
stub xdg-mime '
printf "[%s]" "$@" >>"$SBX/args"; echo >>"$SBX/args"
p=$SBX/padroes
case "$1 $2" in
  "query default") [[ -f $p ]] && { grep -m1 "^$3=" "$p" | cut -d= -f2-; } ;;
  "default "*)
    app=$2; shift 2
    for tipo in "$@"; do
      [[ -f $p ]] && { grep -v "^$tipo=" "$p" >"$p.novo"; mv "$p.novo" "$p"; }
      echo "$tipo=$app" >>"$p"
    done ;;
  *) exit 2 ;;
esac
exit 0'
stub gio

run() {
  PATH=$SBX/bin:$SBX/limpo XDG_DATA_DIRS=$SBX/sys-data:$SBX/nao-existe "$bash_bin" "$apps_sh" "$@"
}
app() { # app <pasta> <arquivo.desktop> <linhas do [Desktop Entry]...>
  local dir=$1 file=$2
  shift 2
  mkdir -p "$(dirname "$dir/$file")"
  { echo "[Desktop Entry]"; echo "Type=Application"; printf '%s\n' "$@"; } >"$dir/$file"
}
zera() {
  rm -f "$padroes" "$SBX/args"
  : >"$SBX/args"
  calls_reset
}
papel() { jq -c --arg r "$1" '.roles[] | select(.id == $r)' <<<"$out"; }
ids() { jq -c --arg r "$1" '[.roles[] | select(.id == $r) | .apps[].id]' <<<"$out"; }

app "$sysd" firefox.desktop "Name=Firefox" "Icon=firefox" "Categories=Network;WebBrowser;" \
  "MimeType=text/html;x-scheme-handler/http;x-scheme-handler/https;application/xhtml+xml;application/pdf;image/png;image/jpeg;"
app "$sysd" chromium.desktop "Name=Chromium" "Name[pt]=Chromium (pt)" "Name[pt_BR]=Navegador Chromium" "Icon=chromium" \
  "Categories=Network;WebBrowser;" "MimeType=text/html;x-scheme-handler/http;x-scheme-handler/https;"
app "$sysd" org.gnome.Nautilus.desktop "Name=Files" "Name[pt]=Ficheiros" "Icon=org.gnome.Nautilus" \
  "Categories=GNOME;GTK;Utility;Core;FileManager;" "MimeType=inode/directory;application/x-7z-compressed;"
app "$sysd" code.desktop "Name=Visual Studio Code" "Icon=vscode" "Categories=TextEditor;Development;IDE;" \
  "MimeType=text/plain;inode/directory;application/x-code-workspace;"
app "$sysd" nvim.desktop "Name=Neovim" "Icon=nvim" "Terminal=true" "Categories=Utility;TextEditor;" \
  "MimeType=text/plain;text/markdown;text/x-lua;text/english;"
app "$sysd" imv.desktop "Name=imv" "NoDisplay=true" "Icon=multimedia-photo-viewer" "Categories=Graphics;2DGraphics;Viewer;" \
  "MimeType=image/png;image/jpeg;image/gif;image/bmp;"
app "$sysd" mpv.desktop "Name=mpv Media Player" "Icon=mpv" "Categories=AudioVideo;Audio;Video;Player;TV;" \
  "MimeType=video/mp4;video/x-matroska;video/webm;video/ogg;audio/mpeg;audio/flac;audio/ogg;"
app "$sysd" kdenlive.desktop "Name=Kdenlive" "Categories=Qt;KDE;AudioVideo;Player;AudioVideoEditing;" \
  "MimeType=video/mp4;video/x-matroska;audio/mpeg;audio/flac;"
app "$sysd" thunderbird.desktop "Name=Thunderbird" "Icon=thunderbird" "Categories=Network;Email;" \
  "MimeType=x-scheme-handler/mailto;message/rfc822;"
app "$sysd" kde/okular.desktop "Name=Okular" "Icon=okular" "Categories=Qt;KDE;Graphics;Office;Viewer;" "MimeType=application/pdf;"
# Os que não podem aparecer.
app "$sysd" oculto.desktop "Name=Oculto" "Hidden=true" "Categories=WebBrowser;" "MimeType=x-scheme-handler/http;"
printf '%s\n' "[Desktop Entry]" "Type=Link" "Name=Um atalho" "Categories=WebBrowser;" \
  "MimeType=x-scheme-handler/http;" >"$sysd/atalho.desktop"
echo "isto não é um .desktop" >"$sysd/lixo.desktop"
echo "MimeType=x-scheme-handler/http;" >"$sysd/leiame.txt"

# ---------------------------------------------------------------- list

t "list: JSON com os oito papéis, os terminais e terminalExec"
zera
out=$(run list)
eq 0 $?
ok jq -e . <<<"$out" >/dev/null
eq '["navegador","arquivos","editor","imagens","video","musica","pdf","email"]' "$(jq -c '[.roles[].id]' <<<"$out")"
eq '["ghostty","kitty","foot","alacritty","wezterm"]' "$(jq -c '[.terminals[].id]' <<<"$out")"
eq '["roles","terminalExec","terminals"]' "$(jq -c 'keys' <<<"$out")"

t "list: cada papel lista os instalados que servem, por nome, com id, name, icon e terminal"
eq '["firefox.desktop","chromium.desktop"]' "$(ids navegador)"
eq '["org.gnome.Nautilus.desktop"]' "$(ids arquivos)"
eq '["nvim.desktop","code.desktop"]' "$(ids editor)"
eq '["imv.desktop"]' "$(ids imagens)"
eq '["mpv.desktop"]' "$(ids video)"
eq '["mpv.desktop"]' "$(ids musica)"
eq '["firefox.desktop","kde-okular.desktop"]' "$(ids pdf)"
eq '["thunderbird.desktop"]' "$(ids email)"
eq '{"id":"firefox.desktop","name":"Firefox","icon":"firefox","terminal":false}' "$(jq -c '.roles[0].apps[0]' <<<"$out")"

t "list: a categoria decide: o navegador não vira visualizador de imagens nem o editor de código gerenciador de arquivos"
hasnt "$(ids imagens)" "firefox"
hasnt "$(ids arquivos)" "code"

t "list: editor de vídeo (AudioVideoEditing) não é player"
hasnt "$(ids video)" "kdenlive"
hasnt "$(ids musica)" "kdenlive"

t "list: Hidden=true, Type que não é Application e arquivo sem [Desktop Entry] não existem"
hasnt "$out" "oculto"
hasnt "$out" "atalho"
hasnt "$out" "lixo"
hasnt "$out" "leiame"

t "list: NoDisplay=true continua na lista (é assim que o imv se esconde do menu)"
has "$(ids imagens)" "imv.desktop"

t "list: app de terminal vem marcado"
eq true "$(jq -r '.roles[] | select(.id == "editor") | .apps[] | select(.id == "nvim.desktop") | .terminal' <<<"$out")"

t "list: o nome em pt_BR ganha do pt, que ganha do Name"
eq "Navegador Chromium" "$(jq -r '.roles[0].apps[] | select(.id == "chromium.desktop") | .name' <<<"$out")"
eq "Ficheiros" "$(jq -r '.roles[1].apps[0].name' <<<"$out")"

t "list: .desktop numa subpasta tem o id com '-' no lugar da '/'"
has "$(ids pdf)" '"kde-okular.desktop"'

t "list: sem padrão nenhum, current e currentName vêm vazios"
eq '["",""]' "$(jq -c '.roles[0] | [.current, .currentName]' <<<"$out")"

t "list: current é o que o xdg-mime diz do primeiro tipo do papel, com o nome do app"
zera
printf '%s\n' "x-scheme-handler/http=firefox.desktop" "inode/directory=org.gnome.Nautilus.desktop" \
  "application/pdf=kde-okular.desktop" "text/html=chromium.desktop" >"$padroes"
out=$(run list)
eq '["firefox.desktop","Firefox"]' "$(jq -c '.roles[0] | [.current, .currentName]' <<<"$out")"
eq '["org.gnome.Nautilus.desktop","Ficheiros"]' "$(jq -c '.roles[1] | [.current, .currentName]' <<<"$out")"
eq '["kde-okular.desktop","Okular"]' "$(papel pdf | jq -c '[.current, .currentName]')"
eq "" "$(papel email | jq -r .current)"
called "xdg-mime query default x-scheme-handler/http"
called "xdg-mime query default x-scheme-handler/mailto"

t "list: padrão que foi desinstalado: current fica, e o nome é o id sem .desktop"
zera
echo "x-scheme-handler/http=sumiu.do.mapa.desktop" >"$padroes"
out=$(run list)
eq '["sumiu.do.mapa.desktop","sumiu.do.mapa"]' "$(jq -c '.roles[0] | [.current, .currentName]' <<<"$out")"

t "list: o .desktop da home ganha do sistema"
zera
app "$home" firefox.desktop "Name=Firefox da Casa" "Icon=firefox-casa" "Categories=WebBrowser;" \
  "MimeType=x-scheme-handler/http;x-scheme-handler/https;"
out=$(run list)
eq '["Firefox da Casa","firefox-casa"]' "$(jq -c '.roles[0].apps[] | select(.id == "firefox.desktop") | [.name, .icon]' <<<"$out")"
eq 1 "$(jq '[.roles[0].apps[] | select(.id == "firefox.desktop")] | length' <<<"$out")"
hasnt "$(ids pdf)" "firefox"
rm "$home/firefox.desktop"

t "list: Hidden=true na home esconde o app do sistema"
app "$home" chromium.desktop "Name=Chromium" "Hidden=true"
out=$(run list)
hasnt "$(ids navegador)" "chromium"
rm "$home/chromium.desktop"

t "list: dois apps com o mesmo nome ganham o id entre parênteses"
app "$home" visor-a.desktop "Name=Visor" "Categories=Viewer;" "MimeType=application/pdf;"
app "$home" visor-b.desktop "Name=Visor" "Categories=Viewer;" "MimeType=application/pdf;"
out=$(run list)
eq '["Firefox","Okular","Visor (visor-a)","Visor (visor-b)"]' "$(papel pdf | jq -c '[.apps[].name]')"
rm "$home"/visor-*.desktop

t "list: só a seção [Desktop Entry] conta; as ações do app não trocam o nome nem os tipos"
printf '%s\n' "[Desktop Action nova-janela]" "Name=Nova janela" "MimeType=inode/directory;" "Categories=FileManager;" \
  "" "[Desktop Entry]" "Type=Application" "Name=Leitor" "Categories=Viewer;" "MimeType=application/pdf;" \
  "" "[Desktop Action outra]" "Name=Outra" "Hidden=true" >"$home/leitor.desktop"
out=$(run list)
eq "Leitor" "$(papel pdf | jq -r '.apps[] | select(.id == "leitor.desktop") | .name')"
hasnt "$(ids arquivos)" "leitor"
rm "$home/leitor.desktop"

t "list: .desktop com fim de linha do Windows, espaço em volta do '=' e sem Name"
printf '%s\r\n' "[Desktop Entry]" "Type=Application" "Name = Leitor DOS" "Icon=dos" "Categories=Viewer;" \
  "MimeType=application/pdf;" >"$home/dos.desktop"
app "$home" semnome.desktop "Categories=Viewer;" "MimeType=application/pdf;"
out=$(run list)
eq '{"id":"dos.desktop","name":"Leitor DOS","icon":"dos","terminal":false}' "$(papel pdf | jq -c '.apps[] | select(.id == "dos.desktop")')"
eq "semnome.desktop" "$(papel pdf | jq -r '.apps[] | select(.id == "semnome.desktop") | .name')"
rm "$home/dos.desktop" "$home/semnome.desktop"

t "list: nome e arquivo com aspas, acento, espaço e comando embutido viram texto, nada roda"
app "$home" 'Meu App "çã" $(touch pwn).desktop' 'Name=Café "do" Zé $(touch '"$SBX"'/pwn1)`touch '"$SBX"'/pwn2`; \n' \
  'Icon=$(touch '"$SBX"'/pwn3)' "Categories=Viewer;" "MimeType=application/pdf;"
out=$(run list)
ok jq -e . <<<"$out" >/dev/null
eq 'Café "do" Zé $(touch '"$SBX"'/pwn1)`touch '"$SBX"'/pwn2`; \n' \
  "$(papel pdf | jq -r '.apps[] | select(.id | startswith("Meu App")) | .name')"
eq 'Meu App "çã" $(touch pwn).desktop' "$(papel pdf | jq -r '.apps[] | select(.id | startswith("Meu App")) | .id')"
absent "$SBX/pwn1"
absent "$SBX/pwn2"
absent "$SBX/pwn3"
absent "pwn"
rm "$home"/Meu*.desktop

t "list: sem nenhum terminal instalado e sem xdg-terminal-exec"
out=$(run list)
eq '[false,false,false,false,false]' "$(jq -c '[.terminals[].installed]' <<<"$out")"
eq false "$(jq -r .terminalExec <<<"$out")"
eq '{"id":"ghostty","name":"Ghostty","installed":false}' "$(jq -c '.terminals[0]' <<<"$out")"

t "list: terminal no PATH vira installed, e o xdg-terminal-exec vira terminalExec"
stub kitty
stub wezterm
stub xdg-terminal-exec
out=$(run list)
eq '["kitty","wezterm"]' "$(jq -c '[.terminals[] | select(.installed) | .id]' <<<"$out")"
eq true "$(jq -r .terminalExec <<<"$out")"
not_called "kitty"
rm "$SBX/bin/kitty" "$SBX/bin/wezterm" "$SBX/bin/xdg-terminal-exec"

t "list: sem pasta de apps nenhuma, os papéis vêm com a lista vazia"
out=$(PATH=$SBX/bin:$SBX/limpo XDG_DATA_HOME=$SBX/vazio XDG_DATA_DIRS=$SBX/nao-existe "$bash_bin" "$apps_sh" list)
eq 0 $?
eq 8 "$(jq '.roles | length' <<<"$out")"
eq 0 "$(jq '[.roles[].apps[]] | length' <<<"$out")"

t "list: o xdg-mime falhando não derruba a lista (current vazio)"
stub xdg-mime 'exit 4'
out=$(run list)
eq 0 $?
eq "" "$(jq -r '.roles[0].current' <<<"$out")"
eq '["firefox.desktop","chromium.desktop"]' "$(ids navegador)"
# de volta ao xdg-mime que guarda os padrões
stub xdg-mime '
printf "[%s]" "$@" >>"$SBX/args"; echo >>"$SBX/args"
p=$SBX/padroes
case "$1 $2" in
  "query default") [[ -f $p ]] && { grep -m1 "^$3=" "$p" | cut -d= -f2-; } ;;
  "default "*)
    app=$2; shift 2
    for tipo in "$@"; do
      [[ -f $p ]] && { grep -v "^$tipo=" "$p" >"$p.novo"; mv "$p.novo" "$p"; }
      echo "$tipo=$app" >>"$p"
    done ;;
  *) exit 2 ;;
esac
exit 0'

# ---------------------------------------------------------------- mimes

t "mimes <papel>: os core e todos os extra, um por linha"
zera
eq "x-scheme-handler/http
x-scheme-handler/https
text/html
application/xhtml+xml
x-scheme-handler/about
x-scheme-handler/unknown" "$(run mimes navegador)"
eq "application/pdf" "$(run mimes pdf)"
eq "x-scheme-handler/mailto
message/rfc822" "$(run mimes email)"

t "mimes <papel> <app>: os core sempre, e dos extra só os que o app declara"
eq "x-scheme-handler/http
x-scheme-handler/https
text/html
application/xhtml+xml" "$(run mimes navegador firefox.desktop)"
eq "x-scheme-handler/http
x-scheme-handler/https
text/html" "$(run mimes navegador chromium.desktop)"
eq "text/plain
text/markdown
text/x-lua" "$(run mimes editor nvim.desktop)"
eq "image/png
image/jpeg
image/gif
image/bmp" "$(run mimes imagens imv.desktop)"

t "mimes: só confere, não chama o xdg-mime"
not_called "xdg-mime"

t "mimes: papel desconhecido sai com 2 e diz qual"
err=$(run mimes fotos 2>&1 >/dev/null)
eq 2 $?
has "$err" "mare-default-apps: papel desconhecido: fotos"

t "mimes: app que não existe sai com 1 e diz qual"
err=$(run mimes navegador nao-existe.desktop 2>&1 >/dev/null)
eq 1 $?
has "$err" "mare-default-apps: app não encontrado: nao-existe.desktop"

t "mimes: sem papel ou com argumento a mais: uso, sai com 2"
run mimes 2>/dev/null
eq 2 $?
run mimes navegador firefox.desktop extra 2>/dev/null
eq 2 $?

# ---------------------------------------------------------------- set

t "set: um xdg-mime default com o app e os tipos do papel que ele abre"
zera
out=$(run set navegador firefox.desktop)
eq 0 $?
has "$(<"$SBX/args")" "[default][firefox.desktop][x-scheme-handler/http][x-scheme-handler/https][text/html][application/xhtml+xml]"
eq 1 "$(grep -c '^\[default\]' "$SBX/args")"

t "set: devolve o JSON de list já com o padrão novo"
ok jq -e . <<<"$out" >/dev/null
eq '["firefox.desktop","Firefox"]' "$(jq -c '.roles[0] | [.current, .currentName]' <<<"$out")"
eq 8 "$(jq '.roles | length' <<<"$out")"

t "set: não entrega ao app um formato extra que ele não lê"
zera
ok run set imagens imv.desktop >/dev/null
eq "[default][imv.desktop][image/png][image/jpeg][image/gif][image/bmp]" "$(grep '^\[default\]' "$SBX/args")"

t "set: app numa subpasta vale pelo id com '-'"
zera
out=$(run set pdf kde-okular.desktop)
eq 0 $?
eq "[default][kde-okular.desktop][application/pdf]" "$(grep '^\[default\]' "$SBX/args")"
eq "Okular" "$(papel pdf | jq -r .currentName)"

t "set --dry-run: nada é gravado, o que seria feito sai no stderr e a lista ainda vem"
zera
out=$(run --dry-run set navegador chromium.desktop 2>"$SBX/err")
eq 0 $?
not_called "xdg-mime default"
absent "$padroes"
has "$(<"$SBX/err")" "dry-run: xdg-mime default chromium.desktop x-scheme-handler/http x-scheme-handler/https text/html"
eq "" "$(jq -r '.roles[0].current' <<<"$out")"

t "set com MARE_APPS_DRY=1: o mesmo que --dry-run"
zera
out=$(MARE_APPS_DRY=1 run set navegador chromium.desktop 2>"$SBX/err")
eq 0 $?
not_called "xdg-mime default"
has "$(<"$SBX/err")" "dry-run: xdg-mime default chromium.desktop"

t "set: app que não existe sai com 1, diz qual e não muda nada"
zera
err=$(run set navegador nao-existe.desktop 2>&1 >/dev/null)
eq 1 $?
has "$err" "app não encontrado: nao-existe.desktop"
not_called "xdg-mime default"

t "set: papel desconhecido sai com 2 e não muda nada"
err=$(run set fotos firefox.desktop 2>&1 >/dev/null)
eq 2 $?
has "$err" "papel desconhecido: fotos"
not_called "xdg-mime default"

t "set: app e papel hostis não rodam nem viram opção de ninguém"
zera
for a in '$(touch '"$SBX"'/pwn4).desktop' '`touch '"$SBX"'/pwn5`' 'firefox.desktop; touch '"$SBX"'/pwn6' '--help' '../../etc/passwd' '.*' '*'; do
  run set navegador "$a" >/dev/null 2>&1
  eq 1 $? "saída de set navegador '$a'"
done
run set '$(touch '"$SBX"'/pwn7)' firefox.desktop >/dev/null 2>&1
eq 2 $?
run set '.*' firefox.desktop >/dev/null 2>&1
eq 2 $?
absent "$SBX/pwn4"
absent "$SBX/pwn5"
absent "$SBX/pwn6"
absent "$SBX/pwn7"
not_called "xdg-mime default"

t "set: um app de nome estranho que existe de verdade vai como um argumento só"
zera
app "$home" 'Meu App "çã" $(touch pwn8).desktop' "Name=Meu App" "Categories=Viewer;" "MimeType=application/pdf;"
ok run set pdf 'Meu App "çã" $(touch pwn8).desktop' >/dev/null
eq '[default][Meu App "çã" $(touch pwn8).desktop][application/pdf]' "$(grep '^\[default\]' "$SBX/args")"
absent pwn8
rm "$home"/Meu*.desktop

t "set: faltando ou sobrando argumento: uso, sai com 2"
zera
err=$(run set navegador 2>&1 >/dev/null)
eq 2 $?
has "$err" "uso: mare-default-apps"
run set 2>/dev/null
eq 2 $?
run set navegador firefox.desktop extra 2>/dev/null
eq 2 $?
not_called "xdg-mime default"

# ---------------------------------------------------------------- terminal

lista=$XDG_CONFIG_HOME/xdg-terminals.list

t "terminal: cria xdg-terminals.list com o .desktop do terminal"
zera
rm -f "$lista"
ok run terminal kitty
eq "kitty.desktop" "$(<"$lista")"
eq 644 "$(stat -c %a "$lista")"

t "terminal: cada nome que o Mk conhece tem o seu .desktop"
for caso in "ghostty=com.mitchellh.ghostty.desktop" "kitty=kitty.desktop" "foot=foot.desktop" \
  "alacritty=Alacritty.desktop" "wezterm=org.wezfurlong.wezterm.desktop"; do
  rm -f "$lista"
  ok run terminal "${caso%%=*}"
  eq "${caso##*=}" "$(<"$lista")" "${caso%%=*}"
done

t "terminal: o escolhido sobe para o alto; o resto continua abaixo, na mesma ordem e sem repetir"
printf '%s\n' "foot.desktop" "# um comentário" "kitty.desktop" "Alacritty.desktop" >"$lista"
ok run terminal kitty
eq "kitty.desktop
foot.desktop
# um comentário
Alacritty.desktop" "$(<"$lista")"

t "terminal: escolher o que já é o único da lista não quebra"
echo "kitty.desktop" >"$lista"
ok run terminal kitty
eq "kitty.desktop" "$(<"$lista")"

t "terminal: cria a pasta de configuração se ela não existe, e não deixa arquivo temporário"
rm -rf "$XDG_CONFIG_HOME"
ok run terminal foot
eq "foot.desktop" "$(<"$lista")"
eq "xdg-terminals.list" "$(ls -A "$XDG_CONFIG_HOME")"

t "terminal --dry-run: diz o que faria e não grava"
rm -f "$lista"
err=$(run --dry-run terminal kitty 2>&1)
eq 0 $?
has "$err" "dry-run: kitty.desktop no alto de $lista"
absent "$lista"

t "terminal: nome desconhecido (ou hostil) sai com 2 e não grava"
rm -f "$lista"
for nome in xterm "" 'kitty; touch '"$SBX"'/pwn9' '$(touch '"$SBX"'/pwn9)' "kitty.desktop" ".*"; do
  err=$(run terminal "$nome" 2>&1 >/dev/null)
  eq 2 $? "saída de terminal '$nome'"
  has "$err" "terminal desconhecido" "terminal '$nome'"
done
absent "$lista"
absent "$SBX/pwn9"

t "terminal: sem nome: uso, sai com 2"
run terminal 2>/dev/null
eq 2 $?
run terminal kitty foot 2>/dev/null
eq 2 $?

t "terminal: não passa pelo xdg-mime"
not_called "xdg-mime"

# ---------------------------------------------------------------- open

t "open: lança o .desktop do padrão do papel, sem arquivo nenhum"
zera
echo "x-scheme-handler/http=firefox.desktop" >"$padroes"
ok run open navegador
called "gio launch $sysd/firefox.desktop"
eq 1 "$(grep -c '^gio ' "$SBX/calls")"

t "open: o padrão numa subpasta (id com '-') é achado"
zera
echo "application/pdf=kde-okular.desktop" >"$padroes"
ok run open pdf
called "gio launch $sysd/kde/okular.desktop"

t "open: o .desktop da home ganha do sistema"
zera
echo "x-scheme-handler/http=firefox.desktop" >"$padroes"
app "$home" firefox.desktop "Name=Firefox da Casa" "Categories=WebBrowser;" "MimeType=x-scheme-handler/http;"
ok run open navegador
called "gio launch $home/firefox.desktop"
rm "$home/firefox.desktop"

t "open: sem padrão, abre o primeiro instalado que serve"
zera
ok run open navegador
called "gio launch $sysd/firefox.desktop"

t "open: padrão desinstalado, abre o primeiro instalado que serve"
zera
echo "x-scheme-handler/http=sumiu.desktop" >"$padroes"
ok run open navegador
called "gio launch $sysd/firefox.desktop"
not_called "sumiu.desktop"

t "open: padrão hostil no mimeapps.list não roda nem sai da pasta de apps"
zera
echo 'x-scheme-handler/http=../../../bin/sh; $(touch '"$SBX"'/pwn10).desktop' >"$padroes"
ok run open navegador
called "gio launch $sysd/firefox.desktop"
absent "$SBX/pwn10"

t "open: nenhum app para o papel: avisa e sai com 1"
zera
rm "$sysd/thunderbird.desktop"
err=$(run open email 2>&1 >/dev/null)
eq 1 $?
has "$err" "mare-default-apps: nenhum app instalado para email"
not_called "gio"

t "open --dry-run: diz o que lançaria e não lança"
zera
err=$(run --dry-run open navegador 2>&1 >/dev/null)
eq 0 $?
has "$err" "dry-run: gio launch $sysd/firefox.desktop"
not_called "gio"

t "open: papel desconhecido sai com 2; sem papel, uso e 2"
err=$(run open fotos 2>&1 >/dev/null)
eq 2 $?
has "$err" "papel desconhecido: fotos"
run open 2>/dev/null
eq 2 $?
not_called "gio"

# ---------------------------------------------------------------- uso

t "sem subcomando, subcomando desconhecido ou só --dry-run: uso, sai com 2"
zera
err=$(run 2>&1 >/dev/null)
eq 2 $?
has "$err" "uso: mare-default-apps [--dry-run] list | set <papel> <app.desktop> | terminal <nome> | open <papel> | mimes <papel> [app.desktop]"
run bobagem 2>/dev/null
eq 2 $?
run --dry-run 2>/dev/null
eq 2 $?
not_called "xdg-mime"
not_called "gio"
