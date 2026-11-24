#!/usr/bin/env bash
# shell/mare/scripts/mare-emoji: pick (copiar, colar, o aviso que se troca),
# tem-wtype e gerar. O gerar escreve sempre dentro de $SBX: o Emoji.js do
# repositório não é tocado (o último caso confere).
. "$(dirname "$0")/../lib.sh"

emoji_sh=$REPO/shell/mare/scripts/mare-emoji
idfile=$XDG_RUNTIME_DIR/mk-emoji-aviso
soma_antes=$(sha256sum "$REPO/shell/mare/Emoji.js")

# wl-copy: guarda o que chegou pela entrada; clip-falha faz falhar.
stub wl-copy '[[ ! -e $SBX/clip-falha ]] || exit 1; cat >"$SBX/clip"'
# notify-send: devolve o número do aviso (aviso-id), como o -p de verdade.
stub notify-send 'cat "$SBX/aviso-id" 2>/dev/null'
stub wtype '[[ ! -e $SBX/wtype-falha ]]'
stub hyprctl 'cat "$SBX/janela.json" 2>/dev/null'

run() { MK_EMOJI_DELAY=0 bash "$emoji_sh" "$@"; }
# Um PATH só com o básico do sistema, os stubs e os guardas, sem os comandos
# dados: para o ramo em que o wtype não está instalado.
path_sem() {
  local d c p
  d=$(mktemp -d "$SBX/sem.XXXXXX")
  mkdir "$d/s" "$d/g"
  for c in bash sh env jq cat sed sleep rm mkdir dirname basename; do
    p=$(PATH=/usr/bin:/bin command -v "$c") && ln -s "$p" "$d/s/$c"
  done
  cp "$SBX"/guard/* "$d/g/"
  for c; do rm -f "$d/g/$c"; done
  echo "$SBX/bin:$d/g:$d/s"
}
zera() {
  rm -f "$SBX/clip" "$SBX/clip-falha" "$SBX/wtype-falha" "$idfile"
  echo 42 >"$SBX/aviso-id"
  echo '{"address":"0x55d0c0ffee","class":"kitty"}' >"$SBX/janela.json"
  calls_reset
}

# ---------------------------------------------------------------- pick

t "pick copiar: o emoji vai para a área de transferência, sem quebra de linha"
zera
ok run pick "😀" copiar "rosto sorrindo"
eq "😀" "$(<"$SBX/clip")"
eq 4 "$(wc -c <"$SBX/clip")"

t "pick copiar: o emoji chega ao wl-copy pela entrada, nunca como argumento"
eq "wl-copy " "$(grep '^wl-copy' "$SBX/calls")"

t "pick copiar: avisa 'copiado' com o nome, num aviso passageiro, e não digita nada"
called "notify-send -a Mk -p -h boolean:transient:true -- 😀  copiado rosto sorrindo"
not_called "wtype"
not_called "hyprctl"

t "pick sem o modo: é copiar"
zera
ok run pick "🎉"
eq "🎉" "$(<"$SBX/clip")"
called "🎉  copiado"
not_called "wtype"

t "pick colar: com janela em foco, copia e digita o emoji nela"
zera
ok run pick "👍" colar "joinha"
eq "👍" "$(<"$SBX/clip")"
called "hyprctl -j activewindow"
called "wtype -- 👍"
called "notify-send -a Mk -p -h boolean:transient:true -- 👍  colado e copiado joinha"

t "pick colar: sem janela em foco, só copia e o aviso diz só 'copiado'"
zera
echo '{}' >"$SBX/janela.json"
ok run pick "👍" colar "joinha"
eq "👍" "$(<"$SBX/clip")"
not_called "wtype"
called "👍  copiado joinha"
not_called "colado"

t "pick colar: fora de uma sessão do Hyprland (hyprctl sem resposta), só copia"
zera
rm -f "$SBX/janela.json"
ok run pick "👍" colar
not_called "wtype"
called "👍  copiado"

t "pick colar: se o wtype falha, o aviso não diz que colou"
zera
: >"$SBX/wtype-falha"
ok run pick "👍" colar "joinha"
called "wtype -- 👍"
called "👍  copiado joinha"
not_called "colado"

t "pick colar: sem o wtype instalado, só copia"
zera
mv "$SBX/bin/wtype" "$SBX/wtype.fora"
PATH=$(path_sem wtype) MK_EMOJI_DELAY=0 bash "$emoji_sh" pick "👍" colar "joinha"
eq 0 $?
eq "👍" "$(<"$SBX/clip")"
called "👍  copiado joinha"
not_called "hyprctl"
mv "$SBX/wtype.fora" "$SBX/bin/wtype"

t "pick colar: espera o painel fechar antes de digitar (MK_EMOJI_DELAY)"
zera
stub sleep
run pick "👍" colar
called "sleep 0"
calls_reset
bash "$emoji_sh" pick "👍" colar
called "sleep 0.35"
calls_reset
run pick "👍" copiar
not_called "sleep"
rm -f "$SBX/bin/sleep"

t "pick: sem emoji, sai com 1 sem copiar nem avisar"
zera
fails run pick
fails run pick ""
eq "" "$(<"$SBX/calls")"

t "pick: se não deu para copiar, avisa, sai com 1 e não digita"
zera
: >"$SBX/clip-falha"
run pick "😀" colar "rosto sorrindo"
eq 1 $?
called "notify-send -a Mk -h boolean:transient:true -- Não deu para copiar rosto sorrindo"
not_called "wtype"
absent "$idfile"

t "pick: o número do aviso fica guardado para o próximo trocar"
zera
run pick "😀" copiar "um"
eq 42 "$(<"$idfile")"

t "pick: o aviso seguinte troca o anterior (-r), em vez de empilhar"
echo 43 >"$SBX/aviso-id"
calls_reset
run pick "🎉" copiar "dois"
called "notify-send -a Mk -p -r 42 -h boolean:transient:true -- 🎉  copiado dois"
eq 43 "$(<"$idfile")"

t "pick: número de aviso guardado que não é um número positivo é ignorado"
for lixo in "abc" "0" "-r 1 --foo" "12 34" ""; do
  zera
  printf '%s\n' "$lixo" >"$idfile"
  run pick "😀" copiar "nome"
  called "notify-send -a Mk -p -h boolean:transient:true -- 😀  copiado nome"
  not_called " -r "
done

t "pick: se o aviso não devolve um número, o guardado anterior fica como estava"
zera
echo 7 >"$idfile"
echo "erro: sem servidor de avisos" >"$SBX/aviso-id"
ok run pick "😀" copiar
eq 7 "$(<"$idfile")"

t "pick: sem servidor de avisos (notify-send falha), copia e sai com 0 assim mesmo"
zera
stub notify-send 'exit 1'
ok run pick "😀" copiar "nome"
eq "😀" "$(<"$SBX/clip")"
absent "$idfile"
stub notify-send 'cat "$SBX/aviso-id" 2>/dev/null'

t "pick: um 'emoji' que parece opção (-n, --help) é copiado e digitado ao pé da letra"
zera
ok run pick "--help" colar "-nome"
eq "--help" "$(<"$SBX/clip")"
called "wtype -- --help"
called "notify-send -a Mk -p -h boolean:transient:true -- --help  colado e copiado -nome"
zera
ok run pick "-n" copiar
eq "-n" "$(<"$SBX/clip")"

t "pick: emoji de vários códigos (bandeira, família, tom de pele) passa inteiro"
zera
for e in "🇧🇷" "👨‍👩‍👧‍👦" "👍🏽" "❤️"; do
  run pick "$e" colar
  eq "$e" "$(<"$SBX/clip")"
  called "wtype -- $e"
done

t "pick: texto com % e barra invertida não é interpretado"
zera
ok run pick '%s\n%d' copiar '$(id) `x`'
eq '%s\n%d' "$(<"$SBX/clip")"
called 'notify-send -a Mk -p -h boolean:transient:true -- %s\n%d  copiado $(id) `x`'

# ---------------------------------------------------------------- tem-wtype

t "tem-wtype: sai com 0 quando o wtype existe, sem rodá-lo"
zera
ok run tem-wtype
not_called "wtype"

t "tem-wtype: sai diferente de 0 quando falta"
mv "$SBX/bin/wtype" "$SBX/wtype.fora"
PATH=$(path_sem wtype) bash "$emoji_sh" tem-wtype
[[ $? -ne 0 ]] || _fail "devia falhar sem o wtype"
mv "$SBX/wtype.fora" "$SBX/bin/wtype"

# ---------------------------------------------------------------- uso

t "sem subcomando ou com um desconhecido: mostra o uso e sai com 2"
zera
out=$(run 2>&1)
eq 2 $?
has "$out" "mare-emoji pick <emoji> <colar|copiar> [nome]"
has "$out" "mare-emoji gerar [saída]"
out=$(run bobagem 2>&1)
eq 2 $?
has "$out" "mare-emoji tem-wtype"
eq "" "$(<"$SBX/calls")"

# ---------------------------------------------------------------- gerar
# Lê a fonte Noto Color Emoji e o CLDR do GTK 3 instalados (só leitura); o
# emoji-test.txt é um pequeno, montado aqui, com os emoji que o script cita.
if [[ -r /usr/share/fonts/noto/NotoColorEmoji.ttf && -r /usr/share/gtk-3.0/emoji/pt.gresource ]] &&
  python3 -I -c 'import gi, fontTools' 2>/dev/null; then

  python3 -I - "$SBX/emoji-test.txt" <<'PY'
import sys
grupos = [
    ("Smileys & Emotion", "😂 🤣 😅 😍 🥰 😭 😉 😎 🤔 🙄 😴 🥳 🤡 💀 ❤️ 💔"),
    ("People & Body", "👍 👎 🙏 💪 👏 🤝 👌 🤙 ✌️ 👀 🫶"),
    ("Animals & Nature", "🐛"),
    ("Food & Drink", "☕ 🍺 🍻"),
    ("Travel & Places", "🚀 🔥"),
    ("Activities", "🎉 ⚽"),
    ("Objects", "💡 💰 💸"),
    ("Symbols", "✅ ❌ ⚠️"),
    ("Flags", "🇧🇷 🇺🇸 🇬🇧 🇵🇹"),
]
def linha(seq, nome, tipo="fully-qualified"):
    return "%-40s ; %-19s # %s E1.0 %s\n" % (" ".join("%04X" % c for c in seq), tipo, "".join(map(chr, seq)), nome)
with open(sys.argv[1], "w", encoding="utf-8") as f:
    f.write("# emoji-test.txt de teste\n")
    for grupo, emojis in grupos:
        f.write("\n# group: %s\n# subgroup: teste\n" % grupo)
        for e in emojis.split():
            seq = [ord(c) for c in e]
            nome = "thumbs up" if e == "👍" else "nome %s" % "-".join("%x" % c for c in seq)
            f.write(linha(seq, nome))
            if e == "👍":
                for tom in range(0x1F3FB, 0x1F400):
                    f.write(linha(seq + [tom], "thumbs up: tom %x" % tom))
            if e == "❤️":
                f.write(linha(seq[:1], "red heart", "unqualified"))
        if grupo == "Objects":
            # Um código de uso privado: a fonte não desenha.
            f.write(linha([0xE000], "fora da fonte"))
PY

  js=$SBX/saida/Emoji.js
  mkdir -p "$SBX/saida"
  resumo=$(MK_EMOJI_TEST=$SBX/emoji-test.txt bash "$emoji_sh" gerar "$js" 2>&1)
  rc=$?

  t "gerar: escreve o arquivo pedido e resume o que fez"
  eq 0 "$rc"
  exists "$js"
  has "$resumo" "$js: 45 emoji (0 só em inglês, 1 fora por a fonte não desenhar"

  t "gerar: o arquivo é uma biblioteca do QML, marcada como gerada, com a contagem no cabeçalho"
  eq ".pragma library" "$(head -n 1 "$js")"
  has "$(head -n 8 "$js")" "GERADO: não edite à mão"
  has "$(head -n 8 "$js")" "// 45 emoji; 0 só com o nome em inglês"

  linha_de() { grep -F -- "\"$1" "$js" | head -1; }

  t "gerar: cada linha é emoji|nome em português|palavras|nome em inglês"
  like "$(linha_de "🐛|")" '^ +"🐛\|[^|]+\|[^|]*\|nome 1f41b\|'
  like "$(linha_de "☕|")" '\|nome 2615(\||")'

  t "gerar: quem tem os cinco tons de pele leva o ~ onde o tom entra; quem não tem, não"
  like "$(linha_de "👍")" '^ +"👍~\|'
  like "$(linha_de "👎")" '^ +"👎\|'

  t "gerar: as palavras postas à mão (gírias) vão no quinto campo"
  like "$(linha_de "👍~|")" '\|thumbs up\|[^|]*joinha[^|]*"'
  like "$(linha_de "🍺|")" '\|nome 1f37a\|[^|]*breja[^|]*"'

  t "gerar: toda bandeira de país leva o código ISO no quinto campo"
  like "$(linha_de "🇧🇷|")" '\|[^|]*BR"'
  like "$(linha_de "🇵🇹|")" '\|[^|]*PT"'

  t "gerar: o que a fonte não desenha e as formas não qualificadas ficam de fora"
  hasnt "$(<"$js")" "fora da fonte"
  hasnt "$(<"$js")" "red heart"
  eq 45 "$(grep -c '^        "' "$js")"

  t "gerar: variantes de tom não viram linhas"
  eq 0 "$(grep -c 'thumbs up: tom' "$js")"

  t "gerar: cada emoji cai na lista do seu grupo, na ordem do emoji-test.txt"
  grupo_de() { awk -v e="\"$1" '/^    [a-z]+: \[$/ { g = $1 } index($0, e) { print g; exit }' "$js"; }
  eq "rostos:" "$(grupo_de "😂|")"
  eq "gente:" "$(grupo_de "👍~|")"
  eq "natureza:" "$(grupo_de "🐛|")"
  eq "comida:" "$(grupo_de "🍻|")"
  eq "lugares:" "$(grupo_de "🚀|")"
  eq "lazer:" "$(grupo_de "⚽|")"
  eq "objetos:" "$(grupo_de "💸|")"
  eq "simbolos:" "$(grupo_de "✅|")"
  eq "bandeiras:" "$(grupo_de "🇬🇧|")"
  eq "😂 🤣 😅 😍" "$(grep '^        "' "$js" | head -4 | sed 's/^ *"\([^|~]*\).*/\1/' | tr '\n' ' ' | sed 's/ $//')"

  if command -v node >/dev/null; then
    t "gerar: o arquivo gerado é JavaScript válido, e withTone troca o ~ pelo tom"
    out=$(node -e '
      const src = require("fs").readFileSync(process.argv[1], "utf8").replace(/^\.pragma library/, "");
      const m = new Function(src + "; return { groups, raw, withTone };")();
      const total = m.groups.reduce((n, g) => n + m.raw[g.key].length, 0);
      const joinha = m.raw.gente.find(l => l.startsWith("👍")).split("|")[0];
      console.log([m.groups.length, total, m.withTone(joinha, 0), m.withTone(joinha, 3)].join(" "));
    ' "$js" 2>&1)
    eq "9 45 👍 👍🏽" "$out"
  fi

  t "gerar: sem argumento, a saída é o Emoji.js ao lado da pasta scripts/ (aqui, numa cópia)"
  mkdir -p "$SBX/copia/mare/scripts"
  cp "$emoji_sh" "$SBX/copia/mare/scripts/mare-emoji"
  MK_EMOJI_TEST=$SBX/emoji-test.txt bash "$SBX/copia/mare/scripts/mare-emoji" gerar 2>/dev/null
  eq 0 $?
  ok cmp -s "$js" "$SBX/copia/mare/Emoji.js"

  t "gerar: gerar duas vezes dá o mesmo arquivo"
  MK_EMOJI_TEST=$SBX/emoji-test.txt bash "$emoji_sh" gerar "$SBX/saida/de-novo.js" 2>/dev/null
  ok cmp -s "$js" "$SBX/saida/de-novo.js"

  t "gerar: se um emoji das palavras à mão sumiu do emoji-test.txt, falha e não escreve nada"
  grep -v '^1F37A ' "$SBX/emoji-test.txt" >"$SBX/emoji-sem-cerveja.txt"
  out=$(MK_EMOJI_TEST=$SBX/emoji-sem-cerveja.txt bash "$emoji_sh" gerar "$SBX/saida/falhou.js" 2>&1)
  [[ $? -ne 0 ]] || _fail "devia falhar"
  has "$out" "EXTRA sem emoji correspondente: 🍺"
  absent "$SBX/saida/falhou.js"

  t "gerar: sem o emoji-test.txt, falha sem escrever nada"
  MK_EMOJI_TEST=$SBX/nao-existe.txt bash "$emoji_sh" gerar "$SBX/saida/falhou.js" 2>/dev/null
  [[ $? -ne 0 ]] || _fail "devia falhar"
  absent "$SBX/saida/falhou.js"
else
  echo "# gerar: sem a Noto Color Emoji, o CLDR do GTK 3, python-gobject ou python-fonttools; casos pulados" >&2
fi

t "nenhum caso tocou no Emoji.js do repositório"
eq "$soma_antes" "$(sha256sum "$REPO/shell/mare/Emoji.js")"
