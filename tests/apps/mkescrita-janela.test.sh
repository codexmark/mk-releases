#!/usr/bin/env bash
# apps/mkescrita: a janela (qml/) e o lançador (bin/mkescrita). O desenho se
# vê no bin/dev-shot; aqui: a janela só usa as cores do Mk, só pede ao motor o
# que o contrato tem, as decisões de salvar e fechar vêm do Escrita.js, os
# dados de exemplo seguem o contrato, e o lançador abre a janela certa.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mkescrita
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
exemplo=$TESTS/apps/mkescrita-exemplo
pagina=$(<"$qml/Pagina.qml")
casca=$(<"$qml/shell.qml")
ponte=$(<"$qml/Motor.qml")

t "cores: só os tokens do Mk, e só no Theme.qml"
tokens=" #0b0b0b #141211 #1b1917 #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml | sort -u)
eq "" "$(grep -lE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml | grep -v '/Theme.qml$' || true)"
eq "" "$(grep -nE '#[0-9a-fA-F]{3,8}\b' "$qml/Escrita.js" || true)" "o Escrita.js recebe as cores, não inventa"
out=$(grep -ohE 'Qt\.rgba\([^)]*\)' "$qml"/*.qml | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255|0, 0, 0, 0)' || true)
eq "" "$out"
# Nem cor por nome ("red", "white") nem cor feita na hora.
eq "" "$(grep -nE 'color: *"[a-z]+"|Qt\.(hsla|hsva|color|lighter|darker|tint)\(' "$qml"/*.qml || true)"
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta no ícone: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$app"/share/*.svg | sort -u)

t "o Theme e os componentes pequenos são os do Maré, iguais aos do MkUtil"
for f in Theme Botao Rotulo; do
  eq "$(<"$REPO/apps/mkutil/qml/$f.qml")" "$(<"$qml/$f.qml")" "$f.qml"
done

t "a pasta da janela é autossuficiente: todo tipo próprio usado tem o arquivo dele, e nenhuma fonte vem empacotada"
for tipo in $(grep -ohE '^\s*(Botao|Rotulo|Linha|Pergunta|Rodape|Busca|Sumario|Fontes|Ajuda|Previa|Pagina) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  exists "$qml/$tipo.qml"
done
for f in Motor.qml Escrita.js shell.qml Pagina.qml; do exists "$qml/$f"; done
eq "" "$(find "$app" -iname '*.ttf' -o -iname '*.otf' -o -iname '*.woff*' | head -3)"
eq "" "$(grep -n 'FontLoader' "$qml"/*.qml || true)"

t "a fonte: a monoespaçada do tema por padrão, e as instaladas (Qt.fontFamilies) no seletor"
has "$pagina" 'readonly property string fonteEmUso: fonte !== "" && Theme.families.indexOf(fonte) >= 0 ? fonte : Theme.mono'
has "$pagina" "font.family: pagina.fonteEmUso"
has "$pagina" "familias: Theme.families"
ok grep -q 'readonly property var families: Qt.fontFamilies()' "$qml/Theme.qml"

t "a janela só pede ao motor subcomandos que estão no contrato, e usa todos"
n=0
while read -r sub; do
  n=$((n + 1))
  has "$contrato" "\`mkescrita-motor $sub" "subcomando fora do contrato: $sub"
done < <({
  grep -ohE '(pedir|vigia)\(\["[a-z]+"' "$qml"/*.qml
  grep -ohE 'argsTexto\("[a-z]+"' "$qml"/*.qml
  grep -ohE 'const args = \["[a-z]+"' "$qml/Escrita.js"
} | grep -oE '"[a-z]+"' | tr -d '"' | sort -u)
eq 9 "$n"

t "as opções que a janela manda são as do contrato"
n=0
while read -r opcao; do
  n=$((n + 1))
  has "$contrato" "$opcao" "opção fora do contrato: $opcao"
done < <(grep -ohE '"--[a-z-]+"' "$qml"/*.qml "$qml/Escrita.js" | tr -d '"' | grep -v -- '--nova' | sort -u)
ok test "$n" -ge 8

t "a janela não roda nada além do motor (e do jq, para os exemplos, e do próprio lançador)"
eq "" "$(grep -n 'command:' "$qml"/*.qml | grep -v 'Motor.qml' || true)"
eq "" "$(grep -n 'execDetached\|Process {' "$qml"/*.qml | grep -v 'Motor.qml' || true)"
eq "" "$(grep -nE '"(sh|bash|sudo|pkexec|rm|xdg-open|wl-copy)"' "$qml"/*.qml "$qml/Escrita.js" || true)"
eq 1 "$(grep -c 'execDetached' "$qml/Motor.qml")"
has "$ponte" 'Quickshell.execDetached(caminho ? [lancador, "--nova", "--", caminho] : [lancador, "--nova"])'
has "$ponte" 'readonly property string lancador: Quickshell.shellPath("../bin/mkescrita")'

t "o texto vai ao motor pelo stdin, com o stdin fechado depois, e nunca pela linha de comando"
has "$ponte" "p.write(p.entrada);"
has "$ponte" "p.stdinEnabled = false;"
eq "" "$(grep -n 'edit\.text' "$qml/Pagina.qml" | grep -E 'pedir\(\[' || true)"
# Todo pedido que leva texto diz quantos bytes vão (--bytes), para o motor recusar o que chegar cortado.
for f in argsSalvar argsRascunho argsExportar argsTexto; do
  ok grep -qE "function $f\(" "$qml/Escrita.js"
done
eq 4 "$(grep -c '"--bytes", String(bytesUtf8(texto))' "$qml/Escrita.js")"
eq "" "$(grep -nE 'pedir\(\["(salvar|html|contar|estrutura|exportar)"' "$qml"/*.qml || true)" "pedido de texto montado à mão, sem --bytes"

t "MKESCRITA_EXEMPLO troca o motor por respostas gravadas"
has "$ponte" 'Quickshell.env("MKESCRITA_EXEMPLO")'
has "$ponte" 'exemplo + "/" + Escrita.arquivoDeExemplo(args)'
has "$ponte" 'Quickshell.env("MKESCRITA_MOTOR") || Quickshell.shellPath("../bin/mkescrita-motor")'
has "$ponte" 'Quickshell.env("MKESCRITA_ARQUIVO")'

t "o título da janela nasce MkEscrita (a classe da regra de flutuar) e só depois segue o documento"
has "$casca" 'title: assentada ? pagina.titulo : "MkEscrita"'
has "$casca" "property bool assentada: false"
has "$pagina" "readonly property string titulo: Escrita.tituloDaJanela(doc)"
has "$casca" "//@ pragma UseQApplication"
has "$casca" "FloatingWindow {"
has "$casca" "settings.watchFiles: false"

t "a janela nunca nasce maior que a tela (numa tela de 1366x768 cabe abaixo da ilha)"
like "$casca" 'implicitWidth: Math\.min\(1040, .*screen'
like "$casca" 'implicitHeight: Math\.min\(760, .*screen'

t "fechar a janela não encerra o app por conta própria: quem decide é o Escrita.aoFechar"
has "$casca" "function onLastWindowClosed() { pagina.pedeFechar(true); }"
has "$casca" "onSair: Qt.quit()"
has "$casca" "onReaparecer: window.visible = true"
eq 1 "$(grep -c 'Qt.quit()' "$qml"/*.qml | awk -F: '{s+=$2} END {print s}')"
has "$pagina" "const decisao = Escrita.aoFechar(pagina.doc);"
has "$pagina" "interprete: tecla => Escrita.respostaDaTecla(tecla)"
for resposta in '"salvar", texto: "Salvar", tecla: "S"' '"descartar", texto: "Descartar", tecla: "D"' '"cancelar", texto: "Cancelar", tecla: "C", esc: true'; do
  has "$pagina" "{ valor: $resposta"
done

t "o salvamento automático só grava quando o Escrita.deveSalvarSozinho deixa, e sempre pelo argsSalvar"
has "$pagina" "if (Escrita.deveSalvarSozinho(pagina.doc))"
eq 1 "$(grep -c 'Escrita.argsSalvar(' "$qml/Pagina.qml")"
has "$pagina" "pagina.estado = Escrita.transicao(pagina.estado, \"editou\");"
has "$pagina" "if (r.conflito) {"
has "$pagina" "pagina.entraEmConflito(r.conflito);"
# Em conflito o relógio do salvamento para.
like "$pagina" 'function entraEmConflito\(motivo\) \{[^}]*salvarLogo\.stop\(\);'
for resposta in recarregar manter como; do has "$pagina" "valor === \"$resposta\""; done

t "rascunho: guardado a cada pausa, apagado depois de salvar, e o recuperado nunca é gravado sozinho"
has "$pagina" "onTriggered: pagina.guardaRascunho()"
has "$pagina" "if (!Escrita.deveGuardarRascunho(pagina.doc))"
has "$pagina" "Escrita.rascunhoParaOferecer(r.itens, pagina.caminho)"
has "$pagina" "pagina.recuperado = true;"
like "$pagina" 'if \(pagina\.estado\.fase === "salvo"\) \{[[:space:]]*pagina\.apagaRascunho\(null\);'
# Recuperar não chama o salvar: só põe o texto na tela.
recupera=$(sed -n '/^    function recupera(resumo) {/,/^    }/p' "$qml/Pagina.qml")
hasnt "$recupera" "grava("
hasnt "$recupera" "salva("
ok grep -q 'return doc.fase === "sujo" && !!doc.caminho && !doc.recuperado && !doc.soLeitura;' "$qml/Escrita.js"

t "o vigia: o que eu mesmo gravei é ignorado, e o que chega no meio de um salvamento espera"
has "$pagina" 'Motor.vigia(["vigiar", "--esperado", pagina.conhecida || "ausente", "--", pagina.caminho]'
has "$pagina" "const decisao = Escrita.aoMudarPorFora(pagina.doc, e, pagina.conhecida);"
has "$pagina" "pagina.eventoGuardado = e;"

t "os atalhos de teclado prometidos estão na página"
for tecla in 'Ctrl+S' 'Ctrl+Shift+S' 'Ctrl+O' 'Ctrl+N' 'Ctrl+Y' 'F11' 'Ctrl+F' 'Ctrl+H' 'Ctrl+G' 'Ctrl+B' 'Ctrl+I' 'Ctrl+Shift+X' 'Ctrl+K' \
  'Ctrl+=' 'Ctrl+-' 'Ctrl+0' 'Ctrl+Shift+C' 'Ctrl+Shift+P' 'Ctrl+Shift+O' 'Ctrl+Shift+F' 'F1' 'Ctrl+/' 'Ctrl+Shift+E'; do
  has "$pagina" "\"$tecla\"" "falta o atalho $tecla"
done
# E a folha do F1 não promete atalho que a página não tem.
while read -r tecla; do
  case $tecla in
    Enter | Shift+Enter | Tab | Shift+Tab | Ctrl+Z | Ctrl+Shift+Z) continue ;; # do campo de texto e da barra de busca
  esac
  has "$pagina" "\"$tecla\"" "a folha de atalhos promete $tecla"
done < <(sed -n '/^function atalhos()/,/^}/p' "$qml/Escrita.js" | grep -oE '\["[^"]+", "' | sed 's/^\["//; s/", "$//' | tr '/' '\n' | sed 's/^ *//; s/ *$//' | sed 's/^Ctrl+$/Ctrl+\//' | grep -v '^$' | sort -u)
has "$pagina" 'pagina.marca("**")'
has "$pagina" 'pagina.marca("*")'
has "$pagina" 'pagina.marca("~~")'
has "$pagina" "acceptedModifiers: Qt.ControlModifier"

t "o conforto de Markdown ao digitar passa pelo Escrita.js e entra no desfazer (remove/insert, nunca text =)"
for f in aoEnter recuar envolver inserirLink alternarMarca substituirUma substituirTodas ocorrencias proxima; do
  has "$pagina" "Escrita.$f(" "a página não usa Escrita.$f"
done
has "$pagina" "edit.remove(e.inicio, e.fim);"
has "$pagina" "edit.insert(e.inicio, e.texto);"
# edit.text só é trocado inteiro ao carregar um arquivo ou um rascunho, com `carregando` ligado.
eq 2 "$(grep -c '^ *edit\.text = ' "$qml/Pagina.qml")"
eq 2 "$(grep -B1 '^ *edit\.text = ' "$qml/Pagina.qml" | grep -c 'pagina.carregando = true;')"

t "a pré-visualização mostra o HTML do motor num Text em RichText, e só abre links http(s)"
previa=$(<"$qml/Previa.qml")
has "$previa" "textFormat: Text.RichText"
has "$previa" "if (Escrita.linkAbrivel(link))"
has "$pagina" 'Escrita.argsTexto("html", texto)'
has "$pagina" "proporcao: rolo.proporcao"

t "o modo foco some com o rodapé; o zoom e a fonte são lembrados pelo motor"
has "$pagina" "visible: !pagina.foco"
has "$pagina" 'Motor.pedir(["preferencias", "ler"]'
has "$pagina" 'Motor.pedir(["preferencias", "gravar"], JSON.stringify({ fonte: pagina.fonte, zoom: pagina.zoomPct, previa: pagina.previa }), null);'
has "$pagina" "Escrita.zoom(pagina.zoomPct, passos)"

t "abrir e salvar escolhem o arquivo pelo seletor do sistema, como no MkUtil; nunca um nome inventado"
has "$pagina" "import QtQuick.Dialogs"
eq 3 "$(grep -c '^    FileDialog {' "$qml/Pagina.qml")"
ok grep -q 'FileDialog {' "$REPO/apps/mkutil/qml/Solta.qml"
has "$pagina" "fileMode: FileDialog.SaveFile"
like "$pagina" 'if \(pagina\.caminho === ""\) \{[[:space:]]*pagina\.salvaComo\(\);'
has "$pagina" 'if (Escrita.ondeAbrir(pagina.doc) === "aqui")'

t "os dados de exemplo são JSON e existem para tudo que a janela pede"
for f in abrir salvar rascunho-listar rascunho-guardar rascunho-ler rascunho-apagar preferencias-ler preferencias-gravar html html-pagina \
  contar estrutura exportar vigiar; do
  ok jq -e . "$exemplo/$f.json" >/dev/null
done
eq 14 "$(find "$exemplo" -name '*.json' | wc -l)"

t "os exemplos têm os campos do contrato que a janela lê"
ok jq -e 'has("arquivo") and has("nome") and has("novo") and has("texto") and has("fim_de_linha") and has("quebra_final") and has("so_leitura")
  and (.impressao | has("tamanho_bytes") and has("mtime_ns") and has("sha256"))' "$exemplo/abrir.json" >/dev/null
for f in salvar exportar; do
  ok jq -e 'has("arquivo") and has("nome") and (.impressao | has("sha256"))' "$exemplo/$f.json" >/dev/null
done
ok jq -e '.itens | type == "array"' "$exemplo/rascunho-listar.json" >/dev/null
ok jq -e 'has("id") and has("origem") and has("nome") and has("guardado_em_s") and has("tamanho_bytes") and has("trecho") and has("vivo")' "$exemplo/rascunho-guardar.json" >/dev/null
ok jq -e 'has("id") and has("origem") and has("base_sha256") and has("guardado_em_s") and has("texto")' "$exemplo/rascunho-ler.json" >/dev/null
ok jq -e '.ok == true' "$exemplo/rascunho-apagar.json" >/dev/null
for f in preferencias-ler preferencias-gravar; do
  ok jq -e 'has("fonte") and has("zoom") and has("previa") and .zoom >= 50 and .zoom <= 300' "$exemplo/$f.json" >/dev/null
done
ok jq -e '.html | startswith("<h1>")' "$exemplo/html.json" >/dev/null
ok jq -e '.html | startswith("<!doctype html>")' "$exemplo/html-pagina.json" >/dev/null
ok jq -e 'has("palavras") and has("caracteres") and has("linhas") and has("leitura_s")' "$exemplo/contar.json" >/dev/null
ok jq -e '[.titulos[] | has("nivel") and has("texto") and has("linha")] | all' "$exemplo/estrutura.json" >/dev/null
ok jq -e 'type == "array"' "$exemplo/vigiar.json" >/dev/null

t "os exemplos são o que o motor de verdade responde para o texto de exemplo"
if [[ -x $app/bin/mkescrita-motor ]]; then
  # Para um arquivo: o $(…) comeria a quebra de linha do fim.
  jq -j .texto "$exemplo/abrir.json" >"$SBX/exemplo.md"
  for sub in contar estrutura html; do
    jeq "$(<"$exemplo/$sub.json")" "$("$app/bin/mkescrita-motor" "$sub" <"$SBX/exemplo.md")"
  done
  jeq "$(<"$exemplo/html-pagina.json")" "$("$app/bin/mkescrita-motor" html --pagina <"$SBX/exemplo.md")"
  eq "$(sha256sum <"$SBX/exemplo.md" | cut -d' ' -f1)" "$(jq -r .impressao.sha256 "$exemplo/abrir.json")"
fi

t "nada de dados da máquina de quem escreveu nos exemplos"
eq "" "$(grep -lE '/home/[a-z]+/|/tmp/|codexmark' "$exemplo"/*.json || true)"

t "o atalho de menu: abre text/markdown e text/plain, com os arquivos (%F)"
d=$(<"$app/share/mkescrita.desktop")
has "$d" "Name=MkEscrita"
has "$d" "Exec=mkescrita %F"
has "$d" "Icon=mkescrita"
has "$d" "Terminal=false"
has "$d" "MimeType=text/markdown;text/plain;"
exists "$app/share/mkescrita.svg"
has "$(<"$app/share/mkescrita.svg")" "<title>MkEscrita</title>"

# --- o lançador ---------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml" "$SBX/onde estou"
cp "$app/bin/mkescrita" "$casa/bin/"
printf '#!/bin/sh\n' >"$casa/bin/mkescrita-motor"
chmod +x "$casa/bin/mkescrita-motor"
stub notify-send
stub hyprctl '[[ $1 == clients ]] && cat "$SBX/clientes.json" 2>/dev/null; exit 0'
stub qs 'if [[ $1 == list ]]; then cat "$SBX/qs-list" 2>/dev/null; else echo "ARQUIVO=${MKESCRITA_ARQUIVO-nao-definido}" >>"$SBX/calls"; fi'
abre() { (cd "$SBX/onde estou" && bash "$casa/bin/mkescrita" "$@"); }
# As janelas extras sobem soltas: espera as chamadas chegarem.
espera_chamadas() {
  local n=$1
  for _ in $(seq 100); do
    (($(grep -c '^ARQUIVO=' "$SBX/calls") >= n)) && return 0
    sleep 0.05
  done
  return 1
}

t "o título do lançador é o da janela"
eq "titulo=MkEscrita" "$(grep -E '^titulo=' "$app/bin/mkescrita")"

t "mkescrita --help responde no terminal sem abrir janela (nem precisa do Quickshell)"
calls_reset
semqs=$(path_sem qs jq)
for opcao in --help -h; do
  out=$(PATH=$semqs abre "$opcao" 2>&1)
  eq 0 $?
  has "$out" "MkEscrita"
  has "$out" "mkescrita <arquivo"
  has "$out" "--nova"
done
not_called "qs"
not_called "notify-send"
not_called "hyprctl"

t "sem janela aberta, abre uma (qs -p …/qml), sem arquivo"
calls_reset
rm -f "$SBX/qs-list" "$SBX/clientes.json"
ok abre
called "qs -p $casa/qml"
called "ARQUIVO="
eq 1 "$(grep -c '^ARQUIVO=$' "$SBX/calls")"

t "o arquivo chega inteiro e absoluto, mesmo com espaço, aspas e começando com -"
calls_reset
ok abre "/tmp/a b.md"
called "ARQUIVO=/tmp/a b.md"
calls_reset
ok abre "sub/é 'x' \"y\".md"
called "ARQUIVO=$SBX/onde estou/sub/é 'x' \"y\".md"
calls_reset
ok abre -- "-rf.md"
called "ARQUIVO=$SBX/onde estou/-rf.md"
calls_reset
# shellcheck disable=SC2016 # é o texto, não a expansão, que tem de chegar
ok abre '$(touch pwn)`z`.md'
# shellcheck disable=SC2016
called "ARQUIVO=$SBX/onde estou/"'$(touch pwn)`z`.md'
absent "$SBX/onde estou/pwn"

t "janela já aberta e nenhum arquivo: só traz para a frente, não abre outra"
calls_reset
echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
echo '[{"title":"MkPulso","initialTitle":"MkPulso","address":"0x1"},{"title":"MkEscrita","initialTitle":"MkEscrita","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre
called 'hl.dsp.focus({ window = "address:0x55aa" })'
not_called "qs -p"

t "a janela é achada pelo título de nascença, mesmo depois que o título passou a dizer o documento"
calls_reset
echo '[{"title":"● notas.md — MkEscrita","initialTitle":"MkEscrita","address":"0xbeef"}]' >"$SBX/clientes.json"
ok abre
called 'hl.dsp.focus({ window = "address:0xbeef" })'
not_called "qs -p"

t "a janela de outro app não conta: abre a sua"
calls_reset
echo '[{"title":"MkComprimir","initialTitle":"MkComprimir","address":"0x77"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "janela já aberta e um arquivo: abre-o numa janela nova"
calls_reset
echo '[{"title":"MkEscrita","initialTitle":"MkEscrita","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre /tmp/novo.md
not_called "dispatch"
called "ARQUIVO=/tmp/novo.md"

t "--nova abre outra janela vazia mesmo com uma já aberta (é o Ctrl+N)"
calls_reset
ok abre --nova
not_called "dispatch"
called "qs -p $casa/qml"
eq 1 "$(grep -c '^ARQUIVO=$' "$SBX/calls")"
calls_reset
ok abre --nova -- /tmp/outro.md
called "ARQUIVO=/tmp/outro.md"

t "vários arquivos: uma janela para cada um"
calls_reset
ok abre /tmp/um.md "dois dois.md" /tmp/tres.md
ok espera_chamadas 3
called "ARQUIVO=/tmp/um.md"
called "ARQUIVO=$SBX/onde estou/dois dois.md"
called "ARQUIVO=/tmp/tres.md"
eq 3 "$(grep -c '^qs -p ' "$SBX/calls")"

t "um endereço torto vindo do Hyprland não entra no comando: abre a janela normalmente"
calls_reset
echo '[{"title":"MkEscrita","initialTitle":"MkEscrita","address":"0x1\" }) os.execute(\"x"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "opção desconhecida: diz qual, mostra o uso e sai com 2, sem abrir nada"
calls_reset
out=$(abre --apagar-tudo 2>&1)
eq 2 $?
has "$out" "--apagar-tudo"
has "$out" "mkescrita --help"
not_called "qs"

t "sem o motor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$SBX/qs-list" "$casa/bin/mkescrita-motor"
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
