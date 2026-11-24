#!/usr/bin/env bash
# apps/mkutil/src: o motor do MkUtil (MkInstalar e MkComprimir), em Go. Os
# testes dele são `go test` (com homes falsas, arquivos gerados no próprio
# teste e comandos de mentira); aqui eles entram na suíte do Mk, junto com o
# formato (gofmt), o go vet, a conferência de que o binário que vai no pacote
# é o do código que está no repositório e as regras que o código não pode
# quebrar (nunca sudo, nunca shell, nunca executar o que está inspecionando).
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

src=$REPO/apps/mkutil/src
bin=$REPO/apps/mkutil/bin/mkutil-motor
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
done < <(go list ./... 2>/dev/null | grep -v '^mkutil$' | sed 's|^mkutil/||' |
  grep -v -e '/falso$' -e '/dubles$') # esses são os dublês e os arquivos de mentira dos próprios testes
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

# O código que vai no binário (sem os testes nem o que só os testes usam).
mapfile -t fontes < <(find "$src" -name '*.go' ! -name '*_test.go' ! -path '*/falso/*' ! -path '*/dubles/*' | sort)

t "o motor nunca chama o sudo nem pede senha: o que precisa de root vai pelo pkexec"
eq "" "$(grep -nE '"(sudo|su|doas|run0)"' "${fontes[@]}" || true)"
ok grep -q 'exec.Command(pkexec, caminhoDoPacman, "-U", "--needed", "--noconfirm", "--", ficha.Arquivo)' "$src/internal/instala/instala.go"
# O pacman vai por extenso: o pkexec acharia um "pacman" solto no PATH de quem chamou.
ok grep -q 'caminhoDoPacman = "/usr/bin/pacman"' "$src/internal/instala/instala.go"

t "nenhum nome de arquivo passa por shell: não há sh -c nem bash no motor"
eq "" "$(grep -nE '"(sh|bash|/bin/sh|/bin/bash|zsh|env)"|"-c"' "${fontes[@]}" || true)"

t "o motor nunca executa o arquivo que inspeciona (nada de --appimage-extract)"
# Todo comando externo é um nome fixo, achado no PATH (ou o pacman por
# extenso): nenhuma chamada usa o caminho do arquivo da pessoa como programa.
eq "" "$(grep -n 'appimage-extract\|appimage-mount' "${fontes[@]}" | grep -v '^[^:]*:[0-9]*:\s*//' || true)"
while read -r chamada; do
  like "$chamada" 'exec\.Command(Context)?\((caminho|pkexec|ffmpeg|ctx, caminho)[,)]' "programa que não é um comando fixo: $chamada"
done < <(grep -hoE 'exec\.Command(Context)?\([^)]*' "${fontes[@]}")
# E cada um desses caminhos vem de um LookPath de nome fixo.
while read -r nome; do
  like "$nome" '^(nome|"pkexec"|"ffmpeg")$' "LookPath de algo que não é nome fixo: $nome"
done < <(grep -hoE 'exec\.LookPath\([^)]*\)' "${fontes[@]}" | sed 's/exec\.LookPath(//; s/)$//')
# Os nomes que chegam a base.Roda e ao LookPath(nome): só estes.
while read -r nome; do
  like " ffprobe ffmpeg bsdtar pacman update-desktop-database zstd xz " " $nome " "comando que o contrato não lista: $nome"
done < <(grep -hoE 'base\.Roda\("[a-z0-9-]+"' "${fontes[@]}" | grep -oE '"[a-z0-9-]+"' | tr -d '"' | sort -u)

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mkutil/pack depois de mexer em src/)"
# O pack anota a assinatura do código (um resumo de todos os .go e do go.mod)
# ao lado do binário. Comparar os bytes do binário dependeria da versão do Go
# de cada máquina.
eq "$("$REPO/apps/mkutil/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "o binário responde ao contrato: sem subcomando, o uso em JSON e saída 1"
out=$("$bin" 2>/dev/null)
eq 1 $?
like "$(jq -r .erro <<<"$out")" '^uso: mkutil-motor <'
contrato=$(<"$REPO/apps/mkutil/CONTRATO.md")
for c in inspecionar instalar instalados desinstalar codificadores midia comprimir descartar; do
  has "$out" "$c" "subcomando do contrato"
  has "$contrato" "\`mkutil-motor $c" "o contrato não descreve: $c"
done

t "o binário recusa um arquivo compactado que sai da pasta, com saída 2 e sem escrever nada"
# Um zip de verdade (sem compressão, feito à mão) com uma entrada "../fora".
printf 'PK\003\004\024\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\007\000\000\000../foraPK\001\002\024\003\024\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\007\000\000\000\000\000\000\000\000\000\000\000\244\201\000\000\000\000../foraPK\005\006\000\000\000\000\001\000\001\000\065\000\000\000\045\000\000\000\000\000' >"$SBX/mau.zip"
out=$(MKUTIL_HOME=$SBX/casa "$bin" inspecionar "$SBX/mau.zip")
eq 2 $?
has "$(jq -r .erro <<<"$out")" "sai da pasta"
out=$(MKUTIL_HOME=$SBX/casa "$bin" instalar "$SBX/mau.zip")
eq 2 $?
absent "$SBX/fora"
absent "$SBX/casa/fora"
eq "" "$(ls -A "$SBX/casa/Aplicativos" 2>/dev/null)"
