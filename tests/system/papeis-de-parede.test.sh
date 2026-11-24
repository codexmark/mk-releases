#!/usr/bin/env bash
# shell/mare/wallpapers: os papéis de parede que vêm com o Mk. Dois conjuntos
# na mesma pasta: os seis autorais (00-N-nome, que o painel mostra primeiro) e
# os doze desenhados por código (01 a 12, do make-wallpapers.py).
. "$(dirname "$0")/../lib.sh"

dir=$REPO/shell/mare/wallpapers
# Largura, altura e formato de uma imagem, lidos do cabeçalho (JPEG ou PNG).
medida() { python3 -I -c '
import struct, sys
d = open(sys.argv[1], "rb").read()
if d[:8] == b"\x89PNG\r\n\x1a\n":
    w, h = struct.unpack(">II", d[16:24]); print(w, h, "png"); sys.exit(0)
if d[:2] == b"\xff\xd8":
    i = 2
    while i < len(d):
        if d[i] != 0xFF: i += 1; continue
        m = d[i + 1]
        if m in (0xC0, 0xC1, 0xC2):
            h, w = struct.unpack(">HH", d[i + 5:i + 9]); print(w, h, "jpg"); sys.exit(0)
        i += 2 + struct.unpack(">H", d[i + 2:i + 4])[0]
sys.exit("formato desconhecido: " + sys.argv[1])' "$1"; }

t "os seis autorais estão na pasta, com os nomes e a ordem do autor"
autorais=(00-1-entropia 00-2-nucleo 00-3-origem 00-4-monolito 00-5-singularidade 00-6-mare)
for n in "${autorais[@]}"; do
  exists "$dir/$n.jpg"
done
eq 6 "$(find "$dir" -name '00-*' | wc -l)"

t "os doze desenhados por código continuam, com os mesmos nomes (quem escolheu um não o perde)"
eq 12 "$(find "$dir" -name '[01][0-9]-*.jpg' ! -name '00-*' | wc -l)"
for n in 01-arcos 05-eclipse 12-catedral; do
  exists "$dir/$n.jpg"
done

t "os autorais vêm primeiro na ordem do painel (a pasta em ordem de nome)"
primeiros=$(find "$dir" -type f -printf '%f\n' | sort | head -6 | tr '\n' ' ')
eq "00-1-entropia.jpg 00-2-nucleo.jpg 00-3-origem.jpg 00-4-monolito.jpg 00-5-singularidade.jpg 00-6-mare.jpg " "$primeiros"

t "todo arquivo da pasta é uma imagem que o painel lista (jpg, jpeg, png ou webp), com nome NN-..."
while read -r f; do
  like "$f" '^[0-9]{2}-[a-z0-9-]+\.(jpg|jpeg|png|webp)$' "nome fora do padrão: $f"
done < <(find "$dir" -type f -printf '%f\n')

t "toda imagem é paisagem em 16:9, com pelo menos 1600 pontos de largura"
for f in "$dir"/*; do
  read -r w h _ < <(medida "$f") || { _fail "não li $f"; continue; }
  ((w >= 1600)) || _fail "$(basename "$f"): só $w de largura"
  ok python3 -I -c "import sys; sys.exit(0 if abs($w / $h - 16 / 9) < 0.01 else 1)"
done

t "nenhuma imagem pesa mais de 1,5 MB (a pasta inteira vai em cada ~/.config/mk)"
for f in "$dir"/*; do
  ((($(stat -c %s "$f")) <= 1500000)) || _fail "$(basename "$f") tem $(($(stat -c %s "$f") / 1024)) KiB"
done

t "o gerador não apaga nem regrava os autorais: só escreve os nomes dele"
gen=$(<"$REPO/system/mk/art/make-wallpapers.py")
hasnt "$gen" "rmtree"
hasnt "$gen" '"00-'
