#!/usr/bin/env bash
# A estrutura do shell que nenhum compilador confere: cada painel registrado
# tem arquivo e entrada no Tide, cada página dos Ajustes tem o seu arquivo,
# cada script chamado pelo QML existe.
. "$(dirname "$0")/../lib.sh"

mare=$REPO/shell/mare

mapfile -t panels < <(sed -n '/readonly property var panels:/,/\]/p' "$mare/Mare.qml" | grep -oE '"[a-z]+"' | tr -d '"')

t "Mare.panels foi lido (a lista não mudou de formato)"
ok test "${#panels[@]}" -ge 15

t "todo painel de Mare.panels tem case no Tide e arquivo <Nome>Panel.qml"
for p in "${panels[@]}"; do
  [[ $p == settings ]] && continue
  file=${p^}Panel.qml
  [[ -f $mare/$file ]] || _fail "falta o arquivo $file (painel $p)"
  # O launcher tem um Loader próprio (fica carregado); os outros, um case.
  grep -qE "case \"$p\": return|sourceComponent: ${p}Panel" "$mare/Tide.qml" || _fail "o Tide não carrega o painel $p"
  grep -qE "Component \{ id: ${p}Panel; ${p^}Panel \{\} \}" "$mare/Tide.qml" || _fail "falta o Component de $p no Tide"
done

t "todo *Panel.qml está registrado em Mare.panels (nenhum painel órfão)"
for f in "$mare"/*Panel.qml; do
  n=$(basename "$f" Panel.qml)
  n=${n,}
  [[ " ${panels[*]} " == *" $n "* ]] || _fail "$(basename "$f") não está em Mare.panels"
done

t "toda página de SettingsWindow.pageFiles tem o arquivo e existe no esquema"
while IFS=: read -r page file; do
  [[ -f $mare/$file.qml ]] || _fail "falta $file.qml (página $page)"
  grep -qE "key: \"$page\", name:" "$mare/SettingsSchema.js" || _fail "a página $page não está no SettingsSchema.js"
done < <(sed -n '/readonly property var pageFiles:/,/})/p' "$mare/SettingsWindow.qml" | sed -nE 's/.*"([a-z]+)": "([A-Za-z]+)".*/\1:\2/p')

t "todo script scripts/mare-* citado no QML existe e é executável"
while read -r s; do
  [[ -x $mare/scripts/$s ]] || _fail "o QML chama scripts/$s, que não existe ou não tem +x"
done < <(grep -ohE 'scripts/mare-[a-z-]+' "$mare"/*.qml | sed 's|scripts/||' | sort -u)

t "todo script de scripts/ é usado por alguém (QML, outro script, config ou bin)"
for s in "$mare"/scripts/*; do
  n=$(basename "$s")
  mapfile -t others < <(find "$mare/scripts" -type f ! -name "$n")
  grep -rqF -- "$n" "$mare"/*.qml "$REPO/config" "$REPO/bin" "$REPO/installer/mk-install" "$REPO/migrations" \
    "${others[@]}" || _fail "ninguém chama scripts/$n"
done

t "todo tipo QML próprio usado existe como arquivo"
# Tipos com inicial maiúscula seguidos de { que não são do Qt/Quickshell:
# confere só os que têm cara de ser do Maré (terminam em Panel, Page ou
# Block, ou começam com Mk/Settings).
while read -r type; do
  [[ -f $mare/$type.qml ]] || grep -qE "component $type:" "$mare"/*.qml || _fail "tipo $type usado e não existe $type.qml"
done < <(grep -ohE '^\s*(Mk[A-Z][A-Za-z]+|Settings[A-Z][A-Za-z]+|[A-Z][A-Za-z]+(Panel|Page|Block)) \{' "$mare"/*.qml | tr -d ' {' | sort -u)

t "singletons: todo arquivo com 'pragma Singleton' é referenciado"
for f in "$mare"/*.qml; do
  grep -q '^pragma Singleton' "$f" || continue
  n=$(basename "$f" .qml)
  mapfile -t others < <(find "$mare" -maxdepth 1 -name '*.qml' ! -name "$n.qml")
  grep -lqE "\b$n\b" "${others[@]}" || _fail "singleton $n sem uso"
done

t "o fundo não traz nenhum lugar escrito à mão: a cidade e as coordenadas vêm do clima"
fundo=$(<"$mare/Backdrop.qml")
hasnt "$fundo" "PORTO VELHO"
out=$(grep -nE '[0-9]+°[0-9]+′' "$mare/Backdrop.qml" || true)
eq "" "$out"
has "$fundo" "Place.coords(Weather.data.lat, Weather.data.lon)"
has "$fundo" "Place.label(Weather.data.city, Weather.data.country)"

t "papel de parede de fábrica: o Maré aplica um dos que vêm com ele, uma vez, só em instalação nova"
w=$(<"$mare/Wallpaper.qml")
fabrica=$(sed -nE 's/.*readonly property string factory: bundled \+ "\/([^"]+)".*/\1/p' "$mare/Wallpaper.qml")
[[ -n $fabrica ]] || _fail "não achei o papel de parede de fábrica no Wallpaper.qml"
exists "$mare/wallpapers/$fabrica"
# Só com a migração já feita (o mk-update reinicia o Maré antes de rodá-la) e
# sem a marca, e nunca por cima de uma escolha da pessoa.
has "$w" "fresh && !marked"
has "$w" 'current === ""'
exists "$REPO/migrations/user/20261128-papel-de-fabrica.sh"
