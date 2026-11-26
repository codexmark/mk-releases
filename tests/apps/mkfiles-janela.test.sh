#!/usr/bin/env bash
# apps/mkfiles: a janela (qml/) e o lançador (bin/mkfiles). O desenho se vê no
# bin/dev-shot; aqui: a janela só usa as cores do Mk, só manda ao motor pedidos
# que estão no contrato, não mexe em arquivo nem monta comando (executa só o
# argv que o motor devolve, ou o próprio lançador), as visões são
# virtualizadas, todo atalho pedido existe, os dados de exemplo têm a forma do
# contrato, e o lançador abre uma janela por chamada com os caminhos inteiros.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mkfiles
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
# O contrato numa linha só: um `"op":` no fim de uma linha ainda casa.
contrato1=$(tr -s ' \n' ' ' <<<"$contrato")
exemplo=$TESTS/apps/mkfiles-exemplo
janela=$(<"$qml/App.qml")
painel=$(<"$qml/Painel.qml")
ponte=$(<"$qml/Motor.qml")
casca=$(<"$qml/shell.qml")

t "cores: só os tokens do Mk, e só no Theme.qml"
tokens=" #0b0b0b #141211 #1b1917 #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml "$app/share/mkfiles.svg" | sort -u)
eq "" "$(grep -lE '#[0-9a-fA-F]{3,8}\b' "$qml"/*.qml | grep -v '/Theme.qml$' || true)"
out=$(grep -ohE 'Qt\.rgba\([^)]*\)' "$qml"/*.qml | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255|0, 0, 0, 0)' || true)
eq "" "$out"
eq "" "$(grep -nE 'color: *"[a-z]+"|Qt\.(darker|lighter|hsla|hsva|tint|color)\(' "$qml"/*.qml || true)"

t "o Theme e os componentes pequenos são os mesmos do MkPorto (cada app é uma pasta que se basta)"
for f in Theme Botao Campo Segmento Rotulo Chave Fundo; do
  ok cmp -s "$qml/$f.qml" "$REPO/apps/mkporto/qml/$f.qml"
done

t "todo tipo próprio usado na janela tem o arquivo dele, e nenhum arquivo sobra"
n=0
for tipo in $(grep -ohE '^\s*([A-Z][A-Za-z]+) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  [[ -e $qml/$tipo.qml ]] || continue
  n=$((n + 1))
done
for f in "$qml"/*.qml; do
  tipo=$(basename "$f" .qml)
  [[ $tipo == shell || $tipo == Theme || $tipo == Motor ]] && continue
  # O tocador da prévia é criado por um Loader, pelo nome do arquivo.
  if [[ $tipo == PreviaMidia ]]; then
    grep -qF 'source: "PreviaMidia.qml"' "$qml/Previa.qml" || _fail "ninguém carrega: $tipo.qml"
    continue
  fi
  grep -qE "^\s*(delegate: )?$tipo \{" "$qml"/*.qml || _fail "ninguém usa: $tipo.qml"
done
ok test "$n" -ge 20
for tipo in App Barra Lateral Painel Cartao Linha Tarefas MenuContexto Conflito Propriedades AbrirCom Pergunta Nome Atalhos Folha Icone \
  Rolagem Deslize Opcao Senha PedeSenha Compactar EmLote Previa; do
  exists "$qml/$tipo.qml"
  grep -qE "^\s*(delegate: )?$tipo \{" "$qml"/*.qml || _fail "tipo sem uso: $tipo"
done
for f in Arquivos Selecao Abas Menu Lote; do
  exists "$qml/$f.js"
  exists "$TESTS/apps/mkfiles-$f.test.mjs"
done
# Nada de fora da pasta: nem o shell do Maré nem outro app.
eq "" "$(grep -nE '^\s*import +"(\.\./|/)' "$qml"/*.qml || true)"
eq "" "$(grep -n 'shell/mare' "$qml"/*.qml "$qml"/*.js || true)"
# Só os módulos que o Mk sempre tem. O QtMultimedia (o tocador da prévia) é
# importado num arquivo só, que a janela cria por um Loader: se o módulo
# faltar, é só esse pedaço que não carrega.
eq "QtQuick QtQuick.Layouts Quickshell Quickshell.Io" "$(grep -ohE '^import [A-Z][A-Za-z.]+' "$qml"/*.qml | awk '{print $2}' | grep -v '^QtMultimedia$' | sort -u | tr '\n' ' ' | sed 's/ $//')"
eq "$qml/PreviaMidia.qml" "$(grep -lE '^import QtMultimedia' "$qml"/*.qml)"
eq "" "$(grep -nE '^\s*PreviaMidia \{' "$qml"/*.qml || true)"

t "o título da janela é MkFiles, nasce com 1000x640 (ou o que couber) e o mínimo é 520x420"
has "$casca" 'title: "MkFiles"'
has "$casca" "FloatingWindow {"
has "$casca" "//@ pragma UseQApplication"
has "$casca" "minimumSize: Qt.size(520, 420)"
like "$casca" 'implicitWidth: Math\.min\(1000, .*screen'
like "$casca" 'implicitHeight: Math\.min\(640, .*screen'
has "$contrato" '**`MkFiles`**'
eq "titulo=MkFiles" "$(grep -E '^titulo=' "$app/bin/mkfiles")"

t "a janela só manda ao motor pedidos que estão no contrato"
mapfile -t pedidos < <(grep -ohE '\bop: "[a-z-]+"' "$qml"/*.qml "$qml"/*.js | grep -oE '"[a-z-]+"' | tr -d '"' | sort -u)
ok test "${#pedidos[@]}" -ge 45
for op in "${pedidos[@]}"; do
  [[ $contrato1 == *"\`$op\`"* || $contrato1 == *"\"op\": \"$op\""* ]] || _fail "pedido fora do contrato: $op"
done
# Toda chamada diz o pedido na hora (um objeto com op:), ou vem do Arquivos.js.
eq "" "$(grep -nE 'Motor\.pedir\(' "$qml"/*.qml | grep -vE 'Motor\.pedir\((\{ op: "[a-z-]+"|\{$|pedido,|Arquivos\.pedidoDeCopia\()' || true)"
# Os pedidos do coração do app estão todos lá.
for op in sobre prefs prefs-gravar estado-gravar lugares discos vigiar-discos listar item vigiar desvigiar cancelar espaco apps definir-padrao abrir terminal \
  modelos copiar mover ligar criar-pasta criar-arquivo renomear lixeira apagar restaurar lixeira-apagar lixeira-esvaziar desfazer refazer responder \
  tarefa-pausar tarefa-cancelar lixeira-listar lixeira-estado buscar miniatura miniatura-cancelar favorito-por favorito-tirar favorito-mover \
  favorito-renomear montar desmontar ejetar propriedades permissoes area-por area-ler area-gravar \
  renomear-lote previa formatos compactado-info compactado-listar compactado-abrir extrair compactar; do
  [[ " ${pedidos[*]} " == *" $op "* ]] || _fail "a janela não usa: $op"
done

t "a janela fala com o motor por um processo só, de pé, com fluxo e eventos"
has "$ponte" 'command: [root.bin, "servir"]'
has "$ponte" "stdinEnabled: true"
has "$ponte" 'Quickshell.env("MKFILES_MOTOR") || Quickshell.shellPath("../bin/mkfiles-motor")'
has "$ponte" 'Quickshell.env("MKFILES_EXEMPLO")'
has "$ponte" 'Quickshell.env("MKFILES_ABRIR")'
has "$ponte" "JSON.stringify(p) + \"\\n\""
has "$ponte" "signal evento(var e)"
# Num fluxo, quem pediu continua esperando até a linha de fim.
has "$ponte" "if (r.fim !== false)"
has "$ponte" "root.desistir()"
has "$janela" "function onEvento(e)"
for evento in pasta miniatura lixeira tarefa conflito historico discos; do
  has "$janela" "case \"$evento\":"
  has "$contrato1" "\"evento\": \"$evento\""
done

t "a janela não roda nada além do motor (e do jq, para os exemplos)"
eq "" "$(grep -n 'command:' "$qml"/*.qml | grep -v 'Motor.qml' || true)"
eq 2 "$(grep -c 'command:' "$qml/Motor.qml")"
eq "" "$(grep -nE '"(sh|bash|sudo|pkexec|rm|mv|cp|ln|mkdir|chmod|wl-copy|wl-paste|xdg-open|gio|udisksctl|python3?)"' "$qml"/*.qml "$qml"/*.js || true)"

t "executar: só o argv que o motor devolveu, ou o próprio lançador para uma janela nova"
eq 1 "$(grep -c 'execDetached' "$qml"/*.qml | awk -F: '{s+=$2} END {print s}')"
has "$janela" "Quickshell.execDetached(argv);"
out=$(grep -ohE '\bexecutar\([^;]*' "$qml"/*.qml | grep -v '^executar(argv) {' | sort -u)
eq 'executar([Quickshell.shellPath("../bin/mkfiles"), "--nova-janela", Arquivos.pastaReal(caminho || painel.caminho)])
executar(l.argv)
executar(r.argv)' "$out"
# No modo de exemplo nada é executado: a janela diz o que abriria.
like "$(tr -s ' \n' ' ' <<<"$janela")" 'if \(Motor\.exemplo !== ""\) \{ avisar\("exemplo: abriria " \+ argv\.join\(" "\)\); return; \}'

t "a janela não mexe em arquivo: copiar, mover e apagar são pedidos ao motor"
eq "" "$(grep -nE 'XMLHttpRequest|FileView|Qt\.openUrlExternally|writeFile|setText\(' "$qml"/*.qml "$qml"/*.js || true)"
has "$janela" 'tarefa({ op: "copiar", origens: caminhos, destino: destino })'
has "$janela" 'tarefa({ op: "lixeira", origens: c })'
# Apagar de vez e esvaziar passam sempre pela pergunta.
like "$(sed -n '/function apagarDeVez/,/^    }/p' "$qml/App.qml" | tr -s ' \n' ' ')" 'perguntar\(\{.*op: "apagar"'
like "$(sed -n '/function esvaziar/,/^    }/p' "$qml/App.qml" | tr -s ' \n' ' ')" 'perguntar\(\{.*op: "lixeira-esvaziar"'
eq 2 "$(grep -c 'op: "apagar"' "$qml/App.qml")"
eq 1 "$(grep -c 'op: "lixeira-esvaziar"' "$qml/App.qml")"

t "as visões são virtualizadas: GridView e ListView com delegates reciclados, sem Repeater sobre os itens"
eq 2 "$(grep -c '^        GridView {' "$qml/Painel.qml")"
eq 1 "$(grep -c '^        ListView {' "$qml/Painel.qml")"
eq 3 "$(grep -c 'reuseItems: true' "$qml/Painel.qml")"
eq 3 "$(grep -c 'model: visible ? painel.total : 0' "$qml/Painel.qml")"
has "$painel" "delegate: Cartao {"
has "$painel" "delegate: Linha {"
# Os itens moram num array; cada delegate lê o seu pelo índice.
has "$(<"$qml/Cartao.qml")" "dono.itens[index]"
has "$(<"$qml/Linha.qml")" "dono.itens[index]"
eq "" "$(grep -nE 'model: *(painel\.|dono\.)?(itens|todos)\b' "$qml"/*.qml || true)"
# Os delegates são leves: sem área de mouse própria (o painel cuida do mouse inteiro).
eq "" "$(grep -n 'MouseArea' "$qml/Cartao.qml" "$qml/Linha.qml" | grep -v '//' || true)"
has "$painel" "Arquivos.visiveis(todos"

t "as três visões, as colunas da lista e o tamanho dos ícones"
for v in icones lista compacta; do
  has "$painel" "\"$v\""
  has "$janela" "app.escolherVisao(\"$v\")"
done
for coluna in tamanho mtime tipo permissoes; do
  has "$painel" "id: \"$coluna\""
done
has "$painel" "painel.janela.ordenarPor(coluna.campo)"
has "$painel" "painel.janela.larguraDaColuna("
has "$painel" "painel.janela.zoom(wheel.angleDelta.y > 0 ? 1 : -1)"

t "seleção: clique, Ctrl, Shift, retângulo, tudo, inverter e por padrão, tudo pelo Selecao.js"
for f in "Selecao.clicar(" "Selecao.so(" "Selecao.noRetangulo(" "Selecao.comIndices(" "Selecao.todos(" "Selecao.inverter(" "Selecao.porPadrao(" \
  "Selecao.mover(" "Selecao.alternar(" "Selecao.reancorar("; do
  has "$painel" "$f"
done

t "todo atalho da tarefa existe na janela e na folha de atalhos"
atalhos=$(sed -n '/^var ATALHOS/,/^\];/p' "$qml/Menu.js")
for s in "Alt+Left" "Alt+Right" "Alt+Up" "Ctrl+L" "Ctrl+F" "Ctrl+T" "Ctrl+W" "Ctrl+Tab" "Ctrl+H" "Ctrl+A" "Ctrl+S" "Ctrl+C" "Ctrl+X" \
  "Ctrl+V" "Shift+Delete" "Ctrl+Z" "Ctrl+Shift+Z" "Ctrl+Shift+N" "Alt+Return" "F1" "Shift+F10" "Escape" "Ctrl++" "Ctrl+-"; do
  grep -qF "sequences: [\"$s\"" "$qml/App.qml" || _fail "falta o atalho $s"
done
# As teclas de navegação e o digitar-para-pular são do painel (não são Shortcut:
# senão roubariam as letras dos campos de texto).
for k in Key_Up Key_Down Key_Left Key_Right Key_Home Key_End Key_PageUp Key_PageDown Key_Return Key_Backspace Key_Tab Key_Menu Key_Space; do
  has "$painel" "Qt.$k"
done
has "$painel" "pular(event.text)"
has "$painel" "janela.alternarPainel()"
eq "" "$(grep -nE 'Shortcut \{ sequences: \["[A-Za-z]"\]' "$qml"/*.qml | grep -v 'app.folha === "conflito"' || true)"
for tecla in "Enter" "Backspace" "Alt+←" "Ctrl+L" "Ctrl+F" "Ctrl+T" "Ctrl+Tab" "F3" "Tab" "F5 · F6" "Ctrl+H" "Ctrl+A" "Ctrl+S" "Ctrl+C" "F2" "Delete" \
  "Shift+Delete" "Ctrl+Z" "Ctrl+Shift+N" "Alt+Enter" "F4" "Shift+F10" "F1"; do
  has "$atalhos" "$tecla"
done
has "$(<"$qml/Atalhos.qml")" "model: Menu.ATALHOS"

t "teclado: com a janela livre, toda tecla sem modificador é do painel, num lugar só"
# Já aconteceu: a primeira seta sumia e Down não andava. As teclas sem
# modificador não podem ser Shortcut enquanto a janela está livre (disputariam
# com o Keys do painel), e o foco tem de voltar ao painel sozinho.
has "$painel" "Keys.onPressed: event => painel.tecla(event)"
eq 1 "$(grep -c 'Keys.onPressed' "$qml/Painel.qml")"
for k in Key_Delete Key_Escape Key_F1 Key_F12; do
  has "$painel" "Qt.$k"
done
for k in Key_F1 Key_F2 Key_F3 Key_F4 Key_F5 Key_F6; do
  has "$janela" "case Qt.$k:"
done
has "$painel" "janela.teclaDeFuncao(k)"
has "$painel" "janela.paraALixeira()"
# Os Shortcut, um por linha: sequências | enabled.
atalhosDaJanela=$(tr '\n' ' ' <"$qml/App.qml" | grep -oE 'Shortcut \{[^}]*sequences: \[[^]]*\][^}]*' |
  sed -E 's/.*sequences: \[([^]]*)\]/\1|/; s/\|.*enabled: ([^;]*)(;.*)?$/|\1/; s/[[:space:]]+/ /g; s/ +$//')
ok test "$(wc -l <<<"$atalhosDaJanela")" -ge 60
eq 0 "$(grep -c 'Shortcut {' "$qml"/[!A]*.qml "$qml"/A[!p]*.qml | awk -F: '{s += $2} END {print s}')"
while IFS='|' read -r seqs quando; do
  semMod=0
  for s in $(tr -d '", ' <<<"${seqs//, /$'\n'}"); do
    [[ $s == *+?* ]] || semMod=1
  done
  ((semMod)) || continue
  # (a) nenhuma sem modificador vale sempre; com a janela livre só as do
  # painel, que têm a condição exata "app.livre" (e passam por porAtalho).
  [[ $quando != "$seqs" && -n $quando ]] || _fail "Shortcut sem modificador e sem enabled: $seqs"
  [[ $quando == "app.livre" || $quando == *"!app.livre"* || $quando == *"menu.aberto"* || $quando == *"app.folha ==="* || $quando == *"app.naLateral"* ]] ||
    _fail "Shortcut sem modificador que pode valer com a janela livre: $seqs ($quando)"
done <<<"$atalhosDaJanela"
# Medido na tela: no FloatingWindow do Quickshell a primeira tecla sem
# modificador não chega ao Keys do painel, mas chega a um Shortcut. Então, com
# a janela livre, as teclas de navegação e ação são Shortcut, uma vez cada, e
# todas entregam ao mesmo tratador do painel (app.porAtalho). O Keys do painel
# fica para o digitar-para-pular.
has "$janela" 'function porAtalho(tecla, modificadores)'
for par in 'Down|Qt.Key_Down' 'Up|Qt.Key_Up' 'Left|Qt.Key_Left' 'Right|Qt.Key_Right' 'Home|Qt.Key_Home' 'End|Qt.Key_End' \
  'PgUp|Qt.Key_PageUp' 'PgDown|Qt.Key_PageDown' 'Space|Qt.Key_Space' 'Delete|Qt.Key_Delete' 'Backspace|Qt.Key_Backspace' \
  'Tab|Qt.Key_Tab' 'Menu|Qt.Key_Menu' 'F2|Qt.Key_F2' 'F3|Qt.Key_F3' 'F4|Qt.Key_F4' 'F5|Qt.Key_F5' 'F6|Qt.Key_F6'; do
  has "$janela" "Shortcut { sequences: [\"${par%%|*}\"]; enabled: app.livre; onActivated: app.porAtalho(${par##*|}, 0) }" "falta o atalho do painel: ${par%%|*}"
done
has "$janela" 'Shortcut { sequences: ["Return", "Enter"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Return, 0) }'
for par in 'Down|Qt.Key_Down' 'Up|Qt.Key_Up' 'Left|Qt.Key_Left' 'Right|Qt.Key_Right' 'Home|Qt.Key_Home' 'End|Qt.Key_End'; do
  has "$janela" "Shortcut { sequences: [\"Shift+${par%%|*}\"]; enabled: app.livre; onActivated: app.porAtalho(${par##*|}, Qt.ShiftModifier) }"
  has "$janela" "Shortcut { sequences: [\"Ctrl+${par%%|*}\"]; enabled: app.livre; onActivated: app.porAtalho(${par##*|}, Qt.ControlModifier) }"
done
has "$janela" 'Shortcut { sequences: ["Shift+Delete"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Delete, Qt.ShiftModifier) }'
has "$janela" 'Shortcut { sequences: ["Ctrl+Space"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Space, Qt.ControlModifier) }'
# (b) a mesma tecla nunca em dois Shortcut com a mesma condição.
eq "" "$(while IFS='|' read -r seqs quando; do for s in $(tr -d '", ' <<<"${seqs//, /$'\n'}"); do echo "$s|$quando"; done; done <<<"$atalhosDaJanela" | sort | uniq -d)"
# `livre` é falso com folha, menu ou barra lateral: é o que separa os dois mundos.
like "$janela" 'readonly property bool livre: folha === "" && !menu\.aberto && .*!naLateral'

t "teclado: o cursor mora num lugar só, e o foco volta ao painel sozinho"
# Nenhuma visão é dona do cursor.
eq "" "$(grep -n 'currentIndex\|keyNavigationEnabled: true\|highlightFollowsCurrentItem' "$qml/Painel.qml" || true)"
has "$painel" "readonly property int cursor: selecao.cursor"
has "$painel" "Selecao.reancorar(selecao, novo)"
# O painel é um FocusScope com um dono do teclado que não é trocado com a visão.
like "$painel" 'FocusScope \{[[:space:]]+id: painel'
has "$painel" "function focar() { teclado.forceActiveFocus(); }"
has "$painel" "readonly property bool temTeclado: teclado.activeFocus"
# O foco volta: ao fim da listagem, ao trocar a visão, ao virar o painel ativo,
# quando a janela fica livre e quando ela ganha o foco.
like "$(tr -s ' \n' ' ' <<<"$painel")" 'refazer\(\); // A listagem chegou: o teclado é do painel \(se nada mais o usa\)\. refocar\(\);'
like "$(tr -s ' \n' ' ' <<<"$painel")" 'onVisaoChanged: \{ refocar\(\);'
has "$painel" "onAtivoChanged: refocar()"
has "$painel" "Component.onCompleted: Qt.callLater(refocar)"
like "$(tr -s ' \n' ' ' <<<"$janela")" 'onLivreChanged: \{ if \(livre\) painel\.focar\(\); \}'
has "$janela" "readonly property bool janelaAtiva: Window.active"
like "$(tr -s ' \n' ' ' <<<"$janela")" 'onJanelaAtivaChanged: \{ if \(janelaAtiva && livre\) painel\.focar\(\); \}'
# E a rede de segurança: tecla que chega à janela com o foco fora do painel não se perde.
like "$(tr -s ' \n' ' ' <<<"$janela")" 'if \(app\.livre && !app\.painel\.temTeclado\) \{ app\.painel\.focar\(\); app\.painel\.tecla\(event\); \}'
like "$janela" 'FocusScope \{[[:space:]]+id: app'
# F2 seleciona só o nome, sem a extensão.
has "$painel" "campoDoNome.select(0, Arquivos.semExtensao(it.nome, Arquivos.ePasta(it)))"
# Os atalhos da janela se calam com uma folha, um menu ou um campo de texto em uso.
has "$janela" 'readonly property bool livre: folha === "" && !menu.aberto && !barra.comFoco && !painelA.renomeando && !painelB.renomeando && !naLateral'

t "abas, painel duplo e janela estreita"
for f in "Abas.abrir(" "Abas.fechar(" "Abas.duplicar(" "Abas.trocar(" "Abas.mover(" "Abas.escolher(" "Abas.comAba("; do
  has "$painel" "$f"
done
has "$janela" "readonly property bool duploVisivel: duplo && width >= 900"
has "$janela" "readonly property bool estreita: width < 720"
has "$janela" "Layout.preferredWidth: app.estreita ? 44 : app.lateralPx"
has "$janela" "pronto: app.pronto && app.duploVisivel"
has "$janela" "function paraOOutro(mover)"
has "$janela" 'op: "estado-gravar", estado: Abas.paraEstado([painelA.abas, painelB.abas], app.ativo, app.duplo, app.lateralPx)'

t "área de transferência pelos pedidos area-*: copiar e recortar põem sempre o gnome-copied-files; colar lê esse, depois uri-list, depois texto com caminhos"
copia=$(sed -n '/^function pedidoDeCopia/,/^}/p' "$qml/Arquivos.js")
has "$copia" 'tipo: "x-special/gnome-copied-files", texto: copiados(caminhos, recortar === true)'
hasnt "$copia" "text/uri-list"
has "$(sed -n '/^function lerArea/,/^}/p' "$qml/Arquivos.js")" "caminhosDeTexto(r.texto)"
has "$janela" "Motor.pedir(Arquivos.pedidoDeCopia(c, recortar)"
has "$janela" 'Motor.pedir({ op: "area-ler" }'
has "$janela" "Arquivos.lerArea(r)"
has "$janela" 'op: "area-gravar"'
has "$(<"$qml/Arquivos.js")" '"x-special/gnome-copied-files"'
has "$(<"$qml/Arquivos.js")" '"text/uri-list"'
eq "" "$(grep -nE '\bTextEdit \{|\.copy\(\)|\.paste\(\)' "$qml"/*.qml || true)"

t "arrastar e soltar: dentro da janela e com outros apps, por text/uri-list"
has "$painel" "Drag.dragType: Drag.Automatic"
has "$painel" 'Drag.mimeData: ({ "text/uri-list": Arquivos.listaDeUris(arrasto.caminhos) })'
has "$painel" "Drag.supportedActions: Qt.CopyAction | Qt.MoveAction | Qt.LinkAction"
ok test "$(grep -c 'DropArea {' "$qml/Painel.qml")" -ge 2
has "$(<"$qml/Lateral.qml")" "DropArea {"
for acao in CopyAction LinkAction; do
  has "$(sed -n '/function soltar(/,/^    }/p' "$qml/App.qml")" "Qt.$acao"
done

t "menus de contexto: do item, de vários, do fundo e de um lugar, com os pontos de extensão"
has "$janela" 'Menu.menu(c.n === 0 ? "fundo" : c.n === 1 ? "item" : "varios", c)'
has "$janela" 'Menu.menu("lugar", { lugar: linha })'
has "$janela" "extensoes: extensoes"
# Toda entrada que um menu pode ter é tratada no App.
while read -r id; do
  has "$janela" "case \"$id\":" "entrada de menu sem ação: $id"
done < <(grep -ohE '\be\("[a-z-]+"' "$qml/Menu.js" | grep -oE '"[a-z-]+"' | tr -d '"' | sort -u)
for id in compactar extrair extrair-pasta extrair-para extrair-tudo extrair-selecao renomear-lote previa; do
  has "$janela" "case \"$id\":"
done

t "folhas: conflito, propriedades, abrir com, perguntas, nome e atalhos"
conflito=$(<"$qml/Conflito.qml")
for r in substituir pular manter renomear cancelar; do
  has "$conflito" "responder(\"$r\")"
  has "$contrato" "| \`$r\` |"
done
has "$conflito" "conflito.todos = !conflito.todos"
has "$janela" 'Motor.pedir({ op: "responder", tarefa: c.tarefa, conflito: c.conflito, resposta: resposta, nome: nome === "" ? null : nome, todos: todos }'
props=$(<"$qml/Propriedades.qml")
has "$props" '{ v: "geral", texto: "geral" }, { v: "permissoes", texto: "permissões" }'
has "$props" "Arquivos.trocarBit(props.modo"
has "$janela" 'op: "permissoes", caminhos: [item.caminho], modo: mudanca.modo, executavel: null, grupo: mudanca.grupo, recursivo: mudanca.recursivo'
has "$(<"$qml/AbrirCom.qml")" "outro app…"
has "$(<"$qml/AbrirCom.qml")" "definir como padrão"
has "$janela" 'op: "definir-padrao", mime: alvo.mime, app: escolha.id'

t "pastas que não abrem: a janela diz o que houve e oferece subir; nunca fica em branco"
has "$painel" "Arquivos.erroDePasta(painel.erro.motivo, painel.erro.frase).titulo"
has "$painel" 'texto: "subir um nível"'
has "$painel" 'erro = { motivo: "sumiu", frase: "" }'
for texto in "Pasta vazia" "A lixeira está vazia" "Nada encontrado" "Lendo a pasta…"; do
  has "$painel" "$texto"
done

t "pasta vigiada, miniaturas só do que aparece, e a lixeira como lugar"
has "$painel" 'Motor.pedir({ op: "vigiar", caminho: onde }, null)'
has "$painel" 'Motor.pedir({ op: "desvigiar", caminho: vigiado }, null)'
has "$painel" "Arquivos.aplicarEvento(todos, e)"
has "$(<"$qml/Cartao.qml")" "dono.querMiniatura(it)"
has "$painel" 'Motor.pedir({ op: "miniatura-cancelar", grupo: grupoDeMinis }, null)'
has "$painel" 'pedido = Motor.pedir({ op: "lixeira-listar" }, chegou)'
has "$painel" 'painel.janela.acao("restaurar")'
has "$painel" 'painel.janela.acao("esvaziar")'

t "os dados de exemplo são JSON, uma lista de linhas, e existem para todo pedido que a janela faz"
for op in "${pedidos[@]}"; do
  [[ -f $exemplo/$op.json ]] || { _fail "falta o exemplo de: $op"; continue; }
  ok jq -e 'type == "array" and length >= 1 and ([.[] | type == "object" and (has("id") | not)] | all)' "$exemplo/$op.json" >/dev/null
  # A primeira linha é a resposta; os eventos vêm depois.
  ok jq -e '.[0] | has("evento") | not' "$exemplo/$op.json" >/dev/null
done
for f in "$exemplo"/*.json; do
  op=$(basename "$f" .json)
  [[ " ${pedidos[*]} " == *" $op "* ]] || _fail "exemplo que ninguém pede: $op.json"
done

t "os exemplos têm a forma do contrato: o item inteiro, a pasta de mentira, a lixeira, os discos e o conflito"
campos='["nome","caminho","tipo","tamanho","mtime","modo","permissoes","dono","grupo","oculto","mime","icone","icone_reserva","alvo","quebrado","alvo_tipo"]'
item="(. as \$i | $campos | all(. as \$c | \$i | has(\$c)))"
ok jq -e "[.[].itens[] | $item] | all" "$exemplo/listar.json" >/dev/null
ok jq -e '[.[].itens[]] | length >= 38 and length <= 45' "$exemplo/listar.json" >/dev/null
ok jq -e '(.[:-1] | all(.fim == false)) and (.[-1] | .fim == true and has("total") and has("gravavel"))' "$exemplo/listar.json" >/dev/null
ok jq -e '[.[].itens[]] | (map(select(.tipo == "pasta")) | length >= 5) and (map(select(.tipo == "link")) | length >= 2)
  and any(.quebrado) and any(.oculto) and any(.mime | startswith("image/")) and any(.nome | test("[áéíóúãõç]")) and any(.nome | test("[0-9]"))' "$exemplo/listar.json" >/dev/null
ok jq -e "[.[].itens[] | $item and has(\"nome_original\") and has(\"origem\") and has(\"apagado_s\") and has(\"lixeira\")] | length == 3 and all" "$exemplo/lixeira-listar.json" >/dev/null
ok jq -e '.[0].discos | length == 2 and (map(select(.montado != null)) | length == 1)
  and all(has("dispositivo") and has("nome") and has("rotulo") and has("sistema") and has("tamanho") and has("livre") and has("total") and has("removivel") and has("uuid"))' "$exemplo/discos.json" >/dev/null
ok jq -e '.[0] | (.lugares | map(.id) == ["pessoal","documentos","downloads","imagens","musicas","videos","lixeira","raiz"])
  and (.favoritos | length >= 3 and all(has("caminho") and has("uri") and has("nome") and has("existe") and has("remoto")))' "$exemplo/lugares.json" >/dev/null
ok jq -e "(.[0] | has(\"tarefa\")) and ([.[] | select(.evento == \"conflito\")] | length == 1 and (.[0] | has(\"tarefa\") and has(\"conflito\") and has(\"sugestao\") and has(\"restam\")
  and (.origem | $item) and (.destino | $item)))" "$exemplo/mover.json" >/dev/null
ok jq -e '[.[] | select(.evento == "tarefa")] | length >= 1 and all(has("tarefa") and has("acao") and has("estado") and has("titulo") and has("itens") and has("itens_total")
  and has("bytes") and has("bytes_total") and has("atual") and has("velocidade") and has("destino") and has("criados") and has("erros") and has("sem_lixeira") and has("desfazivel"))' \
  "$exemplo"/{copiar,mover,responder,apagar,lixeira}.json >/dev/null
ok jq -e '.[0] | (.prefs | has("visao") and has("visao_por_pasta") and has("ordem") and has("ocultos") and has("icone_px") and has("pastas_antes") and has("confirmar_lixeira") and has("clique") and has("colunas"))
  and (.estado | has("paineis") and has("ativo") and has("duplo") and has("lateral_px"))' "$exemplo/prefs.json" >/dev/null
ok jq -e '.[0] | has("versao") and has("ops") and (.extensoes | has("lote") and has("compactados") and has("previa") and has("remotos"))' "$exemplo/sobre.json" >/dev/null
ok jq -e '.[0] | has("padrao") and (.apps | length >= 2 and all(has("id") and has("nome") and has("icone") and has("terminal") and has("padrao")))' "$exemplo/apps.json" >/dev/null
ok jq -e '.[0] | (.itens | length == 1) and has("onde") and has("sou_dono") and has("grupos") and (.contagem | has("arquivos") and has("pastas") and has("tamanho"))' "$exemplo/propriedades.json" >/dev/null
ok jq -e '.[-1].fim == true' "$exemplo/propriedades.json" >/dev/null

t "os exemplos de listagem são o que o motor de verdade diria de um item (os mesmos campos)"
bin=$app/bin/mkfiles-motor
if [[ -x $bin ]]; then
  mkdir -p "$SBX/pasta"
  echo oi >"$SBX/pasta/a.txt"
  out=$("$bin" listar "$SBX/pasta" 2>/dev/null | tail -1)
  eq "$(jq -c '.[0].itens[0] | keys' "$exemplo/listar.json")" "$(jq -c '.itens[0] | keys' <<<"$out")"
else
  echo "  # sem o motor compilado (apps/mkfiles/pack): pulado" >&2
fi

t "o jq dos exemplos solta as linhas gravadas, uma por linha"
if command -v node >/dev/null; then
  cat >"$SBX/cmd.mjs" <<MJS
import { load } from "$TESTS/js/load.mjs";
const A = load("$qml/Arquivos.js");
for (const p of JSON.parse(process.argv[2]))
    console.log(A.comandoDeExemplo(process.argv[3], p).map((a) => Buffer.from(a).toString("base64")).join(" "));
MJS
  linhas=()
  while read -r linha; do
    cmd=()
    for parte in $linha; do cmd+=("$(base64 -d <<<"$parte")"); done
    eq jq "${cmd[0]}"
    linhas+=("$("${cmd[@]}" | wc -l)")
  done < <(node "$SBX/cmd.mjs" '[{"id":1,"op":"listar","caminho":"/x"},{"id":2,"op":"mover"},{"id":3,"op":"sobre"}]' "$exemplo")
  eq "2 3 1" "${linhas[*]}"
else
  echo "  # node ausente: pulado" >&2
fi

t "nada de dados da máquina de quem escreveu nos exemplos nem na janela"
eq "" "$(grep -lE '/home/[a-z]+/|codexmark' "$exemplo"/*.json | xargs -r grep -LE '/home/ana/' || true)"
eq "" "$(grep -ohE '/home/[a-z]+' "$exemplo"/*.json | sort -u | grep -v '^/home/ana$' || true)"
eq "" "$(grep -nE '/home/|codexmark' "$qml"/*.qml "$qml"/*.js "$app"/share/* "$app/bin/mkfiles" || true)"

t "o atalho de menu: abre o mkfiles com o que for passado, é o gerenciador de arquivos e abre pastas"
d=$(<"$app/share/mkfiles.desktop")
has "$d" "Name=MkFiles"
has "$d" "Exec=mkfiles %U"
has "$d" "Icon=mkfiles"
has "$d" "Terminal=false"
has "$d" "Categories=System;FileManager;Utility;"
has "$d" "MimeType=inode/directory;"
has "$d" "Exec=mkfiles --nova-janela"
exists "$app/share/mkfiles.svg"
has "$(<"$app/share/mkfiles.svg")" "<title>MkFiles</title>"

t "o pack, a assinatura e o lançador são executáveis"
for f in pack assinatura bin/mkfiles; do
  ok test -x "$app/$f"
done

# --- o lançador ---------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml" "$SBX/onde estou"
cp "$app/bin/mkfiles" "$casa/bin/"
printf '#!/bin/sh\n' >"$casa/bin/mkfiles-motor"
chmod +x "$casa/bin/mkfiles-motor"
stub notify-send
stub hyprctl 'exit 0'
stub qs 'if [[ $1 == list ]]; then cat "$SBX/qs-list" 2>/dev/null; else printf "ABRIR=%s\n" "${MKFILES_ABRIR-<sem>}" >>"$SBX/calls"; fi'
abre() { (cd "$SBX/onde estou" && bash "$casa/bin/mkfiles" "$@"); }

t "mkfiles: abre a janela (qs -p …/qml), sem nada para abrir"
calls_reset
# shellcheck disable=SC2119
ok abre
called "qs -p $casa/qml"
called "ABRIR=<sem>"

t "mkfiles: uma janela por chamada, mesmo com outra aberta (não foca a que existe)"
calls_reset
echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
ok abre
called "qs -p $casa/qml"
not_called "hyprctl"
not_called "qs list"
ok abre --nova-janela
eq 2 "$(grep -c "^qs -p $casa/qml" "$SBX/calls")"
called "ABRIR=<sem>"

t "mkfiles pasta arquivo: os caminhos chegam absolutos, um por linha, inteiros"
calls_reset
# shellcheck disable=SC2016 # é o texto, não a expansão, que tem de chegar
ok abre "minha pasta" '/tmp/a $(touch pwn) `z`'"'x'"'"y"' ../fora /etc/../etc/hosts
# shellcheck disable=SC2016
eq "ABRIR=$SBX/onde estou/minha pasta
/tmp/a \$(touch pwn) \`z\`'x'\"y\"
$SBX/fora
/etc/hosts" "$(sed -n '/^ABRIR=/,$p' "$SBX/calls")"
absent "$SBX/onde estou/pwn"
has "$janela" "Abas.pedidosDe(Motor.abrir)"

t "mkfiles file://…: o URI é decodificado sem executar nada; --nova-janela e -- são aceitos"
calls_reset
# shellcheck disable=SC2016
ok abre --nova-janela 'file:///tmp/x%20y%24%28touch%20pwn%29/%C3%A7' -- --com-traco 'file://localhost/tmp/z'
# shellcheck disable=SC2016
eq "ABRIR=/tmp/x y\$(touch pwn)/ç
$SBX/onde estou/--com-traco
/tmp/z" "$(sed -n '/^ABRIR=/,$p' "$SBX/calls")"
absent "$SBX/onde estou/pwn"

t "mkfiles com uma opção que não existe: diz o uso e sai com 2, sem abrir nada"
calls_reset
out=$(abre --apagar-tudo 2>&1)
eq 2 $?
has "$out" "uso: mkfiles"
not_called "qs -p"

t "o lançador não passa nada por shell nem mexe em arquivo"
eq "" "$(grep -nE '\beval\b|sh -c|bash -c|\brm\b|\bmv\b|>\s*"?\$' "$app/bin/mkfiles" | grep -v '^[0-9]*:\s*#' || true)"
has "$(<"$app/bin/mkfiles")" 'exec qs -p "$raiz/qml"'

t "sem o motor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$casa/bin/mkfiles-motor"
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

t "a folha de nome abre com o nome sugerido selecionado (sem a extensão): digitar troca, não emenda"
has "$(<"$qml/Campo.qml")" 'function selecionar(ate)'
like "$(<"$qml/Nome.qml")" 'onTriggered: \{[^}]*campo\.focar\(\);[^}]*campo\.selecionar\(nome\.eNome \? Arquivos\.semExtensao\(campo\.text, false\) : campo\.text\.length\)'

# --- fase 2: acertos, renomear em lote, compactados e prévia rápida -------------

t "barra de rolagem arrastável nas visões e nas folhas; o retângulo de seleção rola a lista na beira"
rolagem=$(<"$qml/Rolagem.qml")
has "$rolagem" "preventStealing: true"
has "$rolagem" "function levar(p)"
has "$rolagem" "alvo.contentY = destino"
has "$rolagem" "alvo.contentX = destino"
ok test "$(grep -c '^        Rolagem {' "$qml/Painel.qml")" -eq 2
for f in AbrirCom Atalhos EmLote Previa; do
  has "$(<"$qml/$f.qml")" "Rolagem {" "sem barra de rolagem: $f"
done
# A barra antiga, que só mostrava, saiu.
hasnt "$painel" "só mostra onde se está"
like "$(tr -s ' \n' ' ' <<<"$painel")" 'Timer \{ interval: 30 repeat: true running: mouse\.retangulo && mouse\.pressed'
has "$painel" "mouse.marcar();"
has "$painel" "function marcar()"

t "busca: os controles de tamanho e de data viram os filtros do pedido, pelo Arquivos.js"
has "$painel" "const f = Arquivos.filtrosDaBusca(filtros, Math.floor(Date.now() / 1000));"
has "$painel" "tamanho_min: f.tamanho_min, tamanho_max: f.tamanho_max, depois_s: f.depois_s, antes_s: f.antes_s,"
hasnt "$painel" "tamanho_min: null"
for c in campoMin campoMax campoDe campoAte; do
  has "$painel" "id: $c"
done
has "$painel" "Arquivos.PERIODOS"
# Quem digita num filtro não dispara os atalhos da janela.
has "$painel" "readonly property bool filtrando: campoMin.comFoco || campoMax.comFoco || campoDe.comFoco || campoAte.comFoco"
has "$janela" "&& !painelA.filtrando && !painelB.filtrando"

t "renomear em lote: F2 com vários abre a folha; a conta é do Lote.js, a prévia é uma lista virtualizada e o motor recebe os pares"
lote=$(<"$qml/EmLote.qml")
has "$janela" 'case Qt.Key_F2: acao(painel.selecao.n > 1 && extensoes.lote === true ? "renomear-lote" : "renomear"); return true;'
has "$janela" 'case "renomear-lote": if (extensoes.lote === true) abrirLote(); break;'
has "$janela" 'Motor.pedir({ op: "renomear-lote", pares: pares }'
has "$janela" "folhaDoLote.erro = Lote.recusa(r);"
has "$janela" "folhaDoLote.existentes = painel.busca ? null : { pasta: painel.caminho, nomes: painel.todos.map(it => it.nome) };"
has "$lote" "Lote.calcular(itens, opcoes, existentes)"
has "$lote" "readonly property bool pode: resultado.mudam > 0 && resultado.conflitos === 0"
has "$lote" "ligado: lote.pode"
has "$lote" "ListView {"
has "$lote" "model: lote.resultado.linhas.length"
has "$lote" "reuseItems: true"
hasnt "$lote" "Repeater"
# A folha não faz a conta dos nomes: nada de replace nem de regex nela.
eq "" "$(grep -nE '\.replace\(|new RegExp|toUpperCase|toLowerCase' "$qml/EmLote.qml" || true)"
for campo in de para curinga caixaExata modelo inicio passo caixa acentos espacos aparar extensao; do
  has "$lote" "$campo:" "opção do lote sem controle: $campo"
  ok grep -q "o\.$campo\b" "$qml/Lote.js"
done

t "compactados: extrair pergunta antes ao motor (ferramenta, senha, bomba); compactar pede nome e formato; tudo é tarefa"
extrai=$(sed -n '/^    function extrair(/,/^    }/p' "$qml/App.qml")
has "$extrai" 'Motor.pedir({ op: "compactado-info", caminho: arquivo, senha: segredo || null }'
has "$extrai" "r.disponivel === false"
has "$extrai" "r.suspeito === true"
has "$extrai" "feito: () => vai(true)"
has "$extrai" "app.pedirSenha("
has "$extrai" 'op: "extrair", caminho: arquivo, destino: destino, modo: modo, internos:'
has "$extrai" "sem_limite: semLimite"
has "$janela" 'Motor.pedir({ op: "formatos" }'
has "$janela" 'app.tarefa({ op: "compactar", origens: d.origens, destino: d.destino, nome: nome, formato: formato, nivel: nivel, senha: segredo });'
compactar=$(<"$qml/Compactar.qml")
has "$compactar" "ligado: modelData.disponivel === true"
has "$compactar" '" precisa do pacote " + f.falta'
has "$compactar" 'readonly property bool comSenha: formato === "zip" || formato === "7z"'

t "abrir um compactado como pasta: lista pelo motor, só leitura, sem vigiar; de dentro, abrir extrai só o arquivo"
has "$painel" "readonly property var compactado: Arquivos.partesDoCompactado(caminho)"
has "$painel" 'Motor.pedir({ op: "compactado-listar", caminho: compactado.arquivo, dentro: compactado.interno, senha: segredoDe(compactado.arquivo) }, chegou)'
has "$painel" "ir(Arquivos.noCompactado(arquivos[0].caminho, \"\"));"
has "$painel" "janela.abrirDeDentro(compactado.arquivo, arquivos, segredoDe(compactado.arquivo));"
has "$janela" 'Motor.pedir({ op: "compactado-abrir", caminho: arquivo, interno: it.interno, senha: segredo || null }'
# O compactado não é vigiado: o pedido dele sai antes do vigiar.
abre=$(sed -n '/^    function abrirPasta()/,/^    }/p' "$qml/Painel.qml")
like "$(tr -s ' \n' ' ' <<<"$abre")" 'if \(compactado !== null\) \{ .*compactado-listar.*chegou\); return; \} const onde = caminho;'
# Só leitura: nada se cria, cola, renomeia nem apaga lá dentro.
has "$janela" 'if (dentro !== null && ["colar", "colar-dentro", "recortar", "renomear", "ligar", "lixeira", "apagar", "nova-pasta", "novo-arquivo", "novo-modelo",'
for f in paraALixeira apagarDeVez novaPasta novoArquivo abrirLote compactar; do
  has "$(sed -n "/^    function $f(/,/^    }/p" "$qml/App.qml")" "avisar(soLeitura);" "sem a trava de só leitura: $f"
done
has "$(sed -n '/^    function colar(/,/^    }/p' "$qml/App.qml")" "Abas.eCompactado(destino)"
has "$(sed -n '/^    function soltar(/,/^    }/p' "$qml/App.qml")" "Abas.eCompactado(destino)"
has "$painel" "if (lixeira || compactado !== null || i < 0"
has "$painel" "enabled: painel.erro === null && !painel.busca && painel.compactado === null"
# Copiar de dentro e colar numa pasta extrai só o que foi copiado.
has "$janela" "Arquivos.marcaDoCompactado(painel.compactado.arquivo, internos)"
has "$janela" 'app.extrair(marca.arquivo, destino, "aqui", marca.internos, null);'

t "a senha de um compactado: campo oculto, só na memória; nunca vai para as preferências, o estado nem o registro"
senha=$(<"$qml/Senha.qml")
has "$senha" "echoMode: TextInput.Password"
has "$(<"$qml/PedeSenha.qml")" "Senha {"
has "$compactar" "Senha {"
# Quem pediu a senha a esquece ao fechar.
has "$(<"$qml/PedeSenha.qml")" "campo.limpar();"
has "$compactar" "segredo.limpar();"
eq "" "$(grep -nE 'console\.(log|warn|error|info|debug)' "$qml"/*.qml "$qml"/*.js || true)"
eq "" "$(grep -nE '(prefs-gravar|estado-gravar|preferir\(|avisar\()[^\n]*(segredo|senha:)' "$qml"/*.qml | grep -v 'essa senha\|pede a senha' || true)"
eq "" "$(grep -nE 'segredo|senha' "$qml/Abas.js" || true)"
# A senha só sai da janela dentro de um pedido de compactado.
eq "" "$(grep -ohE 'op: "[a-z-]+"[^}]*senha:' "$qml"/*.qml | grep -oE 'op: "[a-z-]+"' | sort -u | grep -vE '"(compactado-info|compactado-listar|compactado-abrir|extrair|compactar)"' || true)"

t "prévia rápida: Espaço abre, Espaço ou Esc fecham, as setas passam para o vizinho e Enter abre"
previa=$(<"$qml/Previa.qml")
has "$painel" 'janela.acao("previa");'
has "$painel" "else if (!event.isAutoRepeat)"
has "$janela" 'case "previa": abrirPrevia(); break;'
has "$(sed -n '/^    function abrirPrevia(/,/^    }/p' "$qml/App.qml")" "if (extensoes.previa !== true)"
has "$janela" 'Shortcut { sequences: ["Space"]; autoRepeat: false; enabled: app.folha === "previa" && !menu.aberto; onActivated: app.fecharFolha() }'
has "$janela" 'Shortcut { sequences: ["Left", "Up"]; enabled: app.folha === "previa" && !menu.aberto; onActivated: app.vizinhoDaPrevia(-1) }'
has "$janela" 'Shortcut { sequences: ["Right", "Down"]; enabled: app.folha === "previa" && !menu.aberto; onActivated: app.vizinhoDaPrevia(1) }'
has "$janela" 'Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "previa" && !menu.aberto; onActivated: folhaDaPrevia.abrirPedido() }'
has "$janela" 'Motor.pedir({ op: "previa", caminho: it.caminho }'
# Cada classe que o motor diz tem o seu pedaço na folha.
for classe in imagem texto audio video pdf pasta compactado outro; do
  has "$previa" "\"$classe\"" "classe sem prévia: $classe"
done
has "$previa" "AnimatedImage {"
has "$previa" "WheelHandler {"
has "$previa" 'source: "PreviaMidia.qml"'
has "$previa" "midia.status === Loader.Error"
has "$previa" "textFormat: Text.PlainText"
# O PDF é a miniatura grande do sistema; a pasta, a contagem das propriedades; o compactado, a listagem.
has "$janela" 'Motor.pedir({ op: "miniatura", caminhos: [it.caminho], tamanho: "x-large", grupo: "previa", prioridade: 10 }, null);'
has "$janela" 'app.pedidoDaContagem = Motor.pedir({ op: "propriedades", caminhos: [it.caminho] }'
has "$janela" 'app.pedidoDaContagem = Motor.pedir({ op: "compactado-listar", caminho: it.caminho, dentro: "", senha: null }'
has "$previa" "É só uma prévia da primeira página."
# O que há dentro (pasta, compactado) é uma lista virtualizada.
has "$previa" "ListView {"
has "$previa" "model: parent.visible ? previa.lista.length : 0"
hasnt "$previa" "Repeater"
midia=$(<"$qml/PreviaMidia.qml")
for peca in "MediaPlayer {" "VideoOutput {" "AudioOutput {" "tocador.play()" "tocador.pause()" "som.volume = v"; do
  has "$midia" "$peca"
done

t "as extensões novas estão nos atalhos (F1) e nos exemplos"
has "$atalhos" '["Espaço", "prévia rápida'
has "$atalhos" "em lote"
ok jq -e '.[0].extensoes | .lote == true and .compactados == true and .previa == true and .remotos == true' "$exemplo/sobre.json" >/dev/null
ok jq -e '.[0] | has("item") and has("classe") and has("texto") and has("cortado") and has("binario") and has("itens") and has("total")' "$exemplo/previa.json" >/dev/null
ok jq -e '.[0] | (.extrair | length >= 10) and (.compactar | map(.formato) == ["zip","tar.gz","tar.zst","tar.xz","7z"])
  and ([.extrair[], .compactar[]] | all(has("formato") and has("disponivel") and has("falta"))) and ([.extrair[] | select(.disponivel | not)] | length >= 1)' "$exemplo/formatos.json" >/dev/null
ok jq -e '.[0] | has("formato") and has("disponivel") and has("falta") and has("cifrado") and has("itens") and has("tamanho") and has("taxa") and has("suspeito")' "$exemplo/compactado-info.json" >/dev/null
ok jq -e "([.[].itens[] | $item and has(\"interno\")] | length >= 5 and all) and (.[-1].fim == true)" "$exemplo/compactado-listar.json" >/dev/null
ok jq -e '.[0] | has("caminho")' "$exemplo/compactado-abrir.json" >/dev/null
ok jq -e '(.[0] | has("tarefa")) and ([.[] | select(.evento == "tarefa")] | length >= 1)' "$exemplo"/{renomear-lote,extrair,compactar}.json >/dev/null

# --- fase 3: rede ---------------------------------------------------------------
conecta=$(<"$qml/Conectar.qml")
remota=$(<"$qml/PerguntaRemota.qml")
rede=$(<"$qml/Remoto.js")
lateral=$(<"$qml/Lateral.qml")

t "rede: conectar monta pelo motor (Ctrl+K, a barra lateral, um endereço no caminho), e as perguntas do servidor viram folha"
has "$janela" 'Shortcut { sequences: ["Ctrl+K"]; enabled: app.folha === "" && !menu.aberto; onActivated: app.abrirConectar() }'
has "$janela" 'Motor.pedir({ op: "remoto-montar", uri: uri, anonimo: false }'
has "$janela" 'Motor.pedir({ op: "remoto-hosts" }'
has "$janela" 'Motor.pedir({ op: "remotos", rede: comRede }'
has "$janela" 'Motor.pedir({ op: "remoto-desmontar", uri: linha.uri }'
has "$janela" 'case "pergunta-remota":'
has "$janela" 'case "remotos":'
# A resposta volta com o pedido e o número da pergunta; fechar a folha cancela a conexão.
has "$janela" 'Motor.pedir({ op: "remoto-responder", pedido: p.pedido, pergunta: p.pergunta, resposta: resposta }, null);'
has "$janela" 'Motor.pedir({ op: "remoto-responder", pedido: perguntaRemota.pedido, pergunta: perguntaRemota.pergunta, resposta: null }, null);'
has "$janela" 'Motor.pedir({ op: "cancelar", pedido: conexao.pedido }, null);'
# Uma pergunta que ninguém espera não fica pendurada: a montagem é cancelada.
has "$janela" 'Motor.pedir({ op: "remoto-responder", pedido: e.pedido, pergunta: e.pergunta, resposta: null }, null);'
# O que já está montado não é montado de novo.
has "$janela" "const ja = Remoto.montadoEm(uri, remotos);"
# O formulário monta o endereço no .js; os hosts do ssh e a rede local conectam com um clique.
has "$conecta" "Remoto.montarUri({ tipo: tipo, host: host.text, porta: porta.text, usuario: usuario.text,"
has "$conecta" "Remoto.erroDoEndereco(digitado)"
has "$conecta" "Remoto.uriDoHost(h)"
has "$conecta" 'texto: conectar.conectando ? "cancelar a conexão" : "fechar"'
has "$conecta" "text: conectar.saida"
has "$janela" "app.conectar(caminho.trim(), onde => app.painel.ir(onde));"

t "rede: a senha do servidor só existe na folha, em campo oculto; nunca vai para as preferências, o estado, o aviso nem o registro"
# O único campo de senha é o Senha.qml (echoMode Password); a folha a apaga ao responder e ao fechar.
has "$remota" "Senha {"
has "$remota" "visible: remota.forma.oculto"
like "$(tr -s ' \n' ' ' <<<"$remota")" 'const resposta = forma\.oculto \? segredo\.text : claro\.text; segredo\.limpar\(\);'
like "$(tr -s ' \n' ' ' <<<"$remota")" 'onVisibleChanged: \{ if \(!visible\) \{ segredo\.limpar\(\);'
# O padrão que o gio sugere nunca preenche um campo oculto.
has "$remota" "dados && dados.padrao && !forma.oculto"
# A folha de conectar não tem campo de senha, e o formulário não a põe no endereço.
hasnt "$conecta" "Senha {"
hasnt "$conecta" "echoMode"
eq "" "$(grep -nE 'senha|password' <<<"$rede" | grep -vE '^\s*[0-9]+:\s*//|case "senha"|"senha"|A senha não serviu|ou a senha não serviram' || true)"
# A resposta só sai da janela dentro do pedido remoto-responder: nada de log, aviso, prefs ou estado.
eq "" "$(grep -n 'console\.' "$qml"/*.qml "$qml"/*.js || true)"
eq "" "$(grep -nE '(prefs-gravar|estado-gravar|preferir\(|avisar\(|area-por)[^\n]*(resposta|perguntaRemota)' "$qml"/*.qml || true)"
# A resposta aparece em quatro pedidos e só neles: o do conflito de arquivos e os três remoto-responder.
eq 4 "$(grep -c 'resposta: ' "$qml/App.qml")"
eq 3 "$(grep -c 'op: "remoto-responder"' "$qml/App.qml")"
# A conexão em andamento guarda o pedido e o endereço, e mais nada.
has "$janela" "conexao = { pedido: pedido, uri: uri, naFolha: naFolha, cancelada: false };"
hasnt "$(<"$qml/Abas.js")" "resposta"

t "rede: a barra lateral tem a seção Rede (montados com desconectar, e conectar); um favorito de rede monta ao abrir"
has "$lateral" 'l.push({ tipo: "titulo", nome: "rede" });'
has "$lateral" "for (const r of Remoto.linhas(remotos))"
has "$lateral" '(linha.modelData.tipo === "disco" || linha.modelData.tipo === "remoto") && !!linha.modelData.montado'
has "$janela" "remotos: app.remotos"
like "$(tr -s ' \n' ' ' <<<"$janela")" 'if \(linha\.tipo === "remoto"\) app\.desconectar\(linha\); else app\.soltarDisco'
like "$(tr -s ' \n' ' ' <<<"$janela")" 'if \(linha\.tipo === "conectar"\) \{ abrirConectar\(\); return; \}'
like "$(tr -s ' \n' ' ' <<<"$janela")" 'if \(linha\.remoto && linha\.uri\) \{ conectar\(linha\.uri, vai\); return; \}'
hasnt "$janela" "fase seguinte"
# Desconectar tira os painéis de dentro do servidor antes.
like "$(tr -s ' \n' ' ' <<<"$janela")" 'Arquivos\.dentro\(linha\.caminho, p\.caminho\)\) p\.ir\(Motor\.home\); Motor\.pedir\(\{ op: "remoto-desmontar"'

t "rede: numa pasta de servidor as migalhas são «servidor › pasta», Delete apaga de vez (com pergunta) e o endereço pode ser copiado"
has "$(<"$qml/Barra.qml")" "Remoto.migalhas(barra.onde, barra.remotos) || Arquivos.migalhas(barra.onde, barra.home)"
like "$(tr -s ' \n' ' ' <<<"$janela")" 'if \(Remoto\.eRemoto\(painel\.caminho, gvfs\)\) \{ apagarDeVez\(\); return; \}'
has "$janela" "remoto: Remoto.eRemoto(painel.caminho, gvfs),"
has "$janela" 'case "copiar-endereco": copiarEndereco(); break;'
has "$janela" "Remoto.endereco(x, remotos)"
has "$janela" "app.gvfs = r.gvfs || \"\";"
# Subir da raiz do servidor vai para a pasta pessoal, não para a pasta do gvfs.
like "$(tr -s ' \n' ' ' <<<"$painel")" 'if \(remoto && janela\.remotos\.some\(r => r\.raiz === de\)\) \{ ir\(Motor\.home\); return; \}'

t "rede: miniaturas só dos arquivos pequenos e desligáveis; a pasta é relida ao fim de cada tarefa"
has "$painel" "if (Remoto.eRemoto(it.caminho, janela.gvfs) && !Remoto.cabeMiniatura(it, prefs))"
has "$janela" 'case "miniaturas-remotas": preferir({ miniaturas_remotas: prefs.miniaturas_remotas === false }); break;'
has "$janela" 'case "miniatura-remota-mb": preferir({ miniatura_remota_mb: Remoto.proximoLimite(prefs.miniatura_remota_mb || 5) }); break;'
like "$(tr -s ' \n' ' ' <<<"$janela")" 'if \(p\.pronto && p\.remoto && !p\.busca && !p\.carregando\) p\.recarregar\(\);'
# Sair da pasta cancela a listagem que ainda corre (um servidor lento não segura a janela).
like "$(tr -s ' \n' ' ' <<<"$painel")" 'function largar\(\) \{ if \(carregando && pedido > 0\) Motor\.pedir\(\{ op: "cancelar", pedido: pedido \}, null\);'

t "rede: conexão que cai não deixa o painel em branco: avisa e volta para a pasta pessoal"
like "$(tr -s ' \n' ' ' <<<"$painel")" 'Remoto\.caiu\(r\.motivo, caminho, janela\.gvfs\)\) \{ caiuAConexao\(r\.motivo === "conexao" \? r\.erro : ""\); return; \}'
like "$(tr -s ' \n' ' ' <<<"$painel")" 'function caiuAConexao\(frase\) \{.*janela\.avisar\(.*ir\(Motor\.home\); \}'
# Nem entra em laço se a própria pasta pessoal não lista.
has "$painel" "caminho !== Motor.home && Remoto.caiu("
# O servidor saiu da lista dos montados: quem estava nele volta para casa.
like "$(tr -s ' \n' ' ' <<<"$janela")" 'Remoto\.eRemoto\(p\.caminho, gvfs\) && Remoto\.doRemoto\(p\.caminho, lista\) === null\) p\.caiuAConexao\('

t "rede: os exemplos têm a forma do contrato"
ok jq -e '.[0] | (.gvfs | type == "string") and .extensoes.remotos == true' "$exemplo/sobre.json" >/dev/null
ok jq -e '.[0] | (.remotos | length >= 2 and all(has("uri") and has("raiz") and has("nome") and has("protocolo") and has("host"))) and (.rede | all(has("uri") and has("nome")))' "$exemplo/remotos.json" >/dev/null
ok jq -e '.[0].hosts | length >= 2 and all(has("apelido") and has("host") and has("usuario") and has("porta"))' "$exemplo/remoto-hosts.json" >/dev/null
ok jq -e '.[0] | has("uri") and has("caminho") and has("raiz") and has("nome")' "$exemplo/remoto-montar.json" >/dev/null
ok jq -e '.[0].remotos | type == "array"' "$exemplo/remoto-desmontar.json" >/dev/null
ok jq -e '.[0].prefs | .miniaturas_remotas == true and .miniatura_remota_mb == 5' "$exemplo/prefs.json" >/dev/null
# Nenhum exemplo traz senha.
eq "" "$(grep -liE '"(senha|resposta|password)": *"[^"]' "$exemplo"/remot*.json || true)"
