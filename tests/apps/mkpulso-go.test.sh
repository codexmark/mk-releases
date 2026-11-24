#!/usr/bin/env bash
# apps/mkpulso/src: o coletor do MkPulso, em Go. Os testes dele são `go test`
# (com /proc, /sys e comandos falsos); aqui eles entram na suíte do Mk, junto
# com o formato (gofmt), o go vet e a conferência de que o binário que vai no
# pacote é o do código que está no repositório.
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

src=$REPO/apps/mkpulso/src
bin=$REPO/apps/mkpulso/bin/mkpulso-coletor
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
  compgen -G "$src/${pacote#mkpulso}"'/*_test.go' >/dev/null || compgen -G "$src/${pacote#mkpulso/}"'/*_test.go' >/dev/null ||
    _fail "sem teste: $pacote"
done < <(go list ./... 2>/dev/null | grep -v '^mkpulso$' | sed 's|^mkpulso/||' |
  grep -v -e '/falso$' -e '/dubles$') # esses dois são os dublês dos próprios testes
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mkpulso/pack depois de mexer em src/)"
# O pack anota a assinatura do código (um resumo de todos os .go e do go.mod)
# ao lado do binário. Comparar os bytes do binário dependeria da versão do Go
# de cada máquina.
eq "$("$REPO/apps/mkpulso/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "o binário responde ao contrato: sem subcomando, o uso em JSON e saída 1"
out=$("$bin" 2>/dev/null)
eq 1 $?
like "$(jq -r .erro <<<"$out")" '^uso: mkpulso-coletor <'
for c in watch sistema sinal grupo prioridade servicos servico registro inicio; do
  has "$out" "$c" "subcomando do contrato"
done
