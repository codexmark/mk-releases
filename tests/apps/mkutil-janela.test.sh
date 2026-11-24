#!/usr/bin/env bash
# apps/mkutil: as janelas (qml/) e os dois lançadores (bin/mkinstalar e
# bin/mkcomprimir). O desenho se vê no bin/dev-shot; aqui: as janelas só usam
# as cores do Mk, só pedem ao motor o que o contrato tem, os dados de exemplo
# seguem o contrato, e cada lançador abre a janela certa nos arquivos dados.
. "$(dirname "$0")/../lib.sh"

app=$REPO/apps/mkutil
qml=$app/qml
contrato=$(<"$app/CONTRATO.md")
exemplo=$TESTS/apps/mkutil-exemplo

t "cores: só os tokens do Mk, e só no Theme.qml"
tokens=" #0b0b0b #141211 #1b1917 #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml | sort -u)
eq "" "$(grep -lE '#[0-9a-fA-F]{6}\b' "$qml"/*.qml | grep -v '/Theme.qml$' || true)"
out=$(grep -ohE 'Qt\.rgba\([^)]*\)' "$qml"/*.qml | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255|0, 0, 0, 0)' || true)
eq "" "$out"
# Nem cor por nome ("red", "white") nem cor feita na hora.
eq "" "$(grep -nE 'color: *"[a-z]+"|Qt\.(hsla|hsva|color|lighter|darker|tint)\(' "$qml"/*.qml || true)"
# Os ícones do menu também.
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta num ícone: $c"
done < <(grep -ohE '#[0-9a-fA-F]{6}\b' "$app"/share/*.svg | sort -u)

t "o Theme e os componentes pequenos são os do Maré, iguais aos do MkPulso"
for f in Theme Botao Segmento Rotulo Chave; do
  eq "$(<"$REPO/apps/mkpulso/qml/$f.qml")" "$(<"$qml/$f.qml")" "$f.qml"
done

t "toda página que a janela carrega existe"
n=0
while read -r f; do
  n=$((n + 1))
  exists "$qml/$f"
done < <(grep -ohE '"[A-Z][A-Za-z]+\.qml"' "$qml/shell.qml" | tr -d '"' | sort -u)
eq 2 "$n"

t "todo tipo próprio usado nas janelas tem o arquivo dele"
for tipo in $(grep -ohE '^\s*(Botao|Campo|Segmento|Rotulo|Cabecalho|Chave|Barra|Solta) \{' "$qml"/*.qml | tr -d ' {' | sort -u); do
  exists "$qml/$tipo.qml"
done
exists "$qml/Motor.qml"
exists "$qml/Util.js"

t "as janelas só pedem ao motor subcomandos que estão no contrato, e usam todos"
n=0
while read -r sub; do
  n=$((n + 1))
  has "$contrato" "\`mkutil-motor $sub" "subcomando fora do contrato: $sub"
done < <({
  grep -ohE 'pedir\(\["[a-z]+"' "$qml"/*.qml
  grep -ohE '\["[a-z]+", (ficha|midia)\.arquivo' "$qml/Util.js"
} | grep -oE '"[a-z]+"' | tr -d '"' | sort -u)
eq 8 "$n"

t "as opções que as janelas mandam são as do contrato"
n=0
while read -r opcao; do
  n=$((n + 1))
  has "$contrato" "$opcao" "opção fora do contrato: $opcao"
done < <(grep -ohE '"--[a-z-]+"' "$qml"/*.qml "$qml/Util.js" | tr -d '"' | sort -u)
ok test "$n" -ge 12

t "as predefinições, os formatos e as fases que as janelas usam são os do contrato"
for palavra in menor equilibrio alta caber resolucao sem-audio gif webp imagem; do
  has "$contrato" "| \`$palavra\` |" "predefinição fora da tabela do contrato: $palavra"
  ok grep -q "\"$palavra\"" "$qml/Util.js"
done
for palavra in copiando extraindo pacman atalho fim cancelado codificando plano; do
  has "$contrato" "\"fase\": \"$palavra\"" "fase fora do contrato: $palavra"
done
for palavra in copiando extraindo pacman atalho fim cancelado codificando; do
  ok grep -q "\"$palavra\"" "$qml"/Instalar.qml "$qml"/Comprimir.qml "$qml/Util.js"
done
for palavra in h264 h265 av1 jpeg webp avif; do
  has "$contrato" "$palavra"
  ok grep -q "\"$palavra\"" "$qml"/Comprimir.qml "$qml/Util.js"
done

t "as janelas não rodam nada além do motor (e do jq, para os exemplos)"
eq "" "$(grep -n 'command:' "$qml"/*.qml | grep -v 'Motor.qml' || true)"
eq "" "$(grep -nE '"(sh|bash|sudo|pkexec|ffmpeg|rm)"' "$qml"/*.qml "$qml/Util.js" || true)"

t "os títulos das janelas são MkInstalar e MkComprimir"
casca=$(<"$qml/shell.qml")
has "$casca" 'title: comprimir ? "MkComprimir" : "MkInstalar"'

t "as janelas nunca nascem maiores que a tela (numa tela de 1366x768 cabem abaixo da ilha)"
like "$casca" 'implicitWidth: Math\.min\(comprimir \? 940 : 820, .*screen'
like "$casca" 'implicitHeight: Math\.min\(comprimir \? 640 : 580, .*screen'

t "os dados de exemplo são JSON e existem para tudo que as janelas pedem"
for f in inspecionar-appimage inspecionar-pacote inspecionar-compactado instalar instalados desinstalar codificadores midia-video midia-imagem comprimir descartar; do
  ok jq -e . "$exemplo/$f.json" >/dev/null
done

t "os exemplos têm os campos do contrato que as janelas leem"
ficha='has("arquivo") and has("tipo") and has("id") and has("nome") and has("versao") and has("descricao") and has("tamanho_bytes")
  and has("destino") and has("atalho") and has("ja_instalado") and has("precisa_senha") and has("plano") and has("avisos")
  and has("appimage") and has("pacote") and has("compactado")'
for tipo in appimage pacote compactado; do
  ok jq -e "$ficha and .tipo == \"$tipo\" and (.$tipo | type == \"object\")" "$exemplo/inspecionar-$tipo.json" >/dev/null
done
ok jq -e '.appimage | has("tipo") and has("tem_desktop") and has("tem_icone") and has("fuse")' "$exemplo/inspecionar-appimage.json" >/dev/null
ok jq -e '.pacote | has("nome") and has("versao") and has("descricao") and has("arquitetura") and has("tamanho_instalado_bytes")
  and has("dependencias") and has("instalado")' "$exemplo/inspecionar-pacote.json" >/dev/null
ok jq -e '.compactado | has("formato") and has("entradas") and has("tamanho_total_bytes") and has("pasta_unica") and has("executaveis")
  and has("atalhos") and has("executavel_sugerido")' "$exemplo/inspecionar-compactado.json" >/dev/null
ok jq -e '[.itens[] | has("id") and has("tipo") and has("nome") and has("versao") and has("origem") and has("destino") and has("atalho")
  and has("icone") and has("instalado_em_s") and has("tamanho_bytes") and has("existe")] | all' "$exemplo/instalados.json" >/dev/null
ok jq -e 'has("ffmpeg") and has("codecs") and has("aceleracao") and has("imagens") and has("animados")' "$exemplo/codificadores.json" >/dev/null
for f in midia-video midia-imagem; do
  ok jq -e 'has("arquivo") and has("tipo") and has("tamanho_bytes") and has("duracao_s") and has("largura") and has("altura") and has("fps")
    and has("codec") and has("taxa_bps") and has("audio")' "$exemplo/$f.json" >/dev/null
done
ok jq -e '.tipo == "video" and (.audio | has("codec") and has("taxa_bps") and has("canais"))' "$exemplo/midia-video.json" >/dev/null
ok jq -e '.tipo == "imagem" and .duracao_s == null and .audio == null' "$exemplo/midia-imagem.json" >/dev/null
# As respostas em fases: uma lista, cada item com "fase", e a última é o fim.
for f in instalar comprimir; do
  ok jq -e 'type == "array" and ([.[] | has("fase")] | all) and (.[-1].fase == "fim") and .[-1].ok' "$exemplo/$f.json" >/dev/null
done
ok jq -e '.[-1] | has("id") and has("tipo") and has("nome") and has("destino") and has("atalho") and has("avisos")' "$exemplo/instalar.json" >/dev/null
ok jq -e '(.[0] | .fase == "plano" and has("saida") and has("passadas") and has("largura") and has("altura"))
  and ([.[] | select(.fase == "codificando") | has("passada") and has("passadas") and has("pct") and has("velocidade") and has("resta_s")] | all)
  and (.[-1] | has("saida") and has("antes_bytes") and has("depois_bytes") and has("economia_pct") and has("maior"))' "$exemplo/comprimir.json" >/dev/null

t "nada de dados da máquina de quem escreveu nos exemplos"
eq "" "$(grep -lE '/home/[a-z]+/|codexmark' "$exemplo"/*.json || true)"

t "os atalhos de menu: abrem com os arquivos (%F) e não se oferecem para .deb nem .flatpakref"
for nome in mkinstalar mkcomprimir; do
  d=$(<"$app/share/$nome.desktop")
  has "$d" "Exec=$nome %F"
  has "$d" "Icon=$nome"
  has "$d" "Terminal=false"
  exists "$app/share/$nome.svg"
done
d=$(<"$app/share/mkinstalar.desktop")
has "$d" "application/vnd.appimage;"
has "$d" "application/x-xz-compressed-tar;"
has "$d" "application/zip;"
hasnt "$d" "debian"
hasnt "$d" "flatpak"
hasnt "$d" "rpm"
has "$(<"$app/share/mkcomprimir.desktop")" "video/mp4;"

# --- os lançadores ------------------------------------------------------------
casa=$SBX/app
mkdir -p "$casa/bin" "$casa/qml" "$SBX/onde estou"
cp "$app/bin/mkinstalar" "$app/bin/mkcomprimir" "$casa/bin/"
printf '#!/bin/sh\n' >"$casa/bin/mkutil-motor"
chmod +x "$casa/bin/mkutil-motor"
stub notify-send
stub hyprctl '[[ $1 == clients ]] && cat "$SBX/clientes.json" 2>/dev/null; exit 0'
stub qs 'if [[ $1 == list ]]; then cat "$SBX/qs-list" 2>/dev/null; else echo "JANELA=${MKUTIL_JANELA:-} ARQUIVOS=${MKUTIL_ARQUIVOS:-}" >>"$SBX/calls"; fi'
abre() { (cd "$SBX/onde estou" && bash "$casa/bin/$1" "${@:2}"); }

t "os dois lançadores são o mesmo script, só muda a janela"
eq "titulo=MkInstalar janela=instalar" "$(grep -E '^(titulo|janela)=' "$app/bin/mkinstalar" | tr '\n' ' ' | sed 's/ $//')"
eq "titulo=MkComprimir janela=comprimir" "$(grep -E '^(titulo|janela)=' "$app/bin/mkcomprimir" | tr '\n' ' ' | sed 's/ $//')"
eq "$(grep -vE '^(#|titulo=|janela=)' "$app/bin/mkinstalar")" "$(grep -vE '^(#|titulo=|janela=)' "$app/bin/mkcomprimir")"

for par in "mkinstalar instalar MkInstalar" "mkcomprimir comprimir MkComprimir"; do
  read -r cmd janela titulo <<<"$par"

  t "$cmd: sem janela aberta, abre a janela certa (qs -p …/qml), sem arquivo"
  calls_reset
  rm -f "$SBX/qs-list" "$SBX/clientes.json"
  ok abre "$cmd"
  called "qs -p $casa/qml"
  called "JANELA=$janela ARQUIVOS=[]"

  t "$cmd: os arquivos chegam inteiros e absolutos, mesmo com espaço, aspas e começando com -"
  calls_reset
  # shellcheck disable=SC2016 # é o texto, não a expansão, que tem de chegar
  ok abre "$cmd" "/tmp/a b.zip" "-rf 'x' \"y\".mp4" "sub/é.png" '$(touch pwn)`z`'
  # shellcheck disable=SC2016
  called "JANELA=$janela ARQUIVOS=[\"/tmp/a b.zip\",\"$SBX/onde estou/-rf 'x' \\\"y\\\".mp4\",\"$SBX/onde estou/sub/é.png\",\"$SBX/onde estou/"'$(touch pwn)`z`"]'
  absent "$SBX/onde estou/pwn"

  t "$cmd: janela já aberta e nenhum arquivo: só traz para a frente, não abre outra"
  calls_reset
  echo "Config path: $casa/qml/shell.qml" >"$SBX/qs-list"
  echo "[{\"title\":\"MkPulso\",\"address\":\"0x1\"},{\"title\":\"$titulo\",\"address\":\"0x55aa\"}]" >"$SBX/clientes.json"
  ok abre "$cmd"
  called 'hl.dsp.focus({ window = "address:0x55aa" })'
  not_called "qs -p"

  t "$cmd: a outra janela do MkUtil aberta não conta: abre a sua"
  calls_reset
  outra=$([[ $titulo == MkInstalar ]] && echo MkComprimir || echo MkInstalar)
  echo "[{\"title\":\"$outra\",\"address\":\"0x77\"}]" >"$SBX/clientes.json"
  ok abre "$cmd"
  not_called "dispatch"
  called "qs -p $casa/qml"

  t "$cmd: janela já aberta e um arquivo novo: abre nele"
  calls_reset
  echo "[{\"title\":\"$titulo\",\"address\":\"0x55aa\"}]" >"$SBX/clientes.json"
  ok abre "$cmd" /tmp/novo.zip
  not_called "dispatch"
  called 'ARQUIVOS=["/tmp/novo.zip"]'

  t "$cmd: um endereço torto vindo do Hyprland não entra no comando: abre a janela normalmente"
  calls_reset
  echo "[{\"title\":\"$titulo\",\"address\":\"0x1\\\" }) os.execute(\\\"x\"}]" >"$SBX/clientes.json"
  ok abre "$cmd"
  not_called "dispatch"
  called "qs -p $casa/qml"
done

t "sem o motor: avisa na tela, diz o que fazer e sai com 1"
calls_reset
rm -f "$SBX/qs-list" "$casa/bin/mkutil-motor"
for cmd in mkinstalar mkcomprimir; do
  out=$(abre "$cmd" 2>&1)
  eq 1 $?
  has "$out" "motor"
  has "$out" "mk atualizar"
done
called "notify-send"
not_called "qs -p"

t "sem o Quickshell: avisa e sai com 1"
calls_reset
semqs=$(path_sem qs)
out=$(PATH=$semqs abre mkinstalar 2>&1)
eq 1 $?
has "$out" "Quickshell"
called "notify-send"
