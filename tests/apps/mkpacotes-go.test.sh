#!/usr/bin/env bash
# apps/mkpacotes/src: o coletor do MkPacotes, em Go. Os testes dele são `go test`
# (com bases de pacman, comandos e um AUR de mentira); aqui eles entram na suíte do Mk, junto
# com o formato (gofmt), o go vet e a conferência de que o binário que vai no
# pacote é o do código que está no repositório.
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

src=$REPO/apps/mkpacotes/src
bin=$REPO/apps/mkpacotes/bin/mkpacotes-coletor
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

t "go test: a suíte do coletor passa"
out=$(go test ./... 2>&1) || _fail "$(tail -40 <<<"$out")"
hasnt "$out" "FAIL"

t "todo pacote do coletor tem teste"
while read -r pacote; do
  compgen -G "$src/${pacote#mkpacotes}"'/*_test.go' >/dev/null || compgen -G "$src/${pacote#mkpacotes/}"'/*_test.go' >/dev/null ||
    _fail "sem teste: $pacote"
done < <(go list ./... 2>/dev/null | grep -v '^mkpacotes$' | sed 's|^mkpacotes/||' |
  grep -v -e '/falso$' -e '/dubles$') # esses dois são os dublês dos próprios testes
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mkpacotes/pack depois de mexer em src/)"
# O pack anota a assinatura do código (um resumo de todos os .go e do go.mod)
# ao lado do binário. Comparar os bytes do binário dependeria da versão do Go
# de cada máquina.
eq "$("$REPO/apps/mkpacotes/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "o binário responde ao contrato: sem subcomando, o uso em JSON e saída 1"
out=$("$bin" 2>/dev/null)
eq 1 $?
like "$(jq -r .erro <<<"$out")" '^uso: mkpacotes-coletor <'
for c in instalados procurar detalhe atualizacoes historico estado plano agir; do
  has "$out" "$c" "subcomando do contrato"
done

t "todo subcomando do contrato tem o seu teste de contrato em Go"
for c in Instalados Procurar Detalhe Atualizacoes Historico Estado Plano Agir; do
  grep -qE "^func Test${c}_Contrato" "$src"/*_test.go || _fail "sem teste de contrato: $c"
done

t "a suíte do coletor nunca chama o comando de verdade: todo teste que roda comando usa a mesa de dublês"
# dubles.Nova faz do PATH do teste uma pasta só com os dublês. Um teste que
# chega a rodar(…), Planeja ou Age sem ela usaria o pacman da máquina.
for f in "$src"/*_test.go "$src"/internal/acoes/*_test.go "$src"/internal/pacman/*_test.go; do
  awk '/^func Test/ { nome = $2; mesa = 0 }
       /maquina\(t\)|dubles\.Nova\(t\)/ { mesa = 1 }
       /(roda|quer|querErro)\(t, |Planeja\(|Age\(|Repos\(\)|Busca\(\[|EnsaioInstalar\(|EnsaioRemover\(|Atualizacoes\(\)|Roda\("/ {
         if (nome != "" && !mesa) print FILENAME ": " nome }' "$f"
done | sort -u >"$SBX/sem-mesa"
eq "" "$(grep -v 'TestSemSubcomandoOuComUmDesconhecido_UsoEmJSON' "$SBX/sem-mesa" || true)"

t "nenhum arquivo do coletor chama sudo, e o pkexec e o yay só aparecem nas ações"
# ("sudo" como nome de pacote protegido pode; como comando, nunca.)
eq "" "$(grep -rlE --include='*.go' --exclude='*_test.go' '\{"sudo"|(Command|LookPath|Roda)\("sudo"' "$src" || true)"
eq "internal/acoes/plano.go" "$(cd "$src" && grep -rlE --include='*.go' --exclude='*_test.go' '\{"pkexec", |\{"yay", ' . | sed 's|^\./||' | sort | tr '\n' ' ' | sed 's/ $//')"

t "a suíte Go do coletor é rápida (poucos segundos)"
inicio=$(date +%s)
out=$(go test -count=1 ./... 2>&1) || _fail "$(tail -20 <<<"$out")"
ok test $(($(date +%s) - inicio)) -le 30
