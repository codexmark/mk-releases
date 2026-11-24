#!/usr/bin/env bash
# apps/mkcorte: a janela (qml/) e o lançador (bin/mkcorte). O desenho se vê no
# bin/dev-shot; aqui: a janela só usa as cores do Mk, só pede ao motor o que o
# contrato tem, os dados de exemplo seguem o contrato, todo atalho de teclado
# tem quem o atenda e o lançador abre a janela nos arquivos dados.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mkcorte
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
exemplo=$TESTS/apps/mkcorte-exemplo

t "cores: só os tokens do Mk, e só no Theme.qml"
tokens=" #0b0b0b #141211 #1b1917 #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml | sort -u)
eq "" "$(grep -lE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml | grep -v '/Theme.qml$' || true)"
out=$(grep -ohE 'Qt\.rgba\([^)]*\)' "$qml"/*.qml | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255|0, 0, 0, 0)' || true)
eq "" "$out"
# Nem cor por nome ("red", "white") nem cor feita na hora.
eq "" "$(grep -nE 'color: *"[a-z]+"|Qt\.(hsla|hsva|color|lighter|darker|tint)\(' "$qml"/*.qml || true)"
# O ícone do menu também.
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta no ícone: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$app"/share/*.svg | sort -u)

t "o Theme e os componentes pequenos são os do Maré, iguais aos do MkPulso e do MkUtil"
for f in Theme Botao Segmento Rotulo; do
  eq "$(<"$REPO/apps/mkpulso/qml/$f.qml")" "$(<"$qml/$f.qml")" "$f.qml"
done
eq "$(<"$REPO/apps/mkutil/qml/Barra.qml")" "$(<"$qml/Barra.qml")" "Barra.qml"

t "todo tipo próprio usado na janela tem o arquivo dele, e todo arquivo é usado"
for tipo in $(grep -ohE '^\s*(Botao|Segmento|Rotulo|Barra|Editor|Previa|Linha|Exportar|Confirma|Atalhos) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  exists "$qml/$tipo.qml"
done
for f in "$qml"/*.qml; do
  tipo=$(basename "$f" .qml)
  [[ $tipo == shell || $tipo == Theme || $tipo == Motor ]] && continue
  ok grep -qE "^\s*$tipo \{" "$qml"/*.qml
done
exists "$qml/Motor.qml"
exists "$qml/Corte.js"

t "a janela só pede ao motor subcomandos que estão no contrato, e usa todos"
n=0
while read -r sub; do
  n=$((n + 1))
  has "$contrato" "\`mkcorte-motor $sub" "subcomando fora do contrato: $sub"
done < <(grep -ohE 'return \["[a-z]+"' "$qml/Corte.js" | grep -oE '"[a-z]+"' | tr -d '"' | sort -u)
eq 5 "$n"
# Nenhum pedido é montado solto num .qml: todos vêm das funções args* do Corte.js.
eq "" "$(grep -nE 'Motor\.(pedir|tarefa)\(' "$qml"/*.qml | grep -v 'Corte\.args[A-Z][a-z]*(' || true)"
eq 5 "$(grep -ohE 'Motor\.(pedir|tarefa)\(Corte\.args[A-Za-z]+' "$qml"/*.qml | sed 's/.*Corte\.//' | sort -u | wc -l)"

t "as opções que a janela manda são as do contrato"
n=0
while read -r opcao; do
  n=$((n + 1))
  has "$contrato" "$opcao" "opção fora do contrato: $opcao"
done < <(grep -ohE '"--[a-z-]+"' "$qml"/*.qml "$qml/Corte.js" | tr -d '"' | sort -u)
eq 2 "$n"

t "as qualidades, os modos e as fases que a janela usa são os do contrato"
for palavra in original 1080 720; do
  has "$contrato" "\`$palavra\`" "qualidade fora do contrato: $palavra"
  ok grep -q "v: \"$palavra\"" "$qml/Exportar.qml"
done
for palavra in exato rapido; do
  has "$contrato" "\`$palavra\`" "modo fora do contrato: $palavra"
  ok grep -q "v: \"$palavra\"" "$qml/Exportar.qml"
done
for palavra in plano exportando fim cancelado; do
  has "$contrato" "\"fase\": \"$palavra\"" "fase fora do contrato: $palavra"
done
for palavra in exportando fim cancelado; do
  ok grep -q "fase === \"$palavra\"" "$qml/Exportar.qml"
done

t "a janela não roda nada além do motor (e do jq, para os exemplos)"
eq "" "$(grep -n 'command:' "$qml"/*.qml | grep -v 'Motor.qml' || true)"
eq "" "$(grep -nE '"(sh|bash|sudo|pkexec|ffmpeg|ffprobe|rm|xdg-open)"' "$qml"/*.qml "$qml/Corte.js" || true)"
eq "" "$(grep -n 'Process\b' "$qml"/*.qml | grep -v 'Motor.qml' || true)"

t "o título da janela é MkCorte, e ela nunca nasce maior que a tela"
casca=$(<"$qml/shell.qml")
has "$casca" 'title: "MkCorte"'
has "$casca" 'settings.watchFiles: false'
like "$casca" 'implicitWidth: Math\.min\(1120, .*screen'
like "$casca" 'implicitHeight: Math\.min\(740, .*screen'
has "$casca" 'function onLastWindowClosed() { Qt.quit(); }'

t "a pré-visualização é do QtMultimedia: um tocador, a imagem e o som"
previa=$(<"$qml/Previa.qml")
has "$previa" "import QtMultimedia"
for tipo in "MediaPlayer {" "VideoOutput {" "AudioOutput {"; do
  has "$previa" "$tipo"
done
# Quem decide o salto entre clipes é a função pura, testada no node.
has "$previa" "Corte.avancar("
eq "" "$(grep -ln 'import QtMultimedia' "$qml"/*.qml | grep -v '/Previa.qml$' || true)"

t "todo atalho do Corte.js tem quem o atenda no Editor, e o Editor não atende nada a mais"
editor=$(<"$qml/Editor.qml")
mapfile -t acoes < <(sed -n "/^function acaoDaTecla/,/^}/p" "$qml/Corte.js" | grep -oE '"[a-zA-Z]+"' | tr -d '"' | sort -u)
eq 22 "${#acoes[@]}"
for acao in "${acoes[@]}"; do
  has "$editor" "case \"$acao\":" "ação sem quem atenda: $acao"
done
eq "${#acoes[@]}" "$(grep -cE '^\s+case "[a-zA-Z]+":' "$qml/Editor.qml")"
# O teclado todo entra por um lugar só.
eq 1 "$(grep -c 'Keys.onPressed' "$qml"/*.qml | awk -F: '{s+=$2} END {print s}')"
has "$editor" "Corte.acaoDaTecla(event.key, event.modifiers)"
has "$editor" "Corte.acaoDoDialogo(event.key, event.modifiers)"

t "as confirmações respondem pelo teclado: cada diálogo recebe as teclas do Editor"
for dialogo in exportar sair trocar; do
  has "$editor" "$dialogo.tecla(acao)"
done
for f in Confirma Exportar; do
  d=$(<"$qml/$f.qml")
  has "$d" "function tecla(acao)"
  for acao in proximo anterior confirmar cancelar; do
    has "$d" "\"$acao\"" "$f não trata: $acao"
  done
done
# Sair com edição não exportada pergunta; a pergunta tem como desistir.
has "$editor" 'if (Corte.haMudancas(editor.h))'
has "$(grep -A6 'id: sair' "$qml/Editor.qml")" 'v: "cancelar", texto: "continuar editando"'
has "$editor" 'onExportou: editor.h = Corte.marcarExportado(editor.h)'

t "abrir arquivo: linha de comando, soltar na janela e o seletor"
has "$editor" "Motor.arquivos"
has "$editor" "DropArea {"
has "$editor" "FileDialog {"
has "$editor" "fileMode: FileDialog.OpenFiles"
has "$editor" "solte um vídeo aqui"

t "a linha do tempo tem um MouseArea só, e pergunta ao Corte.toque o que há sob o ponteiro"
eq 1 "$(grep -c 'MouseArea {' "$qml/Linha.qml")"
has "$(<"$qml/Linha.qml")" "Corte.toque("
for sinal in irPara aparando aparou dividirEm removerClipe moverClipe; do
  has "$(<"$qml/Linha.qml")" "signal $sinal("
  has "$editor" "on${sinal^}:"
done

t "o diálogo de exportar mostra o plano, o aviso do modo rápido, o progresso, cancelar e mostrar na pasta"
d=$(<"$qml/Exportar.qml")
has "$d" "Corte.argsPlano("
has "$d" "Corte.argsExportar("
has "$d" "sem_recodificar.aviso"
has "$d" "sem_recodificar.motivo"
has "$d" "Corte.textoDoPlano("
has "$d" "Corte.textoDoProgresso("
has "$d" "exportar.tarefa.running = false"
has "$d" "mostrar na pasta"
has "$d" "Qt.openUrlExternally(Corte.urlDeCaminho(Corte.pasta(exportar.fim.saida)))"
# Fechar a janela no meio cancela a exportação.
has "$d" "Component.onDestruction: cancela()"

t "os dados de exemplo são JSON e existem para tudo que a janela pede"
for f in sondar miniaturas plano exportar quadro; do
  ok jq -e . "$exemplo/$f.json" >/dev/null
done
eq 5 "$(find "$exemplo" -type f | wc -l)"

t "os exemplos têm os campos do contrato que a janela lê"
ok jq -e 'has("arquivo") and has("tamanho_bytes") and has("duracao_s") and has("largura") and has("altura") and has("fps") and has("fps_fracao")
  and has("codec") and has("perfil") and has("formato_de_pixel") and has("sar") and has("rotacao") and has("taxa_bps") and has("trilha")
  and has("tem_audio") and (.audio | has("codec") and has("taxa_bps") and has("canais") and has("amostragem_hz") and has("trilha"))' "$exemplo/sondar.json" >/dev/null
ok jq -e 'has("arquivo") and has("duracao_s") and has("altura") and has("pasta") and has("do_cache") and (.quadros | type == "array")' "$exemplo/miniaturas.json" >/dev/null
plano='has("modo") and has("duracao_s") and has("clipes") and has("largura") and has("altura") and has("fps") and has("tem_audio")
  and has("tamanho_estimado_bytes") and has("saida") and (.comando | type == "array") and (.avisos | type == "array")
  and (.sem_recodificar | has("pode") and has("motivo") and has("aviso"))'
ok jq -e "$plano" "$exemplo/plano.json" >/dev/null
# As fases: uma lista, cada item com "fase"; a primeira é o plano e a última o fim.
ok jq -e 'type == "array" and ([.[] | has("fase")] | all) and .[0].fase == "plano" and .[-1].fase == "fim" and .[-1].ok' "$exemplo/exportar.json" >/dev/null
ok jq -e ".[0] | $plano" "$exemplo/exportar.json" >/dev/null
ok jq -e '([.[] | select(.fase == "exportando") | has("pct") and has("velocidade") and has("resta_s") and has("tamanho_bytes")] | all and length > 0)
  and (.[-1] | has("saida") and has("modo") and has("duracao_s") and has("tamanho_bytes"))' "$exemplo/exportar.json" >/dev/null
ok jq -e '.ok and has("saida") and has("tempo_s") and has("tamanho_bytes")' "$exemplo/quadro.json" >/dev/null
# O aviso do modo rápido no exemplo é o que o motor escreve.
aviso=$(grep -A1 '^const AvisoDoRapido' "$app/src/internal/midia/plano.go" | grep -oE '"[^"]+"' | tr -d '"' | tr -d '\n')
eq "$aviso" "$(jq -r .sem_recodificar.aviso "$exemplo/plano.json")"

t "nada de dados da máquina de quem escreveu nos exemplos"
eq "" "$(grep -lE '/home/[a-z]+/|codexmark' "$exemplo"/*.json || true)"

t "o atalho de menu: abre com os arquivos (%F) e se oferece para vídeos"
d=$(<"$app/share/mkcorte.desktop")
has "$d" "Name=MkCorte"
has "$d" "Exec=mkcorte %F"
has "$d" "Icon=mkcorte"
has "$d" "Terminal=false"
for tipo in video/mp4 video/x-matroska video/webm video/quicktime video/x-msvideo; do
  has "$d" "$tipo;"
done
hasnt "$d" "image/"
hasnt "$d" "audio/"
exists "$app/share/mkcorte.svg"
has "$(<"$app/share/mkcorte.svg")" "<title>MkCorte</title>"

# --- o lançador -----------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml" "$SBX/onde estou"
cp "$app/bin/mkcorte" "$casa/bin/"
printf '#!/bin/sh\n' >"$casa/bin/mkcorte-motor"
chmod +x "$casa/bin/mkcorte-motor"
stub notify-send
stub hyprctl '[[ $1 == clients ]] && cat "$SBX/clientes.json" 2>/dev/null; exit 0'
stub qs 'if [[ $1 == list ]]; then cat "$SBX/qs-list" 2>/dev/null; else echo "ARQUIVOS=${MKCORTE_ARQUIVOS:-}" >>"$SBX/calls"; fi'
# shellcheck disable=SC2120 # os argumentos chegam pelo `ok abre …`
abre() { (cd "$SBX/onde estou" && bash "$casa/bin/mkcorte" "$@"); }

t "o lançador tem o bit de execução, e é o do MkComprimir com outro nome"
ok test -x "$app/bin/mkcorte"
ok test -x "$app/pack"
ok test -x "$app/assinatura"
eq "$(grep -vE '^(#|titulo=|janela=|export MKUTIL_JANELA)' "$REPO/apps/mkutil/bin/mkcomprimir" | sed 's/MKUTIL/MKCORTE/g; s/mkutil/mkcorte/g')" \
  "$(grep -vE '^(#|titulo=)' "$app/bin/mkcorte")"
eq "titulo=MkCorte" "$(grep -E '^titulo=' "$app/bin/mkcorte")"

t "mkcorte: sem janela aberta, abre a janela (qs -p …/qml), sem arquivo"
calls_reset
rm -f "$SBX/qs-list" "$SBX/clientes.json"
ok abre
called "qs -p $casa/qml"
called "ARQUIVOS=[]"

t "mkcorte: os arquivos chegam inteiros e absolutos, mesmo com espaço, aspas, dois-pontos e começando com -"
calls_reset
# shellcheck disable=SC2016 # é o texto, não a expansão, que tem de chegar
ok abre "/tmp/a b.mp4" "-rf 'x' \"y\": z.mkv" "sub/é.webm" '$(touch pwn)`z`.mov'
# shellcheck disable=SC2016
called "ARQUIVOS=[\"/tmp/a b.mp4\",\"$SBX/onde estou/-rf 'x' \\\"y\\\": z.mkv\",\"$SBX/onde estou/sub/é.webm\",\"$SBX/onde estou/"'$(touch pwn)`z`.mov"]'
absent "$SBX/onde estou/pwn"

t "mkcorte: janela já aberta e nenhum arquivo: só traz para a frente, não abre outra"
calls_reset
echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
echo '[{"title":"MkPulso","address":"0x1"},{"title":"MkCorte","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre
called 'hl.dsp.focus({ window = "address:0x55aa" })'
not_called "qs -p"

t "mkcorte: a janela de outro app do Mk aberta não conta: abre a sua"
calls_reset
echo '[{"title":"MkComprimir","address":"0x77"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "mkcorte: janela já aberta e um arquivo novo: abre nele"
calls_reset
echo '[{"title":"MkCorte","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre /tmp/novo.mp4
not_called "dispatch"
called 'ARQUIVOS=["/tmp/novo.mp4"]'

t "mkcorte: um endereço torto vindo do Hyprland não entra no comando: abre a janela normalmente"
calls_reset
echo '[{"title":"MkCorte","address":"0x1\" }) os.execute(\"x"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "sem o motor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$SBX/qs-list" "$casa/bin/mkcorte-motor"
out=$(abre 2>&1)
eq 1 $?
has "$out" "motor"
has "$out" "mk atualizar"
called "notify-send"
not_called "qs -p"

t "sem o Quickshell: avisa e sai com 1"
calls_reset
semqs=$(path_sem qs)
out=$(PATH=$semqs abre 2>&1)
eq 1 $?
has "$out" "Quickshell"
called "notify-send"

# Vistos na VM, com a janela lado a lado com um terminal (uns 600 px):
t "a janela aceita ficar estreita, e os controles se ajustam em vez de se sobrepor"
has "$(<"$qml/shell.qml")" "minimumSize: Qt.size(520, 420)"
ed=$(<"$qml/Editor.qml")
has "$ed" "readonly property bool estreito: width < 720"
has "$ed" "visible: !editor.estreito"

t "no fim da edição o último quadro fica na tela (o tocador não limpa a imagem)"
has "$(<"$qml/Previa.qml")" "endOfStreamPolicy: VideoOutput.KeepLastFrame"

t "a folha de atalhos não declara mais linhas do que tem (aviso do Grid no log)"
hasnt "$(<"$qml/Atalhos.qml")" "rows: Math.ceil"
