#!/usr/bin/env bash
# apps/mkpacotes: a janela (qml/) e o lançador (bin/mkpacotes). O desenho se vê
# no bin/dev-shot; aqui: a janela só usa as cores do Mk, só pede ao coletor o
# que o contrato tem, toda ação passa pela folha de confirmação, os dados de
# exemplo têm a forma do que o coletor responde, e o lançador abre uma janela
# só.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mkpacotes
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
exemplo=$TESTS/apps/mkpacotes-exemplo
coletor=$app/bin/mkpacotes-coletor

t "cores: só os tokens do Mk, e só no Theme.qml"
tokens=" #0b0b0b #141211 #1b1917 #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml "$app/share/mkpacotes.svg" | sort -u)
eq "" "$(grep -lE '#[0-9a-fA-F]{3,8}\b' "$qml"/*.qml | grep -v '/Theme.qml$' || true)"
out=$(grep -ohE 'Qt\.rgba\([^)]*\)' "$qml"/*.qml | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255|0, 0, 0, 0)' || true)
eq "" "$out"
eq "" "$(grep -nE 'color: *"[a-z]+"|Qt\.(darker|lighter|hsla|hsva|tint)\(' "$qml"/*.qml || true)"

t "o Theme e os componentes pequenos são os mesmos do MkPulso (cada app é uma pasta que se basta)"
for f in Theme Botao Segmento Rotulo; do
  ok cmp -s "$qml/$f.qml" "$REPO/apps/mkpulso/qml/$f.qml"
done

t "toda página que a janela carrega existe"
n=0
while read -r f; do
  n=$((n + 1))
  exists "$qml/$f"
done < <(grep -ohE '"[A-Z][A-Za-z]+\.qml"' "$qml/App.qml" | tr -d '"' | sort -u)
eq 4 "$n"

t "todo tipo próprio usado na janela tem o arquivo dele"
for tipo in $(grep -ohE '^\s*(Botao|Campo|Segmento|Rotulo|Cabecalho|Marca|Detalhe|Folha|App) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  exists "$qml/$tipo.qml"
done

t "a janela só pede ao coletor subcomandos que estão no contrato, e usa todos eles"
mapfile -t pedidos < <(grep -ohE '(pedir|transmitir)\(\["[a-z]+"' "$qml"/*.qml | grep -oE '"[a-z]+"' | tr -d '"' | sort -u)
for sub in "${pedidos[@]}"; do
  has "$contrato" "mkpacotes-coletor $sub" "subcomando fora do contrato: $sub"
done
mapfile -t docontrato < <(grep -oE '^## `mkpacotes-coletor [a-z]+' <<<"$contrato" | awk '{print $3}' | sort -u)
eq "${docontrato[*]}" "${pedidos[*]}"
eq 8 "${#pedidos[@]}"

t "toda chamada ao coletor diz o subcomando na hora (nada de argumentos montados longe)"
eq "" "$(grep -nE '(pedir|transmitir)\(' "$qml"/*.qml | grep -vE '(pedir|transmitir)\(\["[a-z]+"|function (pedir|transmitir)\(|^[^:]+:[0-9]+: *//' || true)"

t "as ações que a janela pede são as do contrato"
mapfile -t acoes < <(grep -ohE 'Acao\.planejar\("[a-z]+"' "$qml"/*.qml | grep -oE '"[a-z]+"' | tr -d '"' | sort -u)
eq "atualizar cache instalar orfaos remover" "${acoes[*]}"
for a in "${acoes[@]}"; do
  like "$contrato" "\`${a}[ \`]" "o contrato não tem a ação: $a"
done

t "nenhuma página age sozinha: plano e agir só saem do Acao.qml, com os nomes validados e depois de --"
eq "$qml/Acao.qml" "$(grep -lE '\["(plano|agir)"\]' "$qml"/*.qml | tr '\n' ' ' | sed 's/ $//')"
eq 2 "$(grep -cE '\["(plano|agir)"\]\.concat\(Pacotes\.argsDeAcao\(' "$qml/Acao.qml")"
has "$(<"$qml/Pacotes.js")" 'args.concat(["--"], bons)'
# O agir só roda depois de o plano estar na tela e a pessoa confirmar.
like "$(<"$qml/Acao.qml")" 'function confirmar\(\) \{[[:space:]]+if \(root\.fase !== "plano"'
eq "$qml/Folha.qml" "$(grep -lE 'Acao\.confirmar\(\)' "$qml"/*.qml | tr '\n' ' ' | sed 's/ $//')"
# O --mesmo-assim só entra depois de uma recusa, por um botão da folha.
eq "$qml/Folha.qml" "$(grep -lE 'Acao\.insistir\(\)' "$qml"/*.qml | tr '\n' ' ' | sed 's/ $//')"
eq "" "$(grep -nE 'Acao\.planejar\(.*, true\)' "$qml"/*.qml | grep -v '/Acao.qml:' || true)"

t "a ficha e a busca mandam o que a pessoa escreveu depois de --"
has "$(<"$qml/Detalhe.qml")" 'Coletor.pedir(["detalhe", "--", pedido]'
like "$(<"$qml/Procurar.qml")" 'pedir\(\["procurar"\]\.concat\(.*\["--"\], q\.split'

t "os filtros e as ordens da tabela são os que a lógica conhece"
logica=$(<"$qml/Pacotes.js")
for f in pedidos aur orfaos atualizacoes; do
  has "$logica" "case \"$f\":" "filtro que a lógica não conhece: $f"
  has "$(<"$qml/Instalados.qml")" "v: \"$f\"" "filtro que a página não oferece: $f"
done
for c in nome tamanho_bytes instalado_em_s; do
  has "$(<"$qml/Instalados.qml")" "v: \"$c\"" "ordem que a página não oferece: $c"
  has "$contrato" "\"$c\"" "coluna fora do contrato: $c"
done

t "os dados de exemplo são JSON, um por subcomando do contrato"
for f in "${docontrato[@]}"; do
  ok jq -e . "$exemplo/$f.json" >/dev/null
done
ok jq -e 'type == "array" and (.[-1].ok == true) and ([.[:-1][] | has("linha")] | all)' "$exemplo/agir.json" >/dev/null
ok jq -e '[.pacotes[] | .motivo == "pedido" or .motivo == "dependencia"] | all' "$exemplo/instalados.json" >/dev/null
ok jq -e '[.eventos[].acao] | all(. == "instalado" or . == "atualizado" or . == "removido" or . == "reinstalado" or . == "rebaixado")' "$exemplo/historico.json" >/dev/null
ok jq -e '(.totais.pacotes == (.pacotes | length)) and (.totais.orfaos == ([.pacotes[] | select(.orfao)] | length))
  and (.totais.aur == ([.pacotes[] | select(.origem == "aur")] | length))' "$exemplo/instalados.json" >/dev/null

t "nada de dados da máquina de quem escreveu nos exemplos"
eq "" "$(grep -lE '/home/[a-z]+/|codexmark' "$exemplo"/*.json || true)"

# --- os exemplos contra o coletor de verdade -----------------------------------
# Uma máquina de mentira (base do pacman, cache, registro) e o pacman trocado
# por um dublê: o coletor responde, e o exemplo tem de ter os mesmos campos.
db=$SBX/pacman
mkdir -p "$db/local/bash-5.3.20-1" "$db/sync" "$SBX/cache" "$SBX/mk"
printf '%%NAME%%\nbash\n\n%%VERSION%%\n5.3.20-1\n\n%%DESC%%\nThe shell\n\n%%SIZE%%\n1000\n\n%%INSTALLDATE%%\n1792797774\n\n%%DEPENDS%%\nglibc\n\n%%OPTDEPENDS%%\nx: y\n\n' >"$db/local/bash-5.3.20-1/desc"
printf '%%FILES%%\nusr/\nusr/bin/\nusr/bin/bash\n\n' >"$db/local/bash-5.3.20-1/files"
: >"$SBX/cache/fd-10.2.0-1-x86_64.pkg.tar.zst"
printf '[2026-11-21T08:00:00-0400] [ALPM] upgraded htop (3.5.2-1 -> 3.5.3-1)\n' >"$SBX/pacman.log"
stub pacman 'case " $* " in
  *" -Sl "*) echo "core bash 5.3.21-1" ;;
  *" -Ss "*) printf "extra/htop 3.5.3-1\n    Interactive process viewer\n" ;;
  *" -Qu "*) echo "bash 5.3.20-1 -> 5.3.21-1" ;;
  *) exit 1 ;;
esac'
stub pkexec
stub yay
# O checkupdates "sem rede": as atualizações vêm do pacman -Qu do dublê.
stub checkupdates 'exit 1'
responde() {
  MKPACOTES_PACMAN_DB=$db MKPACOTES_CACHE=$SBX/cache MKPACOTES_LOG=$SBX/pacman.log MKPACOTES_MK=$SBX/mk \
    MKPACOTES_TRAVA=$SBX/trava MKPACOTES_AUR=http://127.0.0.1:1 "$coletor" "$@"
}
# Os campos de um JSON, em qualquer profundidade, sem os índices das listas.
campos() { jq -r '[paths | map(select(type == "string")) | join(".")] | unique | .[]' <<<"$1"; }
# Todo campo que o coletor respondeu existe no exemplo, e o exemplo não inventa nenhum.
mesmos_campos() {
  local nome=$1 veio=$2 falta sobra
  falta=$(comm -23 <(campos "$veio") <(campos "$(<"$exemplo/$nome.json")"))
  sobra=$(comm -13 <(campos "$veio") <(campos "$(<"$exemplo/$nome.json")"))
  eq "" "$falta" "$nome.json: o coletor responde e o exemplo não tem"
  eq "" "$sobra" "$nome.json: o exemplo tem e o coletor não responde"
}

t "exemplos: instalados, estado e histórico têm exatamente os campos que o coletor responde"
mesmos_campos instalados "$(responde instalados)"
mesmos_campos estado "$(responde estado)"
mesmos_campos historico "$(responde historico)"

t "exemplos: procurar, atualizações e a ficha também"
mesmos_campos procurar "$(responde procurar htop --sem-aur)"
mesmos_campos atualizacoes "$(responde atualizacoes --sem-aur)"
# A ficha de exemplo é de um pacote do AUR (tem o bloco `aur`); a do coletor
# aqui é de um oficial: os campos de dentro de `aur` ficam de fora da conta.
veio=$(responde detalhe bash)
eq "" "$(comm -23 <(campos "$veio") <(campos "$(<"$exemplo/detalhe.json")"))"
eq "" "$(comm -13 <(campos "$veio") <(campos "$(<"$exemplo/detalhe.json")") | grep -v '^aur\.' || true)"

t "exemplos: o plano tem os campos do coletor (menos os de dentro de cache e de entra, que o exemplo de remoção não usa)"
veio=$(responde plano cache)
eq "" "$(comm -23 <(campos "$veio") <(campos "$(<"$exemplo/plano.json")") | grep -v '^cache\.' || true)"
eq "" "$(comm -13 <(campos "$veio") <(campos "$(<"$exemplo/plano.json")") | grep -v '^sai\.' || true)"
for c in nome versao tamanho_bytes pedido; do
  ok jq -e --arg c "$c" '[.sai[] | has($c)] | all' "$exemplo/plano.json" >/dev/null
done

t "o coletor de verdade recusa o que a janela nunca deveria mandar, sem chamar comando nenhum"
calls_reset
for nome in -Rdd --noconfirm 'htop;reboot' 'a b' HTOP '$(reboot)'; do
  out=$(responde agir remover -- "$nome")
  eq 1 $? "agir remover $nome"
  has "$out" "não é nome de pacote"
  out=$(responde plano instalar -- "$nome")
  eq 1 $?
done
out=$(responde agir remover -Rdd)
eq 1 $?
has "$out" "não conheço a opção"
not_called "pkexec"
not_called "yay"
not_called "pacman"

# --- o lançador ---------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml"
cp "$app/bin/mkpacotes" "$casa/bin/mkpacotes"
printf '#!/bin/sh\n' >"$casa/bin/mkpacotes-coletor"
chmod +x "$casa/bin/mkpacotes-coletor"
stub notify-send
stub hyprctl '[[ $1 == clients ]] && cat "$SBX/clientes.json" 2>/dev/null; exit 0'
stub qs 'if [[ $1 == list ]]; then cat "$SBX/qs-list" 2>/dev/null; else echo "AREA=${MKPACOTES_AREA:-}" >>"$SBX/calls"; fi'
abre() { bash "$casa/bin/mkpacotes" "$@"; }

t "o lançador, o pack e a assinatura têm o bit de execução"
for f in bin/mkpacotes pack assinatura bin/mkpacotes-coletor; do
  ok test -x "$app/$f"
done

t "sem janela aberta: abre a janela do app (qs -p …/qml)"
calls_reset
ok abre
called "qs -p $casa/qml"
called "AREA="

t "com uma área: abre direto nela, e são as áreas que a janela tem"
for area in instalados procurar atualizacoes historico; do
  calls_reset
  ok abre "$area"
  called "AREA=$area"
  has "$(<"$qml/App.qml")" "id: \"$area\"" "a janela não tem a área $area"
done

t "área que não existe: uso e saída 2, sem abrir nada"
calls_reset
abre banana 2>/dev/null
eq 2 $?
not_called "qs -p"

t "janela já aberta: só traz para a frente, não abre outra"
calls_reset
echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
echo '[{"title":"MkPulso","address":"0x1"},{"title":"MkPacotes","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre
called 'hl.dsp.focus({ window = "address:0x55aa" })'
not_called "qs -p"

t "um endereço torto vindo do Hyprland não entra no comando: abre a janela normalmente"
calls_reset
echo '[{"title":"MkPacotes","address":"0x1\" }) os.execute(\"x"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "sem o coletor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$SBX/qs-list" "$casa/bin/mkpacotes-coletor"
out=$(abre 2>&1)
eq 1 $?
has "$out" "coletor"
called "notify-send"
not_called "qs -p"

t "o lançador nunca chama pacman, yay, pkexec nem sudo: quem age é o coletor"
eq "" "$(grep -nE '\b(pacman|yay|pkexec|sudo)\b' "$app/bin/mkpacotes" || true)"

t "a janela se chama MkPacotes e nunca nasce maior que a tela (numa tela de 1366x768 ela cabe abaixo da ilha)"
casca=$(<"$qml/shell.qml")
has "$casca" 'title: "MkPacotes"'
like "$casca" 'implicitWidth: Math\.min\(1220, .*screen'
like "$casca" 'implicitHeight: Math\.min\(780, .*screen'
# O menor tamanho que ela aceita também cabe em 1366x768 com a folga.
like "$casca" 'minimumSize: Qt\.size\(([0-9]+), ([0-9]+)\)'
ok test "${BASH_REMATCH[1]}" -le 1306
ok test "${BASH_REMATCH[2]}" -le 658

t "o atalho do menu chama o lançador e usa o ícone do app"
atalho=$(<"$app/share/mkpacotes.desktop")
has "$atalho" "Exec=mkpacotes"
has "$atalho" "Icon=mkpacotes"
has "$atalho" "Name=MkPacotes"
exists "$app/share/mkpacotes.svg"
