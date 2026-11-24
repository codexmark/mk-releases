#!/usr/bin/env bash
# apps/mkcalc/src: o motor do MkCalc (a conta exata e o histórico), em Go. Os
# testes dele são `go test` (com o histórico em pasta temporária); aqui eles
# entram na suíte do Mk, junto com o formato (gofmt), o go vet, a conferência
# de que o binário que vai no pacote é o do código que está no repositório e
# o binário de verdade respondendo ao contrato.
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

src=$REPO/apps/mkcalc/src
bin=$REPO/apps/mkcalc/bin/mkcalc-motor
contrato=$(<"$REPO/apps/mkcalc/CONTRATO.md")
# O cache de compilação fica fora da caixa de areia, para não recompilar tudo
# a cada rodada; o resto (HOME, módulos) continua dentro dela.
export GOCACHE=${MK_GOCACHE:-/var/tmp/mk-gocache-$(id -u)}
# -buildvcs=false: sem isso o go chama o git (que aqui é um guarda) para carimbar o binário.
export GOFLAGS="-mod=mod -buildvcs=false" GOPROXY=off GOTOOLCHAIN=local
cd "$src" || exit 1

t "o módulo não tem dependência de fora: só a biblioteca padrão"
eq "" "$(grep -E '^\s*require|^\s+[a-z0-9.-]+\.[a-z]+/' go.mod || true)"
absent "$src/go.sum"

t "gofmt: todo arquivo está formatado"
eq "" "$(gofmt -l . 2>&1)"

t "go vet: nenhum aviso"
out=$(go vet ./... 2>&1) || _fail "$out"

t "go test: a suíte do motor passa"
out=$(go test ./... 2>&1) || _fail "$(tail -40 <<<"$out")"
hasnt "$out" "FAIL"

t "todo pacote do motor tem teste"
while read -r pacote; do
  compgen -G "$src/$pacote"'/*_test.go' >/dev/null || _fail "sem teste: $pacote"
done < <(go list ./... 2>/dev/null | grep -v '^mkcalc$' | sed 's|^mkcalc/||')
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

# O código que vai no binário (sem os testes).
mapfile -t fontes < <(find "$src" -name '*.go' ! -name '*_test.go' | sort)

t "a conta é exata: math/big, e nenhum float64 decide um resultado"
ok grep -q '"math/big"' "$src/internal/conta/conta.go"
# O único float do motor é o chute inicial da raiz (o Newton confere depois).
eq "valor.go" "$(grep -lE 'float64|math\.(Pow|Sqrt|Log)' "${fontes[@]}" | xargs -r -n1 basename | sort -u | tr '\n' ' ' | sed 's/ $//')"
eq "" "$(grep -nE 'strconv\.ParseFloat|big\.NewFloat|\.Float64\(\)' "$src/internal/conta/conta.go" "$src/internal/conta/numero.go" "$src/internal/conta/formato.go" || true)"

t "o motor não roda nenhum programa nem usa a rede"
eq "" "$(grep -nE '"os/exec"|"net"|"net/http"|"syscall"' "${fontes[@]}" || true)"

t "o histórico é o único arquivo que o motor escreve, e a escrita é atômica"
eq "historico.go" "$(grep -lE 'os\.(Create|WriteFile|OpenFile|Rename|MkdirAll|Remove)' "${fontes[@]}" | xargs -r -n1 basename | tr '\n' ' ' | sed 's/ $//')"
ok grep -q 'os.CreateTemp(pasta' "$src/internal/historico/historico.go"
ok grep -q 'os.Rename(tmp.Name(), caminho)' "$src/internal/historico/historico.go"

t "os números do contrato são os do código"
ok grep -qE 'CasasExatas += 40$' "$src/internal/conta/conta.go"
ok grep -qE 'CasasAproximadas += 12$' "$src/internal/conta/conta.go"
ok grep -qE 'LimiteDeBits += 10240$' "$src/internal/conta/conta.go"
ok grep -qE 'LimiteDoTexto += 5000$' "$src/internal/conta/conta.go"
ok grep -qE 'LimiteDaRaiz += 1000$' "$src/internal/conta/conta.go"
ok grep -qE 'LimitePadrao = 200$' "$src/internal/historico/historico.go"
has "$contrato" "até **40 casas**"
has "$contrato" "arredondado para 12 casas decimais"
has "$contrato" "10.240 bits"
has "$contrato" "até 5.000 caracteres no motor e 3.500 na janela"
ok grep -q "^var LIMITE = 3500;\$" "$REPO/apps/mkcalc/qml/Calc.js"
has "$contrato" "denominador acima de 1.000"
has "$contrato" "últimas 200 contas"

t "toda frase de erro de conta do motor está na tabela do contrato"
n=0
while read -r frase; do
  n=$((n + 1))
  has "$contrato" "| \`${frase%%:*}" "frase fora do contrato: $frase"
done < <(grep -ohE '(erro|incompleta)\("[^"]+' "$src"/internal/conta/*.go | grep -v _test | sed -E 's/^(erro|incompleta)\("//; s/ «$//; s/: «$//' | sort -u)
ok test "$n" -ge 12

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mkcalc/pack depois de mexer em src/)"
# O pack anota a assinatura do código (um resumo de todos os .go e do go.mod)
# ao lado do binário. Comparar os bytes do binário dependeria da versão do Go
# de cada máquina.
eq "$("$REPO/apps/mkcalc/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "o binário responde ao contrato: sem subcomando, o uso em JSON e saída 1"
out=$("$bin" 2>/dev/null)
eq 1 $?
like "$(jq -r .erro <<<"$out")" '^uso: mkcalc-motor <'
for c in calcular historico apagar limpar servir; do
  has "$out" "$c" "subcomando do contrato"
  has "$contrato" "\`mkcalc-motor $c" "o contrato não descreve: $c"
done

t "calcular: 0,1 + 0,2 é 0,3, e a resposta tem os quatro campos do contrato"
out=$("$bin" calcular "0,1 + 0,2")
eq 0 $?
jeq '{"expressao":"0,1 + 0,2","resultado":"0,3","cru":"0,3","aproximado":false}' "$out"

t "calcular: os exemplos do contrato dão o que o contrato diz"
calc() { "$bin" calcular "$1" | jq -r '.resultado // .erro'; }
eq "220" "$(calc "200 + 10%")"
eq "180" "$(calc "200 - 10%")"
eq "165" "$(calc "100 + 50 + 10%")"
eq "220" "$(calc "200 + (5 + 5)%")"
eq "20" "$(calc "200 × 10%")"
eq "0,5" "$(calc "50%")"
eq "200,2" "$(calc "200 + 10% × 2")"
eq "200,1" "$(calc "200 + (10%)")"
eq "1" "$(calc "1 ÷ 3 × 3")"
eq "512" "$(calc "2^3^2")"
eq "-4" "$(calc "-2^2")"
eq "0,25" "$(calc "2^-2")"
eq "14" "$(calc "2(3 + 4)")"
eq "6" "$(calc "2√9")"
eq "9" "$(calc "raiz(81)")"
eq "2" "$(calc "8^(1 ÷ 3)")"
eq "0,666666666667" "$(calc "2 ÷ 3")"
eq "0,000333333333333" "$(calc "1 ÷ 3000")"
eq "2" "$(calc "√2 × √2")"
eq "2" "$(calc "√(4 ÷ 9) × 3")"
eq "1.234" "$(calc "1.234")"
eq "1,234" "$(calc "1,234")"
eq "1,5" "$(calc "1.5")"
eq "0,123" "$(calc "0.123")"
eq "1.234.567" "$(calc "1.234.567")"
eq "1.234,56" "$(calc "1.234,56")"
eq "1.234,56" "$(calc "1,234.56")"
eq "número mal formado: «1.23.456»" "$(calc "1.23.456")"
eq "1.234.567,891234" "$(calc "1234567,891234")"
eq "1234567,891234" "$("$bin" calcular "1234567,891234" | jq -r .cru)"
eq "true" "$("$bin" calcular "2 ÷ 3" | jq -r .aproximado)"
eq "true" "$("$bin" calcular "√2 × √2" | jq -r .aproximado)"
eq "false" "$("$bin" calcular "√16" | jq -r .aproximado)"

t "calcular: número grande não perde algarismo"
eq "1.219.326.311.370.217.952.237.463.801.111.263.526.900" "$(calc "12345678901234567890 × 98765432109876543210")"

t "calcular: a conta recusada sai com 2, a frase e se está incompleta"
recusa() {
  local out rc
  out=$("$bin" calcular "$1")
  rc=$?
  eq 2 "$rc" "saída de: $1"
  jeq "{\"erro\":\"$2\",\"incompleta\":$3}" "$out"
}
recusa "1 ÷ 0" "divisão por zero" false
recusa "(3 × 4" "parêntese sem par" true
recusa "3 × 4)" "parêntese sem par" false
recusa "2 +" "expressão incompleta" true
recusa "" "expressão vazia" true
recusa "√-4" "raiz de número negativo" false
recusa "2 + × 3" "operador fora de lugar: «×»" false
recusa "2 3" "falta um operador antes de «3»" false
recusa "abc" "não entendi «abc»" false
recusa "2^0,0001" "expoente com casas demais" false

t "calcular: o maior resultado que o motor dá cabe de volta na expressão, para continuar a conta"
limite=$(sed -nE 's/^var LIMITE = ([0-9]+);$/\1/p' "$REPO/apps/mkcalc/qml/Calc.js")
for conta in "2^10239" "0 - 2^10239" "1 ÷ 2^10239" "(2^10239 - 1) ÷ 2^40" "5^4409"; do
  cru=$("$bin" calcular "$conta" | jq -r .cru)
  ok test "${#cru}" -gt 1000
  ok test $((${#cru} + 100)) -lt "$limite"
  eq "$cru" "$("$bin" calcular "$cru + 0" | jq -r .cru)" "continuar de: $conta"
done

t "calcular: 9^9^9 é recusado na hora, sem travar"
antes=$(date +%s%N)
recusa "9^9^9" "expoente grande demais" false
recusa "9^9^9^9^9^9" "expoente grande demais" false
ok test $((($(date +%s%N) - antes) / 1000000)) -lt 2000

t "historico: --guardar grava em XDG_STATE_HOME/mk/mkcalc/historico.json, e só ele"
arq=$XDG_STATE_HOME/mk/mkcalc/historico.json
ok "$bin" calcular "1 + 1" >/dev/null
absent "$arq"
out=$("$bin" calcular --guardar "200 + 10%")
eq 0 $?
exists "$arq"
ok jq -e '.versao == 1 and (.itens | length == 1) and (.itens[0] | .id == 1 and .expressao == "200 + 10%" and .resultado == "220"
  and .cru == "220" and .aproximado == false and (.quando_s | type == "number"))' "$arq" >/dev/null
jeq "$(jq -c .itens "$arq")" "$(jq -c .itens <<<"$out")"
eq "historico.json" "$(ls -A "$XDG_STATE_HOME/mk/mkcalc")"

t "historico, apagar e limpar respondem o histórico como ficou"
"$bin" calcular --guardar "2 ÷ 3" >/dev/null
eq "2 1" "$("$bin" historico | jq -r '[.itens[].id] | join(" ")')"
eq "2" "$("$bin" apagar 1 | jq -r '[.itens[].id] | join(" ")')"
out=$("$bin" apagar 1)
eq 1 $?
jeq '{"erro":"essa conta não está mais no histórico"}' "$out"
jeq '{"itens":[]}' "$("$bin" limpar)"
jeq '{"itens":[]}' "$("$bin" historico)"

t "historico: arquivo corrompido não derruba, começa vazio e a próxima conta o regrava"
printf '{"versao": 1, "itens": [{"id": 1, "expr' >"$arq"
jeq '{"itens":[]}' "$("$bin" historico)"
eq "1" "$("$bin" calcular --guardar "3 × 3" | jq -r '.itens | length')"
ok jq -e '.itens[0].cru == "9"' "$arq" >/dev/null

t "historico: o limite de itens tira a mais velha"
"$bin" limpar >/dev/null
for i in 1 2 3 4 5; do MKCALC_HISTORICO_LIMITE=3 "$bin" calcular --guardar "$i + 0" >/dev/null; done
eq "5 4 3" "$("$bin" historico | jq -r '[.itens[].cru] | join(" ")')"

t "servir: uma linha JSON por pedido, com o id de cada um, e continua de pé depois de um erro"
"$bin" limpar >/dev/null
out=$(printf '%s\n' '{"id":1,"op":"calcular","expr":"0,1+0,2"}' 'lixo' '{"id":"b","op":"calcular","expr":"1÷0"}' \
  '{"id":3,"op":"calcular","expr":"6x7","guardar":true}' '{"id":4,"op":"historico"}' '{"id":5,"op":"apagar","item":1}' \
  '{"id":6,"op":"limpar"}' '{"id":7,"op":"dançar"}' | "$bin" servir)
eq 0 $?
eq 8 "$(wc -l <<<"$out")"
eq '1 null "b" 3 4 5 6 7' "$(jq -c .id <<<"$out" | tr '\n' ' ' | sed 's/ $//')"
linha() { sed -n "$1p" <<<"$out"; }
eq "0,3" "$(linha 1 | jq -r .resultado)"
eq "pedido que não é JSON" "$(linha 2 | jq -r .erro)"
eq "divisão por zero false" "$(linha 3 | jq -r '"\(.erro) \(.incompleta)"')"
eq "42 1" "$(linha 4 | jq -r '"\(.cru) \(.itens | length)"')"
eq "6 × 7" "$(linha 5 | jq -r '.itens[0].expressao')"
eq "0" "$(linha 6 | jq -r '.itens | length')"
eq "0" "$(linha 7 | jq -r '.itens | length')"
eq "não conheço o pedido «dançar»" "$(linha 8 | jq -r .erro)"

t "servir: responde cada pedido na hora, sem esperar o stdin fechar"
# A janela deixa o motor de pé: a resposta tem de sair a cada linha.
coproc MOTOR { "$bin" servir; }
echo '{"id":1,"op":"calcular","expr":"2+2"}' >&"${MOTOR[1]}"
resposta=""
read -r -t 5 resposta <&"${MOTOR[0]}"
eq "4" "$(jq -r .cru <<<"$resposta")"
echo '{"id":2,"op":"calcular","expr":"2+"}' >&"${MOTOR[1]}"
read -r -t 5 resposta <&"${MOTOR[0]}"
eq "2 true" "$(jq -r '"\(.id) \(.incompleta)"' <<<"$resposta")"
entrada=${MOTOR[1]}
exec {entrada}>&-
wait "$MOTOR_PID"
eq 0 $?
