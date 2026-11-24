#!/usr/bin/env bash
# A documentação contra o código: o que os docs citam existe, e o que existe
# está documentado.
. "$(dirname "$0")/../lib.sh"

cd "$REPO" || exit 1
docs=(README.md docs/*.md installer/README.md iso/README.md migrations/README.md shell/mare/README.md tests/README.md)

t "todo comando de bin/ aparece no README ou em docs/"
for b in bin/*; do
  grep -qF -- "$(basename "$b")" README.md docs/*.md || _fail "sem documentação: $b"
done

t "todo script de shell/mare/scripts tem linha no README do shell"
for s in shell/mare/scripts/*; do
  grep -qF -- "\`$(basename "$s")\`" shell/mare/README.md || _fail "sem linha no README: $s"
done

t "todo arquivo QML e JS do shell é citado no README do shell"
for q in shell/mare/*.qml shell/mare/*.js; do
  n=$(basename "$q")
  grep -qF -- "${n%.*}" shell/mare/README.md || _fail "não citado: $n"
done

t "todo painel de Mare.panels está na tabela de painéis do README"
while read -r p; do
  grep -qE "^\| \`$p\` \|" shell/mare/README.md || [[ $p == settings ]] || _fail "painel fora da tabela: $p"
done < <(sed -n '/readonly property var panels:/,/\]/p' shell/mare/Mare.qml | grep -oE '"[a-z]+"' | tr -d '"')

t "caminhos do repositório citados nos docs existem"
for d in "${docs[@]}"; do
  while read -r p; do
    p=${p%/}
    # Um doc de uma pasta pode citar o caminho a partir dela.
    base=$(dirname "$d")
    [[ -e $p || -e $base/$p ]] || compgen -G "$p" >/dev/null || compgen -G "$base/$p" >/dev/null ||
      _fail "$d cita $p, que não existe"
# (iso/out e iso/work não vão no git: só existem depois de um iso/build)
  done < <(grep -oE '`(bin|shell|config|system|installer|iso|migrations|scripts|docs|tests|apps)/[A-Za-z0-9_./*-]+`' "$d" | tr -d '`' |
    grep -vE '^iso/(out|work)(/|$)|AAAA|<' | sort -u)
done

t "toda migração citada pelo nome nos docs existe"
for d in "${docs[@]}"; do
  while read -r m; do
    compgen -G "migrations/*/$m" >/dev/null || _fail "$d cita a migração $m, que não existe"
  done < <(grep -oE '20[0-9]{6}-[a-z0-9-]+\.sh' "$d" | sort -u)
done

t "os docs não prometem o que foi retirado"
# A hibernação saiu: nenhum doc pode mandar preparar ou escolher hibernar.
out=$(grep -nE 'preparar a hibernação|escolha "?hibernar|mk-hibernate setup' "${docs[@]}" | grep -v 'recusa' || true)
eq "" "$out"
# O HyprlandFocusGrab e o hyprpolkitagent como agente ativo também saíram.
out=$(grep -n 'HyprlandFocusGrab' "${docs[@]}" | grep -viE 'antes|era|saiu|removid|não ' || true)
eq "" "$out"

t "nada de dados pessoais nos docs"
out=$(grep -nE '/home/[a-z]+/|[A-Za-z0-9._-]+@(gmail|hotmail|outlook)\.' "${docs[@]}" || true)
eq "" "$out"

t "nada de dados pessoais na vault"
if [[ -d vault ]]; then
  out=$(grep -rnE '/home/[a-z]+/|[A-Za-z0-9._-]+@(gmail|hotmail|outlook)\.' vault || true)
  eq "" "$out"
fi

t "vault: nenhum wikilink aponta para nota que não existe"
if [[ -d vault ]]; then
  while IFS= read -r alvo; do
    [[ -n $(find vault -name "$alvo.md" -print -quit) ]] || _fail "wikilink quebrado: [[$alvo]]"
  # (um `[[ -e arquivo ]]` do bash dentro de um trecho ou bloco de código não é link)
  done < <(awk '/^```/ { dentro = !dentro; next } !dentro' vault/*/*.md | sed 's/`[^`]*`//g' | grep -oE '\[\[[^]|#]+' | sed 's/^\[\[//; s/[[:space:]]*$//; s|.*/||' | grep -v '^$' | sort -u)
fi

t "o número de páginas dos Ajustes dito nos docs bate com o esquema"
n=$(grep -cE '^\s*\{ id: "[a-z]+", ' shell/mare/SettingsSchema.js)
while read -r line; do
  like "$line" "\\b$n páginas" "páginas nos docs ($n no esquema)"
done < <(grep -hoE '\b[0-9]+ páginas' README.md shell/mare/README.md | sort -u)
