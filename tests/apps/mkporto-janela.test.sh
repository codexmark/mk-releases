#!/usr/bin/env bash
# apps/mkporto: a janela (qml/) e o lançador (bin/mkporto). O desenho se vê no
# bin/dev-shot; aqui: a janela só usa as cores do Mk, só pede ao motor o que o
# contrato tem, não monta comando nenhum (abre o argv que o motor devolve),
# não tem campo de senha, não sonda em laço, valida o formulário com as mesmas
# regras do motor, os dados de exemplo têm a forma do que o motor responde, e
# o lançador abre uma janela só.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mkporto
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
exemplo=$TESTS/apps/mkporto-exemplo
motor=$app/bin/mkporto-motor

t "cores: só os tokens do Mk, e só no Theme.qml"
tokens=" #0b0b0b #141211 #1b1917 #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml "$app/share/mkporto.svg" | sort -u)
eq "" "$(grep -lE '#[0-9a-fA-F]{3,8}\b' "$qml"/*.qml | grep -v '/Theme.qml$' || true)"
out=$(grep -ohE 'Qt\.rgba\([^)]*\)' "$qml"/*.qml | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255|0, 0, 0, 0)' || true)
eq "" "$out"
eq "" "$(grep -nE 'color: *"[a-z]+"|Qt\.(darker|lighter|hsla|hsva|tint|color)\(' "$qml"/*.qml || true)"

t "o Theme e os componentes pequenos são os mesmos do MkPulso (cada app é uma pasta que se basta)"
for f in Theme Botao Segmento Rotulo; do
  ok cmp -s "$qml/$f.qml" "$REPO/apps/mkpulso/qml/$f.qml"
done
ok cmp -s "$qml/Cabecalho.qml" "$REPO/apps/mkutil/qml/Cabecalho.qml"

t "todo tipo próprio usado na janela tem o arquivo dele"
n=0
for tipo in $(grep -ohE '^\s*(Botao|Campo|Entrada|Segmento|Rotulo|Cabecalho|Bolinha|Fundo|Lista|Detalhe|Formulario|Confirma|Escolha|NovaChave|Chaves|Atalhos|App) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  n=$((n + 1))
  exists "$qml/$tipo.qml"
done
eq 17 "$n"
exists "$qml/Motor.qml"
exists "$qml/Porto.js"
# E nenhum arquivo sobra sem uso.
for f in "$qml"/*.qml; do
  tipo=$(basename "$f" .qml)
  [[ $tipo == shell || $tipo == Theme || $tipo == Motor ]] && continue
  grep -qE "^\s*$tipo \{" "$qml"/*.qml || _fail "ninguém usa: $tipo.qml"
done

t "a janela só pede ao motor subcomandos que estão no contrato, e usa todos eles"
mapfile -t pedidos < <(grep -ohE 'Motor\.(pedir|tarefa)\(\["[a-z]+"' "$qml"/*.qml | grep -oE '"[a-z]+"' | tr -d '"' | sort -u)
for sub in "${pedidos[@]}"; do
  has "$contrato" "## \`mkporto-motor $sub" "subcomando fora do contrato: $sub"
done
mapfile -t docontrato < <(grep -oE '^## `mkporto-motor [a-z]+' <<<"$contrato" | awk '{print $3}' | sort -u)
eq "${docontrato[*]}" "${pedidos[*]}"
eq 7 "${#pedidos[@]}"

t "toda chamada ao motor diz o subcomando na hora, e só o App fala com ele"
eq "" "$(grep -nE 'Motor\.(pedir|tarefa)\(' "$qml"/*.qml | grep -vE 'Motor\.(pedir|tarefa)\(\["[a-z]+"' || true)"
eq "$qml/App.qml" "$(grep -lE 'Motor\.(pedir|tarefa|abrir)\(' "$qml"/*.qml | tr '\n' ' ' | sed 's/ $//')"

t "o que a pessoa escolheu vai para o motor depois de --, nunca como opção"
casca=$(<"$qml/App.qml")
has "$casca" 'Motor.tarefa(["sondar", "--"].concat(quais)'
eq 2 "$(grep -c 'Motor.pedir(\["comando", "conectar", "--", app.host.apelido\]' "$qml/App.qml")"
has "$casca" 'Motor.pedir(["comando", "chave", "--", app.host.apelido, nome]'
has "$casca" 'Motor.pedir(["remover", "--", alvo]'
# O host novo vai como JSON pela entrada padrão, não na linha de comando.
like "$casca" 'Motor\.pedir\(\["guardar"\], r => \{'
has "$casca" '}, JSON.stringify(pedido));'
# O nome e o comentário da chave nova também: depois de --, e conferidos pelo motor.
has "$casca" 'Motor.pedir(["comando", "gerar", "--"].concat(Porto.argsDaChave(form))'
has "$casca" 'Motor.pedir(["entrar", "--", alvo]'

t "a janela não monta comando nenhum: abre o argv que o motor devolveu, e nada além do motor (e do jq, para os exemplos)"
eq "" "$(grep -n 'command:' "$qml"/*.qml | grep -v '/Motor.qml:' || true)"
eq "$qml/Motor.qml:        Quickshell.execDetached(resposta.terminal);" "$(grep -n 'execDetached' "$qml"/*.qml | sed -E 's/:[0-9]+:/:/')"
eq "" "$(grep -nE '"(sh|bash|ssh|ssh-copy-id|ssh-keygen|scp|sftp|sudo|pkexec|sshpass|ghostty|kitty|foot|alacritty|wezterm|xdg-open|rm)"' "$qml"/*.qml "$qml/Porto.js" || true)"
ponte=$(<"$qml/Motor.qml")
has "$ponte" 'return ["jq", "-c", emLinhas ? ".[]" : ".", exemplo + "/" + Porto.arquivoDeExemplo(args)];'
has "$ponte" 'return [bin].concat(args);'

t "com MKPORTO_EXEMPLO a janela não abre terminal nenhum: só diz o que abriria"
like "$ponte" 'function abrir\(resposta\) \{[[:space:]]+if \(exemplo !== ""\)[[:space:]]+return Porto\.abriria\(resposta\);'
has "$ponte" 'Quickshell.env("MKPORTO_EXEMPLO")'
has "$ponte" 'Quickshell.env("MKPORTO_MOTOR") || Quickshell.shellPath("../bin/mkporto-motor")'

t "não há campo de senha em lugar nenhum, e a janela não lê arquivo"
eq "" "$(grep -niE 'echoMode|password|passphrase|FileView|XMLHttpRequest' "$qml"/*.qml "$qml/Porto.js" || true)"
eq "" "$(grep -nE 'rotulo: "[^"]*senha' "$qml"/*.qml || true)"
mapfile -t campos_do_form < <(grep -oE 'rotulo: "[^"]+"' "$qml/Formulario.qml" | cut -d'"' -f2)
eq "apelido endereço usuário porta chave salto" "${campos_do_form[*]}"
# A folha da chave nova só tem o nome e o comentário: o que protege a chave é
# perguntado pelo ssh-keygen, no terminal.
mapfile -t campos_da_chave < <(grep -oE 'rotulo: "[^"]+"' "$qml/NovaChave.qml" | cut -d'"' -f2)
eq "nome comentário" "${campos_da_chave[*]}"

t "a sondagem roda ao abrir, no botão e no F5, e depois de guardar: nunca em laço"
eq "" "$(grep -nE '\bTimer\b|setInterval|setTimeout' "$qml"/*.qml "$qml/Porto.js" || true)"
has "$casca" 'Component.onCompleted: ler(() => sondar())'
has "$casca" 'Shortcut { sequences: ["F5"]; enabled: app.livre && app.area === "hosts"; onActivated: app.sondar() }'
has "$casca" '"sondar de novo"'
has "$casca" 'app.ler(() => app.sondar([pedido.apelido]));'
eq 4 "$(grep -cE '\bsondar\(' <(grep -v 'function sondar' "$qml/App.qml"))"
eq "" "$(grep -n 'sondar(' "$qml"/*.qml | grep -v '/App.qml:' || true)"

t "editar e remover só valem para o que é do MkPorto, e remover passa pela confirmação"
detalhe=$(<"$qml/Detalhe.qml")
eq 2 "$(grep -c 'ligado: detalhe.doApp' "$qml/Detalhe.qml")"
has "$detalhe" 'readonly property bool doApp: Porto.editavel(host)'
has "$casca" 'onRemover: app.abrirFolha("remover")'
like "$casca" 'if \(qual === "editar" \|\| qual === "remover"\) \{[[:space:]]+if \(!Porto\.editavel\(app\.host\)\)'
like "$casca" 'function remover\(\) \{[[:space:]]+if \(!app\.host \|\| !Porto\.editavel\(app\.host\)'
# Quem chama o remover de verdade: a folha de confirmação e o Enter com ela na tela.
eq "Shortcut { sequences: [\"Return\", \"Enter\"]; enabled: app.folha === \"remover\"; onActivated: app.remover() }|onConfirmado: app.remover()" \
  "$(grep -E 'app\.remover\(\)' "$qml/App.qml" | sed 's/^ *//' | sort | tr '\n' '|' | sed 's/|$//')"

t "as ações do detalhe são as do pedido: conectar, copiar comando, testar entrada, enviar chave, editar e remover"
mapfile -t acoes < <(grep -oE 'texto: "[^"]+"' "$qml/Detalhe.qml" | cut -d'"' -f2)
eq "‹ voltar|conectar|copiar comando|testar entrada|enviar chave|editar|remover" "$(IFS='|'; echo "${acoes[*]}")"
has "$(<"$qml/Chaves.qml")" 'texto: "copiar a pública"'

t "testar entrada só roda quando a pessoa pede, e o resultado é do host em que foi pedido"
eq 1 "$(grep -c 'Motor.pedir(\["entrar"' "$qml/App.qml")"
eq "Shortcut { sequences: [\"Ctrl+T\"]; enabled: app.livre && app.area === \"hosts\"; onActivated: app.testarEntrada() }|onTestarEntrada: app.testarEntrada()" \
  "$(grep -E 'app\.testarEntrada\(\)' "$qml/App.qml" | sed 's/^ *//' | sort | tr '\n' '|' | sed 's/|$//')"
has "$casca" 'onApelidoChanged: entrada = null'
has "$casca" 'if (app.entrada === null || app.entrada.apelido !== alvo)'
has "$detalhe" 'readonly property string textoDaEntrada: Porto.textoDaEntrada(entrada, host)'
has "$detalhe" 'visible: detalhe.textoDaEntrada !== ""'
# O detalhe mostra o servidor que respondeu e de onde vêm os valores.
has "$detalhe" '{ k: "servidor", v: (detalhe.sonda !== null && detalhe.sonda.servidor) || "—" }'
has "$detalhe" '{ k: "valores", v: Porto.fonteDosValores(detalhe.host) }'

t "a área Chaves tem nova chave (que só abre um terminal) e atualizar; F5 ali relê as chaves"
chaves=$(<"$qml/Chaves.qml")
has "$chaves" 'texto: "nova chave"'
has "$chaves" 'texto: "atualizar"'
has "$casca" 'onNovaChave: app.abrirFolha("gerar")'
has "$casca" 'onAtualizar: app.lerChaves()'
has "$casca" 'Shortcut { sequences: ["F5"]; enabled: app.livre && app.area === "chaves"; onActivated: app.lerChaves() }'
has "$casca" 'Shortcut { sequences: ["Ctrl+G"]; enabled: app.livre; onActivated: { app.area = "chaves"; app.abrirFolha("gerar"); } }'
has "$casca" 'visible: app.folha === "gerar"'
# A folha confere o nome e o comentário com as regras do motor, e quem gera é o terminal.
nova=$(<"$qml/NovaChave.qml")
has "$nova" 'readonly property var erros: Porto.validarChave(form, chaves)'
has "$nova" 'texto: "gerar num terminal"'
like "$casca" 'function gerarChave\(form\) \{[^}]*Motor\.pedir\(\["comando", "gerar", "--"\]'
has "$casca" 'app.recado = Motor.abrir(r) || "gerando a chave num terminal; quando terminar, atualize (F5)";'

t "os atalhos do contrato existem na janela, e a folha (F1) lista os mesmos"
for tecla in '"Ctrl+F"' '"Ctrl+N"' '"Ctrl+E"' '"Ctrl+T"' '"Ctrl+G"' '"Ctrl+K"' '"Delete"' '"Ctrl+Shift+C"' '"F5"' '"Ctrl+1"' '"Ctrl+2"' '"Up"' '"Down"' '"Return", "Enter"' '"Escape"' '"F1"'; do
  has "$casca" "Shortcut { sequences: [$tecla]" "falta o atalho $tecla"
done
has "$casca" 'Shortcut { sequences: ["Return", "Enter"]; enabled: app.livre && app.area === "hosts"; onActivated: app.conectar() }'
has "$casca" 'Shortcut { sequences: ["Ctrl+N"]; enabled: app.livre; onActivated: app.abrirFolha("novo") }'
has "$casca" 'Shortcut { sequences: ["Escape"]; onActivated: app.fechar() }'
for tecla in Ctrl+F Ctrl+N Ctrl+E Ctrl+T Ctrl+G Ctrl+K Delete Ctrl+Shift+C F5 Ctrl+1 Ctrl+2 Enter Esc F1; do
  has "$(<"$qml/Porto.js")" "[\"$tecla\", " "a folha de atalhos não tem: $tecla"
  has "$contrato" "\`$tecla\`" "o contrato não tem o atalho: $tecla"
done
has "$(<"$qml/Atalhos.qml")" 'model: Porto.ATALHOS'

t "a janela se chama MkPorto, nasce por volta de 900x600 (ou o que couber) e aceita até 520x420"
janela=$(<"$qml/shell.qml")
has "$janela" 'title: "MkPorto"'
like "$janela" 'implicitWidth: Math\.min\(900, .*screen'
like "$janela" 'implicitHeight: Math\.min\(600, .*screen'
has "$janela" 'minimumSize: Qt.size(520, 420)'

t "abaixo de 720 de largura o detalhe sai do lado da lista e ganha um voltar"
has "$casca" 'readonly property bool estreita: Porto.estreita(width)'
has "$casca" 'visible: !app.estreita || !app.detalheAberto'
has "$casca" 'visible: !app.estreita || app.detalheAberto'
has "$casca" 'width: app.estreita ? parent.width'
like "$detalhe" 'visible: detalhe\.estreita[[:space:]]+texto: "‹ voltar"'
has "$(<"$qml/Porto.js")" 'var LARGURA_DE_DUAS_COLUNAS = 720;'

t "sem host nenhum a janela explica e oferece o novo host"
like "$casca" 'app\.hosts\.length === 0[[:space:]]'
has "$casca" '"Nenhum host ainda"'
ok test "$(grep -c 'onAcionado: app.abrirFolha("novo")' "$qml/App.qml")" -ge 2

t "os dados de exemplo são JSON, um por pedido que a janela faz"
for f in listar sondar entrar chaves guardar remover comando-conectar comando-chave comando-gerar; do
  ok jq -e . "$exemplo/$f.json" >/dev/null
  has "$contrato" "\`$f.json\`" "o contrato não cita o exemplo $f.json"
done
eq 9 "$(find "$exemplo" -type f | wc -l)"

t "os exemplos têm uns 6 hosts de mentira: só domínios .example/.test e IPs de documentação"
eq 6 "$(jq '.hosts | length' "$exemplo/listar.json")"
eq "" "$(jq -r '.hosts[].host' "$exemplo/listar.json" | grep -vE '^(192\.0\.2\.[0-9]+|[a-z0-9.-]+\.(example|test))$' || true)"
ok jq -e '[.hosts[].origem] | (index("mkporto") != null) and (index("config") != null) and all(. == "mkporto" or . == "config")' "$exemplo/listar.json" >/dev/null
ok jq -e '[.hosts[] | select(.salto != null)] | length >= 1' "$exemplo/listar.json" >/dev/null
# Cada host do exemplo tem a sua sonda, e há de tudo: responde (com o servidor),
# não responde, porta que não é ssh, e responde pelo salto.
jeq "$(jq -c '[.hosts[].apelido] | sort' "$exemplo/listar.json")" "$(jq -c '[.[].apelido] | sort' "$exemplo/sondar.json")"
ok jq -e '([.[] | select(.responde == true and .servidor != null and .via == null)] | length >= 1)
  and ([.[] | select(.responde == false)] | length >= 1)
  and ([.[] | select(.motivo == "a porta abre, mas não é um servidor ssh")] | length == 1)
  and ([.[] | select(.responde == true and .via != null)] | length >= 1)' "$exemplo/sondar.json" >/dev/null
# Quem respondeu "por" alguém tem esse alguém como salto na lista.
jeq "$(jq -c '[.hosts[] | select(.salto != null) | {apelido, via: .salto}] | sort_by(.apelido)' "$exemplo/listar.json")" \
  "$(jq -c '[.[] | select(.via != null) | {apelido, via}] | sort_by(.apelido)' "$exemplo/sondar.json")"
# O que o ssh completou não se mistura com o que está escrito no arquivo do app.
ok jq -e '[.hosts[] | (.origem == "mkporto") == (.escrito != null)] | all' "$exemplo/listar.json" >/dev/null
ok jq -e '[.hosts[] | select(.escrito != null and .escrito.usuario == null and .usuario != null)] | length >= 1' "$exemplo/listar.json" >/dev/null
ok jq -e '.chaves | length >= 2' "$exemplo/chaves.json" >/dev/null

t "nada de dados da máquina de quem escreveu nos exemplos, nem chave de verdade"
eq "" "$(grep -lE '/home/[a-z]+/|codexmark|BEGIN [A-Z ]*PRIVATE' "$exemplo"/*.json || true)"
eq "" "$(grep -ohE '[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}' "$exemplo"/*.json | grep -v '^192\.0\.2\.' || true)"
eq "" "$(grep -ohE '[A-Za-z0-9._-]+@[A-Za-z0-9.-]+' "$exemplo"/*.json | grep -vE '\.(example|test)$' || true)"
ok jq -e '[.chaves[].publica | test("EXEMPLO")] | all' "$exemplo/chaves.json" >/dev/null

# --- os exemplos contra o motor de verdade --------------------------------------
# Uma casa de mentira com dois hosts e uma chave: o motor responde, e o exemplo
# tem de ter os mesmos campos.
ssh=$SBX/home/.ssh
mkdir -p "$ssh"
printf 'Host farol\n  HostName 192.0.2.10\n  User ana\n  IdentityFile ~/.ssh/id_ed25519\nHost cofre\n  HostName cofre.interno.test\n  ProxyJump farol\n' >"$ssh/config"
printf 'ssh-ed25519 QUFBQUVYRU1QTE8= ana@oficina.example\n' >"$ssh/id_ed25519.pub"
stub ssh-keygen 'echo "256 SHA256:EXEMPLO ana@oficina.example (ED25519)"'
stub foot
# O ssh de mentira: não sabe a configuração de ninguém (vale o leitor próprio),
# o túnel pelo salto traz uma saudação e a entrada é aceita.
stub ssh 'case " $* " in
  *" -G "*) exit 1 ;;
  *" -W "*) printf "SSH-2.0-Falso_1.0\r\n" ;;
  *" true "*) exit 0 ;;
  *) exit 9 ;;
esac'
responde() { MKPORTO_SSH_DIR=$ssh MKPORTO_TERMINAL=foot "$motor" "$@"; }
# Os campos de um JSON, em qualquer profundidade, sem os índices das listas.
campos() { jq -r '[paths | map(select(type == "string")) | join(".")] | unique | .[]' <<<"$1"; }
mesmos_campos() {
  local nome=$1 veio=$2
  eq "$(campos "$veio")" "$(campos "$(<"$exemplo/$nome.json")")" "$nome.json não tem os campos que o motor responde"
}

t "exemplos: guardar, listar, chaves e remover têm exatamente os campos que o motor responde"
mesmos_campos guardar "$(responde guardar <<<'{"apelido":"atlas","host":"atlas.oficina.example","usuario":"marina","porta":2200}')"
mesmos_campos listar "$(responde listar)"
mesmos_campos chaves "$(responde chaves)"
mesmos_campos remover "$(responde remover -- atlas)"

t "exemplos: o comando de conectar e o de enviar chave também, e o sondar linha a linha"
mesmos_campos comando-conectar "$(responde comando conectar -- farol)"
mesmos_campos comando-chave "$(responde comando chave -- farol id_ed25519.pub)"
# (o cofre só se alcança por salto: o motor pede o túnel ao ssh de mentira e não disca para ninguém)
veio=$(responde sondar -- cofre)
eq "$(campos "$veio")" "$(campos "$(jq -c '.[0]' "$exemplo/sondar.json")")"
jeq '{"apelido":"cofre","responde":true,"motivo":null,"servidor":"Falso_1.0","via":"farol"}' "$(jq -c 'del(.ms)' <<<"$veio")"
called "ssh -o BatchMode=yes -o ConnectTimeout=4 -W cofre.interno.test:22 -- farol"
ok jq -e 'type == "array" and ([.[] | keys == ["apelido", "motivo", "ms", "responde", "servidor", "via"]] | all)' "$exemplo/sondar.json" >/dev/null
mesmos_campos entrar "$(responde entrar -- farol)"
mesmos_campos comando-gerar "$(responde comando gerar -- nova)"
jeq '["ssh-keygen","-t","ed25519","-f"]' "$(jq -c '.argv[0:4]' "$exemplo/comando-gerar.json")"
not_called "ssh-keygen -t"
jeq '["ssh","--","atlas"]' "$(jq -c .argv "$exemplo/comando-conectar.json")"
eq "ssh-copy-id -i" "$(jq -r '.argv[0:2] | join(" ")' "$exemplo/comando-chave.json")"
not_called "foot"

t "o formulário recusa o mesmo que o motor: as regras do Porto.js são o espelho das do guardar"
command -v node >/dev/null || skip "falta o node"
cat >"$SBX/espelho.mjs" <<MJS
import { load, REPO } from "$TESTS/js/load.mjs";
const P = load(REPO + "/apps/mkporto/qml/Porto.js");
const f = JSON.parse(process.argv[2]);
const form = { apelido: f.apelido ?? "", host: f.host ?? "", usuario: f.usuario ?? "", porta: String(f.porta ?? ""), chave: f.chave ?? "", salto: f.salto ?? "" };
const erros = { apelido: P.erroDoApelido(form.apelido), host: P.erroDoHost(form.host), usuario: P.erroDoUsuario(form.usuario),
    porta: P.erroDaPorta(form.porta), chave: P.erroDaChave(form.chave), salto: P.erroDoSalto(form.salto) };
const campo = ["apelido", "host", "usuario", "porta", "chave", "salto"].find((c) => erros[c] !== "") || "";
console.log(campo + "|" + (campo ? erros[campo] : ""));
MJS
n=0
while IFS= read -r pedido; do
  n=$((n + 1))
  limpa=$SBX/espelho-$n
  mkdir -p "$limpa"
  veio=$(MKPORTO_SSH_DIR=$limpa/.ssh "$motor" guardar <<<"$pedido")
  do_motor="$(jq -r '.campo // ""' <<<"$veio")|$(jq -r '.erro // ""' <<<"$veio")"
  da_janela=$(node "$SBX/espelho.mjs" "$pedido")
  eq "$do_motor" "$da_janela" "o motor e a janela discordam em: $pedido"
done <<'PEDIDOS'
{"apelido":"atlas","host":"atlas.oficina.example","usuario":"marina","porta":2200,"chave":"~/.ssh/minha chave","salto":"ana@farol:2222"}
{"apelido":"Web_1.prod-2","host":"2001:db8::1","porta":65535}
{"apelido":"a","host":"192.0.2.1","porta":1}
{"apelido":"","host":"x.test"}
{"apelido":"-atlas","host":"x.test"}
{"apelido":"-oProxyCommand=touch pwn","host":"x.test"}
{"apelido":"a b","host":"x.test"}
{"apelido":"web*","host":"x.test"}
{"apelido":"!x","host":"x.test"}
{"apelido":"máquina","host":"x.test"}
{"apelido":"a/b","host":"x.test"}
{"apelido":"a;b","host":"x.test"}
{"apelido":"a\nHost *","host":"x.test"}
{"apelido":"a\rb","host":"x.test"}
{"apelido":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","host":"x.test"}
{"apelido":"a","host":""}
{"apelido":"a","host":"-oProxyCommand=touch pwn"}
{"apelido":"a","host":"x test"}
{"apelido":"a","host":"x.test\nProxyCommand touch pwn"}
{"apelido":"a","host":"x.test\r\nb"}
{"apelido":"a","host":"x\tb"}
{"apelido":"a","host":"x\u0000b"}
{"apelido":"a","host":"x\u007fb"}
{"apelido":"a","host":"x b"}
{"apelido":"a","host":"x\"b"}
{"apelido":"a","host":"#x"}
{"apelido":"a","host":"x.test","usuario":"-l"}
{"apelido":"a","host":"x.test","usuario":"ana silva"}
{"apelido":"a","host":"x.test","usuario":"ana\nPort 1"}
{"apelido":"a","host":"x.test","usuario":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
{"apelido":"a","host":"x.test","porta":0}
{"apelido":"a","host":"x.test","porta":65536}
{"apelido":"a","host":"x.test","porta":-1}
{"apelido":"a","host":"x.test","chave":"-k"}
{"apelido":"a","host":"x.test","chave":"~/.ssh/k\nProxyCommand x"}
{"apelido":"a","host":"x.test","chave":"~/.ssh/k\" x \""}
{"apelido":"a","host":"x.test","chave":"~/.ssh/k\\"}
{"apelido":"a","host":"x.test","chave":"a\u0007b"}
{"apelido":"a","host":"x.test","salto":"-oProxyCommand=x"}
{"apelido":"a","host":"x.test","salto":"a b"}
{"apelido":"a","host":"x.test","salto":"a\nb"}
{"apelido":"a","host":"x.test","salto":"a#b"}
PEDIDOS
ok test "$n" -ge 40

t "a folha da chave nova recusa o mesmo que o motor: nome e comentário pelas mesmas regras"
cat >"$SBX/espelho-chave.mjs" <<MJS
import { load, REPO } from "$TESTS/js/load.mjs";
const P = load(REPO + "/apps/mkporto/qml/Porto.js");
const f = JSON.parse(process.argv[2]);
const erros = { nome: P.erroDoNomeDaChave(f.nome ?? ""), comentario: P.erroDoComentario(f.comentario ?? "") };
const campo = ["nome", "comentario"].find((c) => erros[c] !== "") || "";
console.log(campo + "|" + (campo ? erros[campo] : ""));
MJS
n=0
while IFS= read -r pedido; do
  n=$((n + 1))
  limpa=$SBX/espelho-chave-$n
  mkdir -p "$limpa/.ssh"
  # (o nome e o comentário vão como argumentos de verdade: o jq os tira do JSON)
  mapfile -d '' -t partes < <(jq -j '(.nome // ""), "\u0000", (.comentario // ""), "\u0000"' <<<"$pedido")
  veio=$(MKPORTO_SSH_DIR=$limpa/.ssh MKPORTO_TERMINAL=foot "$motor" comando gerar -- "${partes[0]}" "${partes[1]:-}")
  do_motor="$(jq -r '.campo // ""' <<<"$veio")|$(jq -r 'if .campo then .erro else "" end' <<<"$veio")"
  da_janela=$(node "$SBX/espelho-chave.mjs" "$pedido")
  eq "$do_motor" "$da_janela" "o motor e a janela discordam em: $pedido"
done <<'PEDIDOS'
{"nome":"trabalho","comentario":"ana no trabalho"}
{"nome":"id_ed25519_casa","comentario":""}
{"nome":"Chave-2.nova","comentario":"ana d'água $USER"}
{"nome":"","comentario":""}
{"nome":"-x","comentario":""}
{"nome":"-oProxyCommand=x","comentario":""}
{"nome":".oculta","comentario":""}
{"nome":"a b","comentario":""}
{"nome":"a/b","comentario":""}
{"nome":"../x","comentario":""}
{"nome":"chave.pub","comentario":""}
{"nome":"máquina","comentario":""}
{"nome":"a\nb","comentario":""}
{"nome":"a;b","comentario":""}
{"nome":"$(x)","comentario":""}
{"nome":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","comentario":""}
{"nome":"boa","comentario":"-x"}
{"nome":"boa","comentario":"a\nb"}
{"nome":"boa","comentario":"a\rb"}
{"nome":"boa","comentario":"a\tb"}
{"nome":"boa","comentario":"a\u0007b"}
{"nome":"boa","comentario":"ééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééééé"}
{"nome":"-x","comentario":"-y"}
PEDIDOS
ok test "$n" -ge 20
not_called "ssh-keygen -t"

# --- o lançador ---------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml"
cp "$app/bin/mkporto" "$casa/bin/mkporto"
printf '#!/bin/sh\n' >"$casa/bin/mkporto-motor"
chmod +x "$casa/bin/mkporto-motor"
stub notify-send
stub hyprctl '[[ $1 == clients ]] && cat "$SBX/clientes.json" 2>/dev/null; exit 0'
stub qs '[[ $1 == list ]] && cat "$SBX/qs-list" 2>/dev/null; exit 0'
abre() { bash "$casa/bin/mkporto" "$@"; }

t "o lançador, o pack, a assinatura e o motor têm o bit de execução"
for f in bin/mkporto pack assinatura bin/mkporto-motor; do
  ok test -x "$app/$f"
done

t "sem janela aberta: abre a janela do app (qs -p …/qml)"
calls_reset
ok abre
called "qs -p $casa/qml"

t "o lançador não aceita argumento: uso e saída 2, sem abrir nada"
calls_reset
abre atlas 2>/dev/null
eq 2 $?
abre -- -oProxyCommand=x 2>/dev/null
eq 2 $?
not_called "qs -p"

t "janela já aberta: só traz para a frente, não abre outra"
calls_reset
echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
echo '[{"title":"MkPulso","address":"0x1"},{"title":"ssh · atlas","address":"0x2"},{"title":"MkPorto","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre
called 'hl.dsp.focus({ window = "address:0x55aa" })'
not_called "qs -p"

t "o terminal de uma conexão não conta como a janela do app"
calls_reset
echo '[{"title":"ssh · atlas","address":"0x2"},{"title":"MkPorto — atlas","address":"0x3"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "um endereço torto vindo do Hyprland não entra no comando: abre a janela normalmente"
calls_reset
echo '[{"title":"MkPorto","address":"0x1\" }) os.execute(\"x"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "sem o motor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$SBX/qs-list" "$casa/bin/mkporto-motor"
out=$(abre 2>&1)
eq 1 $?
has "$out" "motor"
has "$out" "mk atualizar"
called "notify-send"
not_called "qs -p"

t "sem o Quickshell: avisa e sai com 1"
calls_reset
printf '#!/bin/sh\n' >"$casa/bin/mkporto-motor"
chmod +x "$casa/bin/mkporto-motor"
semqs=$(path_sem qs)
out=$(PATH=$semqs abre 2>&1)
eq 1 $?
has "$out" "Quickshell"
called "notify-send"

t "o lançador nunca chama o ssh nem um terminal: quem conecta é a janela, com o argv do motor"
eq "" "$(grep -nE '\b(ssh|ssh-copy-id|sudo|pkexec|ghostty|kitty|foot|alacritty)\b' "$app/bin/mkporto" | grep -v '^[0-9]*:#' || true)"

t "o atalho do menu chama o lançador e usa o ícone do app"
atalho=$(<"$app/share/mkporto.desktop")
has "$atalho" "Exec=mkporto"
has "$atalho" "Icon=mkporto"
has "$atalho" "Name=MkPorto"
has "$atalho" "Terminal=false"
hasnt "$atalho" "MimeType"
hasnt "$atalho" "%"
exists "$app/share/mkporto.svg"
like "$(<"$app/share/mkporto.svg")" '<title>MkPorto</title>'

t "enviar chave e remover também têm tecla (Ctrl+K e Delete), pelas mesmas folhas dos botões"
has "$casca" 'Shortcut { sequences: ["Ctrl+K"]; enabled: app.livre && app.area === "hosts"; onActivated: app.abrirFolha("chave") }'
has "$casca" 'Shortcut { sequences: ["Delete"]; enabled: app.livre && app.area === "hosts"; onActivated: app.abrirFolha("remover") }'

t "numa coluna apertada (um terço de tela) o alto cede: sem a marca e sem o sondar, o novo cabe"
has "$casca" 'readonly property bool apertada: Porto.apertada(width)'
has "$casca" 'visible: !app.apertada'
has "$casca" 'visible: app.area === "hosts" && !app.apertada'

t "a escolha da chave anda pelo teclado: setas marcam, Enter envia a marcada"
escolha=$(<"$qml/Escolha.qml")
has "$escolha" 'property int marcada: 0'
has "$escolha" 'function andar(passo)'
has "$escolha" 'function confirmar()'
has "$escolha" 'onChavesChanged: marcada = 0'
has "$casca" 'Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "chave"; onActivated: escolhaDeChave.confirmar() }'
has "$casca" 'Shortcut { sequences: ["Up"]; enabled: app.folha === "chave"; onActivated: escolhaDeChave.andar(-1) }'
has "$casca" 'Shortcut { sequences: ["Down"]; enabled: app.folha === "chave"; onActivated: escolhaDeChave.andar(1) }'

t "o Delete chega mesmo com o cursor na busca (campo de texto engole a tecla): vazia, pede para remover"
has "$(<"$qml/Campo.qml")" 'signal apagarVazio()'
has "$(<"$qml/Campo.qml")" 'Keys.onDeletePressed: event => {'
has "$(<"$qml/Lista.qml")" 'signal remover()'
has "$(<"$qml/Lista.qml")" 'onApagarVazio: lista.remover()'
has "$casca" 'onRemover: app.abrirFolha("remover")'
