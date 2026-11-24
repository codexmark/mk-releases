#!/usr/bin/env bash
# shell/mare/scripts/mare-shot: cada modo de captura (os argumentos do slurp,
# do grim, do wl-copy e do aviso), a pasta e o diagnóstico da gravação, tudo
# por stub.
. "$(dirname "$0")/../lib.sh"

shot_sh=$REPO/shell/mare/scripts/mare-shot
pics=$HOME/Imagens
dir=$pics/Capturas

stub xdg-user-dir 'cat "$SBX/xdg-pictures" 2>/dev/null'
echo "$pics" >"$SBX/xdg-pictures"
# slurp: devolve a área escolhida, ou sai com 1 se a pessoa cancelou (Esc).
stub slurp '[[ ! -e $SBX/slurp-cancela ]] || exit 1; echo "100,200 640x480"'
# grim: grava no último argumento; com "-", manda a imagem pela saída.
stub grim '
for a; do f=$a; done
if [[ $f == - ]]; then printf "PNG-da-area"; else printf "PNG-em-arquivo" >"$f"; fi'
stub wl-copy 'cat >"$SBX/clip"'
stub notify-send
stub hyprctl 'cat "$SBX/janela.json" 2>/dev/null'
echo '{"address":"0x55d0c0ffee","at":[12,34],"size":[800,600],"class":"kitty","title":"zsh"}' >"$SBX/janela.json"

run() { bash "$shot_sh" "$@"; }
# Um PATH só com o básico do sistema, os stubs e os guardas: para o ramo em
# que o gpu-screen-recorder não está instalado.
path_sem() {
  local d c p
  d=$(mktemp -d "$SBX/sem.XXXXXX")
  mkdir "$d/s" "$d/g"
  for c in bash sh env jq awk sed grep head tail cut tr cat rm mv mkdir dirname basename date timeout; do
    p=$(PATH=/usr/bin:/bin command -v "$c") && ln -s "$p" "$d/s/$c"
  done
  cp "$SBX"/guard/* "$d/g/"
  for c; do rm -f "$d/g/$c"; done
  echo "$SBX/bin:$d/g:$d/s"
}
zera() {
  rm -rf "${pics:?}" "${HOME:?}/Pictures" "$SBX/clip" "$SBX/slurp-cancela"
  calls_reset
}
arquivos() { find "$dir" -type f 2>/dev/null | sort; }

# ---------------------------------------------------------------- pasta

t "pasta: <Imagens>/Capturas, pela pasta de imagens do XDG, sem criar nada"
zera
eq "$dir" "$(run pasta)"
absent "$dir"
called "xdg-user-dir PICTURES"

t "pasta: sem xdg-user-dir respondendo, ~/Pictures/Capturas"
: >"$SBX/xdg-pictures"
eq "$HOME/Pictures/Capturas" "$(run pasta)"
stub xdg-user-dir 'exit 1'
eq "$HOME/Pictures/Capturas" "$(run pasta)"

t "pasta: quando o XDG devolve a própria pasta pessoal (sem Imagens), ~/Pictures/Capturas"
stub xdg-user-dir 'echo "$HOME"'
eq "$HOME/Pictures/Capturas" "$(run pasta)"
stub xdg-user-dir 'cat "$SBX/xdg-pictures" 2>/dev/null'
echo "$pics" >"$SBX/xdg-pictures"

t "pasta: não captura nem avisa"
not_called "grim"
not_called "slurp"
not_called "notify-send"

# ---------------------------------------------------------------- tela

t "tela: o grim grava a tela inteira num arquivo com data e hora em Capturas"
zera
ok run tela
f=$(arquivos)
like "$f" "^$dir/[0-9]{4}-[0-9]{2}-[0-9]{2}_[0-9]{2}-[0-9]{2}-[0-9]{2}\.png$"
called "grim $f"
not_called "grim -g"
not_called "slurp"

t "tela: a imagem salva vai para a área de transferência como PNG"
called "wl-copy --type image/png"
eq "PNG-em-arquivo" "$(<"$SBX/clip")"

t "tela: o aviso leva a imagem como ícone e o caminho com ~ no lugar da pasta pessoal"
called "notify-send -a Mk -i $f Captura salva ~/Imagens/Capturas/$(basename "$f")"

# ---------------------------------------------------------------- área

t "area: o slurp escolhe a área (com as cores do Mk) e o grim grava só ela"
zera
ok run area
f=$(arquivos)
called "slurp -b 0b0b0b66 -c ce5e32 -w 1"
called "grim -g 100,200 640x480 $f"
eq "PNG-em-arquivo" "$(<"$SBX/clip")"
called "Captura salva"

t "sem argumento, é o mesmo que area"
zera
ok run
called "slurp "
called "grim -g 100,200 640x480 $(arquivos)"

t "area: cancelar a escolha (Esc) sai com 0 sem capturar, copiar nem avisar"
zera
: >"$SBX/slurp-cancela"
ok run area
eq "" "$(arquivos)"
not_called "grim"
not_called "wl-copy"
not_called "notify-send"

# ---------------------------------------------------------------- copiar

t "copiar: a área vai direto para a área de transferência, sem arquivo"
zera
ok run copiar
called "slurp -b 0b0b0b66 -c ce5e32 -w 1"
called "grim -g 100,200 640x480 -"
called "wl-copy --type image/png"
eq "PNG-da-area" "$(<"$SBX/clip")"
eq "" "$(arquivos)"

t "copiar: o aviso diz que copiou, não que salvou"
called "notify-send -a Mk Captura copiada"
not_called "Captura salva"

t "copiar: cancelar a escolha sai com 0 sem copiar nem avisar"
zera
: >"$SBX/slurp-cancela"
ok run copiar
not_called "grim"
not_called "wl-copy"
not_called "notify-send"
absent "$SBX/clip"

# ---------------------------------------------------------------- janela

t "janela: a área é a posição e o tamanho da janela em foco"
zera
ok run janela
f=$(arquivos)
called "hyprctl -j activewindow"
called "grim -g 12,34 800x600 $f"
not_called "slurp"
called "Captura salva"

t "janela: sem janela em foco, avisa, não captura e sai com 1"
zera
echo '{}' >"$SBX/janela.json"
run janela
eq 1 $?
called "notify-send -a Mk Sem janela em foco"
not_called "grim"
not_called "wl-copy"
eq "" "$(arquivos)"
echo '{"address":"0x55d0c0ffee","at":[12,34],"size":[800,600]}' >"$SBX/janela.json"

t "janela: o hyprctl falhando (Hyprland fora do ar) dá o mesmo aviso, não um fim mudo"
zera
stub hyprctl 'echo "HYPRLAND_INSTANCE_SIGNATURE not set!" >&2; exit 1'
run janela 2>/dev/null
eq 1 $?
called "notify-send -a Mk Sem janela em foco"
not_called "grim"
not_called "wl-copy"
absent "$dir"

t "janela: resposta que não é JSON, ou sem posição e tamanho em números, também avisa"
for resp in 'isto não é JSON' '{"at":[12,34]}' '{"at":[12,34],"size":null}' '{"at":["a","b"],"size":[1,2]}' \
  '{"at":[12,34],"size":[0,0]}' '{"at":[1,2],"size":["3 -o /tmp/x","4"]}'; do
  zera
  printf '%s\n' "$resp" >"$SBX/janela.json"
  stub hyprctl 'cat "$SBX/janela.json" 2>/dev/null'
  run janela 2>/dev/null
  eq 1 $? "$resp"
  called "notify-send -a Mk Sem janela em foco"
  not_called "grim"
done

t "janela: sem janela em foco, a pasta de Capturas não é criada"
absent "$dir"

t "janela: num monitor à esquerda (posição negativa) a área segue inteira para o grim"
zera
echo '{"at":[-1920,-40],"size":[800,600]}' >"$SBX/janela.json"
ok run janela
called "grim -g -1920,-40 800x600 $(arquivos)"
echo '{"address":"0x55d0c0ffee","at":[12,34],"size":[800,600]}' >"$SBX/janela.json"

# ---------------------------------------------------------------- erros

t "o grim falhando: sai com erro, sem copiar e sem dizer que salvou"
zera
stub grim 'exit 1'
fails run tela
not_called "wl-copy"
not_called "notify-send"
stub grim '
for a; do f=$a; done
if [[ $f == - ]]; then printf "PNG-da-area"; else printf "PNG-em-arquivo" >"$f"; fi'

t "subcomando desconhecido: uso, saída 2 e nenhuma captura"
zera
out=$(run bobagem 2>&1)
eq 2 $?
has "$out" "uso: mare-shot copiar | area | janela | tela | pasta | gravacao"
not_called "grim"
not_called "slurp"
not_called "notify-send"

t "subcomando desconhecido: não cria a pasta de Capturas nem nada na pasta de imagens"
absent "$dir"
absent "$pics"
for arg in "--help" "Tela" "tela extra" $'tela\nx'; do
  zera
  run "$arg" >/dev/null 2>&1
  eq 2 $? "'$arg'"
  absent "$pics"
done

t "area e copiar cancelados (Esc): a pasta de Capturas não é criada"
zera
: >"$SBX/slurp-cancela"
ok run area
absent "$dir"
ok run copiar
absent "$dir"
rm -f "$SBX/slurp-cancela"

t "copiar: não cria a pasta de Capturas (nada é salvo)"
zera
ok run copiar
absent "$dir"

t "o wl-copy falhando depois de salvar: a captura fica, o aviso sai e o erro não some"
zera
stub wl-copy 'cat >/dev/null; exit 1'
run tela
eq 1 $?
f=$(arquivos)
like "$f" "\.png$"
called "notify-send -a Mk -i $f Captura salva"
stub wl-copy 'cat >"$SBX/clip"'

t "pasta de imagens com espaço no nome"
zera
echo "$HOME/Minhas Imagens" >"$SBX/xdg-pictures"
ok run tela
f=$(find "$HOME/Minhas Imagens/Capturas" -type f)
like "$f" "/Minhas Imagens/Capturas/[0-9_-]+\.png$"
called "grim $f"
rm -rf "${HOME:?}/Minhas Imagens"
echo "$pics" >"$SBX/xdg-pictures"

# ---------------------------------------------------------------- gravacao

t "gravacao: com o gpu-screen-recorder respondendo a ajuda, não imprime nada"
zera
stub gpu-screen-recorder 'echo "usage: gpu-screen-recorder -w <window_id|monitor|focused|portal|region> [-c <container_format>]" >&2; exit 1'
out=$(run gravacao)
eq 0 $?
eq "" "$out"

t "gravacao: só lê a ajuda; não grava, não captura e não cria a pasta"
eq "gpu-screen-recorder --help" "$(grep '^gpu-screen-recorder' "$SBX/calls")"
not_called "grim"
absent "$dir"

t "gravacao: sem o gpu-screen-recorder instalado, diz que falta"
rm -f "$SBX/bin/gpu-screen-recorder"
out=$(PATH=$(path_sem gpu-screen-recorder) bash "$shot_sh" gravacao)
eq 0 $?
eq "falta o gpu-screen-recorder" "$out"

t "gravacao: se ele não abre (biblioteca faltando), diz que não abre neste sistema"
stub gpu-screen-recorder 'echo "gpu-screen-recorder: error while loading shared libraries: libavcodec.so.61" >&2; exit 127'
out=$(run gravacao)
eq 0 $?
eq "o gpu-screen-recorder não abre neste sistema" "$out"

t "gravacao: se ele trava (o timeout corta, 124 ou 137), diz que não respondeu"
for rc in 124 137; do
  stub timeout "exit $rc"
  out=$(run gravacao)
  eq 0 $?
  eq "o gpu-screen-recorder não respondeu" "$out" "timeout saiu com $rc"
done
rm -f "$SBX/bin/timeout"

t "gravacao: o timeout é de 3 segundos"
stub timeout 'shift; "$@"'
stub gpu-screen-recorder 'echo "usage: ..."'
run gravacao >/dev/null
called "timeout 3 gpu-screen-recorder --help"
rm -f "$SBX/bin/timeout"
