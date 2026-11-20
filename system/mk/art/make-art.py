#!/usr/bin/env python3
# Gera as imagens do Mk a partir das fontes (Fraunces) e da linguagem Folha:
#   emblem.png      o desenho do fastfetch: arco, colunas, sol de cobre e o Mk
#                   (a marca vem de mark.png, feita por make-logo.py: rode ele antes)
#   grain.png       o grão que faz o preto nunca ser preto
# Rode de novo só quando mudar o desenho: python3 system/mk/art/make-art.py
import math, os, random
from PIL import Image, ImageDraw, ImageFont, ImageFilter

here = os.path.dirname(os.path.abspath(__file__))
BONE = (205, 196, 186)
COPPER = (206, 94, 50)
GROUND = (11, 11, 11)
FRAUNCES = "/usr/share/fonts/TTF/Fraunces[SOFT,WONK,opsz,wght].ttf"
FRAUNCES_I = "/usr/share/fonts/TTF/Fraunces-Italic[SOFT,WONK,opsz,wght].ttf"
MONO = "/usr/share/fonts/TTF/SpaceMonoNerdFont-Regular.ttf"


def font(path, size, weight=400, opsz=None):
    f = ImageFont.truetype(path, size)
    try:
        axes = {a["name"]: a for a in []}
        values = []
        for axis in f.get_variation_axes():
            name = axis["name"]
            name = name.decode() if isinstance(name, bytes) else name
            if name.lower().startswith("weight"):
                values.append(weight)
            elif name.lower().startswith("optical"):
                values.append(opsz or min(144, size))
            else:
                values.append(axis["default"])
        f.set_variation_by_axes(values)
    except Exception:
        pass
    return f


def mark(size):
    """A marca do Mk (mark.png, de make-logo.py) com `size` px de altura."""
    im = Image.open(os.path.join(here, "mark.png")).convert("RGBA")
    big = os.path.join(here, "logo", "mk-mark-2000.png")
    if os.path.exists(big):
        im = Image.open(big).convert("RGBA")
    return im.resize((round(im.width * size / im.height), size), Image.LANCZOS)


def old_mark(size):
    """O Mk antigo (k itálico), desenhado pela fonte. Não usado; fica de referência."""
    fm, fk = font(FRAUNCES, size, 380), font(FRAUNCES_I, size, 420)
    w = int(size * 1.9)
    h = int(size * 1.35)
    im = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    d = ImageDraw.Draw(im)
    bm = d.textbbox((0, 0), "M", font=fm)
    bk = d.textbbox((0, 0), "k", font=fk)
    total = (bm[2] - bm[0]) + (bk[2] - bk[0]) - size * 0.04
    x = (w - total) / 2 - bm[0]
    base = h * 0.82
    asc_m = fm.getmetrics()[0]
    d.text((x, base - asc_m), "M", font=fm, fill=BONE + (255,))
    xk = x + bm[2] - size * 0.04
    d.text((xk, base - fk.getmetrics()[0]), "k", font=fk, fill=COPPER + (255,))
    return im.crop(im.getbbox())


def grain(size=256, seed=7):
    rnd = random.Random(seed)
    im = Image.new("LA", (size, size))
    px = im.load()
    for y in range(size):
        for x in range(size):
            v = rnd.randint(0, 255)
            px[x, y] = (v, rnd.randint(0, 40))
    return im


def emblem(S=1024):
    """O desenho técnico do Mk num quadrado: moldura, arco triplo com raios de
    construção, colunas, horizonte, a linha e o sol de cobre, e o Mk."""
    k = 4  # desenha grande e reduz, para linhas finas e lisas
    W = S * k
    im = Image.new("RGBA", (W, W), (0, 0, 0, 0))
    d = ImageDraw.Draw(im)
    bone = lambda a: BONE + (int(255 * a),)
    lw = lambda px: max(1, int(px * k))

    L, T, R, B = 0.06 * W, 0.06 * W, 0.94 * W, 0.94 * W
    d.rectangle([L, T, R, B], outline=bone(0.22), width=lw(1.2))
    for (x, y, dx, dy) in [(L, T, -1, -1), (R, T, 1, -1), (L, B, -1, 1), (R, B, 1, 1)]:
        cx, cy = x + dx * 0.02 * W, y + dy * 0.02 * W
        d.line([cx + dx * 0.006 * W, cy, cx + dx * 0.03 * W, cy], fill=bone(0.4), width=lw(1.2))
        d.line([cx, cy + dy * 0.006 * W, cx, cy + dy * 0.03 * W], fill=bone(0.4), width=lw(1.2))
    # régua no topo e na esquerda
    n = 0
    x = L
    while x < R:
        x += W * 0.025
        n += 1
        d.line([x, T, x, T + (0.018 if n % 5 == 0 else 0.009) * W], fill=bone(0.35 if n % 5 == 0 else 0.2), width=lw(1))
    n = 0
    y = T
    while y < B:
        y += W * 0.025
        n += 1
        d.line([L, y, L + (0.018 if n % 5 == 0 else 0.009) * W, y], fill=bone(0.35 if n % 5 == 0 else 0.2), width=lw(1))
    # grade de pontos
    step = W * 0.05
    yy = T + step
    while yy < B:
        xx = L + step
        while xx < R:
            r = 1.2 * k
            d.ellipse([xx - r, yy - r, xx + r, yy + r], fill=bone(0.12))
            xx += step
        yy += step

    cx, hy, r1 = W * 0.5, W * 0.68, W * 0.3
    r2, r3 = r1 * 0.8, r1 * 0.46
    for a in range(15, 180, 15):
        t = math.radians(a)
        d.line([cx + math.cos(t) * r3, hy - math.sin(t) * r3, cx + math.cos(t) * r1 * 1.18, hy - math.sin(t) * r1 * 1.18],
               fill=bone(0.1), width=lw(1))
    rr = r1 * 1.18
    d.ellipse([cx - rr, hy - rr, cx + rr, hy + rr], outline=bone(0.09), width=lw(1))
    for r, a, w in [(r1, 0.75, 2.2), (r2, 0.4, 1.4), (r3, 0.3, 1.2)]:
        d.arc([cx - r, hy - r, cx + r, hy + r], 180, 360, fill=bone(a), width=lw(w))
    for x, a in [(cx - r1, 0.55), (cx - r2, 0.25), (cx + r2, 0.25), (cx + r1, 0.55)]:
        d.line([x, hy, x, B], fill=bone(a), width=lw(1.4 if a > 0.3 else 1))
    d.line([L, hy, R, hy], fill=bone(0.6), width=lw(1.4))
    d.line([cx - r1 * 1.45, hy - r1 * 0.9, cx - r1 * 0.2, hy - r1 * 0.9], fill=COPPER + (230,), width=lw(2.4))
    sr = W * 0.022
    d.ellipse([cx - sr, hy - r1 - sr, cx + sr, hy - r1 + sr], fill=COPPER + (255,))
    so = W * 0.045
    d.ellipse([cx - so, hy - r1 - so, cx + so, hy - r1 + so], outline=COPPER + (110,), width=lw(1.4))
    # cotas do vão, embaixo do arco
    yv = hy + (B - hy) * 0.55
    d.line([cx - r1, yv, cx + r1, yv], fill=bone(0.35), width=lw(1))
    for x in (cx - r1, cx + r1):
        d.line([x - 0.008 * W, yv + 0.008 * W, x + 0.008 * W, yv - 0.008 * W], fill=bone(0.5), width=lw(1.2))
    fmono = font(MONO, int(0.026 * W))
    d.text((cx, yv - 0.012 * W), "VÃO", font=fmono, fill=bone(0.55), anchor="ms")

    im = im.resize((S, S), Image.LANCZOS)
    # o Mk dentro do arco
    m = mark(int(S * 0.17))
    im.alpha_composite(m, (int(S * 0.5 - m.width / 2), int(S * 0.68 - m.height - S * 0.035)))
    return im


if __name__ == "__main__":
    emblem().save(os.path.join(here, "emblem.png"))
    grain().save(os.path.join(here, "grain.png"))
    print("ok:", ", ".join(sorted(f for f in os.listdir(here) if f.endswith(".png"))))
