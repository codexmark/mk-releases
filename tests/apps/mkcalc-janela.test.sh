#!/usr/bin/env bash
# apps/mkcalc: a janela (qml/) e o lançador (bin/mkcalc). O desenho se vê no
# bin/dev-shot; aqui: a janela só usa as cores do Mk, só pede ao motor o que o
# contrato tem, não faz conta nem roda nada além do motor, tem todas as teclas
# na tela e no teclado físico, os dados de exemplo seguem o contrato, e o
# lançador abre uma janela só.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mkcalc
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
exemplo=$TESTS/apps/mkcalc-exemplo
janela=$(<"$qml/App.qml")
ponte=$(<"$qml/Motor.qml")

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

t "o Theme e os componentes pequenos são os do Maré, iguais aos do MkUtil"
for f in Theme Botao Rotulo; do
  eq "$(<"$REPO/apps/mkutil/qml/$f.qml")" "$(<"$qml/$f.qml")" "$f.qml"
done

t "a pasta da janela é autossuficiente: todo tipo próprio usado tem o arquivo dele"
for tipo in $(grep -ohE '^\s*(App|Botao|Rotulo|Tecla|Historico|Campo|Segmento|Cabecalho|Chave) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  exists "$qml/$tipo.qml"
done
for f in shell App Motor Tecla Historico Theme Botao Rotulo; do
  exists "$qml/$f.qml"
done
exists "$qml/Calc.js"
# Nada de fora da pasta: nem o shell do Maré nem outro app.
eq "" "$(grep -nE '^\s*import +"(\.\./|/)' "$qml"/*.qml || true)"
eq "" "$(grep -n 'shell/mare' "$qml"/*.qml "$qml/Calc.js" || true)"

t "o título da janela é MkCalc (é por ele que a regra do Hyprland a faz flutuar)"
casca=$(<"$qml/shell.qml")
has "$casca" 'title: "MkCalc"'
has "$casca" "FloatingWindow {"
has "$contrato" '**`MkCalc`**'
eq "titulo=MkCalc" "$(grep -E '^titulo=' "$app/bin/mkcalc")"

t "a janela é pequena (340x520), cabe numa tela pequena e funciona estreita"
like "$casca" 'implicitWidth: Math\.min\(340, .*screen'
like "$casca" 'implicitHeight: Math\.min\(520, .*screen'
has "$casca" "minimumSize: Qt.size(240, 380)"
# Nada de largura fixa no caminho: o visor e o teclado esticam com a janela.
eq "" "$(grep -nE '^\s*(width|Layout\.minimumWidth|Layout\.preferredWidth): [0-9]{3,}' "$qml"/App.qml "$qml"/Historico.qml "$qml"/Tecla.qml || true)"
has "$janela" "Layout.fillWidth: true"
has "$janela" "Layout.fillHeight: true"

t "o teclado da tela tem as 24 teclas, e cada uma aciona uma tecla que o Calc.js conhece"
teclas=$(sed -n '/readonly property var teclas:/,/^    \]/p' "$qml/App.qml")
mapfile -t acoes < <(grep -oE 'a: "[^"]+"' <<<"$teclas" | sed 's/^a: "//; s/"$//')
eq 24 "${#acoes[@]}"
eq "% ( ) + , - 0 1 2 3 4 5 6 7 8 9 ^ apagar igual limpar sinal × ÷ √" "$(printf '%s\n' "${acoes[@]}" | LC_ALL=C sort | tr '\n' ' ' | sed 's/ $//')"
has "$janela" "columns: 4"
has "$janela" "onAcionado: app.agir(modelData.a)"
for rotulo in C "⌫" "±" "=" "√"; do
  has "$teclas" "t: \"$rotulo\""
done

t "o teclado físico passa inteiro pelo Calc.js, e toda ação que ele devolve é tratada"
has "$janela" "Calc.acoes(event.key, event.text, event.modifiers)"
has "$janela" "focus: true"
has "$janela" "forceActiveFocus()"
for acao in igual esc copiar colar historico; do
  has "$janela" "case \"$acao\":"
  ok grep -q "\"$acao\"" "$qml/Calc.js"
done
has "$janela" "estado = Calc.tecla(estado, a)"
# Esc: fecha o histórico, senão limpa, e com tudo vazio fecha a janela.
like "$(tr -s ' \n' ' ' <<<"$janela")" 'case "esc": if \(historicoAberto\) \{ historicoAberto = false; \} else if \(Calc\.vazio\(estado\) && erro === ""\) \{ Qt\.quit\(\); \} else \{ erro = ""; estado = Calc\.novo\(\); \}'

t "toda tecla da tabela do contrato existe no Calc.js"
tabela=$(sed -n '/^### O teclado/,$p' <<<"$contrato")
for tecla in '`Enter` `=`' '`Backspace`' '`Esc`' '`Ctrl+C`' '`Ctrl+V`' '`Ctrl+H`' '`S`' '`R`' '`C` `Delete`' 'NumLock'; do
  has "$tabela" "$tecla"
done

t "a janela não faz conta: o resultado do visor é o que o motor respondeu"
eq "" "$(grep -nE '\beval\(|parseFloat|parseInt|Number\(|Math\.(pow|sqrt|round|floor|ceil)|toFixed|\*\*' "$qml"/App.qml "$qml"/Historico.qml "$qml"/Tecla.qml || true)"
has "$janela" "Calc.visor(estado, previa, erro)"
has "$janela" "app.estado = Calc.confirmado(app.estado, r)"
has "$janela" "Calc.mostrar(app.estado.expr)"

t "a prévia sai a cada mudança da expressão, e só a resposta do último pedido vale"
has "$janela" "onEstadoChanged: prever()"
has "$janela" 'pedidoDaPrevia = Motor.pedir({ op: "calcular", expr: Calc.paraMotor(estado.expr) }'
has "$janela" "if (r.id === app.pedidoDaPrevia)"

t "o = guarda no histórico, e depois dele a expressão continua à vista"
has "$janela" 'Motor.pedir({ op: "calcular", expr: Calc.paraMotor(pedida), guardar: true }'
hasnt "$(sed -n '/function confirmar/,/^    }/p' "$qml/App.qml")" "Calc.novo()"

t "a janela só manda ao motor pedidos que estão no contrato, e usa todos"
n=0
while read -r op; do
  n=$((n + 1))
  has "$contrato" "\"op\": \"$op\"" "pedido fora do contrato: $op"
done < <(grep -ohE 'op: "[a-z]+"' "$qml"/*.qml | grep -oE '"[a-z]+"' | tr -d '"' | sort -u)
eq 4 "$n"
# E os campos dos pedidos são os do contrato.
while read -r campo; do
  has "$contrato" "\"$campo\":" "campo de pedido fora do contrato: $campo"
done < <(grep -ohE 'Motor\.pedir\(\{[^}]*\}' "$qml"/*.qml | grep -oE '[a-z]+:' | tr -d ':' | sort -u)

t "a janela fala com o motor só pelo modo servir, um processo só, de pé"
has "$ponte" 'command: [root.bin, "servir"]'
has "$ponte" "stdinEnabled: true"
has "$ponte" 'Quickshell.env("MKCALC_MOTOR") || Quickshell.shellPath("../bin/mkcalc-motor")'
has "$ponte" 'Quickshell.env("MKCALC_EXEMPLO")'
has "$ponte" "JSON.stringify(p) + \"\\n\""
# Se o motor cai, quem esperava resposta recebe um erro em vez de esperar para sempre.
has "$ponte" "root.desistir()"

t "a janela não roda nada além do motor (e do jq, para os exemplos)"
eq "" "$(grep -n 'command:' "$qml"/*.qml | grep -v 'Motor.qml' || true)"
eq "" "$(grep -nE 'execDetached|"(sh|bash|sudo|pkexec|rm|wl-copy|wl-paste|xclip|bc|python3?)"' "$qml"/*.qml "$qml/Calc.js" || true)"
eq 2 "$(grep -c 'command:' "$qml/Motor.qml")"

t "copiar e colar passam pela área de transferência do Qt, sem comando de fora"
has "$janela" "prancheta.copy()"
has "$janela" "prancheta.paste()"
has "$janela" "Calc.paraCopiar(estado, previa)"
has "$janela" "Calc.colar(estado, prancheta.text)"

t "o histórico abre e fecha, traz o resultado de volta, apaga uma conta e limpa"
painel=$(<"$qml/Historico.qml")
has "$janela" "historicoAberto = !historicoAberto"
has "$janela" "visible: app.historicoAberto"
has "$janela" "app.estado = Calc.doHistorico(app.estado, item)"
has "$janela" 'Motor.pedir({ op: "apagar", item: item.id }'
has "$janela" 'Motor.pedir({ op: "limpar" }'
has "$janela" 'Motor.pedir({ op: "historico" }'
has "$painel" "onClicked: painel.escolhido(linha.modelData)"
has "$painel" "onClicked: painel.apagado(linha.modelData)"
has "$painel" "painel.limpo()"

t "os dados de exemplo são JSON e existem para tudo que a janela pede"
for f in calcular historico apagar limpar; do
  ok jq -e . "$exemplo/$f.json" >/dev/null
done
eq "apagar.json calcular.json historico.json limpar.json" "$(ls "$exemplo" | tr '\n' ' ' | sed 's/ $//')"

t "os exemplos têm os campos do contrato que a janela lê"
item='has("id") and has("expressao") and has("resultado") and has("cru") and has("aproximado") and has("quando_s")'
for f in historico apagar limpar; do
  ok jq -e "(.itens | type == \"array\") and ([.itens[] | $item] | all)" "$exemplo/$f.json" >/dev/null
done
ok jq -e '.itens | length >= 3' "$exemplo/historico.json" >/dev/null
ok jq -e '.itens == []' "$exemplo/limpar.json" >/dev/null
ok jq -e 'has("*") and ([.[] | (has("erro") and has("incompleta")) or (has("expressao") and has("resultado") and has("cru") and has("aproximado"))] | all)' "$exemplo/calcular.json" >/dev/null
ok jq -e "[.[\"*\"].itens[] | $item] | all" "$exemplo/calcular.json" >/dev/null
ok jq -e '[.[] | select(has("erro"))] | (map(select(.incompleta)) | length >= 1) and (map(select(.incompleta | not)) | length >= 1)' "$exemplo/calcular.json" >/dev/null

t "os exemplos são respostas que o motor de verdade daria"
bin=$app/bin/mkcalc-motor
if [[ -x $bin ]]; then
  while IFS=$'\t' read -r expr esperado; do
    jeq "$esperado" "$("$bin" calcular "$expr")" "exemplo de: $expr"
  done < <(jq -r 'to_entries[] | select(.key != "*") | [.key, (.value | tojson)] | @tsv' "$exemplo/calcular.json")
  while IFS=$'\t' read -r expr resultado cru aproximado; do
    jeq "{\"expressao\":$(jq -Rn --arg e "$expr" '$e'),\"resultado\":\"$resultado\",\"cru\":\"$cru\",\"aproximado\":$aproximado}" "$("$bin" calcular "$expr")" "item de exemplo: $expr"
  done < <(jq -r '.itens[] | [.expressao, .resultado, .cru, (.aproximado | tostring)] | @tsv' "$exemplo/historico.json")
else
  _fail "falta o motor compilado (apps/mkcalc/pack)"
fi

t "o jq dos exemplos responde como o motor: a resposta gravada mais o id do pedido"
if command -v node >/dev/null; then
  cat >"$SBX/cmd.mjs" <<MJS
import { load } from "$TESTS/js/load.mjs";
const C = load("$qml/Calc.js");
for (const p of JSON.parse(process.argv[2]))
    console.log(C.comandoDeExemplo(process.argv[3], p).map((a) => Buffer.from(a).toString("base64")).join(" "));
MJS
  pedidos='[{"id":7,"op":"calcular","expr":"200+10%"},{"id":8,"op":"calcular","expr":"2+"},{"id":9,"op":"calcular","expr":"3×3"},{"id":10,"op":"historico"},{"id":11,"op":"limpar"}]'
  respostas=()
  while read -r linha; do
    cmd=()
    for parte in $linha; do cmd+=("$(base64 -d <<<"$parte")"); done
    eq jq "${cmd[0]}"
    respostas+=("$("${cmd[@]}")")
  done < <(node "$SBX/cmd.mjs" "$pedidos" "$exemplo")
  eq 5 "${#respostas[@]}"
  jeq '{"id":7,"expressao":"200 + 10%","resultado":"220","cru":"220","aproximado":false}' "${respostas[0]}"
  jeq '{"id":8,"erro":"expressão incompleta","incompleta":true}' "${respostas[1]}"
  eq "9 1234500" "$(jq -r '"\(.id) \(.cru)"' <<<"${respostas[2]}")"
  eq "10 3" "$(jq -r '"\(.id) \(.itens | length)"' <<<"${respostas[3]}")"
  jeq '{"id":11,"itens":[]}' "${respostas[4]}"
else
  echo "  # node ausente: pulado" >&2
fi

t "nada de dados da máquina de quem escreveu nos exemplos"
eq "" "$(grep -lE '/home/[a-z]+/|codexmark' "$exemplo"/*.json || true)"

t "o atalho de menu: abre o mkcalc, com o ícone dele, na categoria de calculadora"
d=$(<"$app/share/mkcalc.desktop")
has "$d" "Name=MkCalc"
has "$d" "Exec=mkcalc"
has "$d" "Icon=mkcalc"
has "$d" "Terminal=false"
has "$d" "Categories=Utility;Calculator;"
hasnt "$d" "MimeType"
exists "$app/share/mkcalc.svg"
has "$(<"$app/share/mkcalc.svg")" "<title>MkCalc</title>"

t "o pack, a assinatura e o lançador são executáveis"
for f in pack assinatura bin/mkcalc bin/mkcalc-motor; do
  ok test -x "$app/$f"
done

# --- o lançador ---------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml" "$SBX/onde estou"
cp "$app/bin/mkcalc" "$casa/bin/"
printf '#!/bin/sh\n' >"$casa/bin/mkcalc-motor"
chmod +x "$casa/bin/mkcalc-motor"
stub notify-send
stub hyprctl '[[ $1 == clients ]] && cat "$SBX/clientes.json" 2>/dev/null; exit 0'
stub qs 'if [[ $1 == list ]]; then cat "$SBX/qs-list" 2>/dev/null; else echo "EXPR=${MKCALC_EXPR-<sem>}" >>"$SBX/calls"; fi'
# shellcheck disable=SC2120 # os argumentos chegam por `ok abre …`
abre() { (cd "$SBX/onde estou" && bash "$casa/bin/mkcalc" "$@"); }

t "mkcalc: sem janela aberta, abre a janela (qs -p …/qml), sem expressão"
calls_reset
rm -f "$SBX/qs-list" "$SBX/clientes.json"
ok abre
called "qs -p $casa/qml"
called "EXPR=<sem>"

t "mkcalc: janela já aberta: só traz para a frente, não abre outra"
calls_reset
echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
echo '[{"title":"MkPulso","address":"0x1"},{"title":"MkCalc","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre
called 'hl.dsp.focus({ window = "address:0x55aa" })'
not_called "qs -p"

t "mkcalc: a janela de outro app do Mk aberta não conta: abre a sua"
calls_reset
echo '[{"title":"MkPulso","address":"0x77"},{"title":"MkCalculadora","address":"0x78"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "mkcalc: um endereço torto vindo do Hyprland não entra no comando: abre a janela normalmente"
calls_reset
echo '[{"title":"MkCalc","address":"0x1\" }) os.execute(\"x"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "mkcalc \"2 + 2\": abre uma janela já com a expressão, inteira, mesmo com outra aberta"
calls_reset
echo '[{"title":"MkCalc","address":"0x55aa"}]' >"$SBX/clientes.json"
# shellcheck disable=SC2016 # é o texto, não a expansão, que tem de chegar
ok abre '2 + 2 * $(touch pwn) `z`' "; 'x'"
not_called "dispatch"
# shellcheck disable=SC2016
called 'EXPR=2 + 2 * $(touch pwn) `z` ; '"'x'"
absent "$SBX/onde estou/pwn"
# A expressão passa pelo filtro do que é colado: o que não é conta não entra.
has "$janela" "Calc.colar(Calc.novo(), Motor.inicial)"
has "$ponte" 'Quickshell.env("MKCALC_EXPR")'

t "sem o motor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$SBX/qs-list" "$casa/bin/mkcalc-motor"
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
