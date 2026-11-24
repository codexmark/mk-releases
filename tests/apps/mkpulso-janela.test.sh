#!/usr/bin/env bash
# apps/mkpulso: a janela (qml/) e o lançador (bin/mkpulso). O desenho se vê no
# bin/dev-shot; aqui: a janela só usa as cores do Mk, só pede ao coletor o que
# o contrato tem, os dados de exemplo seguem o contrato, e o lançador abre uma
# janela só.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mkpulso
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
exemplo=$TESTS/apps/mkpulso-exemplo

t "cores: só os tokens do Mk, e só no Theme.qml"
tokens=" #0b0b0b #141211 #1b1917 #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml | sort -u)
eq "" "$(grep -lE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml | grep -v '/Theme.qml$' || true)"
out=$(grep -ohE 'Qt\.rgba\([^)]*\)' "$qml"/*.qml | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255|0, 0, 0, 0)' || true)
eq "" "$out"

t "toda página que a janela carrega existe"
while read -r f; do
  exists "$qml/$f"
done < <(grep -ohE '"[A-Z][A-Za-z]+\.qml"' "$qml/App.qml" | tr -d '"' | sort -u)

t "todo tipo próprio usado na janela tem o arquivo dele"
for tipo in $(grep -ohE '^\s*(Botao|Campo|Segmento|Rotulo|Cabecalho|Cartao|Grafico|Chave|App) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  exists "$qml/$tipo.qml"
done

t "a janela só pede ao coletor subcomandos que estão no contrato"
n=0
while read -r sub; do
  n=$((n + 1))
  has "$contrato" "mkpulso-coletor $sub" "subcomando fora do contrato: $sub"
done < <(grep -ohE 'pedir\((args\()?\["[a-z]+"' "$qml"/*.qml | grep -oE '"[a-z]+"' | tr -d '"' | sort -u)
ok test "$n" -ge 6

t "os nomes de sinal e de ação que a janela manda são os do contrato"
for palavra in encerrar matar pausar continuar iniciar parar reiniciar habilitar desabilitar ligar desligar; do
  has "$contrato" "$palavra" "o contrato não tem: $palavra"
  ok grep -q "\"$palavra\"" "$qml"/Processos.qml "$qml"/Servicos.qml "$qml"/Inicio.qml
done

t "os dados de exemplo são JSON e têm os campos do contrato que a janela lê"
for f in watch sistema inicio servicos registro; do
  ok jq -e . "$exemplo/$f.json" >/dev/null
done
ok jq -e '.t_ms and .maquina and .nucleos and (.cpu | has("pct") and has("por_nucleo") and has("temp_c") and has("carga"))
  and (.mem | has("total_kb") and has("usado_kb") and has("cache_kb") and has("swap_total_kb"))
  and (.totais | has("processos") and has("grupos")) and (.grupos | length > 0)' "$exemplo/watch.json" >/dev/null
ok jq -e '[.grupos[] | has("id") and has("tipo") and has("nome") and has("unidade") and has("janelas") and has("n") and has("pid")
  and has("cpu_pct") and has("mem_kb") and has("disco_bps") and has("gpu_pct") and has("energia") and has("media") and has("processos")] | all' \
  "$exemplo/watch.json" >/dev/null
ok jq -e '[.grupos[].processos[] | has("pid") and has("nome") and has("cmd") and has("usuario") and has("estado") and has("cpu_pct")
  and has("mem_kb") and has("nice") and has("inicio_s")] | all' "$exemplo/watch.json" >/dev/null
ok jq -e '[.grupos[].tipo] | all(. == "app" or . == "servico" or . == "sistema")' "$exemplo/watch.json" >/dev/null
ok jq -e '[.servicos[] | has("unidade") and has("estado") and has("habilitado") and has("descricao") and has("mem_kb")] | all' "$exemplo/servicos.json" >/dev/null
ok jq -e '[.itens[] | has("id") and has("origem") and has("nome") and has("comando") and has("ligado") and has("mudavel")] | all' "$exemplo/inicio.json" >/dev/null

t "nada de dados da máquina de quem escreveu nos exemplos"
eq "" "$(grep -lE '/home/[a-z]+/|codexmark' "$exemplo"/*.json || true)"

# --- o lançador ---------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml"
cp "$app/bin/mkpulso" "$casa/bin/mkpulso"
printf '#!/bin/sh\n' >"$casa/bin/mkpulso-coletor"
chmod +x "$casa/bin/mkpulso-coletor"
stub notify-send
stub hyprctl '[[ $1 == clients ]] && cat "$SBX/clientes.json" 2>/dev/null; exit 0'
stub qs 'if [[ $1 == list ]]; then cat "$SBX/qs-list" 2>/dev/null; else echo "AREA=${MKPULSO_AREA:-}" >>"$SBX/calls"; fi'
abre() { bash "$casa/bin/mkpulso" "$@"; }

t "sem janela aberta: abre a janela do app (qs -p …/qml)"
ok abre
called "qs -p $casa/qml"
called "AREA="

t "com uma área: abre direto nela"
calls_reset
ok abre servicos
called "AREA=servicos"

t "área que não existe: uso e saída 2, sem abrir nada"
calls_reset
abre banana 2>/dev/null
eq 2 $?
not_called "qs -p"

t "janela já aberta: só traz para a frente, não abre outra"
calls_reset
echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
echo '[{"title":"Outra","address":"0x1"},{"title":"MkPulso","address":"0x55aa"}]' >"$SBX/clientes.json"
ok abre
called 'hl.dsp.focus({ window = "address:0x55aa" })'
not_called "qs -p"

t "um endereço torto vindo do Hyprland não entra no comando: abre a janela normalmente"
calls_reset
echo '[{"title":"MkPulso","address":"0x1\" }) os.execute(\"x"}]' >"$SBX/clientes.json"
ok abre
not_called "dispatch"
called "qs -p $casa/qml"

t "sem o coletor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$SBX/qs-list" "$casa/bin/mkpulso-coletor"
out=$(abre 2>&1)
eq 1 $?
has "$out" "coletor"
called "notify-send"
not_called "qs -p"

t "a janela nunca nasce maior que a tela (numa tela de 1366x768 ela cabe abaixo da ilha)"
# Visto na VM (1256x651): flutuando em 1220x780, o alto da janela ficava
# cortado atrás da ilha.
casca=$(<"$qml/shell.qml")
like "$casca" 'implicitWidth: Math\.min\(1220, .*screen'
like "$casca" 'implicitHeight: Math\.min\(780, .*screen'
