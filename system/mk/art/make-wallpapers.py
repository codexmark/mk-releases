#!/usr/bin/env python3
# Os papéis de parede do Mk, desenhados por código: cada um é um SVG montado
# aqui, renderizado em 2560x1440 (rsvg-convert) e finalizado com um grão leve
# (que também evita faixas nos degradês). Variam de assunto e de cor, mas falam
# a mesma língua da linha Folha: preto quente, traço fino, formas de desenho
# técnico e arquitetura, um ponto de cobre. Fundo sempre calmo no alto (onde
# fica a ilha) e sem contraste forte onde as janelas ficam.
#
#   python3 system/mk/art/make-wallpapers.py [nome ...]
# Saída: shell/mare/wallpapers/*.jpg (o Maré lista essa pasta junto com a sua).
import math
import os
import random
import subprocess
import sys
import tempfile

import numpy as np
from PIL import Image

here = os.path.dirname(os.path.abspath(__file__))
root = os.path.normpath(os.path.join(here, "..", "..", ".."))
out = os.path.join(root, "shell", "mare", "wallpapers")
W, H = 2560, 1440
BONE, COPPER, GROUND = "#cdc4ba", "#ce5e32", "#0b0b0b"
MONO = "SpaceMono Nerd Font, Space Mono, monospace"


def svg(body, defs=""):
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}" width="{W}" height="{H}">'
            f"<defs>{defs}</defs>{body}</svg>")


def lin(id_, stops, x1=0, y1=0, x2=0, y2=1):
    s = "".join(f'<stop offset="{o}" stop-color="{c}" stop-opacity="{a}"/>' for o, c, a in
                [(t + (1,))[:3] if len(t) == 2 else t for t in stops])
    return f'<linearGradient id="{id_}" x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}">{s}</linearGradient>'


def rad(id_, stops, cx=0.5, cy=0.5, r=0.7):
    s = "".join(f'<stop offset="{o}" stop-color="{c}" stop-opacity="{a}"/>' for o, c, a in
                [(t + (1,))[:3] if len(t) == 2 else t for t in stops])
    return f'<radialGradient id="{id_}" cx="{cx}" cy="{cy}" r="{r}">{s}</radialGradient>'


def rect(fill):
    return f'<rect width="{W}" height="{H}" fill="{fill}"/>'


def label(x, y, text, color=BONE, op=0.5, size=17, anchor="start", spacing=4):
    return (f'<text x="{x}" y="{y}" fill="{color}" fill-opacity="{op}" font-family="{MONO}" font-size="{size}" '
            f'letter-spacing="{spacing}" text-anchor="{anchor}">{text}</text>')


def sun(x, y, r=14, glow=220, color=COPPER, id_="sun"):
    d = rad(id_, [(0, color, 0.55), (0.35, color, 0.16), (1, color, 0)], r=0.5)
    b = (f'<circle cx="{x}" cy="{y}" r="{glow}" fill="url(#{id_})"/>'
         f'<circle cx="{x}" cy="{y}" r="{r}" fill="{color}"/>'
         f'<circle cx="{x}" cy="{y}" r="{r * 2.3}" fill="none" stroke="{color}" stroke-opacity="0.4" stroke-width="1.5"/>')
    return d, b


# ── 01 arcos: uma galeria de arcos em perspectiva ──────────────────────
def arcos():
    defs = rad("bg", [(0, "#1a1613"), (1, GROUND)], cx=0.5, cy=0.58, r=0.75)
    vx, vy = W / 2, 830
    b = [rect("url(#bg)")]
    for i in range(-12, 13):  # piso: linhas que fogem para o ponto de fuga
        b.append(f'<line x1="{vx}" y1="{vy}" x2="{vx + i * 420}" y2="{H + 200}" stroke="{BONE}" stroke-opacity="0.07" stroke-width="1.5"/>')
    for i in range(8, -1, -1):
        s = 0.66 ** i
        w = 2100 * s
        base = vy + (H + 260 - vy) * s
        spring = vy - 330 * s
        op = 0.10 + 0.5 * s
        b.append(f'<path d="M{vx - w / 2} {base} L{vx - w / 2} {spring} A{w / 2} {w / 2} 0 0 1 {vx + w / 2} {spring} L{vx + w / 2} {base}" '
                 f'fill="{GROUND}" fill-opacity="{0.25 * s}" stroke="{BONE}" stroke-opacity="{op}" stroke-width="{1.2 + 2.2 * s}"/>')
        b.append(f'<line x1="{vx - w / 2}" y1="{base}" x2="{vx + w / 2}" y2="{base}" stroke="{BONE}" stroke-opacity="{op * 0.5}" stroke-width="1.2"/>')
    d2, s2 = sun(vx, vy - 6, r=9, glow=260)
    b.append(s2)
    b.append(label(110, H - 90, "GALERIA · CORTE AA", op=0.35))
    return svg("".join(b), defs + d2)


# ── 02 duna: camadas de areia ao entardecer ────────────────────────────
def duna():
    rnd = random.Random(2)
    defs = lin("sky", [(0, "#120f0e"), (0.38, "#2a1a15"), (0.62, "#6b3423"), (0.76, "#b9633a"), (0.84, "#e0a070")])
    b = [rect("url(#sky)"), f'<circle cx="1820" cy="1010" r="150" fill="#f1d3ae" fill-opacity="0.92"/>']
    cols = ["#8a4a2e", "#6a3724", "#4c281c", "#321b15", "#1e1210", "#120d0c"]
    for i, c in enumerate(cols):
        y0 = 1010 + i * 78
        ph = rnd.uniform(0, 6.28)
        amp = 46 + i * 16
        pts = [f"0,{H}"]
        for x in range(0, W + 40, 40):
            y = y0 + amp * math.sin(x / (520 + i * 70) + ph) + amp * 0.4 * math.sin(x / 190 + ph * 2)
            pts.append(f"{x},{y:.1f}")
        pts.append(f"{W},{H}")
        b.append(f'<polygon points="{" ".join(pts)}" fill="{c}"/>')
    return svg("".join(b), defs)


# ── 03 madeira: curvas de nível em volta de um rio ─────────────────────
def madeira():
    defs = rad("bg", [(0, "#12201c"), (1, "#090e0d")], cx=0.45, cy=0.5, r=0.8)
    river = "M-100 1180 C 420 1250, 620 760, 1020 800 S 1500 1220, 1880 900 S 2300 300, 2700 380"
    b = [rect("url(#bg)")]
    for w in range(1500, 60, -58):
        op = 0.10 + 0.22 * (1 - w / 1500)
        b.append(f'<path d="{river}" fill="none" stroke="#9fb8a6" stroke-opacity="{op:.3f}" stroke-width="{w}" stroke-linecap="round"/>')
        b.append(f'<path d="{river}" fill="none" stroke="#0c1614" stroke-width="{w - 2.4}" stroke-linecap="round"/>')
    b.append(f'<path d="{river}" fill="none" stroke="#1d3a36" stroke-width="54" stroke-linecap="round"/>')
    b.append(f'<path d="{river}" fill="none" stroke="#9fb8a6" stroke-opacity="0.5" stroke-width="1.6"/>')
    d2, s2 = sun(1020, 800, r=9, glow=150)
    b += [s2, f'<line x1="1020" y1="800" x2="1180" y2="640" stroke="{COPPER}" stroke-width="1.5" stroke-opacity="0.8"/>',
          label(1196, 634, "PORTO VELHO", color="#d9e3da", op=0.7), label(1196, 664, "8°45′ S · 63°54′ O", color="#d9e3da", op=0.4, size=14)]
    return svg("".join(b), defs + d2)


# ── 04 planta: uma planta baixa em papel heliográfico ──────────────────
def planta():
    rnd = random.Random(4)
    defs = (rad("bg", [(0, "#132238"), (1, "#0a1220")], r=0.8)
            + '<pattern id="g" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M40 0H0V40" fill="none" stroke="#bcd3ee" stroke-opacity="0.06"/></pattern>'
            + '<pattern id="G" width="200" height="200" patternUnits="userSpaceOnUse"><path d="M200 0H0V200" fill="none" stroke="#bcd3ee" stroke-opacity="0.09"/></pattern>')
    ink = "#d6e4f5"
    b = [rect("url(#bg)"), rect("url(#g)"), rect("url(#G)")]
    x0, y0 = 520, 360
    rooms = [(0, 0, 560, 400), (560, 0, 440, 240), (560, 240, 440, 480), (0, 400, 560, 320), (1000, 0, 520, 720)]
    for x, y, w, h in rooms:
        b.append(f'<rect x="{x0 + x}" y="{y0 + y}" width="{w}" height="{h}" fill="none" stroke="{ink}" stroke-opacity="0.75" stroke-width="5"/>')
        b.append(f'<rect x="{x0 + x + 12}" y="{y0 + y + 12}" width="{w - 24}" height="{h - 24}" fill="none" stroke="{ink}" stroke-opacity="0.3" stroke-width="1.2"/>')
    for x, y, r, a in [(560, 120, 90, 0), (280, 400, 90, 1), (1000, 460, 90, 0), (760, 240, 80, 1)]:  # portas
        cx, cy = x0 + x, y0 + y
        if a:
            b.append(f'<path d="M{cx} {cy} h{r} A{r} {r} 0 0 1 {cx} {cy + r}" fill="#0f1b2d" stroke="{ink}" stroke-opacity="0.6" stroke-width="1.5"/>')
        else:
            b.append(f'<path d="M{cx} {cy} v{r} A{r} {r} 0 0 0 {cx + r} {cy}" fill="#0f1b2d" stroke="{ink}" stroke-opacity="0.6" stroke-width="1.5"/>')
    for i in range(6):  # pilares
        for j in range(3):
            b.append(f'<circle cx="{x0 + 1080 + i * 72}" cy="{y0 + 110 + j * 250}" r="9" fill="none" stroke="{ink}" stroke-opacity="0.5" stroke-width="1.5"/>')
    for k in range(9):  # escada
        b.append(f'<line x1="{x0 + 60 + k * 34}" y1="{y0 + 460}" x2="{x0 + 60 + k * 34}" y2="{y0 + 660}" stroke="{ink}" stroke-opacity="0.45" stroke-width="1.4"/>')
    b.append(f'<line x1="{x0}" y1="{y0 + 790}" x2="{x0 + 1520}" y2="{y0 + 790}" stroke="{COPPER}" stroke-width="2"/>')
    for x in (0, 560, 1000, 1520):
        b.append(f'<line x1="{x0 + x - 9}" y1="{y0 + 799}" x2="{x0 + x + 9}" y2="{y0 + 781}" stroke="{COPPER}" stroke-width="2"/>')
    b += [label(x0 + 280, y0 + 826, "5,60", color=ink, anchor="middle"), label(x0 + 780, y0 + 826, "4,40", color=ink, anchor="middle"),
          label(x0 + 1260, y0 + 826, "5,20", color=ink, anchor="middle"), label(x0, y0 - 40, "PLANTA BAIXA · PAVIMENTO α · 1:50", color=ink, op=0.6),
          f'<circle cx="{x0 + 1700}" cy="{y0 + 80}" r="46" fill="none" stroke="{ink}" stroke-opacity="0.5" stroke-width="1.5"/>',
          f'<path d="M{x0 + 1700} {y0 + 26} l14 70 l-14 -16 l-14 16z" fill="{COPPER}"/>', label(x0 + 1700, y0 + 6, "N", color=ink, anchor="middle")]
    return svg("".join(b), defs)


# ── 05 eclipse ─────────────────────────────────────────────────────────
def eclipse():
    cx, cy, r = 1760, 760, 400
    defs = (rad("bg", [(0, "#17120f"), (1, "#090808")], cx=0.69, cy=0.53, r=0.8)
            + rad("cor", [(0.50, "#ffb07a", 0.0), (0.56, "#f08a4a", 0.85), (0.68, COPPER, 0.28), (1, COPPER, 0)], r=0.5)
            + '<filter id="bl" x="-50%" y="-50%" width="200%" height="200%"><feGaussianBlur stdDeviation="9"/></filter>')
    b = [rect("url(#bg)"), f'<circle cx="{cx}" cy="{cy}" r="{r * 1.75}" fill="url(#cor)"/>',
         f'<circle cx="{cx + 10}" cy="{cy - 8}" r="{r + 5}" fill="#ffd0a8" fill-opacity="0.55" filter="url(#bl)"/>',
         f'<circle cx="{cx}" cy="{cy}" r="{r}" fill="#0a0908"/>']
    for k, rr in enumerate((560, 700, 860, 1040, 1240)):
        b.append(f'<circle cx="{cx}" cy="{cy}" r="{rr}" fill="none" stroke="{BONE}" stroke-opacity="{0.16 - k * 0.025}" stroke-width="1.3"/>')
    for a in range(0, 360, 5):
        t = math.radians(a)
        l = 22 if a % 30 == 0 else 10
        b.append(f'<line x1="{cx + math.cos(t) * 560:.1f}" y1="{cy + math.sin(t) * 560:.1f}" x2="{cx + math.cos(t) * (560 + l):.1f}" y2="{cy + math.sin(t) * (560 + l):.1f}" stroke="{BONE}" stroke-opacity="0.28" stroke-width="1.3"/>')
    b.append(label(140, H - 100, "ECLIPSE ANULAR · FASE 03", op=0.4))
    return svg("".join(b), defs)


# ── 06 malha: um terreno em linhas ─────────────────────────────────────
def malha():
    defs = lin("bg", [(0, "#0c0b0b"), (0.55, "#15110f"), (1, "#0b0a0a")])
    b = [rect("url(#bg)")]
    rows = 46

    def height(x, z):
        return (math.sin(x * 2.1 + z * 0.9) * 0.5 + math.sin(x * 0.7 - z * 1.7 + 1.3) * 0.9
                + math.exp(-((x - 0.9) ** 2 + (z - 2.6) ** 2) * 0.55) * 2.3)

    for j in range(rows, 0, -1):
        z = 0.35 + j * 0.12
        scale = 1 / z
        pts = []
        for i in range(0, 161):
            x = (i / 160 - 0.5) * 9
            sx = W / 2 + x * 560 * scale
            sy = 500 + 620 * scale - height(x, z) * 150 * scale
            pts.append((sx, sy))
        poly = " ".join(f"{x:.1f},{y:.1f}" for x, y in pts)
        op = max(0.05, min(0.6, 0.9 * scale - 0.08))
        b.append(f'<polygon points="{poly} {W + 800},{H + 50} -800,{H + 50}" fill="#0d0c0b"/>')
        copper = j == 20
        b.append(f'<polyline points="{poly}" fill="none" stroke="{COPPER if copper else BONE}" stroke-opacity="{0.95 if copper else op:.3f}" stroke-width="{2.4 if copper else 1.3}"/>')
    return svg("".join(b), defs)


# ── 07 papel: a folha clara, em nanquim ────────────────────────────────
def papel():
    ink = "#201c19"
    defs = rad("bg", [(0, "#ded5c9"), (1, "#c3b9ac")], r=0.8)
    cx, cy = 1560, 1080
    b = [rect("url(#bg)")]
    for k, r in enumerate(range(160, 900, 74)):
        b.append(f'<path d="M{cx - r} {cy} A{r} {r} 0 0 1 {cx + r} {cy}" fill="none" stroke="{ink}" stroke-opacity="{0.75 if k % 3 == 0 else 0.32}" stroke-width="{2.6 if k % 3 == 0 else 1.2}"/>')
    for a in range(0, 181, 6):
        t = math.radians(a)
        b.append(f'<line x1="{cx + math.cos(t) * 160:.1f}" y1="{cy - math.sin(t) * 160:.1f}" x2="{cx + math.cos(t) * 826:.1f}" y2="{cy - math.sin(t) * 826:.1f}" stroke="{ink}" stroke-opacity="0.14" stroke-width="1"/>')
    b.append(f'<line x1="240" y1="{cy}" x2="{W - 160}" y2="{cy}" stroke="{ink}" stroke-opacity="0.8" stroke-width="2.4"/>')
    for i in range(60):  # hachura do chão
        x = 240 + i * 36
        b.append(f'<line x1="{x}" y1="{cy + 8}" x2="{x - 26}" y2="{cy + 40}" stroke="{ink}" stroke-opacity="0.3" stroke-width="1.2"/>')
    b += [f'<line x1="{cx}" y1="{cy}" x2="{cx}" y2="{cy - 900}" stroke="{ink}" stroke-opacity="0.35" stroke-width="1.2" stroke-dasharray="18 8 3 8"/>',
          f'<circle cx="{cx}" cy="{cy - 826}" r="15" fill="{COPPER}"/>',
          f'<line x1="{cx - 826}" y1="{cy + 110}" x2="{cx + 826}" y2="{cy + 110}" stroke="{COPPER}" stroke-width="2"/>',
          label(cx, cy + 146, "VÃO 16,52", color=ink, op=0.75, anchor="middle"), label(240, 330, "CÚPULA · ELEVAÇÃO", color=ink, op=0.7, size=19),
          label(240, 364, "FOLHA 07 · ESC 1:100", color=ink, op=0.45, size=14)]
    return svg("".join(b), defs)


# ── 08 crepúsculo: o rio no fim da tarde ───────────────────────────────
def crepusculo():
    hz = 930
    defs = (lin("sky", [(0, "#0c0a14"), (0.34, "#241430"), (0.62, "#6e2a33"), (0.84, "#c85a35"), (1, "#f2b071")])
            + lin("water", [(0, "#b0603c"), (0.12, "#5a2a2c"), (0.45, "#1e1220"), (1, "#0a0910")])
            + lin("path", [(0, "#ffd9a8", 0.85), (1, "#ffd9a8", 0)])
            + f'<clipPath id="up"><rect width="{W}" height="{hz}"/></clipPath>')
    b = [f'<rect width="{W}" height="{hz}" fill="url(#sky)"/>', f'<rect y="{hz}" width="{W}" height="{H - hz}" fill="url(#water)"/>',
         f'<circle cx="1500" cy="{hz + 10}" r="92" fill="#ffe2bb" clip-path="url(#up)"/>',
         f'<polygon points="1440,{hz} 1560,{hz} 1640,{H} 1360,{H}" fill="url(#path)" fill-opacity="0.35"/>']
    rnd = random.Random(8)
    for i in range(70):  # reflexos
        y = hz + 6 + (i / 70) ** 1.6 * (H - hz)
        wdt = rnd.uniform(60, 420) * (0.4 + i / 70)
        x = 1500 + rnd.uniform(-260, 260) * (0.3 + i / 60)
        b.append(f'<line x1="{x - wdt / 2:.0f}" y1="{y:.0f}" x2="{x + wdt / 2:.0f}" y2="{y:.0f}" stroke="#ffd9a8" stroke-opacity="{rnd.uniform(0.05, 0.3):.2f}" stroke-width="{1.2 + i / 30:.1f}"/>')
    pts = [f"0,{hz}"]  # a mata na outra margem
    rnd = random.Random(81)
    y = hz - 26
    for x in range(0, W + 16, 16):
        y = max(hz - 52, min(hz - 12, y + rnd.uniform(-7, 7)))
        pts.append(f"{x},{y:.0f}")
    pts.append(f"{W},{hz}")
    b.append(f'<polygon points="{" ".join(pts)}" fill="#160d16"/>')
    return svg("".join(b), defs)


# ── 09 constelação: uma carta celeste ──────────────────────────────────
def constelacao():
    rnd = random.Random(9)
    cx, cy = 1500, 700
    defs = rad("bg", [(0, "#111725"), (1, "#07090e")], cx=0.58, cy=0.48, r=0.8)
    b = [rect("url(#bg)")]
    for r in range(180, 1500, 180):
        b.append(f'<circle cx="{cx}" cy="{cy}" r="{r}" fill="none" stroke="#b9c6dc" stroke-opacity="0.10" stroke-width="1.2"/>')
    for a in range(0, 360, 15):
        t = math.radians(a)
        b.append(f'<line x1="{cx + math.cos(t) * 180:.0f}" y1="{cy + math.sin(t) * 180:.0f}" x2="{cx + math.cos(t) * 1700:.0f}" y2="{cy + math.sin(t) * 1700:.0f}" stroke="#b9c6dc" stroke-opacity="0.06" stroke-width="1"/>')
    b.append(f'<ellipse cx="{cx}" cy="{cy}" rx="1080" ry="430" transform="rotate(-18 {cx} {cy})" fill="none" stroke="{COPPER}" stroke-opacity="0.5" stroke-width="1.4" stroke-dasharray="3 9"/>')
    for _ in range(520):
        x, y = rnd.uniform(0, W), rnd.uniform(0, H)
        m = rnd.random() ** 3
        b.append(f'<circle cx="{x:.0f}" cy="{y:.0f}" r="{0.7 + m * 2.6:.1f}" fill="#eef2fa" fill-opacity="{0.25 + m * 0.7:.2f}"/>')
    cons = [(980, 420), (1130, 520), (1290, 480), (1400, 620), (1360, 800), (1540, 880), (1700, 760)]
    b.append(f'<polyline points="{" ".join(f"{x},{y}" for x, y in cons)}" fill="none" stroke="#dfe6f3" stroke-opacity="0.45" stroke-width="1.3"/>')
    for i, (x, y) in enumerate(cons):
        c = COPPER if i == 3 else "#f4f6fb"
        b.append(f'<circle cx="{x}" cy="{y}" r="{7 if i == 3 else 4.5}" fill="{c}"/>')
        b.append(f'<circle cx="{x}" cy="{y}" r="15" fill="none" stroke="{c}" stroke-opacity="0.35" stroke-width="1"/>')
    b += [label(1424, 612, "α", color="#f4f6fb", op=0.8, size=22, spacing=0), label(140, H - 100, "CARTA CELESTE · HEMISFÉRIO SUL", color="#b9c6dc", op=0.45)]
    return svg("".join(b), defs)


# ── 10 cobre: metal derretido, sem assunto ─────────────────────────────
# O fundo é somado em numpy (manchas gaussianas exatas): o desfoque do SVG
# deixava degraus quadrados e cortava na borda.
def cobre_bg():
    y, x = np.mgrid[0:H, 0:W].astype(np.float32)
    im = np.zeros((H, W, 3), np.float32) + np.array([12, 10, 10], np.float32)
    blobs = [(420, 1260, 560, (122, 47, 28), 0.95), (1250, 1380, 640, (206, 94, 50), 0.85), (2150, 1180, 560, (229, 154, 90), 0.6),
             (1800, 420, 520, (64, 26, 44), 0.8), (640, 380, 460, (26, 24, 34), 0.7), (2500, 420, 380, (96, 44, 30), 0.55)]
    for cx, cy, r, col, a in blobs:
        g = np.exp(-(((x - cx) ** 2 + (y - cy) ** 2) / (2 * (r * 0.62) ** 2)))[..., None] * a
        im = im * (1 - g) + np.array(col, np.float32) * g
    top = np.clip(1 - y / 420, 0, 1)[..., None] ** 1.5 * 0.8   # calmo no alto, onde fica a ilha
    return im * (1 - top) + np.array([11, 10, 10], np.float32) * top


def cobre():
    b = []
    for k in range(9):
        y = 900 + k * 52
        b.append(f'<path d="M-50 {y} C 600 {y - 260 + k * 22}, 1300 {y + 220 - k * 14}, {W + 50} {y - 180 + k * 30}" fill="none" stroke="#ffe3c4" stroke-opacity="{0.26 - k * 0.024:.3f}" stroke-width="1.4"/>')
    return svg("".join(b)), cobre_bg


# ── 11 mata: troncos na neblina ────────────────────────────────────────
def mata():
    rnd = random.Random(11)
    fog = "#3f5249"
    defs = (lin("bg", [(0, "#16211d"), (0.5, "#33453d"), (0.78, "#4d6156"), (1, "#26332d")])
            + rad("glow", [(0, "#f2c08a", 0.75), (0.4, "#e9a56a", 0.25), (1, "#e9a56a", 0)], r=0.5)
            + lin("floor", [(0, "#060908", 0), (1, "#060908", 0.9)]) + lin("top", [(0, "#080c0b", 0.85), (1, "#080c0b", 0)]))
    b = [rect("url(#bg)"), '<circle cx="1640" cy="930" r="620" fill="url(#glow)"/>', '<circle cx="1640" cy="930" r="46" fill="#f6d3a4" fill-opacity="0.9"/>']
    layers = [(54, "#3a4d44", 6, 18, 0.30), (30, "#273830", 10, 30, 0.22), (13, "#16221d", 22, 58, 0.12), (4, "#090e0c", 60, 130, 0.0)]
    for n, col, wmin, wmax, haze in layers:
        for _ in range(n):
            x = rnd.uniform(-80, W + 80)
            w = rnd.uniform(wmin, wmax)
            lean = rnd.uniform(-0.012, 0.012) * H
            b.append(f'<polygon points="{x:.0f},-20 {x + w:.0f},-20 {x + w * 1.12 + lean:.0f},{H + 20} {x + lean - w * 0.12:.0f},{H + 20}" fill="{col}"/>')
        if haze:  # a neblina entre uma fileira e a seguinte
            b.append(f'<rect width="{W}" height="{H}" fill="{fog}" fill-opacity="{haze}"/>')
    b += [f'<rect y="{H - 420}" width="{W}" height="420" fill="url(#floor)"/>', f'<rect width="{W}" height="380" fill="url(#top)"/>']
    return svg("".join(b), defs)


# ── 12 catedral: um hiperboloide de linhas retas ───────────────────────
def catedral():
    defs = rad("bg", [(0, "#1b1614"), (1, "#0a0909")], cx=0.62, cy=0.55, r=0.75)
    cx, top, base, rx, ry = 1600, 250, 1250, 600, 96
    n, twist = 72, math.radians(118)
    b = [rect("url(#bg)")]

    def pt(a, y, k=1.0):
        return cx + math.cos(a) * rx * k, y + math.sin(a) * ry * k

    for fam, sign in ((0, 1), (1, -1)):
        for i in range(n):
            a = 2 * math.pi * i / n
            x1, y1 = pt(a, base)
            x2, y2 = pt(a + sign * twist, top, 0.9)
            front = math.sin(a) > 0
            copper = fam == 0 and i % 12 == 3
            op = 0.85 if copper else (0.34 if front else 0.13)
            b.append(f'<line x1="{x1:.1f}" y1="{y1:.1f}" x2="{x2:.1f}" y2="{y2:.1f}" stroke="{COPPER if copper else BONE}" stroke-opacity="{op}" stroke-width="{1.9 if copper else 1.15}"/>')
    b += [f'<ellipse cx="{cx}" cy="{base}" rx="{rx}" ry="{ry}" fill="none" stroke="{BONE}" stroke-opacity="0.5" stroke-width="1.6"/>',
          f'<ellipse cx="{cx}" cy="{top}" rx="{rx * 0.9}" ry="{ry * 0.9}" fill="none" stroke="{BONE}" stroke-opacity="0.4" stroke-width="1.4"/>',
          f'<line x1="200" y1="{base + ry + 30}" x2="{W - 200}" y2="{base + ry + 30}" stroke="{BONE}" stroke-opacity="0.22" stroke-width="1.3"/>',
          f'<circle cx="{cx}" cy="{(top + base) / 2 - 20}" r="9" fill="{COPPER}"/>',
          label(200, 330, "HIPERBOLOIDE · 72 RETAS", op=0.42), label(200, 362, "NENHUMA LINHA CURVA", op=0.28, size=14)]
    return svg("".join(b), defs)


ALL = [("01-arcos", arcos), ("02-duna", duna), ("03-madeira", madeira), ("04-planta", planta), ("05-eclipse", eclipse), ("06-malha", malha),
       ("07-papel", papel), ("08-crepusculo", crepusculo), ("09-constelacao", constelacao), ("10-cobre", cobre), ("11-mata", mata), ("12-catedral", catedral)]


def finish(png, jpg, seed, bg=None):
    """Junta com o fundo em numpy (se houver), põe um grão leve de filme (que
    também quebra as faixas dos degradês) e grava em JPG."""
    layer = Image.open(png).convert("RGBA")
    if bg is not None:
        rgba = np.asarray(layer).astype(np.float32)
        a = rgba[..., 3:4] / 255
        im = bg() * (1 - a) + rgba[..., :3] * a
    else:
        im = np.asarray(layer.convert("RGB")).astype(np.float32)
    rng = np.random.default_rng(seed)
    grain = rng.normal(0, 2.6, im.shape[:2])[..., None]
    im = np.clip(im + grain, 0, 255).astype(np.uint8)
    Image.fromarray(im).save(jpg, quality=90, optimize=True, subsampling=0)


if __name__ == "__main__":
    os.makedirs(out, exist_ok=True)
    want = set(sys.argv[1:])
    with tempfile.TemporaryDirectory() as tmp:
        for i, (name, fn) in enumerate(ALL):
            if want and not any(w in name for w in want):
                continue
            made = fn()
            doc, bg = made if isinstance(made, tuple) else (made, None)
            s, p = os.path.join(tmp, name + ".svg"), os.path.join(tmp, name + ".png")
            with open(s, "w") as f:
                f.write(doc)
            subprocess.run(["rsvg-convert", "-w", str(W), "-o", p, s], check=True)
            finish(p, os.path.join(out, name + ".jpg"), i, bg)
            print("ok", name)
