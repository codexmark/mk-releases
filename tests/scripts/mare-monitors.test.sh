#!/usr/bin/env bash
# shell/mare/scripts/mare-monitors: a lista de telas (hyprctl + limites do
# EDID + telas guardadas + regra pessoal) e o teste com volta automática
# (trial, keep, revert, o vigia), com EDIDs binários e hyprctl falsos.
. "$(dirname "$0")/../lib.sh"

mon_sh=$REPO/shell/mare/scripts/mare-monitors
drm=$SBX/drm
sessao=teste123
token=$XDG_RUNTIME_DIR/mk-monitores-$sessao
mkdir -p "$drm"

# O hyprctl falso responde o que estiver em $SBX/monitors.json; "version"
# falha quando existe $SBX/hypr-morto (o Hyprland caiu).
stub hyprctl '
case "$*" in
  "-j monitors all") cat "$SBX/monitors.json" 2>/dev/null ;;
  version) [[ ! -e $SBX/hypr-morto ]] ;;
esac'

# Sempre pelo caminho inteiro do falso: um vigia que sobrasse depois de a
# caixa de areia sumir não acharia hyprctl nenhum, nunca o de verdade.
run() {
  HYPRLAND_INSTANCE_SIGNATURE=$sessao MARE_HYPRCTL=$SBX/bin/hyprctl MARE_DRM=$drm \
    bash "$mon_sh" "$@"
}

# Uma saída como o `hyprctl -j monitors all` mostra.
# tela <id> <nome> <descrição> <x> [campos a mais em JSON]
tela() {
  jq -nc --argjson id "$1" --arg name "$2" --arg desc "$3" --argjson x "$4" --argjson mais "${5:-{\}}" '{
    id: $id, name: $name, description: $desc, make: "Fabricante", model: "Modelo", serial: "",
    width: 1920, height: 1080, physicalWidth: 530, physicalHeight: 300, refreshRate: 60.00000,
    x: $x, y: 0, activeWorkspace: { id: 1, name: "1" }, specialWorkspace: { id: 0, name: "" },
    reserved: [0, 0, 0, 0], scale: 1.00, transform: 0, focused: false, dpmsStatus: true, vrr: false,
    solitary: "0", activelyTearing: false, directScanoutTo: "0", disabled: false,
    currentFormat: "XRGB8888", mirrorOf: "none",
    availableModes: ["1920x1080@60.00Hz"] } + $mais'
}
telas() { printf '%s\n' "$@" | jq -sc . >"$SBX/monitors.json"; }

# Um EDID de 128 bytes: cabeçalho, versão 1.4, um descritor de 18 bytes na
# posição dada e a soma que fecha em 0. Sem posição, nenhum descritor.
# edid <arquivo> [posição] [bytes do descritor...]
edid() {
  local file=$1 pos=${2:-} i v sum=0 fmt
  local -a b
  (($# >= 2)) && shift 2 || shift
  for ((i = 0; i < 128; i++)); do b[i]=0; done
  for i in 1 2 3 4 5 6; do b[i]=255; done
  b[18]=1 b[19]=4
  if [[ -n $pos ]]; then
    i=$pos
    for v in "$@"; do
      b[i]=$v
      i=$((i + 1))
    done
  fi
  for ((i = 0; i < 127; i++)); do sum=$((sum + b[i])); done
  b[127]=$(((256 - sum % 256) % 256))
  fmt=$(printf '\\x%02x' "${b[@]}")
  # shellcheck disable=SC2059  # o formato é o próprio conteúdo binário
  printf "$fmt" >"$file"
}
# O descritor "Display Range Limits" (marca 0xFD).
# faixa <flags> <vmin> <vmax> <hmin> <hmax> <clock/10MHz> [byte10] [byte11] [byte12]
faixa() { echo 0 0 0 253 "$1" "$2" "$3" "$4" "$5" "$6" "${7:-1}" "${8:-10}" "${9:-32}" 32 32 32 32 32; }
# conector <nome> [placa]: a pasta que o kernel cria para a saída.
conector() {
  mkdir -p "$drm/${2:-card1}-$1"
  echo "$drm/${2:-card1}-$1/edid"
}
# Dois JSON iguais em valor (60.00 e 60 são o mesmo número).
jeq() {
  jq -en --argjson a "$1" --argjson b "$2" '$a == $b' >/dev/null 2>&1 ||
    _fail "${3:-jeq}: esperado" "    $1" "  obtido" "    $2"
}
limites() { run list | jq -c --arg n "${1:-DP-1}" '.monitors[] | select(.name == $n) | .limits'; }

# ---------------------------------------------------------------- list

t "list: sem hyprctl respondendo, sai um JSON vazio e válido"
rm -f "$SBX/monitors.json"
eq '{"monitors":[],"saved":[]}' "$(run list)"

t "list: resposta que não é uma lista (erro do hyprctl em texto, objeto) vira lista vazia"
echo "HYPRLAND_INSTANCE_SIGNATURE not set!" >"$SBX/monitors.json"
eq '{"monitors":[],"saved":[]}' "$(run list)"
echo '{"erro":true}' >"$SBX/monitors.json"
eq '{"monitors":[],"saved":[]}' "$(run list)"

t "list: pergunta ao hyprctl por todas as saídas, ligadas ou não"
calls_reset
telas "$(tela 0 DP-1 "Dell U2720Q" 0)"
run list >/dev/null
called "hyprctl -j monitors all"

t "list: sem MARE_HYPRCTL, o comando é o hyprctl do PATH"
calls_reset
out=$(MARE_DRM=$drm bash "$mon_sh" list)
called "hyprctl -j monitors all"
eq DP-1 "$(jq -r '.monitors[0].name' <<<"$out")"

t "list: cada saída vem no formato do painel"
telas "$(tela 0 DP-1 "Dell Inc. DELL U2720Q 7XYZ" 1920 '{"width":3840,"height":2160,"refreshRate":59.99700,"y":120,"scale":1.5,"transform":1,"focused":true,"make":"Dell Inc.","model":"DELL U2720Q"}')"
out=$(run list)
jeq '{"name":"DP-1","description":"Dell Inc. DELL U2720Q 7XYZ","make":"Dell Inc.","model":"DELL U2720Q","w":3840,"h":2160,"refresh":59.997,"x":1920,"y":120,"scale":1.5,"transform":1,"enabled":true,"focused":true,"mirror":"","modes":[{"w":1920,"h":1080,"r":60}],"limits":null,"ruled":false}' \
  "$(jq -c '.monitors[0]' <<<"$out")"

t "list: campos que o hyprctl não mandou ganham o valor neutro"
echo '[{"id":0,"name":"DP-1"}]' >"$SBX/monitors.json"
eq '{"name":"DP-1","description":"","make":"","model":"","w":0,"h":0,"refresh":0,"x":0,"y":0,"scale":1,"transform":0,"enabled":true,"focused":false,"mirror":"","modes":[],"limits":null,"ruled":false}' \
  "$(run list | jq -c '.monitors[0]')"

t "list: os modos saem sem repetição, do maior para o menor e da taxa mais alta para a mais baixa"
telas "$(tela 0 DP-1 "Tela" 0 '{"availableModes":["1920x1080@60.00Hz","2560x1440@59.95Hz","1920x1080@165.00Hz","1920x1080@60.00Hz","2560x1440@143.91Hz","1280x1440@60.00Hz","2560x720@60.00Hz"]}')"
jeq '[[2560,1440,143.91],[2560,1440,59.95],[1920,1080,165],[1920,1080,60],[2560,720,60],[1280,1440,60]]' \
  "$(run list | jq -c '.monitors[0].modes | map([.w, .h, .r])')"

t "list: um modo que não é LxA@TaxaHz é ignorado, sem derrubar a lista"
telas "$(tela 0 DP-1 "Tela" 0 '{"availableModes":["preferred","1920x1080@60.00Hz","1920x1080","x@Hz",""]}')"
jeq '[{"w":1920,"h":1080,"r":60}]' "$(run list | jq -c '.monitors[0].modes')"

t "list: uma saída desligada vem com enabled falso"
telas "$(tela 0 DP-1 "Tela" 0 '{"disabled":true}')"
eq false "$(run list | jq -c '.monitors[0].enabled')"

t "list: o espelho vem pelo nome da saída espelhada, não pelo número"
telas "$(tela 0 eDP-1 "Painel interno" 0)" "$(tela 1 HDMI-A-1 "Projetor" 0 '{"mirrorOf":0}')" \
  "$(tela 2 DP-2 "Outra tela" 1920 '{"mirrorOf":"1"}')" "$(tela 3 DP-3 "Sem par" 3840 '{"mirrorOf":"9"}')"
out=$(run list)
eq '{"DP-2":"HDMI-A-1","DP-3":"","HDMI-A-1":"eDP-1","eDP-1":""}' \
  "$(jq -cS '.monitors | map({ key: .name, value: .mirror }) | from_entries' <<<"$out")"

t "list: a tela de mentira (FALLBACK) e as virtuais (HEADLESS-n) ficam de fora"
telas "$(tela 0 FALLBACK "" 0)" "$(tela 1 HEADLESS-2 "" 0)" "$(tela 2 DP-1 "Tela" 0)"
eq '["DP-1"]' "$(run list | jq -c '.monitors | map(.name)')"

t "list: as saídas vêm da esquerda para a direita, de cima para baixo, e no empate pelo nome"
telas "$(tela 0 DP-3 "C" 1920)" "$(tela 1 DP-2 "B" 0 '{"y":1080}')" "$(tela 2 DP-1 "A" 0)" "$(tela 3 DP-0 "Z" 1920)"
eq '["DP-1","DP-2","DP-0","DP-3"]' "$(run list | jq -c '.monitors | map(.name)')"

# ---------------------------------------------------------------- saved

t "list: 'saved' é a linha -- mk:telas do arquivo dado"
telas "$(tela 0 DP-1 "Tela" 0)"
cat >"$SBX/monitores.lua" <<'EOF'
-- Gerado pelos Ajustes.
-- mk:telas [{"name":"DP-1","mode":"1920x1080@60"},{"name":"HDMI-A-1","mode":"preferred"}]
hl.monitor({ output = "DP-1", mode = "1920x1080@60" })
EOF
eq '[{"name":"DP-1","mode":"1920x1080@60"},{"name":"HDMI-A-1","mode":"preferred"}]' \
  "$(run list "$SBX/monitores.lua" | jq -c .saved)"

t "list: de duas linhas -- mk:telas, vale a primeira"
printf -- '-- mk:telas [{"name":"DP-1"}]\n-- mk:telas [{"name":"DP-9"}]\n' >"$SBX/monitores.lua"
eq '[{"name":"DP-1"}]' "$(run list "$SBX/monitores.lua" | jq -c .saved)"

t "list: FALLBACK e HEADLESS-n guardados por engano não voltam em 'saved'"
printf -- '-- mk:telas [{"name":"FALLBACK"},{"name":"HEADLESS-3"},{"name":"DP-1"}]\n' >"$SBX/monitores.lua"
eq '[{"name":"DP-1"}]' "$(run list "$SBX/monitores.lua" | jq -c .saved)"

t "list: linha -- mk:telas estragada, ou que não é uma lista, vira 'saved' vazio"
printf -- '-- mk:telas [{"name":"DP-1"\n' >"$SBX/monitores.lua"
out=$(run list "$SBX/monitores.lua")
eq '[]' "$(jq -c .saved <<<"$out")"
eq '["DP-1"]' "$(jq -c '.monitors | map(.name)' <<<"$out")"
printf -- '-- mk:telas {"name":"DP-1"}\n' >"$SBX/monitores.lua"
eq '[]' "$(run list "$SBX/monitores.lua" | jq -c .saved)"

t "list: arquivo que não existe, 'saved' vazio"
eq '[]' "$(run list "$SBX/nao-existe.lua" | jq -c .saved)"

# ---------------------------------------------------------------- limits (EDID)

telas "$(tela 0 DP-1 "Tela" 0)" "$(tela 1 eDP-1 "Painel" 1920)" "$(tela 2 HDMI-A-1 "TV" 3840)"
dp=$(conector DP-1)

t "limits: o bloco 0xFD vira vmin/vmax em Hz, hmin/hmax em kHz e clock em MHz"
# shellcheck disable=SC2046  # os bytes do descritor, um por argumento
edid "$dp" 90 $(faixa 0 48 165 30 180 60)
eq '{"vmin":48,"vmax":165,"hmin":30,"hmax":180,"clock":600}' "$(limites)"

t "limits: o bloco é achado em qualquer um dos quatro descritores"
for pos in 54 72 90 108; do
  # shellcheck disable=SC2046
  edid "$dp" "$pos" $(faixa 0 40 75 30 83 17)
  eq '{"vmin":40,"vmax":75,"hmin":30,"hmax":83,"clock":170}' "$(limites)" "descritor em $pos"
done

t "limits: uma marca 0xFD fora dos quatro descritores não conta"
# shellcheck disable=SC2046
edid "$dp" 60 $(faixa 0 40 75 30 83 17)
eq null "$(limites)"

t "limits: EDID sem o bloco de limites, null"
edid "$dp"
eq null "$(limites)"
# Um descritor de outro tipo (0xFC, o nome do monitor) não serve.
edid "$dp" 72 0 0 0 252 0 77 111 110 105 116 111 114 10 32 32 32 32 32
eq null "$(limites)"

t "limits: EDID 1.4, o byte 4 soma 255 só ao vmax (0x02) ou ao hmax (0x08)"
# shellcheck disable=SC2046
edid "$dp" 54 $(faixa 2 48 105 30 255 99)
eq '{"vmin":48,"vmax":360,"hmin":30,"hmax":255,"clock":990}' "$(limites)"
# shellcheck disable=SC2046
edid "$dp" 54 $(faixa 8 48 240 30 45 99)
eq '{"vmin":48,"vmax":240,"hmin":30,"hmax":300,"clock":990}' "$(limites)"

t "limits: o byte 4 em 0x0F soma 255 aos quatro limites"
# shellcheck disable=SC2046
edid "$dp" 54 $(faixa 15 1 105 5 45 99)
eq '{"vmin":256,"vmax":360,"hmin":260,"hmax":300,"clock":990}' "$(limites)"

t "limits: com fórmula CVT (byte 10 = 4), o byte 12 tira 0,25 MHz por passo do clock"
# 0b00010100 >> 2 = 5 passos: 600 - 1,25 = 598,75 MHz.
# shellcheck disable=SC2046
edid "$dp" 90 $(faixa 0 48 165 30 180 60 4 17 20)
eq '{"vmin":48,"vmax":165,"hmin":30,"hmax":180,"clock":598.75}' "$(limites)"

t "limits: sem fórmula CVT, o byte 12 não mexe no clock"
# shellcheck disable=SC2046
edid "$dp" 90 $(faixa 0 48 165 30 180 60 1 17 20)
eq 600 "$(limites | jq .clock)"

t "limits: limites sem sentido (zero, máximo abaixo do mínimo, clock zero) viram null"
for f in "0 0 165 30 180 60" "0 75 60 30 180 60" "0 48 165 0 180 60" "0 48 165 90 30 60" "0 48 165 30 180 0"; do
  # shellcheck disable=SC2046,SC2086
  edid "$dp" 54 $(faixa $f)
  eq null "$(limites)" "faixa $f"
done

t "limits: EDID com a soma errada é recusado"
# shellcheck disable=SC2046
edid "$dp" 90 $(faixa 0 48 165 30 180 60)
python3 -I -c 'import sys
p = sys.argv[1]; b = bytearray(open(p, "rb").read()); b[127] = (b[127] + 7) % 256
open(p, "wb").write(b)' "$dp"
eq null "$(limites)"

t "limits: EDID sem o cabeçalho 00 FF FF FF FF FF FF 00 é recusado"
# shellcheck disable=SC2046
edid "$dp" 90 $(faixa 0 48 165 30 180 60)
# Troca dois bytes do cabeçalho mantendo a soma: só o cabeçalho denuncia.
python3 -I -c 'import sys
p = sys.argv[1]; b = bytearray(open(p, "rb").read()); b[1] = 254; b[20] = (b[20] + 1) % 256
assert sum(b) % 256 == 0
open(p, "wb").write(b)' "$dp"
eq null "$(limites)"

t "limits: EDID cortado, vazio (saída sem cabo) ou ausente, null"
# shellcheck disable=SC2046
edid "$dp" 90 $(faixa 0 48 165 30 180 60)
head -c 100 "$dp" >"$dp.curto" && mv "$dp.curto" "$dp"
eq null "$(limites)"
: >"$dp"
eq null "$(limites)"
rm -f "$dp"
eq null "$(limites)"

t "limits: só o bloco base conta; os blocos de extensão não atrapalham"
# shellcheck disable=SC2046
edid "$dp" 90 $(faixa 0 48 144 30 160 60)
head -c 128 /dev/urandom >>"$dp"
eq '{"vmin":48,"vmax":144,"hmin":30,"hmax":160,"clock":600}' "$(limites)"

t "limits: a saída é achada em qualquer placa (card0, card1...)"
rm -rf "${drm:?}"/*
# shellcheck disable=SC2046
edid "$(conector HDMI-A-1 card0)" 54 $(faixa 0 24 60 15 68 30)
eq '{"vmin":24,"vmax":60,"hmin":15,"hmax":68,"clock":300}' "$(limites HDMI-A-1)"

t "limits: a DP-1 não herda os limites do painel eDP-1 nem os da DP-10"
rm -rf "${drm:?}"/*
edid "$(conector DP-1)"
# shellcheck disable=SC2046
edid "$(conector eDP-1 card0)" 54 $(faixa 0 40 60 30 90 20)
# shellcheck disable=SC2046
edid "$(conector DP-10)" 54 $(faixa 0 50 75 30 90 25)
telas "$(tela 0 DP-1 "Tela" 0)" "$(tela 1 eDP-1 "Painel" 1920)"
eq null "$(limites DP-1)"
eq '{"vmin":40,"vmax":60,"hmin":30,"hmax":90,"clock":200}' "$(limites eDP-1)"

t "limits: cada saída leva os seus"
# shellcheck disable=SC2046
edid "$(conector DP-1)" 72 $(faixa 0 48 165 30 180 60)
eq '[["DP-1",165],["eDP-1",60]]' "$(run list | jq -c '.monitors | map([.name, .limits.vmax])')"

t "limits: um nome de saída hostil não vira caminho nem derruba a lista"
mkdir -p "$SBX/fora"
# shellcheck disable=SC2046
edid "$SBX/fora/edid" 54 $(faixa 0 48 165 30 180 60)
telas "$(tela 0 "x/../../fora" "Tela" 0)" "$(tela 1 'a"b c' "Outra" 100)" "$(tela 2 '*' "Curinga" 200)"
out=$(run list)
eq '[null,null,null]' "$(jq -c '.monitors | map(.limits)' <<<"$out")"
eq '["x/../../fora","a\"b c","*"]' "$(jq -c '.monitors | map(.name)' <<<"$out")"

# ---------------------------------------------------------------- ruled

rm -rf "${drm:?}"/*
telas "$(tela 0 DP-1 "Dell Inc. DELL U2720Q 7XYZ" 0)" "$(tela 1 eDP-1 "BOE" 1920)" "$(tela 2 HDMI-A-1 "LG TV" 3840)"
regra() { run list "" "$SBX/pessoal.lua" | jq -c '.monitors | map(select(.ruled) | .name)'; }

t "ruled: a saída citada pelo nome no monitors.lua pessoal vem marcada"
echo 'hl.monitor({ output = "HDMI-A-1", mode = "preferred" })' >"$SBX/pessoal.lua"
eq '["HDMI-A-1"]' "$(regra)"

t "ruled: o nome tem de estar inteiro: eDP-1 e DP-10 não marcam a DP-1"
printf '%s\n' 'hl.monitor({ output = "eDP-1" })' 'hl.monitor({ output = "DP-10" })' >"$SBX/pessoal.lua"
eq '["eDP-1"]' "$(regra)"

t "ruled: o nome no começo ou no fim do arquivo, sem nada em volta, também vale"
printf 'DP-1' >"$SBX/pessoal.lua"
eq '["DP-1"]' "$(regra)"

t "ruled: citada só em linha de comentário, não marca"
printf '%s\n' '-- hl.monitor({ output = "DP-1" })' '   -- HDMI-A-1 fica para depois' 'local x = 1' >"$SBX/pessoal.lua"
eq '[]' "$(regra)"

t "ruled: a descrição da tela (6 letras ou mais) também marca; uma curta, não"
printf '%s\n' 'hl.monitor({ output = "desc:Dell Inc. DELL U2720Q 7XYZ" })' 'local marca = "BOE"' >"$SBX/pessoal.lua"
eq '["DP-1"]' "$(regra)"

t "ruled: sem monitors.lua pessoal (ou vazio), ninguém vem marcado"
: >"$SBX/pessoal.lua"
eq '[]' "$(regra)"
rm -f "$SBX/pessoal.lua"
eq '[]' "$(regra)"

t "ruled: descrição com caracteres de padrão do shell é comparada ao pé da letra"
telas "$(tela 0 DP-1 "Tela [*] ?????" 0)"
echo 'output = "desc:Tela boa demais"' >"$SBX/pessoal.lua"
eq '[]' "$(regra)"
echo 'output = "desc:Tela [*] ?????"' >"$SBX/pessoal.lua"
eq '["DP-1"]' "$(regra)"

# ---------------------------------------------------------------- trial / keep / revert

arq=$SBX/cfg/hypr/mare/monitores.lua
novo='hl.monitor({ output = "DP-1", mode = "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync" })'
espera() { # até 6 s por uma condição
  local i
  for i in $(seq 1 60); do
    "$@" && return 0
    sleep 0.1
  done
  return 1
}
sem_vigia() { [[ ! -e $token.pid ]]; }
tem_vigia() { [[ -s $token.pid ]] && kill -0 "$(<"$token.pid")" 2>/dev/null; }
voltou() { [[ ! -e $arq.anterior ]]; }
recargas() { grep -c '^hyprctl reload$' "$SBX/calls"; }
# Começa cada caso sem vigia, sem sobra e com o arquivo dado.
zera() {
  if [[ -s $token.pid ]]; then kill -KILL "$(<"$token.pid")" 2>/dev/null; fi
  rm -rf "$token" "$token.pid" "$SBX/cfg" "$SBX/hypr-morto"
  if (($#)); then
    mkdir -p "$(dirname "$arq")"
    printf '%s' "$1" >"$arq"
  fi
  calls_reset
}

t "trial: sem os quatro argumentos, ou com segundos que não são número, uso e saída 2"
zera
for args in "trial" "trial 15" "trial 15 $arq" "trial quinze $arq texto" "trial -1 $arq texto" "trial 15 $arq a b"; do
  # shellcheck disable=SC2086
  run $args >/dev/null 2>&1
  eq 2 $? "$args"
done
run trial 15 "" texto >/dev/null 2>&1
eq 2 $? "arquivo vazio"
absent "$arq"
not_called "hyprctl"

t "trial: grava o texto como veio, cria a pasta, recarrega o Hyprland e responde ok"
zera
out=$(run trial 30 "$arq" "$novo")
eq ok "$out"
eq "$novo" "$(<"$arq")"
eq "${#novo}" "$(wc -c <"$arq")" "sem quebra de linha a mais"
eq 1 "$(recargas)"

t "trial: deixa um vigia rodando e o arquivo anterior guardado com a sessão na primeira linha"
ok espera tem_vigia
eq "-- mk:teste $sessao" "$(head -n 1 "$arq.anterior")"

t "keep: o vigia mantém o arquivo novo, apaga a sobra e não recarrega de novo"
ok run keep
ok espera sem_vigia
eq "$novo" "$(<"$arq")"
absent "$arq.anterior"
absent "$token"
eq 1 "$(recargas)"

t "revert: o arquivo de antes volta byte a byte e o Hyprland recarrega"
antes=$'-- telas de antes\nhl.monitor({ output = "DP-1", mode = "preferred" })\n\n\n'
zera "$antes"
run trial 30 "$arq" "$novo" >/dev/null
eq "$novo" "$(<"$arq")"
ok run revert
ok espera sem_vigia
ok cmp -s "$arq" <(printf '%s' "$antes")
absent "$arq.anterior"
absent "$token"
eq 2 "$(recargas)"

t "revert: se antes não havia arquivo, ele some de novo"
zera
run trial 30 "$arq" "$novo" >/dev/null
exists "$arq"
run revert
ok espera sem_vigia
absent "$arq"
absent "$arq.anterior"

t "trial: sem resposta no prazo, o vigia devolve o arquivo anterior e recarrega"
zera "antes"
run trial 1 "$arq" "$novo" >/dev/null
eq "$novo" "$(<"$arq")"
ok espera voltou
ok espera sem_vigia
eq "antes" "$(<"$arq")"
eq 2 "$(recargas)"

t "trial: se o Hyprland cai com a configuração nova, o vigia devolve já, sem esperar o prazo"
zera "antes"
: >"$SBX/hypr-morto"
SECONDS=0
run trial 60 "$arq" "$novo" >/dev/null
ok espera voltou
ok espera sem_vigia
eq "antes" "$(<"$arq")"
((SECONDS < 10)) || _fail "levou $SECONDS s"
rm -f "$SBX/hypr-morto"

t "trial: o vigia que recebe o sinal de término (desligar no meio) devolve o arquivo antes de sair"
zera "antes"
run trial 60 "$arq" "$novo" >/dev/null
ok espera tem_vigia
kill -TERM "$(<"$token.pid")"
ok espera voltou
ok espera sem_vigia
eq "antes" "$(<"$arq")"
called "hyprctl reload"

t "trial: um segundo teste por cima do primeiro guarda como 'antes' o arquivo original"
zera "original"
run trial 60 "$arq" "primeiro" >/dev/null
ok espera tem_vigia
vigia1=$(<"$token.pid")
out=$(run trial 60 "$arq" "segundo")
eq ok "$out"
eq "segundo" "$(<"$arq")"
eq "original" "$(tail -n +2 "$arq.anterior")"
ok espera tem_vigia
[[ $(<"$token.pid") != "$vigia1" ]] || _fail "o vigia do primeiro teste continua dono"
fails kill -0 "$vigia1" 2>/dev/null
run revert
ok espera sem_vigia
eq "original" "$(<"$arq")"

t "trial: aplicar logo depois de voltar espera o vigia terminar e parte do arquivo devolvido"
zera "original"
run trial 60 "$arq" "primeiro" >/dev/null
ok espera tem_vigia
run revert
run trial 60 "$arq" "segundo" >/dev/null
eq "segundo" "$(<"$arq")"
eq "original" "$(tail -n +2 "$arq.anterior")"
run revert
ok espera sem_vigia
eq "original" "$(<"$arq")"

t "trial: a sobra de uma sessão que morreu não vale como 'antes'"
zera "atual"
printf -- '-- mk:teste sessao-velha\nde dias atras' >"$arq.anterior"
run trial 60 "$arq" "$novo" >/dev/null
eq "-- mk:teste $sessao" "$(head -n 1 "$arq.anterior")"
run revert
ok espera sem_vigia
eq "atual" "$(<"$arq")"

t "trial: a sobra desta mesma sessão, sem vigia cedendo a vez, também não vale como 'antes'"
zera "atual"
printf -- '-- mk:teste %s\nde um vigia que foi morto' "$sessao" >"$arq.anterior"
run trial 60 "$arq" "$novo" >/dev/null
eq "atual" "$(tail -n +2 "$arq.anterior")"
run revert
ok espera sem_vigia
eq "atual" "$(<"$arq")"

t "trial: um pid velho de vigia que já morreu não segura o teste novo"
zera "atual"
echo 999999999 >"$token.pid"
SECONDS=0
eq ok "$(run trial 60 "$arq" "$novo")"
((SECONDS < 3)) || _fail "esperou $SECONDS s por um vigia morto"
run keep
ok espera sem_vigia
eq "$novo" "$(<"$arq")"

t "keep: uma resposta perdida, sem teste rodando, não confirma o teste seguinte"
zera "atual"
run keep
exists "$token"
run trial 60 "$arq" "$novo" >/dev/null
ok espera tem_vigia
sleep 0.6
exists "$arq.anterior"
ok tem_vigia
run revert
ok espera sem_vigia
eq "atual" "$(<"$arq")"

t "trial: o texto não é interpretado (\$(), %, barras) e o caminho pode ter espaço"
zera
arq_esp="$SBX/cfg/pasta com espaco/monitores.lua"
hostil='-- $(touch '"$SBX"'/invadiu) `id` %s %d \n \\ "aspas" '"'"'simples'"'"
run trial 60 "$arq_esp" "$hostil" >/dev/null
eq "$hostil" "$(<"$arq_esp")"
absent "$SBX/invadiu"
run keep
ok espera sem_vigia
eq "$hostil" "$(<"$arq_esp")"
absent "$arq_esp.anterior"

t "trial: a gravação é por troca de arquivo e não deixa temporário na pasta"
zera "antes"
run trial 60 "$arq" "$novo" >/dev/null
run keep
ok espera sem_vigia
eq "monitores.lua" "$(ls -A "$(dirname "$arq")" | tr '\n' ' ' | sed 's/ $//')"

t "watch: sem os dois argumentos, uso e saída 2"
run watch 15 >/dev/null 2>&1
eq 2 $?

t "sem subcomando ou com um desconhecido: uso e saída 2"
run >/dev/null 2>&1
eq 2 $?
out=$(run bobagem 2>&1)
eq 2 $?
has "$out" "uso: mare-monitors list"

# Nenhum vigia fica para trás.
zera
