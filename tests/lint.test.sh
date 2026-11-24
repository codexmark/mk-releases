#!/usr/bin/env bash
# Sintaxe e lint do repositório inteiro: nada que não passa aqui é lançado.
# apps/mktunes (gerado no repositório do MkTunes) e as pastas de trabalho da
# ISO ficam de fora.
. "$(dirname "$0")/lib.sh"

cd "$REPO" || exit 1
mapfile -t all < <(find . -type f \
  -not -path './.git/*' -not -path './apps/mktunes/*' -not -path './iso/work/*' -not -path './iso/out/*' |
  sed 's|^\./||' | sort)

# shells: os que têm #! de bash; libs: os que são carregados com `.` (sem #!).
shells=() libs=() pys=()
for f in "${all[@]}"; do
  case $(head -c 64 "$f" 2>/dev/null | tr -d '\0' | head -n1) in
    '#!'*bash* | '#!'*/sh*) shells+=("$f") ;;
    '#!'*python*) pys+=("$f") ;;
    *) case $f in
      *.sh | *.bash) libs+=("$f") ;;
      *.py) pys+=("$f") ;;
    esac ;;
  esac
done

t "há scripts para conferir (a busca não ficou vazia)"
ok test "${#shells[@]}" -gt 50

t "bash -n: todo script tem sintaxe válida"
for f in "${shells[@]}" "${libs[@]}"; do
  out=$(bash -n "$f" 2>&1) || _fail "$f" "$out"
done

t "shellcheck: nenhum aviso (nível warning)"
if command -v shellcheck >/dev/null; then
  out=$(shellcheck -S warning -f gcc "${shells[@]}" 2>&1)
  eq "" "$out"
  # Um arquivo carregado com `.` define o que outro usa e recebe argumentos
  # de quem o chama: "não usado" e "nunca passam argumento" não valem nele.
  out=$(shellcheck -s bash -e SC2034,SC2120 -S warning -f gcc "${libs[@]}" 2>&1)
  eq "" "$out"
else
  echo "  # shellcheck ausente: pulado" >&2
fi

t "todo executável de bin/ e scripts do shell tem o bit de execução"
for f in bin/* shell/mare/scripts/* tests/run tests/hooks/* apps/mkpulso/bin/mkpulso apps/mkpacotes/bin/mkpacotes apps/mkutil/bin/mkinstalar \
  apps/mkutil/bin/mkcomprimir apps/mkcalc/bin/mkcalc apps/mkcorte/bin/mkcorte apps/mkescrita/bin/mkescrita apps/mk*/pack apps/mk*/assinatura; do
  [[ -f $f && $f != bin/lib-packages.sh ]] || continue
  [[ -x $f ]] || _fail "sem +x: $f"
done

t "QML: nenhum erro de sintaxe"
if command -v qmllint >/dev/null; then
  for f in "${all[@]}"; do
    [[ $f == *.qml ]] || continue
    out=$(qmllint "$f" 2>&1 | grep -iE 'syntax error|failed to parse|Expected token') && _fail "$f" "$out"
  done
else
  echo "  # qmllint ausente: pulado" >&2
fi

t "JavaScript do shell: sintaxe válida"
if command -v node >/dev/null; then
  for f in shell/mare/*.js apps/mk*/qml/*.js; do
    out=$(grep -vE '^\.(pragma|import) ' "$f" | node --check - 2>&1) || _fail "$f" "$out"
  done
else
  echo "  # node ausente: pulado" >&2
fi

t "Lua: toda configuração compila"
if command -v luac >/dev/null; then
  for f in "${all[@]}"; do
    [[ $f == *.lua ]] || continue
    out=$(luac -p "$f" 2>&1) || _fail "$f" "$out"
  done
else
  echo "  # luac ausente: pulado" >&2
fi

t "JSON: todo arquivo é válido"
for f in "${all[@]}"; do
  [[ $f == *.json ]] || continue
  out=$(jq empty "$f" 2>&1) || _fail "$f" "$out"
done

t "Python: todo script compila"
for f in "${pys[@]}"; do
  out=$(python3 -c 'import ast,sys; ast.parse(open(sys.argv[1]).read(), sys.argv[1])' "$f" 2>&1) || _fail "$f" "$out"
done

t "nenhum arquivo com fim de linha do Windows nem conflito de merge esquecido"
for f in "${shells[@]}"; do
  grep -q $'\r$' "$f" && _fail "CRLF: $f"
done
out=$(grep -rlE '^(<<<<<<<|>>>>>>>) ' --include='*.qml' --include='*.js' --include='*.lua' --include='*.md' \
  --exclude-dir=.git --exclude-dir=apps --exclude-dir=work --exclude-dir=out . 2>/dev/null)
eq "" "$out"

t "rm -rf com variável sempre protegido por \${VAR:?} ou caminho fixo"
# Um rm -rf "$x/..." com x vazio apaga a raiz do caminho. Aceitos: "${x:?}",
# variável vinda de mktemp na mesma função fica por conta da revisão; aqui só
# o caso clássico: rm -rf "$var"/ ou "$var/"* sem :?.
out=$(grep -nE 'rm -r?f?r? .*"\$[A-Za-z_]+"?/(\*|"|$)' "${shells[@]}" | grep -v ':?' | grep -v '^tests/' || true)
eq "" "$out"

t "nenhum teste monta um PATH com /usr/bin ou /bin: os guardas sairiam do caminho"
# Já aconteceu: um caso "sem o yay" com PATH=\$SBX/bin:/usr/bin rodou o yay de
# verdade. Para tirar um comando, use path_sem (tests/lib.sh).
out=$(grep -rnE 'PATH=("|\$\{?[A-Za-z_]*\}?:|[^ ]*:)?/(usr/)?s?bin' tests --include='*.test.sh' |
  grep -v 'command -v' | grep -v '^tests/lint.test.sh' || true)
eq "" "$out"
