#!/usr/bin/env python3
# O logo do Mk em vetor: "Mk" em Fraunces reta (M em osso, k em cobre) sobre
# um plano cartesiano. Eixo vertical à esquerda e a linha de cota embaixo (em
# cobre sob o k); guias quase invisíveis nas alturas e hastes; as diagonais de
# construção do M e do k; e nós nas quinas que as definem.
#
# As letras viram contorno (path) a partir da própria fonte, e as guias saem
# dos pontos reais desse contorno: muda a fonte ou o peso, o desenho acompanha.
#
# Duas formas: o logo inteiro (guias, diagonais e nós), para tamanhos grandes
# (boot, cartaz), e a marca compacta (as letras e os eixos), para tamanhos
# pequenos (ilha, login, bloqueio), onde as linhas finas sumiriam.
#
#   python3 system/mk/art/make-logo.py      gera logo/*.svg e *.png, a marca e
#                                           as cópias que o boot, o login e o
#                                           bloqueio usam
#   python3 system/mk/art/make-logo.py entrada   só as peças da tela de entrada
#                                           (as letras em camadas e a geometria)
import os
import shutil
import sys
import subprocess
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen

here = os.path.dirname(os.path.abspath(__file__))
out = os.path.join(here, "logo")
FONT = "/usr/share/fonts/TTF/Fraunces[SOFT,WONK,opsz,wght].ttf"
BONE, COPPER, GROUND = "#cdc4ba", "#ce5e32", "#0b0b0b"
WGHT, KERN = 380, -60

font = instancer.instantiateVariableFont(TTFont(FONT), {"wght": WGHT, "opsz": 144, "SOFT": 0, "WONK": 0})
glyphs = font.getGlyphSet()
cmap = font.getBestCmap()


def glyph_path(ch, ox):
    """Contorno da letra em coordenadas do SVG (y para baixo), origem em ox."""
    name = cmap[ord(ch)]
    pen = SVGPathPen(glyphs)
    glyphs[name].draw(TransformPen(pen, (1, 0, 0, -1, ox, 0)))
    return pen.getCommands(), glyphs[name].width


# Posições (unidades da fonte; no SVG, y = -altura).
M_ADV = glyphs[cmap[ord("M")]].width
KX = M_ADV + KERN                     # origem do k
K_RIGHT = KX + 954                    # fim da perna do k
HEIGHTS = [0, 156, 867, 1400, 1469]   # base, vértice do M, altura-x, maiúsculas, ascendente do k
VERTS = [215, 722, 1320, KX + 220, KX + 914]  # hastes do M, vértice, haste do k, fim do braço
DIAGS = [((408, 1390), (722, 156)), ((1180, 1388), (804, 415)),          # as diagonais do M
         ((KX + 283, 321), (KX + 914, 867)), ((KX + 431, 479), (KX + 853, 74))]  # braço e perna do k
# Quinas: o alto das duas diagonais do M e o vértice; no k, o alto da haste,
# onde o braço sai da haste, onde a perna encontra o braço, a ponta do braço e o pé da perna.
NODES = [(408, 1390, BONE), (722, 156, BONE), (1180, 1388, BONE),
         (KX + 220, 1469, COPPER), (KX + 303, 360, COPPER), (KX + 548, 600, COPPER),
         (KX + 914, 867, COPPER), (KX + 853, 74, COPPER)]

AX, AY = -230, 150                    # eixo vertical (x) e linha de cota (y, abaixo da base)
TOP, RIGHT = -1640, K_RIGHT + 200     # fim dos eixos
VB = (-520, -1900, RIGHT + 300 + 520, 1900 + AY + 300)


def svg(background):
    m_path, _ = glyph_path("M", 0)
    k_path, _ = glyph_path("k", KX)
    p = []
    p.append(f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{VB[0]} {VB[1]} {VB[2]} {VB[3]}">')
    p.append("<title>Mk</title>")
    if background:
        p.append(f'<rect x="{VB[0]}" y="{VB[1]}" width="{VB[2]}" height="{VB[3]}" fill="{GROUND}"/>')
    # Guias e diagonais ficam dentro do plano (entre os eixos).
    p.append(f'<defs><clipPath id="plano"><rect x="{AX}" y="{TOP}" width="{RIGHT - AX}" height="{AY - TOP}"/></clipPath></defs>')
    p.append('<g clip-path="url(#plano)">')
    # Guias: alturas e hastes, quase invisíveis.
    p.append(f'<g stroke="{BONE}" stroke-width="4" opacity="0.13">')
    for h in HEIGHTS:
        p.append(f'<line x1="{AX}" y1="{-h}" x2="{RIGHT}" y2="{-h}"/>')
    for x in VERTS:
        p.append(f'<line x1="{x}" y1="{AY}" x2="{x}" y2="{TOP}"/>')
    p.append("</g>")
    # Diagonais de construção, um pouco além das letras.
    p.append(f'<g stroke-width="4" opacity="0.2">')
    for i, ((x1, y1), (x2, y2)) in enumerate(DIAGS):
        dx, dy = x2 - x1, y2 - y1
        e = 0.35
        color = BONE if i < 2 else COPPER
        p.append(f'<line x1="{x1 - dx * e:.1f}" y1="{-(y1 - dy * e):.1f}" x2="{x2 + dx * e:.1f}" y2="{-(y2 + dy * e):.1f}" stroke="{color}"/>')
    p.append("</g>")
    p.append("</g>")
    # Eixos: vertical à esquerda e a cota embaixo; cobre sob o k.
    p.append(f'<g stroke="{BONE}" stroke-width="10" stroke-linecap="square" opacity="0.62">')
    p.append(f'<line x1="{AX}" y1="{AY}" x2="{RIGHT}" y2="{AY}"/>')
    p.append(f'<line x1="{AX}" y1="{AY}" x2="{AX}" y2="{TOP}"/>')
    p.append(f'<line x1="{AX - 44}" y1="{TOP}" x2="{AX + 44}" y2="{TOP}"/>')
    p.append(f'<line x1="{RIGHT}" y1="{AY - 44}" x2="{RIGHT}" y2="{AY + 44}"/>')
    p.append("</g>")
    p.append(f'<line x1="{KX - 20}" y1="{AY}" x2="{RIGHT}" y2="{AY}" stroke="{COPPER}" stroke-width="14"/>')
    # As letras.
    p.append(f'<path d="{m_path}" fill="{BONE}"/>')
    p.append(f'<path d="{k_path}" fill="{COPPER}"/>')
    # Nós nas quinas: um furo escuro com aro, para aparecer até sobre a letra.
    for x, y, color in NODES:
        p.append(f'<circle cx="{x}" cy="{-y}" r="22" fill="{GROUND}" stroke="{color}" stroke-width="7"/>')
        p.append(f'<circle cx="{x}" cy="{-y}" r="5" fill="{color}"/>')
    p.append("</svg>")
    return "\n".join(p) + "\n"


def mark_svg():
    """A marca compacta: as letras e os eixos (o vertical à esquerda e a cota
    embaixo, cobre sob o k), com traço mais grosso para tamanhos pequenos."""
    m_path, _ = glyph_path("M", 0)
    k_path, _ = glyph_path("k", KX)
    ax, ay, top, right = -170, 120, -1560, K_RIGHT + 70
    vb = (ax - 40, top - 40, right - ax + 80, ay - top + 80)
    return "\n".join([
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{vb[0]} {vb[1]} {vb[2]} {vb[3]}">',
        "<title>Mk</title>",
        f'<g stroke="{BONE}" stroke-width="26" opacity="0.6">',
        f'<line x1="{ax}" y1="{ay}" x2="{right}" y2="{ay}"/>',
        f'<line x1="{ax}" y1="{ay + 13}" x2="{ax}" y2="{top}"/>',
        "</g>",
        f'<line x1="{KX - 20}" y1="{ay}" x2="{right}" y2="{ay}" stroke="{COPPER}" stroke-width="34"/>',
        f'<path d="{m_path}" fill="{BONE}"/>',
        f'<path d="{k_path}" fill="{COPPER}"/>',
        "</svg>",
    ]) + "\n"


def letter_svg(ch, ox, color):
    """Uma letra só, no quadro do logo inteiro: a entrada (sddm) anima o M e o
    k em camadas separadas, por cima da construção que ela mesma desenha."""
    path, _ = glyph_path(ch, ox)
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{VB[0]} {VB[1]} {VB[2]} {VB[3]}">'
            f'<path d="{path}" fill="{color}"/></svg>\n')


def login_assets(root):
    """O que a tela de entrada precisa para construir a marca em movimento: as
    duas letras em PNG e a geometria (em unidades da fonte) num .js do tema."""
    theme = os.path.join(root, "system/mk/sddm/mk")
    # O descanso de tela do Maré desenha a mesma construção: recebe cópias.
    shell_assets = os.path.join(root, "shell/mare/assets")
    for name, ch, ox, color in (("letra-m", "M", 0, BONE), ("letra-k", "k", KX, COPPER)):
        tmp = os.path.join(out, name + ".svg")
        with open(tmp, "w") as f:
            f.write(letter_svg(ch, ox, color))
        render(tmp, os.path.join(theme, name + ".png"), width=1600)
        os.remove(tmp)
    nodes = ", ".join(f"[{x}, {y}, {0 if c == BONE else 1}]" for x, y, c in NODES)
    diags = ", ".join(f"[{x1}, {y1}, {x2}, {y2}]" for (x1, y1), (x2, y2) in DIAGS)
    with open(os.path.join(theme, "geometria.js"), "w") as f:
        f.write(f""".pragma library

// A geometria da marca "Mk", para a tela de entrada desenhar a construção
// dela (eixos, guias, diagonais e nós) em movimento. GERADO: não edite à mão,
// rode system/mk/art/make-logo.py (lá diz de onde cada número vem).
// Unidades da fonte, y para cima (na tela, y = -altura). vb é o quadro das
// imagens letra-m.png e letra-k.png: [x, y, largura, altura], y para baixo.
var vb = [{VB[0]}, {VB[1]}, {VB[2]}, {VB[3]}];
var ax = {AX};          // o eixo vertical
var ay = {AY};          // a linha de cota, abaixo da base (para baixo)
var top = {TOP};        // o alto do eixo (para baixo: negativo é acima)
var right = {RIGHT};    // o fim da cota
var kx = {KX};          // onde o k começa: daqui em diante a cota é de cobre
// Alturas: base, vértice do M, altura-x, maiúsculas, ascendente do k.
var heights = {HEIGHTS};
// Hastes do M, o vértice dele, a haste do k e o fim do braço.
var verts = {VERTS};
// Diagonais de construção [x1, y1, x2, y2]: duas do M, o braço e a perna do k.
var diags = [{diags}];
// Nós nas quinas [x, y, cor]: 0 = osso (do M), 1 = cobre (do k).
var nodes = [{nodes}];
""")
    for name in ("letra-m.png", "letra-k.png", "geometria.js"):
        shutil.copy(os.path.join(theme, name), os.path.join(shell_assets, name))


def render(svg_path, png_path, width=None, height=None):
    size = ["-w", str(width)] if width else ["-h", str(height)]
    subprocess.run(["rsvg-convert", *size, "-o", png_path, svg_path], check=True)


if __name__ == "__main__":
    os.makedirs(out, exist_ok=True)
    if sys.argv[1:] == ["entrada"]:
        # Só as peças da tela de entrada, sem regravar o resto.
        login_assets(os.path.normpath(os.path.join(here, "..", "..", "..")))
        print("ok: letra-m.png, letra-k.png, geometria.js (entrada e shell/mare/assets)")
        sys.exit(0)
    for name, bg in (("mk-logo", True), ("mk-logo-transparente", False)):
        path = os.path.join(out, name + ".svg")
        with open(path, "w") as f:
            f.write(svg(bg))
        for width in (8000, 2400):
            png = os.path.join(out, f"{name}-{width}.png")
            subprocess.run(["rsvg-convert", "-w", str(width), "-o", png, path], check=True)
    mark = os.path.join(out, "mk-mark.svg")
    with open(mark, "w") as f:
        f.write(mark_svg())
    render(mark, os.path.join(out, "mk-mark-2000.png"), width=2000)

    # As cópias que o resto do Mk usa.
    root = os.path.normpath(os.path.join(here, "..", "..", ".."))
    render(mark, os.path.join(here, "mark.png"), height=160)
    render(mark, os.path.join(root, "shell/mare/assets/mark.png"), height=120)
    shutil.copy(mark, os.path.join(root, "shell/mare/assets/mark.svg"))
    render(os.path.join(out, "mk-logo-transparente.svg"), os.path.join(root, "system/mk/plymouth/mk/logo.png"), width=1200)
    login_assets(root)
    print("ok:", ", ".join(sorted(os.listdir(out))))
