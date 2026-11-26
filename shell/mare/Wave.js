.pragma library

// A onda da ilha: a borda de baixo ondula com o espectro do som, e um rastro
// mais fraco vem atrás dela. Aqui ficam só as contas; o Sound.qml guarda os
// níveis e o Bar.qml desenha. Testado no node (tests/js/Wave.test.mjs).

function clamp01(v) {
    v = Number(v);
    return v > 0 ? Math.min(1, v) : 0;
}

// O nível de cada faixa, a meio caminho entre o quadro anterior e este: sobe
// depressa e desce devagar, como um líquido. `raw` vai de 0 a 1.
function smooth(prev, raw, bands) {
    var out = [];
    for (var i = 0; i < bands; i++) {
        var v = clamp01(raw[i]), p = clamp01(prev[i]);
        out.push(v > p ? p + (v - p) * 0.7 : p + (v - p) * 0.35);
    }
    return out;
}

// O rastro: segue o nível com atraso. Vem atrás na subida e demora a cair, e
// por isso cruza a onda. Perto de zero vira zero (em silêncio não sobra fio).
function trail(prev, levels, bands) {
    var out = [];
    for (var i = 0; i < bands; i++) {
        var v = clamp01(levels[i]), p = clamp01(prev[i]);
        var s = v > p ? p + (v - p) * 0.22 : p + (v - p) * 0.09;
        out.push(s < 0.002 ? 0 : s);
    }
    return out;
}

// A curva: três pontos por faixa, suavizados, presos em zero nas pontas, como
// pares [x, y] (y para baixo, até `amp`). Sem altura, largura ou faixas, uma
// linha reta.
function curve(levels, bands, width, amp) {
    var pts = [[0, 0]];
    var w = Number(width) > 0 ? Number(width) : 0;
    var n = bands, steps = n * 3;
    if (amp > 0.05 && w > 0 && n > 1 && levels && levels.length === n) {
        // Contraste: a faixa mais fraca do momento fica perto de zero e a
        // mais forte usa a altura toda. Sem isso, com todas as faixas soando,
        // a borda só descia por igual.
        var lo = 1, hi = 0, i;
        for (i = 0; i < n; i++) {
            lo = Math.min(lo, levels[i]);
            hi = Math.max(hi, levels[i]);
        }
        var floor = lo * 0.85, span = Math.max(0.08, hi - floor);
        var gain = 0.55 + 0.45 * Math.min(1, hi * 1.6);
        for (i = 1; i < steps; i++) {
            var t = i / steps, p = t * (n - 1), a = Math.floor(p), b = Math.min(n - 1, a + 1);
            var k = (1 - Math.cos((p - a) * Math.PI)) / 2;
            var raw = levels[a] * (1 - k) + levels[b] * k;
            var v = Math.min(1, Math.pow(Math.max(0, (raw - floor) / span), 0.6) * gain);
            var edge = Math.sin(t * Math.PI);
            pts.push([t * w, v * amp * Math.pow(edge, 0.6)]);
        }
    }
    pts.push([w, 0]);
    return pts;
}
