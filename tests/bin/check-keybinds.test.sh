#!/usr/bin/env bash
# bin/check-keybinds: extrai o comando do cabeçalho do Keybinds.js e roda na
# raiz do repositório; aqui com cópias em $SBX (e uma vez com o de verdade).
. "$(dirname "$0")/../lib.sh"

falso=$SBX/repo
mkdir -p "$falso"/{bin,shell/mare,config/hypr/mare}
cp "$REPO/bin/check-keybinds" "$falso/bin/check-keybinds"
confere() { bash "$falso/bin/check-keybinds" 2>&1; }

# Um PATH sem o node: só o que o script usa antes de procurar por ele.
mkdir -p "$SBX/min"
for c in bash env sed dirname; do
  ln -s "$(PATH=/usr/bin:/bin command -v "$c")" "$SBX/min/$c"
done

t "sem o node: diz que precisa dele e sai com 2 (o mk-release não lança)"
echo "//   node -e 'process.exit(0)'" >"$falso/shell/mare/Keybinds.js"
out=$(PATH=$SBX/min confere)
eq 2 $?
has "$out" "check-keybinds: precisa do node"

command -v node >/dev/null || skip "falta o node"

t "sem o comando no cabeçalho do Keybinds.js: sai com 2, dizendo por quê"
printf '.pragma library\n// sem comando aqui\nvar actions = []\n' >"$falso/shell/mare/Keybinds.js"
out=$(confere)
eq 2 $?
has "$out" "comando não encontrado no Keybinds.js"

t "um 'node -e' fora do formato do cabeçalho (sem os três espaços) não é rodado"
printf '// node -e '"'"'process.exit(0)'"'"'\n' >"$falso/shell/mare/Keybinds.js"
confere >/dev/null
eq 2 $?

t "roda o comando do cabeçalho na raiz do repositório, de onde quer que seja chamado"
cat >"$falso/shell/mare/Keybinds.js" <<'JS'
.pragma library
//   node -e 'const fs=require("fs");process.exit(fs.readFileSync("config/hypr/mare/binds.lua","utf8").trim()==="iguais"?0:1)'
JS
echo "iguais" >"$falso/config/hypr/mare/binds.lua"
(cd / && bash "$falso/bin/check-keybinds")
eq 0 $?

t "o comando sai com 1 (os arquivos divergem): o check-keybinds sai com 1"
echo "diferentes" >"$falso/config/hypr/mare/binds.lua"
confere >/dev/null
eq 1 $?

t "o repositório de verdade: binds.lua e Keybinds.js batem"
out=$(bash "$REPO/bin/check-keybinds" 2>&1)
eq 0 $? "$out"

t "com os arquivos de verdade: um combo trocado só no binds.lua é pego"
cp "$REPO/shell/mare/Keybinds.js" "$falso/shell/mare/Keybinds.js"
cp "$REPO/config/hypr/mare/binds.lua" "$falso/config/hypr/mare/binds.lua"
ok confere >/dev/null
# Troca o combo da primeira ação do binds.lua.
sed -i '0,/combo = "[^"]*"/s//combo = "SUPER + F24"/' "$falso/config/hypr/mare/binds.lua"
out=$(confere)
eq 1 $?
has "$out" "SUPER + F24"

t "com os arquivos de verdade: uma ação a menos no Keybinds.js é pega"
cp "$REPO/config/hypr/mare/binds.lua" "$falso/config/hypr/mare/binds.lua"
# Tira a primeira ação (a primeira linha com id:) do Keybinds.js.
n=$(grep -n '^ *{ *id: *"' "$falso/shell/mare/Keybinds.js" | head -1 | cut -d: -f1)
if [[ -n $n ]]; then
  sed -i "${n}d" "$falso/shell/mare/Keybinds.js"
  out=$(confere)
  eq 1 $?
  has "$out" "só no lua"
else
  _fail "não achei uma ação de uma linha no Keybinds.js para tirar"
fi
