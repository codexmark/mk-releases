#!/usr/bin/env bash
# apps/mkescrita/src: o motor do MkEscrita, em Go. Os testes dele são `go test`
# (com pastas temporárias; nenhum toca na home de quem roda); aqui eles entram
# na suíte do Mk, junto com o formato (gofmt), o go vet, a conferência de que o
# binário que vai no pacote é o do código que está no repositório e as regras
# que o código não pode quebrar (nenhum comando externo, nenhum shell) e o
# contrato de ponta a ponta com o binário: não perder texto.
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

src=$REPO/apps/mkescrita/src
bin=$REPO/apps/mkescrita/bin/mkescrita-motor
# O cache de compilação fica fora da caixa de areia, para não recompilar tudo
# a cada rodada; o resto (HOME, módulos) continua dentro dela.
export GOCACHE=${MK_GOCACHE:-/var/tmp/mk-gocache-$(id -u)}
# -buildvcs=false: sem isso o go chama o git (que aqui é um guarda) para carimbar o binário.
export GOFLAGS="-mod=mod -buildvcs=false" GOPROXY=off GOTOOLCHAIN=local
cd "$src" || exit 1

t "o módulo não tem dependência de fora: só a biblioteca padrão"
eq "" "$(grep -E '^\s*require|^\s+[a-z0-9.-]+\.[a-z]+/' go.mod || true)"
absent "$src/go.sum"
eq "" "$(grep -rhE '^\s*"[a-z0-9.-]+\.[a-z]+/' --include='*.go' . || true)" "import de fora da biblioteca padrão"

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
done < <(go list ./... 2>/dev/null | grep -v '^mkescrita$' | sed 's|^mkescrita/||')
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

# O código que vai no binário (sem os testes).
mapfile -t fontes < <(find "$src" -name '*.go' ! -name '*_test.go' | sort)

t "o motor não roda comando nenhum: sem os/exec, sem shell, sem rede"
eq "" "$(grep -nE '"os/exec"|"net"|"net/http"|"plugin"|"unsafe"' "${fontes[@]}" || true)"
eq "" "$(grep -nE '"(sh|bash|/bin/sh|/bin/bash|sudo|pkexec)"' "${fontes[@]}" || true)"

t "o conversor de Markdown é próprio: nenhum html/template nem biblioteca de fora"
eq "" "$(grep -nE '^\s*"(html/template|text/template|html)"$' "${fontes[@]}" || true)"

t "gravar é sempre atômico: nenhum os.WriteFile nem os.Create no código do motor"
eq "" "$(grep -nE 'os\.(WriteFile|Create|OpenFile)\(' "${fontes[@]}" || true)"
ok grep -q 'os.CreateTemp(pasta' "$src/internal/disco/arquivo.go"
ok grep -q 'tmp.Sync()' "$src/internal/disco/arquivo.go"
ok grep -q 'os.Rename(nome, destino)' "$src/internal/disco/arquivo.go"

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mkescrita/pack depois de mexer em src/)"
eq "$("$REPO/apps/mkescrita/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "o binário responde ao contrato: sem subcomando, o uso em JSON e saída 1"
out=$("$bin" 2>/dev/null)
eq 1 $?
like "$(jq -r .erro <<<"$out")" '^uso: mkescrita-motor <'
contrato=$(<"$REPO/apps/mkescrita/CONTRATO.md")
for c in abrir salvar rascunho vigiar html contar estrutura exportar preferencias; do
  has "$out" "$c" "subcomando do contrato"
  has "$contrato" "\`mkescrita-motor $c" "o contrato não descreve: $c"
done

# --- de ponta a ponta, com o binário ------------------------------------------
export MKESCRITA_PROC=$SBX/proc
mkdir -p "$SBX/proc" "$SBX/textos"
arq="$SBX/textos/-um nome 'difícil' \"x\".md"

t "abrir: o texto e a impressão (tamanho, mtime, sha256)"
printf '# Olá\r\nação\r\n' >"$arq"
out=$("$bin" abrir -- "$arq")
eq 0 $?
eq $'# Olá\nação\n' "$(jq -r .texto <<<"$out")"$'\n'
eq "crlf true false" "$(jq -r '"\(.fim_de_linha) \(.quebra_final) \(.novo)"' <<<"$out")"
sha=$(jq -r .impressao.sha256 <<<"$out")
eq "$(sha256sum <"$arq" | cut -d' ' -f1)" "$sha"
eq "$(stat -c %s "$arq")" "$(jq -r .impressao.tamanho_bytes <<<"$out")"
ok jq -e '.impressao.mtime_ns > 0 and .arquivo != "" and .nome != "" and (.so_leitura == false)' <<<"$out" >/dev/null

t "abrir: recusa binário e arquivo grande demais com saída 2 e frase clara"
printf 'GIF89a\000\001\002' >"$SBX/textos/foto.gif"
out=$("$bin" abrir "$SBX/textos/foto.gif")
eq 2 $?
has "$(jq -r .erro <<<"$out")" "binário"
head -c 1100000 /dev/zero | tr '\0' 'a' >"$SBX/textos/grande.md"
out=$(MKESCRITA_LIMITE_MB=1 "$bin" abrir "$SBX/textos/grande.md")
eq 2 $?
has "$(jq -r .erro <<<"$out")" "grande demais"
has "$contrato" "MKESCRITA_LIMITE_MB"

t "salvar: grava o que veio pelo stdin, mantém as permissões e o CRLF, sem deixar temporário"
chmod 640 "$arq"
out=$(printf '# Olá\nmudei\n' | "$bin" salvar --esperado "$sha" --fim-de-linha crlf --quebra-final --bytes 13 -- "$arq")
eq 0 $?
printf '# Olá\r\nmudei\r\n' >"$SBX/esperado"
ok cmp -s "$SBX/esperado" "$arq"
eq 640 "$(stat -c %a "$arq")"
sha2=$(jq -r .impressao.sha256 <<<"$out")
eq "$(sha256sum <"$arq" | cut -d' ' -f1)" "$sha2"
eq "" "$(find "$SBX/textos" -name '*.mkescrita-*')"

t "salvar: se o arquivo mudou por fora, não grava: responde conflito (saída 2) e a janela decide"
printf 'mexeram por fora\n' >"$arq"
out=$(printf 'o meu texto\n' | "$bin" salvar --esperado "$sha2" -- "$arq")
eq 2 $?
eq "mudou null" "$(jq -r '"\(.conflito) \(.erro)"' <<<"$out")"
eq "$(sha256sum <"$arq" | cut -d' ' -f1)" "$(jq -r .impressao.sha256 <<<"$out")"
eq "mexeram por fora" "$(<"$arq")"

t "salvar --forcar grava assim mesmo"
out=$(printf 'o meu texto\n' | "$bin" salvar --esperado "$sha2" --forcar -- "$arq")
eq 0 $?
eq "o meu texto" "$(<"$arq")"

t "salvar: texto que chegou cortado (--bytes não bate) não troca o arquivo"
out=$(printf 'metad' | "$bin" salvar --forcar --bytes 5000 -- "$arq")
eq 1 $?
has "$(jq -r .erro <<<"$out")" "cortado"
eq "o meu texto" "$(<"$arq")"

t "salvar: segue o link simbólico em vez de trocá-lo por um arquivo"
ln -s "$arq" "$SBX/textos/atalho.md"
out=$(printf 'pelo link\n' | "$bin" salvar --forcar -- "$SBX/textos/atalho.md")
eq 0 $?
ok test -L "$SBX/textos/atalho.md"
eq "pelo link" "$(<"$arq")"

t "rascunhos: moram em \$XDG_STATE_HOME/mk/mkescrita/rascunhos e sobrevivem ao app"
out=$(printf 'ainda não salvei\n' | "$bin" rascunho guardar r-teste-1 --origem "$arq" --base abc)
eq 0 $?
exists "$XDG_STATE_HOME/mk/mkescrita/rascunhos/r-teste-1.json"
eq 600 "$(stat -c %a "$XDG_STATE_HOME/mk/mkescrita/rascunhos/r-teste-1.json")"
out=$("$bin" rascunho listar)
eq "r-teste-1 $arq false" "$(jq -r '.itens[0] | "\(.id) \(.origem) \(.vivo)"' <<<"$out")"
out=$("$bin" rascunho ler r-teste-1)
eq "ainda não salvei abc" "$(jq -r '"\(.texto | rtrimstr("\n")) \(.base_sha256)"' <<<"$out")"
# Guardar um rascunho nunca toca no arquivo de origem.
eq "pelo link" "$(<"$arq")"
ok "$bin" rascunho apagar r-teste-1 >/dev/null
absent "$XDG_STATE_HOME/mk/mkescrita/rascunhos/r-teste-1.json"
out=$("$bin" rascunho guardar ../../fora </dev/null)
eq 1 $?
absent "$XDG_STATE_HOME/mk/fora.json"

t "vigiar: uma linha JSON quando o arquivo muda por fora, e outra quando some"
printf 'um\n' >"$SBX/textos/vigiado.md"
timeout 20 "$bin" vigiar --intervalo-ms 20 -- "$SBX/textos/vigiado.md" >"$SBX/vigia.out" 2>&1 &
vigia=$!
sleep 0.3
eq "" "$(<"$SBX/vigia.out")" "sem mudança, silêncio"
printf 'dois dois\n' >"$SBX/textos/vigiado.md"
for _ in $(seq 100); do [[ -s $SBX/vigia.out ]] && break; sleep 0.05; done
eq "mudou $(sha256sum <"$SBX/textos/vigiado.md" | cut -d' ' -f1)" "$(head -1 "$SBX/vigia.out" | jq -r '"\(.evento) \(.impressao.sha256)"')"
rm -f "$SBX/textos/vigiado.md"
for _ in $(seq 100); do (($(wc -l <"$SBX/vigia.out") >= 2)) && break; sleep 0.05; done
eq "sumiu null" "$(sed -n 2p "$SBX/vigia.out" | jq -r '"\(.evento) \(.impressao)"')"
kill "$vigia" 2>/dev/null
wait "$vigia" 2>/dev/null

t "html: o que vem do texto é escapado (<script> e javascript: em link)"
# shellcheck disable=SC2016 # é Markdown, não expansão
out=$(printf '# T\n\n<script>alert(1)</script> [x](javascript:alert(1)) **n** `c`\n\n- [ ] a\n\n```sh\nls\n```\n' | "$bin" html)
eq 0 $?
html=$(jq -r .html <<<"$out")
has "$html" "&lt;script&gt;alert(1)&lt;/script&gt;"
hasnt "$html" "<script"
hasnt "$html" "javascript:"
has "$html" "<h1>T</h1>"
has "$html" "<strong>n</strong>"
has "$html" "<code>c</code>"
has "$html" '<li class="tarefa">☐ a</li>'
has "$html" '<pre><code class="language-sh">ls'
out=$(printf '# Meu texto\n' | "$bin" html --pagina)
has "$(jq -r .html <<<"$out")" "<!doctype html>"
has "$(jq -r .html <<<"$out")" "<title>Meu texto</title>"
has "$(jq -r .html <<<"$out")" "<style>"

t "contar e estrutura: palavras, tempo de leitura e os títulos com nível e linha"
out=$(printf '# Um\n\nduas palavras\n\n## Dois\n' | "$bin" contar)
eq "4 2" "$(jq -r '"\(.palavras) \(.leitura_s)"' <<<"$out")"
out=$(printf '# Um\n\nduas palavras\n\n## Dois\n' | "$bin" estrutura)
jeq '{"titulos":[{"nivel":1,"texto":"Um","linha":1},{"nivel":2,"texto":"Dois","linha":5}]}' "$out"

t "exportar html: nunca sobrescreve sem --forcar"
out=$(printf '# Um\n' | "$bin" exportar html "$SBX/textos/saida.html")
eq 0 $?
has "$(<"$SBX/textos/saida.html")" "<h1>Um</h1>"
antes=$(sha256sum <"$SBX/textos/saida.html")
out=$(printf '# Outro\n' | "$bin" exportar html "$SBX/textos/saida.html")
eq 2 $?
has "$(jq -r .erro <<<"$out")" "já existe"
eq "$antes" "$(sha256sum <"$SBX/textos/saida.html")"
ok "$bin" exportar html --forcar "$SBX/textos/saida.html" <<<"# Outro" >/dev/null
has "$(<"$SBX/textos/saida.html")" "<h1>Outro</h1>"

t "preferencias: lidas e gravadas em \$XDG_CONFIG_HOME/mk/mkescrita.json, com o zoom entre 50 e 300"
jeq '{"fonte":"","zoom":100,"previa":false}' "$("$bin" preferencias ler)"
out=$(printf '{"fonte":"Fira Code","zoom":1000}' | "$bin" preferencias gravar)
jeq '{"fonte":"Fira Code","zoom":300,"previa":false}' "$out"
jeq '{"fonte":"Fira Code","zoom":300,"previa":false}' "$(<"$XDG_CONFIG_HOME/mk/mkescrita.json")"
jeq '{"fonte":"Fira Code","zoom":300,"previa":false}' "$("$bin" preferencias ler)"

t "nada foi escrito fora da caixa de areia nem na pasta do repositório"
eq "" "$(find "$REPO/apps/mkescrita" -name '*.mkescrita-*' -o -name 'rascunhos' | head -3)"
