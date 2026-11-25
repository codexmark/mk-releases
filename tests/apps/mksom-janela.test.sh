#!/usr/bin/env bash
# apps/mksom: a janela (qml/) e o lançador (bin/mksom). O desenho se vê no
# bin/dev-shot; aqui: a janela só usa as cores do Mk, só pede ao motor o que o
# contrato tem, não chama comando de som nenhum por conta própria, aplica os
# ganhos no ritmo certo, tudo se alcança pelo teclado, cabe em 420×420, os
# dados de exemplo têm a forma do que o motor responde, o nome de uma
# predefinição é recusado pelas mesmas regras do motor, e o lançador abre uma
# janela só.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mksom
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
exemplo=$TESTS/apps/mksom-exemplo
motor=$app/bin/mksom-motor
# Os comandos do PipeWire não estão entre os guardas do lib.sh: aqui eles são
# sempre de mentira, para o de verdade nunca rodar.
for c in pw-dump pw-cli pw-record pw-play pipewire; do stub "$c" 'exit 1'; done

t "cores: só os tokens do Mk, e só no Theme.qml"
tokens=" #0b0b0b #141211 #1b1917 #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml "$app/share/mksom.svg" | sort -u)
eq "" "$(grep -lE '#[0-9a-fA-F]{3,8}\b' "$qml"/*.qml | grep -v '/Theme.qml$' || true)"
out=$(grep -ohE 'Qt\.rgba\([^)]*\)' "$qml"/*.qml | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255|0, 0, 0, 0)' || true)
eq "" "$out"
eq "" "$(grep -nE 'color: *"[a-z]+"|Qt\.(darker|lighter|hsla|hsva|tint|color)\(|(fill|stroke)Style = "' "$qml"/*.qml || true)"

t "o Theme e os componentes pequenos são os mesmos dos outros apps (cada app é uma pasta que se basta)"
for f in Theme Botao Segmento Rotulo; do
  ok cmp -s "$qml/$f.qml" "$REPO/apps/mkpulso/qml/$f.qml"
done
ok cmp -s "$qml/Chave.qml" "$REPO/apps/mkutil/qml/Chave.qml"
ok cmp -s "$qml/Fundo.qml" "$REPO/apps/mkporto/qml/Fundo.qml"

t "todo tipo próprio usado na janela tem o arquivo dele, e nenhum arquivo sobra"
n=0
for tipo in $(grep -ohE '^\s*(Botao|Campo|Segmento|Rotulo|Chave|Fundo|Interruptor|Banda|Deslizante|Curva|Medidor|Equalizador|Microfone|Predefinicoes|Nome|Atalhos|App) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  n=$((n + 1))
  exists "$qml/$tipo.qml"
done
eq 17 "$n"
exists "$qml/Motor.qml"
exists "$qml/Som.js"
for f in "$qml"/*.qml; do
  tipo=$(basename "$f" .qml)
  [[ $tipo == shell || $tipo == Theme || $tipo == Motor ]] && continue
  grep -qE "^\s*$tipo \{" "$qml"/*.qml || _fail "ninguém usa: $tipo.qml"
done

casca=$(<"$qml/App.qml")
ponte=$(<"$qml/Motor.qml")

t "a janela só pede ao motor o que está no contrato, e só o App fala com ele"
eq "$qml/App.qml" "$(grep -lE 'Motor\.(pedir|fluxo|limpar)\(' "$qml"/*.qml | tr '\n' ' ' | sed 's/ $//')"
# Os pedidos, um por um.
has "$casca" 'Motor.pedir(["estado"], r => {'
has "$casca" 'Motor.pedir(["eq", app.ligado ? "desligar" : "ligar"], r => {'
has "$casca" 'Motor.pedir(["eq", "ajustar"], r => {'
has "$casca" '}, JSON.stringify(Som.pedido(app.ganhos, app.pre)));'
has "$casca" 'Motor.pedir(["eq", "predefinicao", "aplicar", "--", p.nome], r => {'
has "$casca" 'const args = de === "" ? ["eq", "predefinicao", "guardar", "--", nome] : ["eq", "predefinicao", "renomear", "--", de, nome];'
has "$casca" 'Motor.pedir(["eq", "predefinicao", "apagar", "--", nome], r => {'
has "$casca" 'pedirAoMic(["mic", "padrao", String(id)]'
has "$casca" 'pedirAoMic(["mic", "volume", String(id), String(v)]'
has "$casca" 'pedirAoMic(["mic", "mudo", String(id), mudo ? "sim" : "nao"]'
has "$casca" 'Motor.fluxo(["mic", "nivel", String(app.entradaId)]'
has "$casca" 'Motor.fluxo(["mic", "gravar", String(app.entradaId)]'
has "$casca" 'Motor.fluxo(["mic", "ouvir"]'
# E nenhum outro: toda lista que começa com um subcomando é uma das de cima.
mapfile -t subs < <(grep -ohE '\["(estado|eq|mic)"(, "[a-z]+")*' "$qml/App.qml" | tr -d '[",' | sed 's/  */ /g' | sort -u)
eq "eq|eq ajustar|eq predefinicao apagar|eq predefinicao aplicar|eq predefinicao guardar|eq predefinicao renomear|estado|mic gravar|mic mudo|mic nivel|mic ouvir|mic padrao|mic volume" \
  "$(IFS='|'; echo "${subs[*]}")"
for sub in "eq ligar" "eq desligar" "eq ajustar" "eq predefinicao" "mic listar" "mic padrao" "mic nivel" "mic gravar" estado; do
  has "$contrato" "## \`mksom-motor $sub" "o contrato não descreve: $sub"
done
eq 10 "$(grep -cE 'Motor\.(pedir|fluxo)\(' "$qml/App.qml")"

t "o nome de uma predefinição vai para o motor depois de --, nunca como opção"
eq 4 "$(grep -oE '"(aplicar|guardar|renomear|apagar)", "--"' "$qml/App.qml" | wc -l)"
eq "" "$(grep -nE '"(aplicar|guardar|renomear|apagar)", [^"]' "$qml/App.qml" || true)"

t "a janela não chama comando de som nenhum: só o motor (e o jq, para os exemplos)"
eq "" "$(grep -n 'command:' "$qml"/*.qml | grep -v '/Motor.qml:' || true)"
eq "$qml/Motor.qml:            Quickshell.execDetached([bin, \"mic\", \"limpar\"]);" "$(grep -n 'execDetached' "$qml"/*.qml | sed -E 's/:[0-9]+:/:/')"
eq "" "$(grep -nE '"(sh|bash|pw-[a-z]+|pipewire|wpctl|pactl|systemctl|sudo|pkexec|rm|easyeffects)"' "$qml"/*.qml "$qml/Som.js" || true)"
eq "" "$(grep -niE 'FileView|XMLHttpRequest|QtMultimedia|MediaPlayer|SoundEffect|AudioInput' "$qml"/*.qml "$qml/Som.js" || true)"
has "$ponte" 'return ["jq", "-c", emLinhas ? ".[]" : ".", exemplo + "/" + Som.arquivoDeExemplo(args)];'
has "$ponte" 'return [bin].concat(args);'
has "$ponte" 'Quickshell.env("MKSOM_MOTOR") || Quickshell.shellPath("../bin/mksom-motor")'

t "com MKSOM_EXEMPLO a janela não chama o motor nem toca ou grava som"
has "$ponte" 'Quickshell.env("MKSOM_EXEMPLO")'
like "$ponte" 'function comando\(args, emLinhas\) \{[[:space:]]+if \(exemplo !== ""\)[[:space:]]+return \["jq"'
like "$ponte" 'function limpar\(\) \{[[:space:]]+if \(exemplo === ""\)[[:space:]]+Quickshell\.execDetached'
# O pedido do ajustar não vai para o jq.
has "$ponte" 'const tem = typeof entrada === "string" && exemplo === "";'
# Todo processo da ponte sai de comando(): não há outro jeito de montar um.
eq 2 "$(grep -c 'command: comando(args, ' "$qml/Motor.qml")"

t "um fluxo segura a entrada padrão do motor aberta, e parar a fecha (o motor mata o pw-record)"
like "$ponte" 'id: longo[[:space:]]+Process \{'
has "$ponte" 'stdinEnabled: true'
like "$ponte" 'function parar\(\) \{[[:space:]]+l\.aCadaLinha = null;[[:space:]]+l\.stdinEnabled = false;[[:space:]]+l\.running = false;'
has "$contrato" "Para quando a entrada padrão do motor fecha"

t "arrastar aplica ao soltar e, no arraste, no máximo 10 vezes por segundo e uma de cada vez"
banda=$(<"$qml/Banda.qml")
# A banda avisa a cada passo do arraste (soltou = false) e ao soltar (true).
has "$banda" 'banda.mexido(v, false);'
has "$banda" 'onReleased: banda.mexido(banda.valor, true)'
eq 2 "$(grep -c 'banda.mexido(v, false);' "$qml/Banda.qml")"
has "$casca" 'onMexido: (banda, valor, soltou) => app.mexer(banda, valor, soltou)'
like "$casca" 'function mexer\(banda, valor, soltou\) \{[[:space:]]+app\.ganhos = Som\.trocar\(app\.ganhos, banda, valor\);[[:space:]]+app\.pedirAplicacao\(soltou\);'
like "$casca" 'function pedirAplicacao\(ja\) \{[[:space:]]+app\.sujo = true;[[:space:]]+if \(app\.emVoo\)[[:space:]]+return;'
has "$casca" 'if (ja || Som.devoAplicar(agora, app.ultimaAplicacao, false)) {'
has "$casca" 'depois.interval = Math.max(1, Som.esperaPara(agora, app.ultimaAplicacao));'
like "$casca" 'function aplicar\(\) \{[[:space:]]+if \(app\.emVoo \|\| !app\.sujo\)[[:space:]]+return;'
# Quando a aplicação volta e os controles mudaram de novo, sai outra.
like "$casca" 'app\.emVoo = false;[[:space:]]+app\.recado = r\.erro \|\| "";[[:space:]]+if \(app\.sujo\)[[:space:]]+app\.pedirAplicacao\(false\);'
has "$(<"$qml/Som.js")" 'var INTERVALO_MS = 100;'
# O único lugar que chama o eq ajustar é o aplicar().
eq 1 "$(grep -c '"eq", "ajustar"' "$qml/App.qml")"
# Os dois relógios da janela: o da aplicação adiada e o do pico que cai.
eq 2 "$(grep -cE '^\s*Timer \{' "$qml/App.qml")"
eq "" "$(grep -lE '\bTimer \{|setInterval|setTimeout' "$qml"/*.qml "$qml/Som.js" | grep -v '/App.qml$' || true)"

t "o equalizador tem a chave grande, as 10 bandas, o pré-ganho, a curva, o seletor, guardar como e plano"
eqz=$(<"$qml/Equalizador.qml")
has "$eqz" 'Interruptor {'
has "$eqz" 'model: Som.BANDAS'
has "$eqz" 'Banda {'
has "$eqz" 'Curva {'
has "$eqz" 'Deslizante {'
has "$eqz" 'min: Som.PRE_MIN'
has "$eqz" 'max: Som.PRE_MAX'
has "$eqz" 'texto: equalizador.estreita ? "guardar…" : "guardar como…"'
has "$eqz" 'texto: "plano"'
has "$eqz" 'onAcionado: equalizador.predefinicoesPedidas()'
# A frequência embaixo e o ganho em cima.
like "$banda" 'id: numero[[:space:]]+anchors\.top: parent\.top'
like "$banda" 'id: nome[[:space:]]+anchors\.bottom: parent\.bottom'
has "$banda" 'text: Som.textoDoGanho(banda.valor)'
has "$banda" 'text: Som.rotulo(banda.frequencia, banda.estreita)'
# A curva vem das contas do Som.js.
has "$(<"$qml/Curva.qml")" 'const pontos = Som.pontos(ganhos, pre, width, height'
# O aviso de distorção e o botão que baixa o pré-ganho.
has "$eqz" 'readonly property string aviso: Som.avisoDeDistorcao(ganhos, pre)'
has "$eqz" 'texto: "baixar"'
has "$casca" 'onPreCorrigido: app.mexerPre(Som.preSugerido(app.ganhos), true)'
has "$(<"$qml/Som.js")" '"pode distorcer: '
# A predefinição mostrada é a que tem os valores de agora.
has "$casca" 'readonly property string predefinicao: Som.predefinicaoDe(predefinicoes, ganhos, pre)'

t "as predefinições: aplicar, guardar como, renomear e apagar (só as da pessoa)"
sel=$(<"$qml/Predefinicoes.qml")
has "$sel" 'onClicked: seletor.escolhida(item.modelData.nome)'
eq 2 "$(grep -c 'visible: item.modelData.fabrica !== true' "$qml/Predefinicoes.qml")"
has "$sel" 'texto: "renomear"'
has "$sel" 'texto: "apagar"'
has "$casca" 'onRenomearPedido: nome => app.abrirFolha("nome", nome)'
has "$casca" 'onApagarPedido: nome => app.apagarPredefinicao(nome)'
has "$(<"$qml/Nome.qml")" 'readonly property string problema: Som.erroDoNome(digitado, lista)'
has "$(<"$qml/Nome.qml")" 'ligado: !nome.ocupado && nome.problema === ""'

t "tudo se alcança pelo teclado: Tab entre as bandas, ↑/↓, Page Up/Down, 0, Espaço, Ctrl+1/2, F1, Esc"
for f in Banda Deslizante Interruptor; do
  has "$(<"$qml/$f.qml")" 'activeFocusOnTab: true' "$f não recebe o Tab"
done
has "$banda" '[Qt.Key_Up]: "cima", [Qt.Key_Down]: "baixo", [Qt.Key_PageUp]: "pagina_cima", [Qt.Key_PageDown]: "pagina_baixo", [Qt.Key_0]: "zero"'
has "$banda" 'banda.mexido(Som.mover(banda.valor, tecla, false), true);'
# Com uma folha por cima, as teclas não mexem na banda escondida.
has "$banda" 'if (!banda.ativo || !tecla ||'
has "$casca" 'ativo: app.livre'
# A banda em foco se vê.
has "$banda" 'visible: banda.activeFocus'
has "$casca" 'Shortcut { sequences: ["Space"]; enabled: app.livre && app.area === "equalizador"; onActivated: app.alternar() }'
has "$casca" 'Shortcut { sequences: ["Ctrl+1"]; enabled: app.livre; onActivated: app.area = "equalizador" }'
has "$casca" 'Shortcut { sequences: ["Ctrl+2"]; enabled: app.livre; onActivated: app.area = "microfone" }'
has "$casca" 'Shortcut { sequences: ["Ctrl+S"]; enabled: app.livre && app.area === "equalizador"; onActivated: app.abrirFolha("nome") }'
has "$casca" 'Shortcut { sequences: ["Ctrl+0"]; enabled: app.livre && app.area === "equalizador"; onActivated: app.aplicarPredefinicao("plano") }'
has "$casca" 'Shortcut { sequences: ["Escape"]; onActivated: app.fechar() }'
has "$casca" 'Shortcut { sequences: ["F1"]; onActivated: app.folha = app.folha === "atalhos" ? "" : app.folha === "" ? "atalhos" : app.folha }'
has "$casca" 'Shortcut { sequences: ["Up"]; enabled: app.folha === "predefinicoes"; onActivated: seletor.andar(-1) }'
has "$casca" 'Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "predefinicoes"; onActivated: seletor.confirmar() }'
# Ao abrir (e ao fechar uma folha) o foco vai para a primeira banda: o Tab tem de onde partir.
like "$casca" 'Component\.onCompleted: \{[[:space:]]+ler\(\);[[:space:]]+if \(app\.area === "equalizador"\)[[:space:]]+equalizador\.focar\(\);'
for tecla in Tab '↑' 'Page Up' 0 'Espaço' 'Ctrl+1' 'Ctrl+2' 'Ctrl+S' 'Ctrl+0' F1 Esc; do
  has "$contrato" "\`$tecla" "o contrato não tem o atalho: $tecla"
done
for tecla in Tab '↑ ↓' 'Page Up / Down' 0 'Espaço' 'Ctrl+0' 'Ctrl+S' 'Ctrl+1' 'Ctrl+2' F1 Esc; do
  has "$(<"$qml/Som.js")" "[\"$tecla\", " "a folha de atalhos não tem: $tecla"
done
has "$(<"$qml/Atalhos.qml")" 'model: Som.ATALHOS'

t "o microfone: entradas, medidor com pico que cai, volume, mudo, gravar e ouvir com contagem, diagnóstico"
mic=$(<"$qml/Microfone.qml")
has "$mic" 'model: microfone.entradas'
has "$mic" 'Medidor {'
has "$mic" 'max: Som.VOLUME_MAX'
has "$mic" 'Chave {'
has "$mic" 'texto: Som.textoDaGravacao(microfone.etapa, microfone.falta)'
has "$mic" 'readonly property var diagnostico: Som.diagnostico(entradas, entrada, niveis, ouvindo)'
has "$mic" 'text: microfone.recado !== "" ? microfone.recado : microfone.diagnostico.texto'
has "$mic" '"Nenhuma entrada de som.'
has "$casca" 'app.pico = Som.picoQueCai(app.pico, l.pico, 0);'
has "$casca" 'onTriggered: app.pico = Som.picoQueCai(app.pico, Som.PISO, interval)'
has "$casca" 'app.niveis = Som.junta(app.niveis, l);'
# As frases do diagnóstico saem do .js puro.
for frase in '"nenhuma entrada"' '"a entrada está muda"' '"o nível está muito baixo (fale perto / suba o volume)"' '"está saturando (baixe o volume)"'; do
  has "$(<"$qml/Som.js")" "$frase"
  has "$contrato" "${frase//\"/}" "o contrato não tem a frase"
done

t "o medidor só roda com a área Microfone à vista, e para (com a gravação) quando a janela fecha"
like "$casca" 'function ouvir\(\) \{[[:space:]]+app\.pararDeOuvir\(\);[[:space:]]+if \(app\.area !== "microfone" \|\| app\.entradaId < 0 \|\| !app\.responde\)[[:space:]]+return;'
like "$casca" 'if \(app\.area === "microfone"\) \{[[:space:]]+app\.ouvir\(\);[[:space:]]+\} else \{[[:space:]]+app\.pararDeOuvir\(\);'
like "$casca" 'Component\.onDestruction: \{[[:space:]]+if \(app\.medidor\)[[:space:]]+app\.medidor\.parar\(\);[[:space:]]+if \(app\.gravador\)[[:space:]]+app\.gravador\.parar\(\);[[:space:]]+Motor\.limpar\(\);'
# Gravar e ouvir: só toca se gravou, e uma gravação de cada vez.
like "$casca" 'function gravarEOuvir\(\) \{[[:space:]]+if \(app\.etapa !== "" \|\| app\.entradaId < 0\)[[:space:]]+return;'
like "$casca" 'if \(!gravou\) \{[[:space:]]+app\.etapa = "";[[:space:]]+return;'

t "a janela se chama MkSom, nasce por volta de 560x520 (ou o que couber) e cabe em 420x420"
janela=$(<"$qml/shell.qml")
has "$janela" 'title: "MkSom"'
like "$janela" 'implicitWidth: Math\.min\(560, .*screen'
like "$janela" 'implicitHeight: Math\.min\(520, .*screen'
has "$janela" 'minimumSize: Qt.size(400, 400)'
has "$janela" 'FloatingWindow {'
has "$contrato" 'Título e classe: `MkSom`'

t "abaixo de 480 de largura os controles estreitam e os rótulos encurtam; abaixo de 470 de altura a curva some"
has "$(<"$qml/Som.js")" 'var LARGURA_ESTREITA = 480;'
has "$(<"$qml/Som.js")" 'var ALTURA_BAIXA = 470;'
has "$casca" 'readonly property bool estreita: Som.estreita(width)'
has "$casca" 'readonly property bool baixa: Som.baixa(height)'
has "$casca" 'anchors.margins: app.estreita ? 12 : 18'
has "$casca" 'visible: !app.estreita'
has "$banda" 'width: Math.min(parent.width, banda.estreita ? 24 : 30)'
has "$banda" 'font.pixelSize: banda.estreita ? 8 : 9'
has "$eqz" 'visible: !equalizador.baixa'
has "$eqz" 'width: fileira.width / Som.BANDAS.length'
has "$eqz" 'Rotulo { text: equalizador.estreita ? "pré" : "pré-ganho" }'
has "$mic" 'texto: microfone.estreita ? "padrão" : "tornar a padrão"'

t "quando o PipeWire não responde a janela diz e oferece tentar de novo"
has "$casca" 'visible: app.pronto && !app.responde'
has "$casca" '"O som não respondeu"'
has "$casca" 'texto: "tentar de novo"'

t "os dados de exemplo são JSON, um por pedido que a janela faz"
arquivos=(estado eq-ligar eq-desligar eq-ajustar eq-predefinicao-aplicar eq-predefinicao-guardar eq-predefinicao-renomear
  eq-predefinicao-apagar mic-listar mic-padrao mic-volume mic-mudo mic-nivel mic-gravar mic-ouvir mic-limpar)
for f in "${arquivos[@]}"; do
  ok jq -e . "$exemplo/$f.json" >/dev/null
  has "$contrato" "\`$f.json\`" "o contrato não cita o exemplo $f.json"
done
eq "${#arquivos[@]}" "$(find "$exemplo" -type f | wc -l)"

t "os exemplos mostram a janela cheia: equalizador ligado, uma predefinição da pessoa, três entradas"
ok jq -e '.som.responde and .eq.ligado and .eq.na_saida and .eq.predefinicao == "graves" and .saida_padrao.nome == "mksom.eq"' "$exemplo/estado.json" >/dev/null
ok jq -e '[.eq.predefinicoes[] | select(.fabrica)] | length == 6' "$exemplo/estado.json" >/dev/null
ok jq -e '[.eq.predefinicoes[] | select(.fabrica | not)] | length >= 1' "$exemplo/estado.json" >/dev/null
ok jq -e '(.entradas | length) == 3 and ([.entradas[] | select(.padrao)] | length == 1) and ([.entradas[] | select(.mudo)] | length == 1)' "$exemplo/estado.json" >/dev/null
ok jq -e '(.eq.ganhos | length) == 10 and (.eq.bandas | length) == 10' "$exemplo/estado.json" >/dev/null
# Os fluxos são listas: uma linha por item.
ok jq -e 'type == "array" and ([.[:-1][] | has("pico") and has("rms")] | all) and .[-1] == {"fim": true}' "$exemplo/mic-nivel.json" >/dev/null
jeq '[{"falta":5},{"falta":4},{"falta":3},{"falta":2},{"falta":1},{"ok":true,"segundos":5}]' "$(<"$exemplo/mic-gravar.json")"

t "nada de dados da máquina de quem escreveu nos exemplos"
eq "" "$(grep -lE '/home/[a-z]+/|codexmark|/tmp/' "$exemplo"/*.json || true)"
eq "" "$(jq -r '.. | objects | .nome? // empty' "$exemplo"/*.json | grep -E '^(alsa|bluez)_' | grep -v 'exemplo' || true)"

# --- os exemplos contra o motor de verdade --------------------------------------
# Um PipeWire de mentira com as mesmas três entradas: o motor responde, e o
# exemplo tem de ter os mesmos campos.
no() { printf '{"id": %s, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "%s", "node.description": "%s", "media.class": "%s", "object.serial": %s}, "params": {"Props": [{"mute": false, "channelVolumes": [0.125, 0.125]}]}}}' "$@"; }
dump() { # dump <com o equalizador? 0|1> <node.name da saída padrão>
  echo "["
  no 51 alsa_output.exemplo.analog-stereo "Alto-falantes" Audio/Sink 310
  echo ,
  no 52 alsa_input.exemplo.analog-stereo "Microfone interno" Audio/Source 311
  echo ,
  if [[ $1 == 1 ]]; then
    no 71 mksom.eq "Mk Equalizador" Audio/Sink 320
    echo ,
  fi
  printf '{"id": 36, "type": "PipeWire:Interface:Metadata", "props": {"metadata.name": "default"}, "metadata": [{"key": "default.audio.sink", "value": {"name": "%s"}}, {"key": "default.audio.source", "value": {"name": "alsa_input.exemplo.analog-stereo"}}]}\n]\n' "$2"
}
dump 0 alsa_output.exemplo.analog-stereo >"$SBX/desligado.json"
dump 1 mksom.eq >"$SBX/ligado.json"
cp "$SBX/desligado.json" "$SBX/dump.json"
stub pw-dump 'cat "$SBX/dump.json"'
stub systemctl 'case "$*" in
  *"enable --now"*) cp "$SBX/ligado.json" "$SBX/dump.json" ;;
  *"disable --now"*) cp "$SBX/desligado.json" "$SBX/dump.json" ;;
esac'
stub wpctl
stub pw-cli
stub pw-play
responde() { "$motor" "$@"; }
# Os campos de um JSON, em qualquer profundidade, sem os índices das listas.
campos() { jq -r '[paths | map(select(type == "string")) | join(".")] | unique | .[]' <<<"$1"; }
mesmos_campos() {
  local nome=$1 veio=$2
  eq "$(campos "$veio")" "$(campos "$(<"$exemplo/$nome.json")")" "$nome.json não tem os campos que o motor responde"
}

t "exemplos: ligar, estado, ajustar e desligar têm exatamente os campos que o motor responde"
mesmos_campos eq-ligar "$(responde eq ligar)"
mesmos_campos estado "$(responde estado)"
mesmos_campos eq-ajustar "$(responde eq ajustar <<<'{"ganhos":[1,0,0,0,0,0,0,0,0,0],"pre":-1}')"
mesmos_campos eq-predefinicao-aplicar "$(responde eq predefinicao aplicar -- graves)"
mesmos_campos eq-desligar "$(responde eq desligar)"

t "exemplos: as predefinições e o microfone também"
mesmos_campos eq-predefinicao-guardar "$(responde eq predefinicao guardar -- Minha)"
mesmos_campos eq-predefinicao-renomear "$(responde eq predefinicao renomear -- Minha Sala)"
mesmos_campos eq-predefinicao-apagar "$(responde eq predefinicao apagar -- Sala)"
mesmos_campos mic-listar "$(responde mic listar)"
mesmos_campos mic-padrao "$(responde mic padrao 52)"
mesmos_campos mic-volume "$(responde mic volume 52 80)"
mesmos_campos mic-mudo "$(responde mic mudo 52 sim)"
mesmos_campos mic-limpar "$(responde mic limpar)"
mkdir -p "$XDG_RUNTIME_DIR/mksom"
printf RIFF >"$XDG_RUNTIME_DIR/mksom/teste.wav"
mesmos_campos mic-ouvir "$(responde mic ouvir < <(sleep 2))"
# O medidor, linha a linha.
stub pw-record 'head -c 2132 /dev/zero'
veio=$(responde mic nivel 52 < <(sleep 2))
eq "$(campos "$(head -1 <<<"$veio")")" "$(campos "$(jq -c '.[0]' "$exemplo/mic-nivel.json")")"
jeq "$(tail -1 <<<"$veio")" "$(jq -c '.[-1]' "$exemplo/mic-nivel.json")"

t "a folha do nome recusa o mesmo que o motor: as regras do Som.js são o espelho das do guardar"
command -v node >/dev/null || skip "falta o node"
cat >"$SBX/espelho.mjs" <<MJS
import { load, REPO } from "$TESTS/js/load.mjs";
const S = load(REPO + "/apps/mksom/qml/Som.js");
const lista = JSON.parse(process.argv[3]);
console.log(S.erroDoNome(JSON.parse(process.argv[2]), lista));
MJS
lista=$(jq -c '.eq.predefinicoes' <<<"$(responde estado)")
n=0
while IFS= read -r pedido; do
  n=$((n + 1))
  # (o nome vai como argumento de verdade: o jq o tira do JSON)
  mapfile -d '' -t partes < <(jq -j '., "\u0000"' <<<"$pedido")
  veio=$(MKSOM_CONFIG=$SBX/espelho-$n "$motor" eq predefinicao guardar -- "${partes[0]:-}")
  do_motor=$(jq -r '.erro // ""' <<<"$veio")
  da_janela=$(node "$SBX/espelho.mjs" "$pedido" "$lista")
  eq "$do_motor" "$da_janela" "o motor e a janela discordam em: $pedido"
done <<'PEDIDOS'
"minha"
"Fone do trabalho"
"  com espaços  "
"rock & jazz (2)"
"água"
"éééééééééééééééééééééééééééééééééééééééé"
"ééééééééééééééééééééééééééééééééééééééééé"
"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
""
"   "
"a/b"
"/"
"a\\b"
".."
"."
"..."
"-rf"
"--"
" -x"
"a\nb"
"a\n"
"\n"
"a\tb"
"a\u001bb"
"a\u007fb"
"a\u0085"
"a "
"plano"
"GRAVES"
"Fone Pequeno"
" voz "
"$(reboot)"
"a;b"
"😀 festa"
PEDIDOS
ok test "$n" -ge 30

# --- o lançador ---------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml"
cp "$app/bin/mksom" "$casa/bin/mksom"
printf '#!/bin/sh\n' >"$casa/bin/mksom-motor"
chmod +x "$casa/bin/mksom-motor"
stub notify-send
stub hyprctl '[[ $1 == clients ]] && cat "$SBX/clientes.json" 2>/dev/null; exit 0'
stub qs '[[ $1 == list ]] && cat "$SBX/qs-list" 2>/dev/null; [[ $1 == -p ]] && echo "${MKSOM_AREA:-}" >"$SBX/area"; exit 0'
abre() { bash "$casa/bin/mksom" "$@"; }

t "o lançador, o pack, a assinatura e o motor têm o bit de execução"
for f in bin/mksom pack assinatura bin/mksom-motor; do
  ok test -x "$app/$f"
done

t "sem janela aberta: abre a janela do app (qs -p …/qml), no Equalizador"
calls_reset
ok abre
called "qs -p $casa/qml"
eq "equalizador" "$(<"$SBX/area")"

t "mksom microfone abre direto no teste de microfone"
calls_reset
ok abre microfone
called "qs -p $casa/qml"
eq "microfone" "$(<"$SBX/area")"
has "$ponte" 'readonly property string areaInicial: Som.area(Quickshell.env("MKSOM_AREA") || "")'
has "$casca" 'property string area: Motor.areaInicial'

t "argumento que o lançador não conhece: uso e saída 2, sem abrir nada"
calls_reset
abre volume 2>/dev/null
eq 2 $?
abre microfone outro 2>/dev/null
eq 2 $?
abre -- -x 2>/dev/null
eq 2 $?
not_called "qs -p"

t "janela já aberta: só traz para a frente, não abre outra"
calls_reset
echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
echo '[{"title":"MkPulso","address":"0x1"},{"title":"MkSom","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre
called 'hl.dsp.focus({ window = "address:0x55aa" })'
not_called "qs -p"

t "uma janela com outro título não conta como a do app"
calls_reset
echo '[{"title":"MkSom — outra coisa","address":"0x3"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "um endereço torto vindo do Hyprland não entra no comando: abre a janela normalmente"
calls_reset
echo '[{"title":"MkSom","address":"0x1\" }) os.execute(\"x"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "sem o motor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$SBX/qs-list" "$casa/bin/mksom-motor"
out=$(abre 2>&1)
eq 1 $?
has "$out" "motor"
has "$out" "mk atualizar"
called "notify-send"
not_called "qs -p"

t "sem o Quickshell: avisa e sai com 1"
calls_reset
printf '#!/bin/sh\n' >"$casa/bin/mksom-motor"
chmod +x "$casa/bin/mksom-motor"
semqs=$(path_sem qs)
out=$(PATH=$semqs abre 2>&1)
eq 1 $?
has "$out" "Quickshell"
called "notify-send"

t "o lançador não mexe no som: nem systemctl, nem wpctl, nem pw-*"
eq "" "$(grep -nE '\b(systemctl|wpctl|pactl|pipewire|pw-[a-z]+|sudo|pkexec)\b' "$app/bin/mksom" | grep -v '^[0-9]*:#' || true)"

t "o atalho do menu chama o lançador e usa o ícone do app"
atalho=$(<"$app/share/mksom.desktop")
has "$atalho" "Exec=mksom"
has "$atalho" "Icon=mksom"
has "$atalho" "Name=MkSom"
has "$atalho" "Terminal=false"
hasnt "$atalho" "MimeType"
hasnt "$atalho" "%"
exists "$app/share/mksom.svg"
like "$(<"$app/share/mksom.svg")" '<title>MkSom</title>'
