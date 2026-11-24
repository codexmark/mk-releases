// shell/mare/Monitors.qml, o trecho entre "puro: início" e "puro: fim": as
// funções de lista para lista e de lista para texto da página Monitores. O
// que os comentários de lá prometem: modelines dentro dos limites do EDID,
// escalas que o Hyprland aceita, o desenho das telas sem buraco e com o canto
// em 0x0, a última tela ligada que não desliga, e um monitores.lua que o Lua
// sempre consegue ler, seja qual for o nome da saída.
import { spawnSync } from "node:child_process";
import { t, eq, igual, perto, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("Monitors.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["arrange", "bounds", "clash", "closeGaps", "copy", "detached", "enabledCount", "enter", "entry", "extraRates", "fileText",
    "find", "fitScale", "hz", "islands", "listed", "logical", "lua", "matches", "merge", "mirrorsOf", "missingScales",
    "modeFor", "modeText", "modeline", "modelineInfo", "nearestMode", "neighbour", "normalize", "overlaps",
    "percent", "pickMode", "place", "placed", "quote", "rates", "references", "relationTo", "resize",
    "resolutions", "round2", "rule", "same", "sameScale", "scaleOptions", "seat", "setEnabled", "shownRate",
    "slim", "summary", "timing", "unmirror"];
const CONSTANTES = ["minVblank", "wantedScales"];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
});

t("o bloco puro não usa nada de fora dele (o loader recusaria)", () => {
    const f = fonte("Monitors.qml");
    assert.ok(f.de > 0 && f.ate > f.de);
    igual(f.funcoes.slice().sort(), FUNCOES);
    // E o loader acusa mesmo: os nomes de fora que ele vigia são os do painel.
    for (const nome of ["live", "draft", "saved", "pending", "apply", "lister", "selected"])
        assert.ok(f.fora.includes(nome), "o loader não vigia '" + nome + "'");
});

// --- ferramentas ---

// Gerador determinístico (mulberry32): a mesma semente dá a mesma sequência.
function gerador(semente) {
    let a = semente >>> 0;
    const rnd = () => {
        a = (a + 0x6D2B79F5) >>> 0;
        let x = a;
        x = Math.imul(x ^ (x >>> 15), x | 1);
        x ^= x + Math.imul(x ^ (x >>> 7), x | 61);
        return ((x ^ (x >>> 14)) >>> 0) / 4294967296;
    };
    rnd.int = (de, ate) => de + Math.floor(rnd() * (ate - de + 1));
    rnd.um = (lista) => lista[Math.floor(rnd() * lista.length)];
    return rnd;
}

function tela(name, w, h, mais) {
    return Object.assign({
        name: name, description: "", w: w, h: h, refresh: 60, mode: w + "x" + h + "@60", x: 0, y: 0, scale: 1,
        transform: 0, enabled: true, mirror: "", auto: false, modes: [{ w: w, h: h, r: 60 }], limits: null
    }, mais || {});
}

// A geometria, refeita aqui sem usar o código testado.
function ret(m) {
    const s = m.scale > 0 ? m.scale : 1;
    const w = Math.round(m.w / s), h = Math.round(m.h / s);
    return m.transform % 2 === 1 ? { x: m.x, y: m.y, w: h, h: w } : { x: m.x, y: m.y, w: w, h: h };
}
const noDesenho = (m) => m.enabled && !m.mirror && !m.auto && m.w > 0 && m.h > 0;
const rets = (lista) => lista.filter(noDesenho).map(ret);

function sobrepostas(lista) {
    const r = rets(lista);
    for (let i = 0; i < r.length; i++)
        for (let j = i + 1; j < r.length; j++)
            if (r[i].x < r[j].x + r[j].w && r[j].x < r[i].x + r[i].w && r[i].y < r[j].y + r[j].h && r[j].y < r[i].y + r[i].h)
                return true;
    return false;
}

// Encostam por um lado (não só pelo canto).
function encostam(a, b) {
    const emY = a.y < b.y + b.h && b.y < a.y + a.h;
    const emX = a.x < b.x + b.w && b.x < a.x + a.w;
    return (emY && (Math.abs(a.x + a.w - b.x) <= 1 || Math.abs(b.x + b.w - a.x) <= 1))
        || (emX && (Math.abs(a.y + a.h - b.y) <= 1 || Math.abs(b.y + b.h - a.y) <= 1));
}

function soltas(lista) {
    const r = rets(lista);
    if (r.length < 2)
        return false;
    const vistos = [0];
    for (let i = 0; i < vistos.length; i++)
        for (let j = 0; j < r.length; j++)
            if (!vistos.includes(j) && encostam(r[vistos[i]], r[j]))
                vistos.push(j);
    return vistos.length < r.length;
}

// Um buraco no eixo: uma faixa que nenhuma tela cobre, com telas dos dois lados.
function buraco(lista, eixo, lado) {
    const r = rets(lista).sort((a, b) => a[eixo] - b[eixo]);
    let borda = -Infinity;
    for (const x of r) {
        if (borda !== -Infinity && x[eixo] > borda)
            return true;
        borda = Math.max(borda, x[eixo] + x[lado]);
    }
    return false;
}

function origem(lista) {
    const r = rets(lista);
    return r.length === 0 ? null : { x: Math.min(...r.map((a) => a.x)), y: Math.min(...r.map((a) => a.y)) };
}

// O que todo resultado tem de ter, com o desenho limpo ou não.
function sempre(lista) {
    const p = [];
    const o = origem(lista);
    if (o && (o.x !== 0 || o.y !== 0))
        p.push("o canto do desenho está em " + o.x + "x" + o.y + ", não em 0x0");
    if (buraco(lista, "x", "w"))
        p.push("há um buraco na horizontal");
    if (buraco(lista, "y", "h"))
        p.push("há um buraco na vertical");
    if (!lista.some((m) => m.enabled))
        p.push("todas as telas estão desligadas");
    for (const m of lista) {
        if (noDesenho(m) && !(Number.isInteger(m.x) && Number.isInteger(m.y)))
            p.push(m.name + " está numa posição quebrada: " + m.x + "x" + m.y);
        if (!m.mirror)
            continue;
        const alvo = lista.find((x) => x.name === m.mirror);
        if (m.mirror === m.name || !alvo || !alvo.enabled || alvo.mirror)
            p.push(m.name + " espelha " + JSON.stringify(m.mirror) + ", que não serve de espelho");
        if (!m.enabled)
            p.push(m.name + " está desligada e espelhando");
    }
    return p;
}

// O desenho inteiro: sem sobreposição e sem tela solta.
function limpo(lista) {
    const p = sempre(lista);
    if (sobrepostas(lista))
        p.push("duas telas estão uma por cima da outra");
    if (soltas(lista))
        p.push("uma tela não encosta em nenhuma outra");
    return p;
}

const pos = (lista) => lista.map((m) => m.name + (m.enabled ? "" : "(off)") + (m.mirror ? "(=" + m.mirror + ")" : "")
    + (m.auto ? "(auto)" : "") + " " + ret(m).w + "x" + ret(m).h + "+" + m.x + "+" + m.y).join(" | ");

const TAMANHOS = [[1920, 1080], [2560, 1440], [3840, 2160], [1366, 768], [1280, 1024], [1920, 1200], [3440, 1440]];
const LADOS = ["esquerda", "direita", "acima", "abaixo"];

// Um desenho limpo de n telas: cada uma encostada num lado de uma anterior.
function desenho(rnd, n) {
    const lista = [];
    for (let i = 0; i < n; i++) {
        const [w, h] = rnd.um(TAMANHOS);
        const m = tela("T" + i, w, h, {
            scale: bloco.fitScale(w, h, rnd.um([1, 1, 1.25, 1.5, 2])), transform: rnd.um([0, 0, 0, 1, 2, 3, 5]),
            modes: [{ w: w, h: h, r: 60 }, { w: w, h: h, r: 50 }, { w: 1280, h: 720, r: 60 }, { w: 1024, h: 768, r: 60 }]
        });
        for (let tentativa = 0; i > 0; tentativa++) {
            const o = lista[rnd.int(0, i - 1)];
            const a = ret(m), b = ret(o), lado = rnd.um(LADOS);
            m.x = lado === "esquerda" ? o.x - a.w : lado === "direita" ? o.x + b.w : o.x;
            m.y = lado === "acima" ? o.y - a.h : lado === "abaixo" ? o.y + b.h : o.y;
            if (!sobrepostas(lista.concat([m])))
                break;
            if (tentativa > 200)
                throw new Error("desenho: não achei lugar");
        }
        lista.push(m);
    }
    const o = origem(lista);
    for (const m of lista) {
        m.x -= o.x;
        m.y -= o.y;
    }
    return lista;
}

// Uma operação do painel, sorteada: [descrição, lista nova, o que o pedido
// tinha de garantir (uma função que devolve os problemas)].
function operacao(rnd, lista) {
    const a = rnd.um(lista).name, b = rnd.um(lista).name;
    switch (rnd.int(0, 7)) {
    case 0: case 1: case 2: {
        const lado = rnd.um(LADOS.concat(["espelhar"]));
        const out = M.place(lista, a, lado, b);
        // O pedido vale se as duas estão ligadas, a referência tem lugar e a tela tem tamanho.
        const ma = lista.find((m) => m.name === a), mb = lista.find((m) => m.name === b);
        const vale = a !== b && ma.enabled && mb.enabled && !mb.mirror && (lado === "espelhar" || (noDesenho(mb) && ma.w > 0));
        return ["place " + a + " " + lado + " " + b, out,
            () => vale && M.relationTo(out, a, b) !== lado ? [a + " ficou '" + M.relationTo(out, a, b) + "' de " + b + ", não '" + lado + "'"] : []];
    }
    case 3:
        return ["setEnabled " + a + " false", M.setEnabled(lista, a, false), () => []];
    case 4:
        return ["setEnabled " + a + " true", M.setEnabled(lista, a, true), () => []];
    case 5: {
        const troca = rnd.um([{ scale: 1 }, { scale: 1.25 }, { scale: 2 }, { transform: 0 }, { transform: 1 }, { transform: 3 }]);
        return ["resize " + a + " " + JSON.stringify(troca), M.resize(lista, a, troca), () => []];
    }
    case 6: {
        const out = M.unmirror(lista, a);
        return ["unmirror " + a, out, () => out.find((m) => m.name === a).mirror ? [a + " continua espelhando"] : []];
    }
    default: {
        const modo = rnd.um(lista.find((m) => m.name === a).modes);
        return ["pickMode " + a + " " + modo.w + "x" + modo.h, M.pickMode(lista, a, modo.w, modo.h), () => []];
    }
    }
}

// Os limites de EDID usados nos casos: { vmin, vmax (Hz), hmin, hmax (kHz), clock (MHz) }.
const FULLHD = { vmin: 48, vmax: 75, hmin: 30, hmax: 83, clock: 170 };
const QHD = { vmin: 48, vmax: 75, hmin: 30, hmax: 120, clock: 300 };
const GAMER = { vmin: 48, vmax: 144, hmin: 30, hmax: 160, clock: 330 };
const QHD144 = { vmin: 40, vmax: 144, hmin: 30, hmax: 222, clock: 600 };

// Os números de um "modeline ...".
function partes(texto) {
    const p = texto.split(" ");
    eq(p[0], "modeline");
    eq(p.length, 12, "modeline com " + p.length + " campos: " + texto);
    const n = p.slice(1, 10).map(Number);
    return { mhz: n[0], hdisp: n[1], hss: n[2], hse: n[3], htotal: n[4], vdisp: n[5], vss: n[6], vse: n[7], vtotal: n[8],
        sinais: p.slice(10) };
}

// O que há de errado num timing, contra os limites do monitor. A taxa e a
// frequência de linha são as do sinal de verdade: o clock (kHz inteiros) sobre
// os totais. O arredondamento do clock para baixo tira no máximo 1 kHz, então
// os pisos têm essa folga; os tetos não têm folga nenhuma.
function foraDosLimites(tm, w, h, r, l) {
    const p = [];
    const ml = partes(M.modeline(tm));
    const linha = ml.mhz * 1e6 / ml.htotal;         // Hz
    const taxa = linha / ml.vtotal;                 // Hz
    const folga = 1000 / (ml.htotal * ml.vtotal);   // 1 kHz de clock, em Hz de taxa
    if (tm.w !== w || tm.h !== h || tm.r !== r)
        p.push("não é o modo pedido");
    if (ml.hdisp !== w || ml.vdisp !== h)
        p.push("o modeline não tem " + w + "x" + h);
    if (Math.abs(ml.mhz * 1000 - tm.khz) > 1e-6 || !Number.isInteger(tm.khz))
        p.push("clock do texto (" + ml.mhz + " MHz) diferente do calculado (" + tm.khz + " kHz)");
    if (!(ml.mhz <= l.clock))
        p.push("pixel clock " + ml.mhz + " MHz acima de " + l.clock);
    if (!(linha <= l.hmax * 1000 - 100 + 1e-6))
        p.push("linha em " + linha / 1000 + " kHz, acima de " + l.hmax);
    if (!(tm.vtotal * r <= l.hmax * 1000 - 100))
        p.push("linha a menos de 0,1 kHz do teto de " + l.hmax + " kHz");
    if (!(linha >= (l.hmin || 0) * 1000 - folga * ml.vtotal))
        p.push("linha em " + linha / 1000 + " kHz, abaixo de " + l.hmin);
    if (!(taxa <= l.vmax + 1e-9))
        p.push("taxa " + taxa + " Hz acima de " + l.vmax);
    if (!(taxa >= (l.vmin || 0) - folga))
        p.push("taxa " + taxa + " Hz abaixo de " + l.vmin);
    if (!(taxa <= r + 1e-9 && taxa > r - 0.01))
        p.push("taxa de verdade " + taxa + " Hz longe dos " + r + " pedidos");
    if (!(ml.vtotal - ml.vdisp >= M.minVblank))
        p.push("intervalo vertical de " + (ml.vtotal - ml.vdisp) + " linhas, menos que " + M.minVblank);
    if (!(ml.hdisp < ml.hss && ml.hss < ml.hse && ml.hse < ml.htotal))
        p.push("hsync fora do total: " + [ml.hdisp, ml.hss, ml.hse, ml.htotal].join(" "));
    if (!(ml.vdisp < ml.vss && ml.vss < ml.vse && ml.vse < ml.vtotal))
        p.push("vsync fora do total: " + [ml.vdisp, ml.vss, ml.vse, ml.vtotal].join(" "));
    if (ml.htotal !== tm.htotal || ml.vtotal !== tm.vtotal)
        p.push("totais do texto diferentes dos calculados");
    if (ml.sinais.join(" ") !== "+hsync -vsync")
        p.push("polaridade " + ml.sinais.join(" ") + " (blanking reduzido é +hsync -vsync)");
    return p.map((x) => w + "x" + h + "@" + r + ": " + x);
}

// O luac e o lua de verdade, se houver. O texto vai pela entrada: nada é gravado.
const temLuac = !spawnSync("luac", ["-v"]).error;
const temLua = !spawnSync("lua", ["-v"]).error;
if (!temLuac)
    console.log("skip - sem luac: o monitores.lua gerado não passa pelo compilador do Lua");
if (!temLua)
    console.log("skip - sem lua: o monitores.lua gerado não é executado com um hl falso");

function erroDoLuac(texto) {
    const r = spawnSync("luac", ["-p", "-"], { input: texto, encoding: "utf8" });
    return r.status === 0 ? "" : (r.stderr || "luac saiu com " + r.status).trim();
}

// Vários arquivos num luac só (cada um no seu `do ... end`); se o lote
// reprova, um por um para dizer qual.
function reprovadosNoLuac(textos) {
    if (erroDoLuac(textos.map((x) => "do\n" + x + "\nend\n").join("")) === "")
        return [];
    return textos.map((x) => [x, erroDoLuac(x)]).filter((x) => x[1] !== "")
        .map((x) => x[1] + "\n    em: " + JSON.stringify(x[0].slice(-400)));
}

// Texto -> string Lua, byte a byte em decimal: o jeito mais burro e mais certo.
const luaBytes = (s) => '"' + [...Buffer.from(String(s), "utf8")].map((b) => "\\" + String(b).padStart(3, "0")).join("") + '"';
const hex = (s) => Buffer.from(String(s), "utf8").toString("hex");

// Roda o Lua gerado com um `hl` falso. `ligadas` é o que hl.get_monitors
// devolve (nomes, ou null para ele falhar); `depois`, o que devolve na hora
// em que o evento registrado dispara. Devolve { regras, eventos, comandos }.
// Nada sai daqui: hl.exec_cmd, os.execute e io.popen só anotam.
function rodaLua(texto, ligadas, depois) {
    const lista = (nomes) => nomes === null ? "nil" : "{ " + nomes.map((n) => "{ name = " + luaBytes(n) + " }").join(", ") + " }";
    const antes = [
        "local function hex(v) return (tostring(v):gsub('.', function(c) return string.format('%02x', c:byte()) end)) end",
        "local MON, CB = " + lista(ligadas) + ", nil",
        "local function anota(c) print('exec ' .. hex(c)) end",
        "os.execute, io.popen, os.remove, os.rename = anota, anota, anota, anota",
        "hl = {}",
        "function hl.monitor(r)",
        "    local k = {}",
        "    for nome in pairs(r) do k[#k + 1] = nome end",
        "    table.sort(k)",
        "    for i, nome in ipairs(k) do k[i] = nome .. '=' .. type(r[nome]):sub(1, 1) .. hex(r[nome]) end",
        "    print('monitor ' .. table.concat(k, ' '))",
        "end",
        "function hl.get_monitors() if MON == nil then error('sem monitores') end return MON end",
        "function hl.on(ev, fn) print('on ' .. hex(ev)) CB = fn end",
        "hl.exec_cmd = anota",
        "local function arquivo()"
    ].join("\n") + "\n";
    const fim = "\nend\narquivo()\n"
        + (depois === undefined ? "" : "MON = " + lista(depois) + "\nif CB then CB() end\n");
    const r = spawnSync("lua", ["-"], { input: antes + texto + fim, encoding: "utf8" });
    if (r.status !== 0)
        throw new Error("o lua recusou o arquivo gerado: " + (r.stderr || "").trim());
    const txt = (h) => Buffer.from(h, "hex").toString("utf8");
    const out = { regras: [], eventos: [], comandos: [] };
    for (const linha of r.stdout.split("\n").filter((l) => l !== "")) {
        const [tipo, ...resto] = linha.split(" ");
        if (tipo === "exec")
            out.comandos.push(txt(resto[0] || ""));
        else if (tipo === "on")
            out.eventos.push(txt(resto[0]));
        else if (tipo === "monitor") {
            const regra = {};
            for (const campo of resto) {
                const [, nome, tp, valor] = /^(\w+)=(.)(.*)$/.exec(campo);
                const v = txt(valor);
                regra[nome] = tp === "n" ? Number(v) : tp === "b" ? v === "true" : v;
            }
            out.regras.push(regra);
        } else
            throw new Error("saída inesperada do lua: " + linha);
    }
    return out;
}

// --- o básico ---

t("copy: uma cópia funda; mexer nela não mexe na original", () => {
    const a = [tela("A", 1920, 1080)];
    const b = M.copy(a);
    igual(b, a);
    b[0].x = 99;
    b[0].modes[0].r = 1;
    eq(a[0].x, 0);
    eq(a[0].modes[0].r, 60);
    igual(M.copy(null), []);
    igual(M.copy(undefined), []);
});

t("find: a tela com aquele nome, ou null", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1280, 720)];
    eq(M.find(l, "B"), l[1]);
    eq(M.find(l, "C"), null);
    eq(M.find(null, "A"), null);
    eq(M.find([], "A"), null);
});

t("round2: duas casas", () => {
    eq(M.round2(59.9512), 59.95);
    eq(M.round2(60.012), 60.01);
    eq(M.round2(144), 144);
});

t("modeText: LxA@taxa, com a taxa em duas casas", () => {
    eq(M.modeText(1920, 1080, 60), "1920x1080@60");
    eq(M.modeText(2560, 1440, 143.912), "2560x1440@143.91");
});

t("hz e percent: do jeito que a página mostra", () => {
    eq(M.hz(60), "60 Hz");
    eq(M.hz(59.94), "59,94 Hz");
    eq(M.hz(60.004), "60 Hz");
    eq(M.percent(1), "100%");
    eq(M.percent(1.25), "125%");
    eq(M.percent(1.3333), "133%");
});

// --- escala ---

t("fitScale: as escalas que dão tamanho lógico inteiro ficam como estão", () => {
    eq(M.fitScale(1920, 1080, 1), 1);
    eq(M.fitScale(1920, 1080, 1.25), 1.25);
    eq(M.fitScale(1920, 1080, 1.5), 1.5);
    eq(M.fitScale(3840, 2160, 2), 2);
});

t("fitScale: sempre um múltiplo de 1/120 que dá tamanho lógico inteiro, o mais perto do pedido", () => {
    const p = [];
    const serve = (w, h, k) => k > 0 && (w * 120) % k === 0 && (h * 120) % k === 0;
    for (const [w, h] of TAMANHOS.concat([[1600, 900], [1680, 1050], [2880, 1800], [1024, 600], [5120, 1440]]))
        for (let pedido = 0.5; pedido <= 2.5; pedido += 0.05) {
            const s = M.fitScale(w, h, pedido);
            const k = Math.round(s * 120);
            if (Math.abs(k / 120 - s) > 1e-12 || !serve(w, h, k)) {
                p.push(w + "x" + h + " a " + pedido.toFixed(2) + " deu " + s + ", que o Hyprland trocaria");
                continue;
            }
            // Nenhuma escala que serve fica mais perto do pedido.
            const dist = Math.abs(k - Math.round(pedido * 120));
            for (let outro = Math.round(pedido * 120) - dist + 1; outro < Math.round(pedido * 120) + dist; outro++)
                if (serve(w, h, outro))
                    p.push(w + "x" + h + " a " + pedido.toFixed(2) + " deu " + s + ", mas " + outro / 120 + " está mais perto");
        }
    vazio(p, "escalas erradas");
});

t("fitScale: sem tamanho sabido devolve o pedido; pedido zero ou torto vale 1", () => {
    eq(M.fitScale(0, 0, 1.5), 1.5);
    eq(M.fitScale(1920, 1080, 0), 1);
    eq(M.fitScale(1920, 1080, -2), 1);
    eq(M.fitScale(1920, 1080, NaN), 1);
    eq(M.fitScale(1920, 1080, undefined), 1);
});

t("wantedScales: as escalas de sempre", () => igual(M.wantedScales, [1, 1.25, 1.5, 2]));

t("scaleOptions: as de sempre mais a da tela, em ordem e sem repetir", () => {
    igual(M.scaleOptions(tela("A", 1920, 1080)), [1, 1.25, 1.5, 2]);
    igual(M.scaleOptions(null), [1, 1.25, 1.5, 2]);
    const com = M.scaleOptions(tela("A", 3840, 2160, { scale: 1.75 }));
    assert.ok(com.includes(M.fitScale(3840, 2160, 1.75)), "falta a escala da própria tela: " + com);
    const p = [];
    for (const [w, h] of TAMANHOS)
        for (const s of [1, 1.2, 1.6, 1.75, 2.5]) {
            const op = M.scaleOptions(tela("A", w, h, { scale: s }));
            for (let i = 0; i < op.length; i++) {
                if (i > 0 && !(op[i] > op[i - 1] + 0.004))
                    p.push(w + "x" + h + ": fora de ordem ou repetida: " + op);
                if (M.fitScale(w, h, op[i]) !== op[i])
                    p.push(w + "x" + h + ": oferece " + op[i] + ", que o Hyprland trocaria");
            }
        }
    vazio(p, "opções de escala");
    // A lista de sempre não é estragada por quem pede as opções.
    igual(M.wantedScales, [1, 1.25, 1.5, 2]);
});

t("missingScales: 1366x768 fica sem 125% e 150%; 1920x1080 tem todas", () => {
    igual(M.missingScales(tela("A", 1366, 768)), [1.25, 1.5]);
    igual(M.missingScales(tela("A", 1920, 1080)), []);
    igual(M.missingScales(null), []);
    igual(M.missingScales(tela("A", 0, 0)), []);
});

t("sameScale: a mesma escala a menos do arredondamento", () => {
    assert.ok(M.sameScale(1.3333, 4 / 3));
    assert.ok(M.sameScale(1.25, 1.25));
    assert.ok(!M.sameScale(1.25, 1.2667));
    assert.ok(!M.sameScale(1, 1.0083));
});

t("logical: o tamanho na tela conta a escala e a rotação", () => {
    igual(M.logical(tela("A", 1920, 1080)), { w: 1920, h: 1080 });
    igual(M.logical(tela("A", 3840, 2160, { scale: 2 })), { w: 1920, h: 1080 });
    igual(M.logical(tela("A", 2560, 1440, { scale: 1.25, transform: 1 })), { w: 1152, h: 2048 });
    igual(M.logical(tela("A", 1920, 1080, { transform: 3 })), { w: 1080, h: 1920 });
    igual(M.logical(tela("A", 1920, 1080, { transform: 2 })), { w: 1920, h: 1080 });
    igual(M.logical(tela("A", 1920, 1080, { transform: 5 })), { w: 1080, h: 1920 });
    igual(M.logical(tela("A", 1920, 1080, { scale: 0 })), { w: 1920, h: 1080 });
});

t("placed: ligada, com lugar próprio e tamanho conhecido", () => {
    assert.ok(M.placed(tela("A", 1920, 1080)));
    assert.ok(!M.placed(tela("A", 1920, 1080, { enabled: false })));
    assert.ok(!M.placed(tela("A", 1920, 1080, { mirror: "B" })));
    assert.ok(!M.placed(tela("A", 1920, 1080, { auto: true })));
    assert.ok(!M.placed(tela("A", 0, 0)));
});

// --- modos e taxas ---

const MODOS = [{ w: 1920, h: 1080, r: 60 }, { w: 1920, h: 1080, r: 59.94 }, { w: 1920, h: 1080, r: 50 },
    { w: 1280, h: 720, r: 60 }, { w: 1920, h: 1080, r: 60 }, { w: 1024, h: 768, r: 75.03 }];

t("resolutions: cada resolução uma vez, na ordem da lista", () => {
    igual(M.resolutions({ modes: MODOS }), [{ w: 1920, h: 1080 }, { w: 1280, h: 720 }, { w: 1024, h: 768 }]);
    igual(M.resolutions(null), []);
    igual(M.resolutions({}), []);
});

t("rates: as taxas da resolução em uso", () => {
    igual(M.rates(tela("A", 1920, 1080, { modes: MODOS })), [60, 59.94, 50, 60]);
    igual(M.rates(tela("A", 1024, 768, { modes: MODOS })), [75.03]);
    igual(M.rates(tela("A", 800, 600, { modes: MODOS })), []);
    igual(M.rates(null), []);
});

t("listed: a taxa está na lista da resolução em uso", () => {
    const m = tela("A", 1920, 1080, { modes: MODOS });
    assert.ok(M.listed(m, 60));
    assert.ok(M.listed(m, 59.94));
    assert.ok(M.listed(m, 59.944));
    assert.ok(!M.listed(m, 75));
    assert.ok(!M.listed(m, 75.03), "75,03 é de outra resolução");
});

t("nearestMode: o Hyprland informa 60.012 e a lista tem 60.01", () => {
    const m = tela("A", 1920, 1080, { refresh: 60.012, modes: [{ w: 1920, h: 1080, r: 59.94 }, { w: 1920, h: 1080, r: 60.01 },
        { w: 1280, h: 720, r: 60.012 }] });
    igual(M.nearestMode(m), { w: 1920, h: 1080, r: 60.01 });
    eq(M.nearestMode(tela("A", 800, 600, { modes: MODOS })), null);
    eq(M.nearestMode({ w: 1, h: 1, refresh: 60 }), null);
});

// --- timing e modeline ---

t("minVblank: 25 linhas (12 foram poucas num Dell P2319H)", () => eq(M.minVblank, 25));

t("timing: o exemplo do comentário, 1920x1080 a 75 Hz", () => {
    const tm = M.timing(1920, 1080, 75, FULLHD);
    eq(M.modeline(tm), "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync");
});

t("timing: 1920x1080 a 60 Hz usa o intervalo vertical do CVT de blanking reduzido (460 µs)", () => {
    const tm = M.timing(1920, 1080, 60, FULLHD);
    igual(tm, { w: 1920, h: 1080, r: 60, htotal: 2000, vtotal: 1111, khz: 133320 });
    // 31 linhas a 66,66 kHz são os 460 µs.
    const us = (tm.vtotal - tm.h) / (tm.vtotal * tm.r) * 1e6;
    assert.ok(us >= 460 && us < 460 + 1e6 / (tm.vtotal * tm.r), "intervalo de " + us + " µs");
});

t("timing: o horizontal é o do CVT-RBv2, 80 pontos de intervalo (8 + 32 + 40)", () => {
    for (const [w, h, r, l] of [[1920, 1080, 60, FULLHD], [2560, 1440, 75, QHD], [1920, 1080, 144, GAMER]]) {
        const ml = partes(M.modeline(M.timing(w, h, r, l)));
        igual([ml.hss - ml.hdisp, ml.hse - ml.hss, ml.htotal - ml.hse], [8, 32, 40]);
    }
});

t("timing: vários monitores, todas as taxas inteiras da faixa dentro de todos os limites", () => {
    const p = [];
    let validos = 0;
    for (const [w, h, l] of [[1920, 1080, FULLHD], [2560, 1440, QHD], [1920, 1080, GAMER], [2560, 1440, QHD144],
        [1366, 768, { vmin: 40, vmax: 60, hmin: 30, hmax: 60, clock: 90 }], [3840, 2160, { vmin: 24, vmax: 75, hmin: 30, hmax: 160, clock: 600 }]])
        for (let r = l.vmin; r <= l.vmax; r++) {
            const tm = M.timing(w, h, r, l);
            if (tm) {
                validos++;
                p.push(...foraDosLimites(tm, w, h, r, l));
            }
        }
    assert.ok(validos > 200, "só " + validos + " timings válidos: o caso não testou nada");
    vazio(p, "timings fora dos limites");
});

t("timing: os monitores de sempre aceitam as taxas que anunciam", () => {
    const p = [];
    for (const [w, h, r, l] of [[1920, 1080, 60, FULLHD], [1920, 1080, 75, FULLHD], [1920, 1080, 48, FULLHD],
        [2560, 1440, 60, QHD], [2560, 1440, 75, QHD], [1920, 1080, 144, GAMER], [1920, 1080, 120, GAMER],
        [2560, 1440, 144, QHD144], [1920, 1080, 59.94, FULLHD], [1920, 1080, 74.5, FULLHD]]) {
        const tm = M.timing(w, h, r, l);
        if (!tm)
            p.push(w + "x" + h + "@" + r + " deu null");
        else
            p.push(...foraDosLimites(tm, w, h, r, l));
    }
    vazio(p, "timings");
});

t("timing: quando o CVT não cabe, o maior intervalo que os limites deixam", () => {
    // A 144 Hz o teto de 160 kHz deixa 1110 linhas (159,9 kHz), menos que as 1157 do CVT.
    const tm = M.timing(1920, 1080, 144, GAMER);
    eq(tm.vtotal, 1110);
    eq(M.timing(1920, 1080, 144, Object.assign({}, GAMER, { hmax: 161 })).vtotal, 1117);
    // E o pixel clock também segura: 300 MHz a 144 Hz em 2000 pontos dão 1041 linhas, que não cabem.
    eq(M.timing(1920, 1080, 144, Object.assign({}, GAMER, { clock: 300 })), null);
    eq(M.timing(1920, 1080, 144, Object.assign({}, GAMER, { clock: 319 })).vtotal, 1107);
});

t("timing: taxa fora da faixa vertical do monitor dá null", () => {
    eq(M.timing(1920, 1080, 76, FULLHD), null);
    eq(M.timing(1920, 1080, 75.01, FULLHD), null);
    eq(M.timing(1920, 1080, 47, FULLHD), null);
    eq(M.timing(1920, 1080, 47.99, FULLHD), null);
    eq(M.timing(1920, 1080, 240, GAMER), null);
});

t("timing: o que estoura a frequência de linha ou o pixel clock dá null", () => {
    // 2560x1440 a 75 Hz num monitor de 170 MHz.
    eq(M.timing(2560, 1440, 75, FULLHD), null);
    // Sem as 25 linhas de intervalo dentro do teto de linha.
    eq(M.timing(1920, 1080, 75, Object.assign({}, FULLHD, { hmax: 82 })), null);
    assert.ok(M.timing(1920, 1080, 74, Object.assign({}, FULLHD, { hmax: 82 })));
    // Abaixo do piso de linha, e o pixel clock não deixa subir até ele.
    eq(M.timing(1920, 1080, 48, Object.assign({}, FULLHD, { hmin: 60, clock: 110 })), null);
    // Nem o teto de linha: piso de 60 kHz com teto de 60 kHz não tem a folga de 0,1 kHz.
    eq(M.timing(640, 480, 60, { vmin: 50, vmax: 75, hmin: 60, hmax: 60, clock: 100 }), null);
});

t("timing: com a linha abaixo do piso, o intervalo vertical cresce até alcançá-lo", () => {
    // 640x480 a 60 Hz: as 505 linhas do CVT dão 30,3 kHz e servem para um piso de 30.
    const l = { vmin: 50, vmax: 75, hmin: 30, hmax: 80, clock: 100 };
    eq(M.timing(640, 480, 60, l).vtotal, 505);
    // Para um piso de 31 kHz são precisas 519 linhas (31,14 kHz: 0,1 kHz de folga, como no teto).
    const alto = Object.assign({}, l, { hmin: 31 });
    const tm = M.timing(640, 480, 60, alto);
    eq(tm.vtotal, 519);
    vazio(foraDosLimites(tm, 640, 480, 60, alto));
    // 1920x1080 a 48 Hz num monitor que só desce a 60 kHz: 1253 linhas.
    const fhd = Object.assign({}, FULLHD, { hmin: 60 });
    eq(M.timing(1920, 1080, 48, fhd).vtotal, 1253);
    vazio(foraDosLimites(M.timing(1920, 1080, 48, fhd), 1920, 1080, 48, fhd));
    // Sem folga para os 0,1 kHz, o piso exato ainda serve.
    eq(M.timing(640, 480, 60, { vmin: 50, vmax: 75, hmin: 31, hmax: 80, clock: 22.4 }).vtotal, 518);
    // Quem já está acima do piso não muda.
    eq(M.timing(1920, 1080, 60, FULLHD).vtotal, 1111);
});

t("timing: limites ausentes ou incompletos dão null", () => {
    eq(M.timing(1920, 1080, 60, null), null);
    eq(M.timing(1920, 1080, 60, undefined), null);
    eq(M.timing(1920, 1080, 60, {}), null);
    for (const falta of ["vmax", "hmax", "clock"]) {
        const l = Object.assign({}, FULLHD);
        delete l[falta];
        eq(M.timing(1920, 1080, 60, l), null, "sem " + falta);
        eq(M.timing(1920, 1080, 60, Object.assign({}, FULLHD, { [falta]: 0 })), null, falta + " zero");
        eq(M.timing(1920, 1080, 60, Object.assign({}, FULLHD, { [falta]: null })), null, falta + " null");
        eq(M.timing(1920, 1080, 60, Object.assign({}, FULLHD, { [falta]: NaN })), null, falta + " NaN");
    }
});

t("timing: modo torto dá null", () => {
    eq(M.timing(0, 1080, 60, FULLHD), null);
    eq(M.timing(1920, 0, 60, FULLHD), null);
    eq(M.timing(1920, 1080, 0, FULLHD), null);
    eq(M.timing(1920, 1080, -60, FULLHD), null);
    eq(M.timing(1920, 1080, NaN, FULLHD), null);
    eq(M.timing(undefined, undefined, undefined, FULLHD), null);
});

t("timing: algumas centenas de monitores sorteados, nenhum sinal fora dos limites", () => {
    const rnd = gerador(20260601);
    const p = [];
    let validos = 0, nulos = 0;
    for (let i = 0; i < 600; i++) {
        const [w, h] = rnd.um(TAMANHOS.concat([[1600, 900], [1024, 768], [800, 600]]));
        const vmin = rnd.int(24, 60);
        const l = { vmin: vmin, vmax: vmin + rnd.int(1, 190), hmin: rnd.int(15, 60), hmax: rnd.int(40, 300), clock: rnd.int(60, 900) };
        const r = rnd() < 0.5 ? rnd.int(vmin - 3, l.vmax + 3) : Math.round((vmin - 3 + rnd() * (l.vmax - vmin + 6)) * 1000) / 1000;
        const tm = M.timing(w, h, r, l);
        if (tm) {
            validos++;
            p.push(...foraDosLimites(tm, w, h, r, l).map((x) => x + " com " + JSON.stringify(l)));
            continue;
        }
        nulos++;
        if (r > l.vmax || r < l.vmin)
            continue;
        // null dentro da faixa: tem de haver um motivo nos limites. Com o
        // máximo de linhas que o teto de linha e o pixel clock deixam, ou não
        // sobram as 25 de intervalo, ou a linha não chega ao piso.
        const cabem = Math.min(Math.floor((l.hmax * 1000 - 100) / r), Math.floor(l.clock * 1e6 / ((w + 80) * r)));
        if (cabem - h >= M.minVblank && cabem * r >= l.hmin * 1000)
            p.push(w + "x" + h + "@" + r + " com " + JSON.stringify(l) + " cabia (" + cabem + " linhas) e deu null");
    }
    assert.ok(validos > 100 && nulos > 100, validos + " válidos e " + nulos + " nulos: o sorteio não cobre os dois lados");
    vazio(p, "timings sorteados");
});

t("modelineInfo: o modeline de um timing volta à resolução e à taxa", () => {
    const p = [];
    for (const [w, h, r, l] of [[1920, 1080, 60, FULLHD], [1920, 1080, 75, FULLHD], [2560, 1440, 75, QHD],
        [1920, 1080, 144, GAMER], [2560, 1440, 144, QHD144], [1920, 1080, 59.94, FULLHD]]) {
        const info = M.modelineInfo(M.modeline(M.timing(w, h, r, l)));
        if (!info || info.w !== w || info.h !== h || !(Math.abs(info.r - r) < 0.01))
            p.push(w + "x" + h + "@" + r + " voltou " + JSON.stringify(info));
    }
    vazio(p, "ida e volta");
    perto(M.modelineInfo("modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync").r, 75);
    igual(M.modelineInfo("  modeline   148.5 1920 2008 2052 2200 1080 1084 1089 1125  "), { w: 1920, h: 1080, r: 60 });
});

t("modelineInfo: o que não é modeline dá null", () => {
    for (const x of ["", null, undefined, "preferred", "1920x1080@60", "modeline", "modeline 165.75 1920 1928 1960 2000 1080",
        "modeline abc 1920 1928 1960 2000 1080 1083 1088 1105", "modeline 165.75 1920 1928 1960 0 1080 1083 1088 1105",
        "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 -1105", "xmodeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105"])
        eq(M.modelineInfo(x), null, JSON.stringify(x));
});

t("extraRates: só taxas acima da maior da lista, decrescentes, todas com timing válido", () => {
    const m = tela("A", 1920, 1080, { modes: MODOS, limits: FULLHD });
    igual(M.extraRates(m), [75, 74, 73, 72, 71, 70, 69, 68, 67, 66, 65, 64, 63, 62, 61]);
    const p = [];
    for (const [w, h, l, max] of [[1920, 1080, FULLHD, 60], [1920, 1080, GAMER, 60], [1920, 1080, GAMER, 119.88],
        [2560, 1440, QHD, 59.95], [2560, 1440, QHD144, 100], [2560, 1440, FULLHD, 60], [1920, 1080, FULLHD, 75]]) {
        const x = tela("A", w, h, { modes: [{ w: w, h: h, r: max }, { w: w, h: h, r: 30 }, { w: 640, h: 480, r: 200 }], limits: l });
        const extras = M.extraRates(x);
        const onde = w + "x" + h + " até " + l.vmax + " Hz, lista até " + max + ": ";
        extras.forEach((r, i) => {
            if (!(r > max))
                p.push(onde + r + " não está acima da maior da lista");
            if (!Number.isInteger(r) || r > l.vmax)
                p.push(onde + r + " não é taxa inteira até o teto");
            if (i > 0 && !(r < extras[i - 1]))
                p.push(onde + "fora de ordem: " + extras);
            const tm = M.timing(w, h, r, l);
            if (!tm)
                p.push(onde + r + " não tem timing");
            else
                p.push(...foraDosLimites(tm, w, h, r, l));
        });
        // E não falta nenhuma: toda taxa inteira acima da lista que cabe está lá.
        for (let r = Math.floor(max) + 1; r <= l.vmax; r++)
            if (r > Math.round(max) && M.timing(w, h, r, l) && !extras.includes(r))
                p.push(onde + "falta " + r);
    }
    vazio(p, "taxas extras");
});

t("extraRates: sem limites, sem modos ou já no teto, nenhuma", () => {
    igual(M.extraRates(null), []);
    igual(M.extraRates(tela("A", 1920, 1080, { modes: MODOS, limits: null })), []);
    igual(M.extraRates(tela("A", 800, 600, { modes: MODOS, limits: FULLHD })), []);
    igual(M.extraRates(tela("A", 1920, 1080, { modes: [{ w: 1920, h: 1080, r: 75 }], limits: FULLHD })), []);
    igual(M.extraRates(tela("A", 1920, 1080, { modes: [{ w: 1920, h: 1080, r: 74.97 }], limits: FULLHD })), []);
});

t("modeFor: taxa da lista vai pelo nome; fora dela, o modeline calculado", () => {
    const m = tela("A", 1920, 1080, { modes: MODOS, limits: FULLHD });
    eq(M.modeFor(m), "1920x1080@60");
    eq(M.modeFor(Object.assign({}, m, { refresh: 59.94 })), "1920x1080@59.94");
    eq(M.modeFor(Object.assign({}, m, { refresh: 75 })), "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync");
    // Uma taxa da lista não vira modeline mesmo com limites.
    eq(M.modeFor(Object.assign({}, m, { refresh: 50 })), "1920x1080@50");
});

t("modeFor: sem limites (o EDID sumiu), vale o modeline guardado para esta resolução e taxa", () => {
    const guardado = "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync";
    const m = tela("A", 1920, 1080, { modes: MODOS, refresh: 75, mode: guardado });
    eq(M.modeFor(m), guardado);
    // De outra taxa ou de outra resolução não vale.
    eq(M.modeFor(Object.assign({}, m, { refresh: 70 })), "1920x1080@70");
    eq(M.modeFor(Object.assign({}, m, { w: 1280, h: 720, refresh: 75 })), "1280x720@75");
    // Fora dos limites e sem nada guardado: o nome, e o Hyprland decide.
    eq(M.modeFor(tela("A", 1920, 1080, { modes: MODOS, refresh: 90, limits: FULLHD })), "1920x1080@90");
});

t("shownRate: num modeline o Hyprland informa 74,66 para 75; o painel mostra 75", () => {
    const m = tela("A", 1920, 1080, { modes: MODOS, refresh: 74.66, limits: FULLHD });
    // Pela taxa que o arquivo guardou.
    const antiga = tela("A", 1920, 1080, { refresh: 75, mode: "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync" });
    eq(M.shownRate(m, antiga), 75);
    eq(M.shownRate(Object.assign({}, m, { limits: null }), antiga), 75);
    // Ou pela que o painel ofereceria.
    eq(M.shownRate(m, null), 75);
    eq(M.shownRate(m, undefined), 75);
});

t("shownRate: perto de um modo da lista, vale o da lista", () => {
    const m = tela("A", 1920, 1080, { modes: MODOS, refresh: 60.012, limits: FULLHD });
    eq(M.shownRate(m, null), 60);
    eq(M.shownRate(Object.assign({}, m, { refresh: 59.951 }), null), 59.94);
    // Um modeline guardado de outra resolução, ou longe da taxa de agora, não manda.
    const outra = tela("A", 1280, 720, { refresh: 60.5, mode: "modeline 74.5 1280 1288 1320 1360 720 723 728 745 +hsync -vsync" });
    eq(M.shownRate(m, outra), 60);
    const longe = tela("A", 1920, 1080, { refresh: 75, mode: "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync" });
    eq(M.shownRate(m, longe), 60);
    // O arquivo guardou uma taxa da lista (não é modeline): vale a lista.
    eq(M.shownRate(m, tela("A", 1920, 1080, { refresh: 60.3, mode: "1920x1080@60.3" })), 60);
});

t("shownRate: sem lista e sem limites, o que o Hyprland informa", () => {
    eq(M.shownRate(tela("A", 1920, 1080, { modes: [], refresh: 61.7 }), null), 61.7);
});

// --- entry ---

t("entry: o que o mare-monitors list devolve vira a tela do painel", () => {
    const m = M.entry({ name: "DP-1", description: "Dell P2319H", w: 1920, h: 1080, refresh: 60.012, x: 1920, y: 0, scale: 1,
        transform: 0, enabled: true, mirror: "", modes: [{ w: 1920, h: 1080, r: 60.01 }, { w: 1280, h: 720, r: 60 }], limits: FULLHD }, []);
    igual(m, { name: "DP-1", description: "Dell P2319H", w: 1920, h: 1080, refresh: 60.01, x: 1920, y: 0, scale: 1, transform: 0,
        enabled: true, mirror: "", auto: false, mode: "1920x1080@60.01",
        modes: [{ w: 1920, h: 1080, r: 60.01 }, { w: 1280, h: 720, r: 60 }], limits: FULLHD });
});

t("entry: campos faltando ou tortos viram o padrão", () => {
    const m = M.entry({ name: "X", w: "1920", h: "1080", refresh: "60" }, null);
    eq(m.description, "");
    igual([m.w, m.h, m.refresh, m.x, m.y, m.scale, m.transform], [1920, 1080, 60, 0, 0, 1, 0]);
    eq(m.enabled, true);
    eq(m.mirror, "");
    igual(m.modes, []);
    eq(m.limits, null);
    eq(m.mode, "1920x1080@60");
    eq(M.entry({ name: "X", w: 1920, h: 1080, transform: 9 }, []).transform, 1);
    eq(M.entry({ name: "X", w: 1920, h: 1080, enabled: false }, []).enabled, false);
    eq(M.entry({ name: 7 }, []).name, "7");
});

t("entry: a escala vira uma que o Hyprland aceita", () => {
    eq(M.entry({ name: "X", w: 1366, h: 768, refresh: 60, scale: 1.25 }, []).scale, M.fitScale(1366, 768, 1.25));
    eq(M.entry({ name: "X", w: 1920, h: 1080, refresh: 60, scale: 1.5 }, []).scale, 1.5);
});

t("entry: sem modo sabido, o Hyprland escolhe (auto, preferred)", () => {
    const m = M.entry({ name: "HDMI-A-1", enabled: false }, []);
    eq(m.auto, true);
    eq(m.mode, "preferred");
    eq(m.enabled, false);
});

t("entry: uma tela desligada vale pelo que o arquivo guardou dela", () => {
    const guardado = [{ name: "HDMI-A-1", w: 2560, h: 1440, refresh: 75, x: 1920, y: 0, scale: 1.25, transform: 1, enabled: false,
        mode: "2560x1440@75" }];
    const m = M.entry({ name: "HDMI-A-1", enabled: false, modes: [{ w: 2560, h: 1440, r: 75 }] }, guardado);
    igual([m.w, m.h, m.refresh, m.x, m.y, m.scale, m.transform], [2560, 1440, 75, 1920, 0, 1.25, 1]);
    eq(m.enabled, false);
    eq(m.auto, false);
    eq(m.mode, "2560x1440@75");
    // Uma tela ligada vale pelo que o Hyprland diz, não pelo arquivo.
    const viva = M.entry({ name: "HDMI-A-1", w: 1920, h: 1080, refresh: 60, x: 0, y: 0, scale: 1, modes: [{ w: 1920, h: 1080, r: 60 }] }, guardado);
    igual([viva.w, viva.h, viva.x, viva.scale, viva.transform], [1920, 1080, 0, 1, 0]);
});

t("entry: o modeline guardado continua valendo, com a taxa pedida e não a que o Hyprland arredonda", () => {
    const modo = "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync";
    const guardado = [{ name: "DP-1", w: 1920, h: 1080, refresh: 75, mode: modo }];
    const cru = { name: "DP-1", w: 1920, h: 1080, refresh: 74.66, modes: [{ w: 1920, h: 1080, r: 60 }] };
    // Com o EDID: recalculado, e dá o mesmo.
    const com = M.entry(Object.assign({ limits: FULLHD }, cru), guardado);
    eq(com.refresh, 75);
    eq(com.mode, modo);
    // Sem o EDID: o guardado.
    const sem = M.entry(cru, guardado);
    eq(sem.refresh, 75);
    eq(sem.mode, modo);
});

// --- o desenho: posição relativa ---

const lado = () => [tela("A", 1920, 1080), tela("B", 2560, 1440, { x: 1920 })];
const tres = () => [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { x: 3840 })];

t("relationTo: de que lado da outra a tela está", () => {
    const l = lado();
    eq(M.relationTo(l, "B", "A"), "direita");
    eq(M.relationTo(l, "A", "B"), "esquerda");
    const pilha = [tela("A", 1920, 1080), tela("B", 1920, 1080, { y: 1080 })];
    eq(M.relationTo(pilha, "B", "A"), "abaixo");
    eq(M.relationTo(pilha, "A", "B"), "acima");
});

t("relationTo: só pelo canto, longe, ela mesma ou tela que não existe não é lado nenhum", () => {
    const canto = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 1080 })];
    eq(M.relationTo(canto, "A", "B"), "");
    eq(M.relationTo(canto, "B", "A"), "");
    const longe = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 3000 })];
    eq(M.relationTo(longe, "B", "A"), "");
    eq(M.relationTo(lado(), "A", "A"), "");
    eq(M.relationTo(lado(), "A", "Z"), "");
    eq(M.relationTo(lado(), "Z", "A"), "");
    // Uma tela desligada não tem lado.
    const off = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, enabled: false })];
    eq(M.relationTo(off, "B", "A"), "");
});

t("relationTo: quem espelha está 'espelhar' só em relação a quem espelha", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { mirror: "A" }), tela("C", 1920, 1080, { x: 1920 })];
    eq(M.relationTo(l, "B", "A"), "espelhar");
    eq(M.relationTo(l, "B", "C"), "");
    eq(M.relationTo(l, "A", "B"), "");
});

t("overlaps: uma por cima da outra, mesmo que só um pouco", () => {
    assert.ok(!M.overlaps(lado()));
    assert.ok(M.overlaps([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1919 })]));
    assert.ok(M.overlaps([tela("A", 1920, 1080), tela("B", 1920, 1080)]));
    assert.ok(!M.overlaps([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 1080 })]));
    // Quem não está no desenho não conta: espelho, desligada, automática.
    assert.ok(!M.overlaps([tela("A", 1920, 1080), tela("B", 1920, 1080, { mirror: "A" })]));
    assert.ok(!M.overlaps([tela("A", 1920, 1080), tela("B", 1920, 1080, { enabled: false })]));
    assert.ok(!M.overlaps([tela("A", 1920, 1080), tela("B", 1920, 1080, { auto: true })]));
    assert.ok(!M.overlaps(null));
    assert.ok(!M.overlaps([]));
    // A rotação e a escala contam.
    assert.ok(M.overlaps([tela("A", 1920, 1080, { transform: 1 }), tela("B", 1920, 1080, { y: 1080 })]));
    assert.ok(!M.overlaps([tela("A", 3840, 2160, { scale: 2 }), tela("B", 1920, 1080, { x: 1920 })]));
});

t("detached: uma tela que só encosta pelo canto, ou longe, está solta", () => {
    assert.ok(!M.detached(lado()));
    assert.ok(!M.detached(tres()));
    assert.ok(M.detached([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 1080 })]));
    assert.ok(M.detached([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 2500 })]));
    assert.ok(M.detached([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { x: 3840, y: 1080 })]));
    // Uma tela só, ou nenhuma, não está solta de nada.
    assert.ok(!M.detached([tela("A", 1920, 1080)]));
    assert.ok(!M.detached([]));
    assert.ok(!M.detached(null));
    assert.ok(!M.detached([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 9999, enabled: false })]));
});

t("overlaps e detached dizem o mesmo que a geometria refeita no teste, em 400 desenhos sorteados", () => {
    const rnd = gerador(7);
    const p = [];
    let por = 0, sol = 0;
    for (let i = 0; i < 400; i++) {
        const l = desenho(rnd, rnd.int(1, 5));
        // Metade fica como está; na outra, uma tela é empurrada de qualquer jeito.
        if (rnd() < 0.6) {
            const m = rnd.um(l);
            m.x += rnd.um([0, 8, -8, 400, -400, ret(m).w, -ret(m).w]);
            m.y += rnd.um([0, 8, -8, 400, -400, ret(m).h, -ret(m).h]);
        }
        if (rnd() < 0.1)
            rnd.um(l).enabled = false;
        const so = sobrepostas(l), st = soltas(l);
        por += so ? 1 : 0;
        sol += st ? 1 : 0;
        if (M.overlaps(l) !== so)
            p.push("overlaps disse " + M.overlaps(l) + " para " + pos(l));
        if (M.detached(l) !== st)
            p.push("detached disse " + M.detached(l) + " para " + pos(l));
    }
    assert.ok(por > 30 && sol > 30, "o sorteio quase não fez sobreposição (" + por + ") nem tela solta (" + sol + ")");
    vazio(p, "divergências");
});

t("bounds: o retângulo que contém o desenho", () => {
    igual(M.bounds(lado()), { x: 0, y: 0, w: 4480, h: 1440 });
    igual(M.bounds([tela("A", 1920, 1080, { x: -100, y: 50 }), tela("B", 1920, 1080, { x: 5000, enabled: false })]),
        { x: -100, y: 50, w: 1920, h: 1080 });
    igual(M.bounds([]), { x: 0, y: 0, w: 1, h: 1 });
    igual(M.bounds(null), { x: 0, y: 0, w: 1, h: 1 });
    igual(M.bounds([tela("A", 1920, 1080, { auto: true })]), { x: 0, y: 0, w: 1, h: 1 });
});

t("closeGaps: uma tela que ficou longe das outras encosta de novo", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 5000 }), tela("C", 1920, 1080, { x: 9000 })];
    M.closeGaps(l, "x", "w");
    igual(l.map((m) => m.x), [0, 1920, 3840]);
    const v = [tela("A", 1920, 1080, { y: 100 }), tela("B", 1920, 1080, { y: 3000 })];
    M.closeGaps(v, "y", "h");
    igual(v.map((m) => m.y), [100, 1180]);
    // Quem não está no desenho fica onde estava.
    const off = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 5000, enabled: false }), tela("C", 1920, 1080, { x: 7000, mirror: "A" })];
    M.closeGaps(off, "x", "w");
    igual(off.map((m) => m.x), [0, 5000, 7000]);
});

t("closeGaps: quem já encosta ou divide a faixa com outra não sai do lugar", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { x: 960, y: 1080 })];
    M.closeGaps(l, "x", "w");
    M.closeGaps(l, "y", "h");
    igual(l.map((m) => [m.x, m.y]), [[0, 0], [1920, 0], [960, 1080]]);
});

t("normalize: leva o canto de cima à esquerda para 0x0, sem buracos", () => {
    const l = M.normalize([tela("A", 1920, 1080, { x: -500, y: 300 }), tela("B", 1920, 1080, { x: 4000, y: 300 })]);
    igual(l.map((m) => [m.x, m.y]), [[0, 0], [1920, 0]]);
    // Posição quebrada vira inteira.
    const q = M.normalize([tela("A", 1920, 1080, { x: 0.4, y: 0.4 }), tela("B", 1920, 1080, { x: 1920.4, y: 0.4 })]);
    igual(q.map((m) => [m.x, m.y]), [[0, 0], [1920, 0]]);
    // Sem nada no desenho, nada muda.
    const nada = [tela("A", 1920, 1080, { x: 77, enabled: false })];
    igual(M.normalize(M.copy(nada)), nada);
    igual(M.normalize([]), []);
});

t("normalize: um desenho limpo só volta para o canto, em 300 desenhos sorteados", () => {
    const rnd = gerador(11);
    const p = [];
    for (let i = 0; i < 300; i++) {
        const l = desenho(rnd, rnd.int(1, 5));
        const dx = rnd.int(-3000, 3000), dy = rnd.int(-3000, 3000);
        const longe = M.copy(l).map((m) => Object.assign(m, { x: m.x + dx, y: m.y + dy }));
        const de_volta = M.normalize(longe);
        if (pos(de_volta) !== pos(l))
            p.push("era " + pos(l) + "\n    deslocado de " + dx + "x" + dy + " voltou " + pos(de_volta));
        p.push(...limpo(de_volta).map((x) => x + ": " + pos(de_volta)));
        if (pos(M.normalize(M.copy(de_volta))) !== pos(de_volta))
            p.push("normalizar de novo mudou: " + pos(de_volta));
    }
    vazio(p, "desenhos");
});

t("normalize: telas espalhadas de qualquer jeito, até por cima uma da outra, ficam limpas e no canto (300 sorteios)", () => {
    const rnd = gerador(13);
    const p = [];
    for (let i = 0; i < 300; i++) {
        const l = desenho(rnd, rnd.int(2, 6));
        for (const m of l) {
            m.x += rnd.int(-2, 2) * rnd.um([2500, 700, 0.4]);
            m.y += rnd.int(-2, 2) * rnd.um([2500, 700, 0.4]);
        }
        if (rnd() < 0.2)
            rnd.um(l).enabled = false;
        if (rnd() < 0.2)
            rnd.um(l).auto = true;
        const depois = M.normalize(M.copy(l));
        p.push(...limpo(depois).map((x) => x + ": " + pos(l) + " -> " + pos(depois)));
        if (pos(M.normalize(M.copy(depois))) !== pos(depois))
            p.push("normalizar de novo mudou: " + pos(depois));
        // Quem não está no desenho não é tocado.
        for (const m of l.filter((x) => !noDesenho(x))) {
            const d = depois.find((x) => x.name === m.name);
            if (d.x !== m.x || d.y !== m.y)
                p.push(m.name + " não está no desenho e mudou de lugar");
        }
    }
    vazio(p, "desenhos");
});

// --- o arranjador ---

t("clash: a primeira tela da lista que está por cima desta, ou null", () => {
    const a = tela("A", 1920, 1080), b = tela("B", 1920, 1080, { x: 1000 }), c = tela("C", 1920, 1080, { x: 1920 });
    eq(M.clash(a, [c, b]), b);
    eq(M.clash(a, [c]), null);
    eq(M.clash(a, [a]), null, "ela mesma não conta");
    eq(M.clash(a, []), null);
    eq(M.clash(b, [a, c]), a);
});

t("islands: os grupos de telas que encostam umas nas outras por um lado", () => {
    const nomes = (l) => M.islands(l).map((g) => g.map((m) => m.name).join(""));
    igual(nomes(tres()), ["ABC"]);
    igual(nomes([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 1080 }), tela("C", 1920, 1080, { x: 3840, y: 1080 })]), ["A", "BC"]);
    igual(nomes([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 5000 }), tela("C", 1920, 1080, { y: 1080 })]), ["AC", "B"]);
    igual(nomes([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, enabled: false }), tela("C", 1920, 1080, { mirror: "A" })]), ["A"]);
    // Um ponto de distância já é solta, aqui.
    igual(nomes([tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1921 })]), ["A", "B"]);
    igual(M.islands([]), []);
    igual(M.islands(null), []);
});

t("enter: a tela vai para a direita do desenho, na altura da que vai mais à direita", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 400 }), tela("C", 2560, 1440, { x: 77, y: 77 })];
    M.enter(l, l[2]);
    igual([l[2].x, l[2].y], [3840, 400]);
    // Desenho fora do canto (ainda não normalizado) também.
    const neg = [tela("A", 1920, 1080, { x: -5000, y: -70 }), tela("B", 1920, 1080, { x: 9 })];
    M.enter(neg, neg[1]);
    igual([neg[1].x, neg[1].y], [-3080, -70]);
    // Sozinha, vai para o canto.
    const so = [tela("A", 1920, 1080, { x: 50, y: 60 }), tela("B", 1920, 1080, { enabled: false })];
    M.enter(so, so[0]);
    igual([so[0].x, so[0].y], [0, 0]);
});

t("arrange: tira uma tela de cima da outra pelo lado mais curto (direita ou abaixo), e encosta as soltas", () => {
    // B em cima de A, mais para a direita que para baixo: vai para além de A. Aí cai em cima de C,
    // que estava depois e cede por sua vez: para baixo, que é menos que a largura inteira.
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1000 }), tela("C", 1920, 1080, { x: 1920 })];
    igual(M.arrange(l).map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 1920, 0], ["C", 1920, 1080]]);
    // Mais para baixo que para a direita: desce, e C nem é tocada.
    const b = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 500, y: 900 }), tela("C", 1920, 1080, { x: 1920 })];
    igual(M.arrange(b).map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 500, 1080], ["C", 1920, 0]]);
    // Só pelo canto: a solta encosta pelo caminho mais curto, alinhada.
    const canto = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 1080 })];
    igual(M.arrange(canto).map((m) => [m.x, m.y]), [[0, 0], [1920, 0]]);
    // Longe mas na mesma faixa: só chega perto, sem perder o desalinho.
    const longe = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 5000, y: 300 })];
    igual(M.arrange(longe).map((m) => [m.x, m.y]), [[0, 0], [1920, 300]]);
    // Sem `keep`, o grupo maior fica e a tela sozinha é que anda.
    const grupo = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 5000, y: 0 }), tela("C", 1920, 1080, { x: 5000, y: 1080 })];
    igual(M.arrange(grupo).map((m) => [m.x, m.y]), [[3080, 0], [5000, 0], [5000, 1080]]);
    // Um grupo solto anda junto, com o arranjo que tinha.
    const junto = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 5000, y: 0 }), tela("C", 1920, 1080, { x: 5000, y: 1080 })];
    igual(M.arrange(junto, ["A"]).map((m) => [m.x, m.y]), [[0, 0], [1920, 0], [1920, 1080]]);
    // Um ponto de distância (ou de sobreposição) também é fechado.
    const ponto = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1921 }), tela("C", 1920, 1080, { y: 1079 })];
    igual(M.arrange(ponto, ["A"]).map((m) => [m.x, m.y]), [[0, 0], [1920, 0], [0, 1080]]);
});

t("arrange: as telas de `keep` não saem do lugar, e as outras cedem para o lado dado", () => {
    const l = () => [tela("A", 1920, 1080, { x: 100, y: 50 }), tela("B", 1920, 1080, { x: 600, y: 50 }), tela("C", 1920, 1080, { x: 9000, y: 9000 })];
    igual(M.arrange(l(), ["B"], "direita").map((m) => [m.name, m.x, m.y]).slice(0, 2), [["A", 2520, 50], ["B", 600, 50]]);
    igual(M.arrange(l(), ["B"]).map((m) => [m.name, m.x, m.y]).slice(0, 2), [["A", 100, 1130], ["B", 600, 50]]);
    for (const lado of [undefined].concat(LADOS))
        assert.ok(!sobrepostas(M.arrange(l(), ["B"], lado)) && !soltas(M.arrange(l(), ["B"], lado)), "com " + lado);
    igual(M.arrange(l(), ["B"], "esquerda").map((m) => [m.name, m.x, m.y])[0], ["A", -1320, 50]);
    igual(M.arrange(l(), ["B"], "abaixo").map((m) => [m.name, m.x, m.y])[0], ["A", 100, 1130]);
    igual(M.arrange(l(), ["B"], "acima").map((m) => [m.name, m.x, m.y])[0], ["A", 100, -1030]);
    // Não leva o desenho para o canto (isso é do normalize) nem toca em quem não está no desenho.
    const fora = [tela("A", 1920, 1080, { x: 300, y: 300 }), tela("B", 1920, 1080, { x: 7, y: 7, enabled: false })];
    igual(M.arrange(M.copy(fora)), fora);
    igual(M.arrange([]), []);
});

t("arrange: de qualquer bagunça de uma a seis telas sai um desenho limpo, e arrumar de novo não muda nada (500 sorteios)", () => {
    const rnd = gerador(73);
    const p = [];
    for (let i = 0; i < 500; i++) {
        const l = desenho(rnd, rnd.int(1, 6));
        for (const m of l) {
            m.x = rnd.int(-3, 3) * rnd.um([1, 640, 1000, 1920]);
            m.y = rnd.int(-3, 3) * rnd.um([1, 540, 1000, 1080]);
        }
        const fixas = l.filter(() => rnd() < 0.2).map((m) => m.name).slice(0, 1);
        const lado = rnd.um([undefined].concat(LADOS));
        const out = M.arrange(M.copy(l), fixas, lado);
        const onde = pos(l) + " (fixas: " + fixas + ", lado: " + lado + ") -> " + pos(out);
        if (sobrepostas(out))
            p.push("ficou por cima: " + onde);
        if (soltas(out))
            p.push("ficou solta: " + onde);
        for (const n of fixas)
            if (pos([out.find((m) => m.name === n)]) !== pos([l.find((m) => m.name === n)]))
                p.push("a fixa " + n + " saiu do lugar: " + onde);
        if (pos(M.arrange(M.copy(out), fixas, lado)) !== pos(out) || pos(M.arrange(M.copy(out))) !== pos(out))
            p.push("arrumar de novo mudou: " + onde);
    }
    vazio(p, "bagunças");
});

// --- place ---

t("place: os quatro lados, sempre em relação à outra", () => {
    const l = lado();
    igual(M.place(l, "B", "esquerda", "A").map((m) => [m.name, m.x, m.y]), [["A", 2560, 0], ["B", 0, 0]]);
    igual(M.place(l, "B", "direita", "A").map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 1920, 0]]);
    igual(M.place(l, "B", "acima", "A").map((m) => [m.name, m.x, m.y]), [["A", 0, 1440], ["B", 0, 0]]);
    igual(M.place(l, "B", "abaixo", "A").map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 0, 1080]]);
});

t("place: conta a escala e a rotação das duas", () => {
    const l = [tela("A", 3840, 2160, { scale: 2 }), tela("B", 1920, 1080, { transform: 1, x: 1920 })];
    igual(M.place(l, "B", "direita", "A").map((m) => [m.x, m.y]), [[0, 0], [1920, 0]]);
    igual(M.place(l, "B", "esquerda", "A").map((m) => [m.x, m.y]), [[1080, 0], [0, 0]]);
    igual(M.place(l, "B", "acima", "A").map((m) => [m.x, m.y]), [[0, 1920], [0, 0]]);
    igual(M.place(l, "A", "abaixo", "B").map((m) => [m.x, m.y]), [[0, 1920], [0, 0]]);
});

t("place: não mexe na lista que recebeu", () => {
    const l = lado();
    const antes = JSON.stringify(l);
    M.place(l, "B", "acima", "A");
    M.place(l, "B", "espelhar", "A");
    eq(JSON.stringify(l), antes);
});

t("place: pedido sem sentido devolve a lista como estava", () => {
    const l = lado();
    igual(M.place(l, "B", "direita", "B"), l);
    igual(M.place(l, "Z", "direita", "A"), l);
    igual(M.place(l, "B", "direita", "Z"), l);
    const off = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, enabled: false })];
    igual(M.place(off, "A", "direita", "B"), off);
    igual(M.place(off, "B", "direita", "A"), off);
    // A referência tem de ter lugar próprio: uma que espelha outra não serve.
    const esp = [tela("A", 1920, 1080), tela("B", 1920, 1080, { mirror: "A" }), tela("C", 1920, 1080, { x: 1920 })];
    igual(M.place(esp, "C", "direita", "B"), esp);
    igual(M.place(esp, "C", "espelhar", "B"), esp);
    // Nem uma sem lugar no desenho, nem uma tela sem tamanho sabido.
    const auto = [tela("A", 1920, 1080), tela("B", 0, 0, { auto: true })];
    igual(M.place(auto, "A", "direita", "B"), auto);
    igual(M.place(auto, "B", "direita", "A"), auto);
});

t("place: espelhar tira a tela do desenho e solta quem a espelhava", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { mirror: "B" })];
    const out = M.place(l, "B", "espelhar", "A");
    eq(M.find(out, "B").mirror, "A");
    eq(M.find(out, "C").mirror, "", "C espelhava B, que agora é espelho");
    vazio(sempre(out));
    eq(M.relationTo(out, "B", "A"), "espelhar");
});

t("place: dar um lado a quem espelhava desfaz o espelho", () => {
    const l = [tela("A", 1920, 1080), tela("B", 2560, 1440, { mirror: "A", x: 77, y: 77 })];
    const out = M.place(l, "B", "abaixo", "A");
    eq(M.find(out, "B").mirror, "");
    igual([M.find(out, "B").x, M.find(out, "B").y], [0, 1080]);
});

t("place: uma tela automática ganha lugar", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { auto: true })];
    const out = M.place(l, "B", "direita", "A");
    eq(M.find(out, "B").auto, false);
    eq(M.find(out, "B").x, 1920);
});

t("place: com duas telas, qualquer pedido deixa o desenho limpo e a tela do lado pedido (300 sorteios)", () => {
    const rnd = gerador(17);
    const p = [];
    for (let i = 0; i < 300; i++) {
        const l = desenho(rnd, 2);
        const [a, b] = rnd() < 0.5 ? ["T0", "T1"] : ["T1", "T0"];
        const rel = rnd.um(LADOS);
        const out = M.place(l, a, rel, b);
        const onde = "place " + a + " " + rel + " " + b + ": " + pos(l) + " -> " + pos(out);
        p.push(...limpo(out).map((x) => x + ": " + onde));
        if (M.relationTo(out, a, b) !== rel)
            p.push("ficou '" + M.relationTo(out, a, b) + "': " + onde);
        // A outra continua do mesmo tamanho e nada some.
        if (out.map((m) => m.name).join() !== l.map((m) => m.name).join())
            p.push("a lista mudou de telas: " + onde);
    }
    vazio(p, "pedidos");
});

t("place: com três a seis telas, a tela fica do lado pedido e o desenho fica limpo (400 sorteios)", () => {
    const rnd = gerador(19);
    const p = [];
    for (let i = 0; i < 400; i++) {
        const l = desenho(rnd, rnd.int(3, 6));
        const a = rnd.um(l).name, b = rnd.um(l.filter((m) => m.name !== a)).name;
        const rel = rnd.um(LADOS);
        const out = M.place(l, a, rel, b);
        const onde = "place " + a + " " + rel + " " + b + ": " + pos(l) + " -> " + pos(out);
        p.push(...limpo(out).map((x) => x + ": " + onde));
        // A referência e a tela movida ficam alinhadas como o pedido as pôs.
        const ma = out.find((m) => m.name === a), mb = out.find((m) => m.name === b);
        if (rel === "esquerda" || rel === "direita" ? ma.y !== mb.y : ma.x !== mb.x)
            p.push("a tela não ficou alinhada com a referência: " + onde);
        if (M.relationTo(out, a, b) !== rel)
            p.push("ficou '" + M.relationTo(out, a, b) + "': " + onde);
        // As outras telas guardam o arranjo que tinham entre si enquanto não sobra buraco.
        if (out.length !== l.length)
            p.push("a lista mudou de tamanho: " + onde);
    }
    vazio(p, "pedidos");
});

t("place: no lado já ocupado a tela entra ali, e a que estava vai para além dela, em cadeia", () => {
    // A | B | C, e C vai para a direita de A: A | C | B.
    igual(M.place(tres(), "C", "direita", "A").map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 3840, 0], ["C", 1920, 0]]);
    // A | B | C | D, e D vai para a direita de A: A | D | B | C.
    const quatro = tres().concat([tela("D", 2560, 1440, { x: 5760 })]);
    igual(M.place(quatro, "D", "direita", "A").map((m) => [m.name, m.x, m.y]),
        [["A", 0, 0], ["B", 4480, 0], ["C", 6400, 0], ["D", 1920, 0]]);
    // Para a esquerda a cadeia vai para a esquerda: A | B | C, A à esquerda de C: B | A | C.
    igual(M.place(tres(), "A", "esquerda", "C").map((m) => [m.name, m.x, m.y]), [["A", 1920, 0], ["B", 0, 0], ["C", 3840, 0]]);
    // E na vertical: A sobre B sobre C, C abaixo de A: A, C, B.
    const pilha = [tela("A", 1920, 1080), tela("B", 1920, 1080, { y: 1080 }), tela("C", 1920, 1080, { y: 2160 })];
    igual(M.place(pilha, "C", "abaixo", "A").map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 0, 2160], ["C", 0, 1080]]);
    igual(M.place(pilha, "A", "acima", "C").map((m) => [m.name, m.x, m.y]), [["A", 0, 1080], ["B", 0, 0], ["C", 0, 2160]]);
});

t("place: quem não está no caminho não sai do lugar", () => {
    // A | B com C embaixo de A. Pôr B à direita de A (onde já está) não muda nada.
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { y: 1080 })];
    igual(M.place(l, "B", "direita", "A"), l);
    // B vai para baixo de C: A e C ficam onde estavam.
    igual(M.place(l, "B", "abaixo", "C").map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 0, 2160], ["C", 0, 1080]]);
    // Pedir o lado em que a tela já está, num desenho limpo, não muda nada (300 sorteios).
    const rnd = gerador(67);
    const p = [];
    for (let i = 0; i < 300; i++) {
        const d = desenho(rnd, rnd.int(2, 6));
        for (const a of d)
            for (const b of d) {
                const rel = M.relationTo(d, a.name, b.name);
                const alinhadas = rel === "esquerda" || rel === "direita" ? a.y === b.y : a.x === b.x;
                if (rel !== "" && alinhadas && pos(M.place(d, a.name, rel, b.name)) !== pos(d))
                    p.push("place " + a.name + " " + rel + " " + b.name + " mudou " + pos(d) + " para " + pos(M.place(d, a.name, rel, b.name)));
            }
    }
    vazio(p, "pedidos que não deviam mudar nada");
});

t("place: a tela que saiu do meio não deixa as outras soltas", () => {
    // A | B | C em fila, B vai para baixo de A: C encosta de novo.
    const out = M.place(tres(), "B", "abaixo", "A");
    eq(M.relationTo(out, "B", "A"), "abaixo");
    vazio(limpo(out).map((x) => x + ": " + pos(out)));
    // Uma escada: tirar o degrau do meio deixaria as pontas só pelo canto.
    const escada = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 540 }), tela("C", 1920, 1080, { x: 3840, y: 1080 })];
    vazio(limpo(escada));
    const sem = M.place(escada, "B", "acima", "A");
    eq(M.relationTo(sem, "B", "A"), "acima");
    vazio(limpo(sem).map((x) => x + ": " + pos(sem)));
});

// --- espelho ---

t("mirrorsOf: quem está espelhando esta tela", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { mirror: "A" }), tela("C", 1920, 1080, { mirror: "A" }),
        tela("D", 1920, 1080, { mirror: "A", enabled: false }), tela("E", 1920, 1080, { x: 1920 })];
    igual(M.mirrorsOf(l, "A"), ["B", "C"]);
    igual(M.mirrorsOf(l, "E"), []);
    igual(M.mirrorsOf(null, "A"), []);
});

t("unmirror: a tela ganha lugar próprio à direita de quem espelhava", () => {
    const l = [tela("A", 1920, 1080), tela("B", 2560, 1440, { mirror: "A" })];
    const out = M.unmirror(l, "B");
    eq(M.find(out, "B").mirror, "");
    igual([M.find(out, "B").x, M.find(out, "B").y], [1920, 0]);
    eq(M.relationTo(out, "B", "A"), "direita");
    vazio(limpo(out));
    eq(M.find(l, "B").mirror, "A", "a lista recebida não muda");
});

t("unmirror: com mais telas, entra à direita do desenho, na altura da que vai mais à direita", () => {
    // A | C, com C mais baixa; B espelhava A e guardava uma posição velha em cima de C.
    const l = [tela("A", 1920, 1080), tela("B", 2560, 1440, { mirror: "A", x: 2000, y: 100 }), tela("C", 1920, 1080, { x: 1920, y: 300 })];
    const out = M.unmirror(l, "B");
    igual(out.map((m) => [m.name, m.mirror, m.x, m.y]), [["A", "", 0, 0], ["B", "", 3840, 300], ["C", "", 1920, 300]]);
    vazio(limpo(out));
});

t("ex-espelho: quando o alvo é desligado ou vira espelho, a tela entra à direita do desenho e não por cima de outra", () => {
    // B espelha A e guarda a posição velha, bem em cima de C.
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { mirror: "A", x: 1920, y: 0 }), tela("C", 1920, 1080, { x: 1920 })];
    const off = M.setEnabled(l, "A", false);
    igual(off.map((m) => [m.name, m.enabled, m.mirror, m.x, m.y]), [["A", false, "", 0, 0], ["B", true, "", 1920, 0], ["C", true, "", 0, 0]]);
    vazio(limpo(off));
    // A vira espelho de C: B, que espelhava A, ganha lugar à direita.
    const esp = M.place(l, "A", "espelhar", "C");
    igual(esp.map((m) => [m.name, m.mirror, m.x, m.y]), [["A", "C", 0, 0], ["B", "", 1920, 0], ["C", "", 0, 0]]);
    vazio(limpo(esp));
    // Duas que espelhavam a mesma: uma depois da outra.
    const duas = l.concat([tela("D", 1280, 720, { mirror: "A", x: 5, y: 5 })]);
    const out = M.setEnabled(duas, "A", false);
    igual(out.map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 1920, 0], ["C", 0, 0], ["D", 3840, 0]]);
    vazio(limpo(out));
});

t("unmirror: sem modo sabido não há onde encostar, o Hyprland escolhe", () => {
    const l = [tela("A", 1920, 1080), tela("B", 0, 0, { mirror: "A" })];
    const b = M.find(M.unmirror(l, "B"), "B");
    eq(b.mirror, "");
    eq(b.auto, true);
});

t("unmirror: quem não espelha, ou não existe, fica como está", () => {
    igual(M.unmirror(lado(), "B"), lado());
    igual(M.unmirror(lado(), "Z"), lado());
});

t("mirror e unmirror: ida e volta com duas telas deixa o desenho limpo (200 sorteios)", () => {
    const rnd = gerador(23);
    const p = [];
    for (let i = 0; i < 200; i++) {
        const l = desenho(rnd, 2);
        const esp = M.place(l, "T1", "espelhar", "T0");
        if (M.find(esp, "T1").mirror !== "T0" || M.mirrorsOf(esp, "T0").join() !== "T1")
            p.push("não espelhou: " + pos(esp));
        p.push(...limpo(esp).map((x) => "espelhada, " + x + ": " + pos(esp)));
        const igualA0 = M.find(esp, "T0");
        if (igualA0.x !== 0 || igualA0.y !== 0)
            p.push("a tela que sobrou não está em 0x0: " + pos(esp));
        const volta = M.unmirror(esp, "T1");
        p.push(...limpo(volta).map((x) => "desfeito, " + x + ": " + pos(volta)));
        if (M.relationTo(volta, "T1", "T0") !== "direita")
            p.push("não voltou para a direita: " + pos(volta));
    }
    vazio(p, "espelhos");
});

// --- referências ---

t("references: as telas que servem de referência, ligadas e com lugar próprio", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { mirror: "A" }), tela("C", 1920, 1080, { enabled: false }),
        tela("D", 1920, 1080, { x: 1920 })];
    igual(M.references(l, "A").map((m) => m.name), ["D"]);
    igual(M.references(l, "D").map((m) => m.name), ["A"]);
    igual(M.references(l, "B").map((m) => m.name), ["A", "D"]);
    igual(M.references(null, "A"), []);
});

t("neighbour: a tela que ela espelha, ou em que encosta, ou a primeira que serve", () => {
    const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { x: 3840 }),
        tela("D", 1920, 1080, { mirror: "B" })];
    eq(M.neighbour(l, "C"), "B");
    eq(M.neighbour(l, "A"), "B");
    eq(M.neighbour(l, "D"), "B");
    // Solta: a primeira referência.
    const solta = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { x: 9000, y: 9000 })];
    eq(M.neighbour(solta, "C"), "A");
    eq(M.neighbour([tela("A", 1920, 1080)], "A"), "");
    eq(M.neighbour([], "A"), "");
});

// --- ligar e desligar ---

t("enabledCount: quantas estão ligadas", () => {
    eq(M.enabledCount([tela("A", 1, 1), tela("B", 1, 1, { enabled: false }), tela("C", 1, 1, { mirror: "A" })]), 2);
    eq(M.enabledCount([]), 0);
});

t("setEnabled: desligar a única tela ligada é recusado", () => {
    const uma = [tela("A", 1920, 1080)];
    igual(M.setEnabled(uma, "A", false), uma);
    const resto = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, enabled: false }), tela("C", 0, 0, { enabled: false, auto: true })];
    igual(M.setEnabled(resto, "A", false), resto);
    // Uma depois da outra: a última fica.
    let l = tres();
    for (const nome of ["A", "B", "C", "A", "B", "C"])
        l = M.setEnabled(l, nome, false);
    eq(M.enabledCount(l), 1);
    igual(l.filter((m) => m.enabled).map((m) => m.name), ["C"]);
});

t("setEnabled: desligar tira a tela do desenho, fecha o buraco e solta quem a espelhava", () => {
    const l = tres().concat([tela("D", 1920, 1080, { mirror: "B" })]);
    const out = M.setEnabled(l, "B", false);
    eq(M.find(out, "B").enabled, false);
    eq(M.find(out, "D").mirror, "");
    igual([M.find(out, "A").x, M.find(out, "C").x], [0, 1920]);
    eq(M.find(l, "B").enabled, true, "a lista recebida não muda");
    // Quem espelhava e é desligada deixa de espelhar.
    const esp = M.setEnabled([tela("A", 1920, 1080), tela("B", 1920, 1080, { mirror: "A" })], "B", false);
    eq(M.find(esp, "B").mirror, "");
});

t("setEnabled: uma tela religada entra à direita das outras", () => {
    const l = [tela("A", 1920, 1080), tela("B", 2560, 1440, { x: 1920 }), tela("C", 1920, 1080, { x: 300, y: 300, enabled: false, mirror: "A" })];
    const out = M.setEnabled(l, "C", true);
    const c = M.find(out, "C");
    igual([c.enabled, c.mirror, c.auto, c.x, c.y], [true, "", false, 4480, 0]);
    vazio(limpo(out));
});

t("setEnabled: religada sem modo sabido, o Hyprland escolhe", () => {
    const l = [tela("A", 1920, 1080), tela("B", 0, 0, { enabled: false, mode: "lixo" })];
    const b = M.find(M.setEnabled(l, "B", true), "B");
    igual([b.enabled, b.auto, b.mode], [true, true, "preferred"]);
});

t("setEnabled: pedir o que já é, ou uma tela que não existe, não muda nada", () => {
    igual(M.setEnabled(lado(), "A", true), lado());
    igual(M.setEnabled(lado(), "Z", false), lado());
    const off = [tela("A", 1920, 1080), tela("B", 1920, 1080, { enabled: false, x: 55 })];
    igual(M.setEnabled(off, "B", false), off);
});

t("setEnabled: religar num desenho limpo deixa o desenho limpo (300 sorteios)", () => {
    const rnd = gerador(29);
    const p = [];
    for (let i = 0; i < 300; i++) {
        const l = desenho(rnd, rnd.int(2, 6));
        const nome = rnd.um(l).name;
        const m = l.find((x) => x.name === nome);
        m.enabled = false;
        m.x = rnd.int(-5000, 5000);
        m.y = rnd.int(-5000, 5000);
        const base = M.normalize(M.copy(l));
        p.push(...limpo(base).map((x) => "antes de religar, " + x + ": " + pos(base)));
        const out = M.setEnabled(base, nome, true);
        // As que já estavam não saem do lugar.
        if (pos(out.filter((x) => x.name !== nome)) !== pos(base.filter((x) => x.name !== nome)))
            p.push("religar " + nome + " mexeu nas outras: " + pos(base) + " -> " + pos(out));
        p.push(...limpo(out).map((x) => x + ": religar " + nome + ": " + pos(base) + " -> " + pos(out)));
    }
    vazio(p, "religadas");
});

t("setEnabled: desligar qualquer tela de um desenho limpo deixa o desenho limpo (400 sorteios)", () => {
    const rnd = gerador(71);
    const p = [];
    for (let i = 0; i < 400; i++) {
        const l = desenho(rnd, rnd.int(2, 6));
        const nome = rnd.um(l).name;
        const out = M.setEnabled(l, nome, false);
        if (out.find((m) => m.name === nome).enabled)
            p.push("não desligou " + nome);
        p.push(...limpo(out).map((x) => x + ": desligar " + nome + ": " + pos(l) + " -> " + pos(out)));
    }
    vazio(p, "desligadas");
});

t("setEnabled: tirar o degrau do meio de uma escada não deixa as pontas só pelo canto", () => {
    const escada = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 1080 }), tela("C", 1920, 1080, { x: 0, y: 1080 })];
    vazio(limpo(escada));
    const out = M.setEnabled(escada, "C", false);
    vazio(limpo(out).map((x) => x + ": " + pos(out)));
});

// --- resize e pickMode ---

t("resize: a tela encolhe e a da direita acompanha a borda", () => {
    const out = M.resize(lado(), "A", { scale: 2 });
    igual(out.map((m) => [m.x, m.y]), [[0, 0], [960, 0]]);
    eq(M.find(out, "A").scale, 2);
});

t("resize: a tela cresce e as de baixo e da direita abrem espaço", () => {
    const l = [tela("A", 3840, 2160, { scale: 2 }), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { y: 1080 })];
    const out = M.resize(l, "A", { scale: 1 });
    igual(out.map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["B", 3840, 0], ["C", 0, 2160]]);
    vazio(limpo(out));
});

t("resize: girar troca largura e altura, e as vizinhas acompanham", () => {
    const out = M.resize(lado(), "A", { transform: 1 });
    igual(out.map((m) => [m.x, m.y]), [[0, 0], [1080, 0]]);
    const pilha = [tela("A", 1920, 1080), tela("B", 1920, 1080, { y: 1080 })];
    igual(M.resize(pilha, "A", { transform: 3 }).map((m) => [m.x, m.y]), [[0, 0], [0, 1920]]);
});

t("resize: a escala pedida vira uma que o Hyprland aceita, e o modo acompanha", () => {
    const l = [tela("A", 1366, 768, { modes: [{ w: 1366, h: 768, r: 60 }, { w: 1366, h: 768, r: 40 }] })];
    eq(M.resize(l, "A", { scale: 1.25 })[0].scale, M.fitScale(1366, 768, 1.25));
    eq(M.resize(l, "A", { refresh: 40 })[0].mode, "1366x768@40");
    // Uma taxa fora da lista, com EDID, vira modeline.
    const edid = [tela("A", 1920, 1080, { limits: FULLHD })];
    eq(M.resize(edid, "A", { refresh: 75 })[0].mode, "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync");
});

t("resize: tela que não existe devolve a lista; a recebida nunca muda", () => {
    igual(M.resize(lado(), "Z", { scale: 2 }), lado());
    const l = lado();
    const antes = JSON.stringify(l);
    M.resize(l, "A", { scale: 2, transform: 1 });
    eq(JSON.stringify(l), antes);
});

t("resize: num desenho em L, a vizinha puxada não vai para cima de uma terceira", () => {
    // A em cima à esquerda, B larga embaixo, C à direita de A e em cima de B.
    const l = [tela("A", 1920, 1080), tela("B", 3840, 2160, { y: 1080 }), tela("C", 1920, 1080, { x: 1920 })];
    vazio(limpo(l));
    const out = M.resize(l, "A", { scale: 2 });
    assert.ok(!sobrepostas(out), "ficou por cima: " + pos(out));
    vazio(sempre(out));
    // A tela de baixo e à direita não era vizinha de A: C fica onde estava sobre B.
    const d = [tela("A", 1920, 1080, { transform: 1, modes: [{ w: 1920, h: 1080, r: 60 }, { w: 1280, h: 720, r: 60 }] }),
        tela("B", 3840, 2160, { y: 1920 }), tela("C", 1920, 1080, { x: 3840, y: 1920, transform: 1 })];
    vazio(limpo(d));
    const out2 = M.pickMode(d, "A", 1280, 720);
    assert.ok(!sobrepostas(out2), "ficou por cima: " + pos(out2));
});

t("resize e pickMode: num desenho limpo, qualquer troca deixa o desenho limpo (600 sorteios)", () => {
    const rnd = gerador(31);
    const p = [];
    for (let i = 0; i < 600; i++) {
        const l = desenho(rnd, rnd.int(2, 6));
        const nome = rnd.um(l).name;
        let out, feito;
        if (rnd() < 0.3) {
            const modo = rnd.um(l.find((m) => m.name === nome).modes);
            feito = "pickMode " + nome + " " + modo.w + "x" + modo.h;
            out = M.pickMode(l, nome, modo.w, modo.h);
        } else {
            const troca = rnd.um([{ scale: 1 }, { scale: 1.25 }, { scale: 1.5 }, { scale: 2 }, { transform: 0 }, { transform: 1 },
                { transform: 2 }, { transform: 3 }, { scale: 2, transform: 1 }]);
            feito = "resize " + nome + " " + JSON.stringify(troca);
            out = M.resize(l, nome, troca);
        }
        const onde = feito + ": " + pos(l) + " -> " + pos(out);
        p.push(...limpo(out).map((x) => x + ": " + onde));
    }
    vazio(p, "trocas");
});

t("pickMode: trocar a resolução fica na taxa mais perto da atual", () => {
    const modos = [{ w: 1920, h: 1080, r: 60 }, { w: 1920, h: 1080, r: 144 }, { w: 1280, h: 720, r: 60 }, { w: 1280, h: 720, r: 120 }];
    const l = [tela("A", 1920, 1080, { refresh: 144, mode: "1920x1080@144", modes: modos }), tela("B", 1920, 1080, { x: 1920 })];
    const out = M.pickMode(l, "A", 1280, 720);
    igual([out[0].w, out[0].h, out[0].refresh, out[0].mode], [1280, 720, 120, "1280x720@120"]);
    eq(out[1].x, 1280, "a vizinha acompanha a borda");
    // Com a taxa dada, vale a dada.
    eq(M.pickMode(l, "A", 1280, 720, 60)[0].mode, "1280x720@60");
    eq(M.pickMode(l, "A", 1920, 1080, 60)[0].refresh, 60);
    // Resolução sem modo na lista: a taxa de agora.
    eq(M.pickMode(l, "A", 800, 600)[0].refresh, 144);
    igual(M.pickMode(l, "Z", 1280, 720), l);
});

// --- sequências de operações ---

t("depois de qualquer sequência de operações, de uma a seis telas: nenhuma por cima, nenhuma solta, canto em 0x0, o lado pedido (400 sequências)", () => {
    const rnd = gerador(37);
    const p = [];
    let passos = 0;
    for (let i = 0; i < 400 && p.length < 30; i++) {
        let l = desenho(rnd, rnd.int(1, 6));
        const historia = [pos(l)];
        for (let k = 0; k < 10; k++) {
            const antes = JSON.stringify(l);
            const [feito, out, pedido] = operacao(rnd, l);
            historia.push(feito + " -> " + pos(out));
            passos++;
            const erros = limpo(out).concat(pedido());
            if (JSON.stringify(l) !== antes)
                erros.push("a operação mexeu na lista que recebeu");
            if (out.map((m) => m.name).join() !== l.map((m) => m.name).join())
                erros.push("a lista mudou de telas");
            if (M.overlaps(out) !== sobrepostas(out))
                erros.push("overlaps diz " + M.overlaps(out));
            if (M.detached(out) !== soltas(out))
                erros.push("detached diz " + M.detached(out));
            if (pos(M.normalize(M.copy(out))) !== pos(out) || pos(M.arrange(M.copy(out))) !== pos(out))
                erros.push("arrumar de novo muda o resultado");
            if (erros.length > 0) {
                p.push(erros.join("; ") + "\n    " + historia.join("\n    "));
                break;
            }
            l = out;
        }
    }
    assert.ok(passos > 3000, "só " + passos + " passos");
    vazio(p, "sequências");
});

t("com duas telas, depois de qualquer sequência o desenho continua limpo (300 sequências)", () => {
    const rnd = gerador(41);
    const p = [];
    for (let i = 0; i < 300 && p.length < 30; i++) {
        let l = desenho(rnd, 2);
        const historia = [pos(l)];
        for (let k = 0; k < 12; k++) {
            const [feito, out] = operacao(rnd, l);
            historia.push(feito + " -> " + pos(out));
            const erros = limpo(out);
            if (erros.length > 0) {
                p.push(erros.join("; ") + "\n    " + historia.join("\n    "));
                break;
            }
            l = out;
        }
    }
    vazio(p, "sequências");
});

// --- comparar ---

t("slim: só o que vira regra", () => {
    const m = tela("A", 1920, 1080, { description: "Dell", x: 10.4, y: 0.6, scale: 1.33333333, limits: FULLHD });
    igual(M.slim(m), { name: "A", description: "Dell", w: 1920, h: 1080, refresh: 60, mode: "1920x1080@60", x: 10, y: 1,
        scale: 1.3333, transform: 0, enabled: true, mirror: "", auto: false });
    igual(Object.keys(M.slim({ name: "A" })), ["name", "description", "w", "h", "refresh", "mode", "x", "y", "scale", "transform",
        "enabled", "mirror", "auto"]);
    eq(M.slim({ name: "A" }).description, "");
    eq(M.slim({ name: "A" }).mirror, "");
    eq(M.slim({ name: "A" }).auto, false);
});

t("same: compara só o que vira regra", () => {
    const a = lado();
    const b = M.copy(a);
    b[0].modes = [];
    b[0].limits = FULLHD;
    b[1].x += 0.2;
    assert.ok(M.same(a, b));
    assert.ok(M.same(null, []));
    for (const [campo, valor] of [["x", 5], ["scale", 2], ["transform", 1], ["enabled", false], ["mirror", "B"], ["refresh", 75],
        ["mode", "1920x1080@75"], ["auto", true], ["w", 1280]]) {
        const c = M.copy(a);
        c[0][campo] = valor;
        assert.ok(!M.same(a, c), "trocar " + campo + " devia contar");
    }
    assert.ok(!M.same(a, a.slice(0, 1)));
    assert.ok(!M.same(a, a.slice().reverse()));
});

t("matches: o Hyprland ficou como o pedido", () => {
    const quer = lado();
    assert.ok(M.matches(quer, M.copy(quer)));
    // A taxa ele arredonda do jeito dele; a posição pode errar por um ponto.
    const veio = M.copy(quer);
    veio[0].refresh = 59.951;
    veio[1].x += 1;
    assert.ok(M.matches(quer, veio));
    assert.ok(M.matches(null, null));
    assert.ok(M.matches([], quer));
});

t("matches: outra regra valendo sobre a nossa não bate", () => {
    const quer = lado();
    for (const [campo, valor] of [["x", 100], ["y", 5], ["scale", 2], ["transform", 1], ["w", 1280], ["h", 720], ["enabled", false],
        ["mirror", "B"], ["refresh", 75]]) {
        const veio = M.copy(quer);
        veio[0][campo] = valor;
        assert.ok(!M.matches(quer, veio), "o Hyprland com " + campo + " = " + valor + " devia não bater");
    }
});

t("matches: só as telas no cabo contam", () => {
    const quer = lado().concat([tela("C", 1920, 1080, { x: 4480 })]);
    assert.ok(M.matches(quer, lado()));
});

t("matches: desligada e sozinha a tela religa, e é o esperado", () => {
    const quer = [tela("A", 1920, 1080, { enabled: false }), tela("B", 1920, 1080)];
    // B saiu do cabo: A religou.
    assert.ok(M.matches(quer, [tela("A", 1920, 1080)]));
    // Com B no cabo, A ligada não é o pedido.
    assert.ok(!M.matches(quer, [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 })]));
    assert.ok(M.matches(quer, [tela("A", 1920, 1080, { enabled: false, x: 999, scale: 3 }), tela("B", 1920, 1080)]));
});

t("matches: no espelho e na tela automática a posição não conta", () => {
    const quer = [tela("A", 1920, 1080), tela("B", 1920, 1080, { mirror: "A" }), tela("C", 1920, 1080, { auto: true })];
    const veio = [tela("A", 1920, 1080), tela("B", 1280, 720, { mirror: "A", x: 500 }), tela("C", 1920, 1080, { x: 1920 })];
    assert.ok(M.matches(quer, veio));
    veio[1].mirror = "";
    assert.ok(!M.matches(quer, veio));
    // Sem modo sabido, o que o Hyprland escolher serve.
    assert.ok(M.matches([tela("A", 0, 0, { auto: true })], [tela("A", 3840, 2160, { scale: 2, x: 7 })]));
});

t("matches: um modeline recusado deixa a tela na taxa de antes, e isso não bate", () => {
    const quer = [tela("A", 1920, 1080, { refresh: 75 })];
    assert.ok(M.matches(quer, [tela("A", 1920, 1080, { refresh: 74.66 })]));
    assert.ok(!M.matches(quer, [tela("A", 1920, 1080, { refresh: 60 })]));
});

// --- merge e seat ---

t("merge: as telas ligadas mais as guardadas que não estão no cabo", () => {
    const ligadas = [tela("A", 1920, 1080)];
    const guardadas = [tela("A", 1920, 1080, { x: 500 }), tela("B", 2560, 1440, { x: 1920, description: "a da mesa" })];
    const out = M.merge(ligadas, guardadas);
    igual(out.map((m) => m.name), ["A", "B"]);
    eq(out[0].x, 0, "a ligada vale pelo que está agora");
    eq(out[1].description, "a da mesa");
    igual(out, out.map((m) => M.slim(m)), "só o que vira regra");
    igual(M.merge(null, null), []);
    igual(M.merge(ligadas, null), ligadas.map((m) => M.slim(m)));
    // Lixo no arquivo não entra.
    igual(M.merge(ligadas, [null, {}, { name: 5 }, "x"]).map((m) => m.name), ["A"]);
    // E a lista guardada não é tocada.
    eq(guardadas[1].x, 1920);
});

t("merge: a tela fora do cabo volta para o lado da vizinha, onde a vizinha está agora", () => {
    // Guardado: A (1920) e B à direita. Agora A está a 200%: B encosta na borda nova.
    const guardadas = [tela("A", 1920, 1080), tela("B", 2560, 1440, { x: 1920 })];
    const out = M.merge([tela("A", 1920, 1080, { scale: 2 })], guardadas);
    igual([out[1].x, out[1].y, out[1].auto], [960, 0, false]);
    // À esquerda, acima e abaixo também.
    const esq = M.merge([tela("A", 1920, 1080)], [tela("A", 1920, 1080, { x: 2560 }), tela("B", 2560, 1440)]);
    igual([esq[1].x, esq[1].y], [-2560, 0]);
    const cima = M.merge([tela("A", 1920, 1080, { x: 300, y: 300 })], [tela("A", 1920, 1080, { y: 1440 }), tela("B", 2560, 1440)]);
    igual([cima[1].x, cima[1].y], [300, 300 - 1440]);
    const baixo = M.merge([tela("A", 1920, 1080, { transform: 1 })], [tela("A", 1920, 1080), tela("B", 2560, 1440, { y: 1080 })]);
    igual([baixo[1].x, baixo[1].y], [0, 1920]);
});

t("merge: uma tela fora do cabo encostada noutra fora do cabo acompanha a cadeia", () => {
    const guardadas = [tela("A", 1920, 1080), tela("C", 1920, 1080, { x: 4480 }), tela("B", 2560, 1440, { x: 1920 })];
    const out = M.merge([tela("A", 1920, 1080, { scale: 2 })], guardadas);
    igual(out.map((m) => [m.name, m.x, m.y]), [["A", 0, 0], ["C", 3520, 0], ["B", 960, 0]]);
    assert.ok(!sobrepostas(out) && !soltas(out));
});

t("merge: sem vizinha no desenho, as que sobram vão juntas para a direita; sem desenho, o Hyprland escolhe", () => {
    const guardadas = [tela("B", 1920, 1080, { x: 7000, y: 300 }), tela("C", 1920, 1080, { x: 8920, y: 300 })];
    const out = M.merge([tela("A", 2560, 1440)], guardadas);
    igual(out.map((m) => [m.name, m.x, m.y, m.auto]), [["A", 0, 0, false], ["B", 2560, 0, false], ["C", 4480, 0, false]]);
    const sem = M.merge([tela("A", 0, 0, { auto: true })], guardadas);
    igual(sem.map((m) => [m.name, m.auto]), [["A", true], ["B", true], ["C", true]]);
    igual(M.merge([], guardadas).map((m) => m.auto), [true, true]);
});

t("merge: a guardada desligada ou espelhada continua como estava", () => {
    const guardadas = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, enabled: false }), tela("C", 1920, 1080, { mirror: "A" })];
    const out = M.merge([tela("A", 1920, 1080, { scale: 2 })], guardadas);
    igual(out.slice(1), guardadas.slice(1).map((m) => M.slim(m)));
});

t("merge: desenho limpo com guardadas limpas dá um desenho limpo, sem mexer nas telas ligadas (300 sorteios)", () => {
    const rnd = gerador(43);
    const p = [];
    for (let i = 0; i < 300; i++) {
        // O que estava guardado: um desenho limpo. Agora algumas telas saíram
        // do cabo e as que ficaram mudaram de tamanho e de arranjo.
        const guardadas = desenho(rnd, rnd.int(2, 6));
        const ficam = guardadas.filter(() => rnd() < 0.5);
        let ligadas = M.copy(ficam);
        for (const m of ligadas) {
            m.scale = bloco.fitScale(m.w, m.h, rnd.um([1, 1.5, 2]));
            m.transform = rnd.um([0, 1]);
        }
        ligadas = desenhoDe(rnd, ligadas);
        const out = M.merge(ligadas, guardadas);
        const onde = pos(guardadas) + " com " + (pos(ligadas) || "nenhuma") + " -> " + pos(out);
        if (sobrepostas(out))
            p.push("ficou por cima: " + onde);
        if (soltas(out))
            p.push("ficou tela solta: " + onde);
        if (out.map((m) => m.name).sort().join() !== guardadas.map((m) => m.name).sort().join())
            p.push("sumiu ou sobrou tela: " + onde);
        for (const m of ligadas) {
            const o = out.find((x) => x.name === m.name);
            if (o.x !== m.x || o.y !== m.y)
                p.push("a tela ligada " + m.name + " mudou de lugar: " + onde);
        }
        if (ligadas.length === 0 && out.some((m) => !m.auto))
            p.push("sem desenho, alguma não ficou automática: " + onde);
    }
    vazio(p, "merges");
});

// Rearruma as telas dadas num desenho limpo (para o caso acima).
function desenhoDe(rnd, telas) {
    const lista = [];
    for (const m of telas) {
        m.x = 0;
        m.y = 0;
        for (let k = 0; lista.length > 0; k++) {
            const o = rnd.um(lista), a = ret(m), b = ret(o), l = rnd.um(LADOS);
            m.x = l === "esquerda" ? o.x - a.w : l === "direita" ? o.x + b.w : o.x;
            m.y = l === "acima" ? o.y - a.h : l === "abaixo" ? o.y + b.h : o.y;
            if (!sobrepostas(lista.concat([m])) || k > 200)
                break;
        }
        lista.push(m);
    }
    return lista;
}

t("seat: o lugar da tela no desenho de agora, do mesmo lado da vizinha que tinha", () => {
    const antes = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 200 })];
    const b = antes[1];
    // A vizinha mudou de lugar: o desalinho de antes continua.
    igual([M.seat(antes, [tela("A", 1920, 1080, { x: 100, y: 50 })], b).x, M.seat(antes, [tela("A", 1920, 1080, { x: 100, y: 50 })], b).y],
        [2020, 250]);
    // A vizinha encolheu e o desalinho de antes não encosta mais: alinha.
    const pequena = [tela("A", 1920, 1080, { scale: 2 })];
    const antes2 = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, y: 700 })];
    igual([M.seat(antes2, pequena, antes2[1]).x, M.seat(antes2, pequena, antes2[1]).y], [960, 0]);
});

t("seat: sem vizinha no desenho, ou com o lugar ocupado, null", () => {
    const antes = [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 }), tela("C", 1920, 1080, { x: 9000 })];
    eq(M.seat(antes, [tela("A", 1920, 1080)], antes[2]), null);
    eq(M.seat(antes, [], antes[1]), null);
    eq(M.seat(antes, [tela("A", 1920, 1080, { auto: true })], antes[1]), null);
    // O lugar à direita de A já tem outra tela.
    eq(M.seat(antes, [tela("A", 1920, 1080), tela("X", 1920, 1080, { x: 1920 })], antes[1]), null);
    // seat não mexe na tela que recebeu.
    M.seat(antes, [tela("A", 1920, 1080, { x: 500 })], antes[1]);
    eq(antes[1].x, 1920);
});

// --- o Lua gerado ---

t("quote: texto entre aspas, com aspas e barras escapadas", () => {
    eq(M.quote("DP-1"), '"DP-1"');
    eq(M.quote('a"b'), '"a\\"b"');
    eq(M.quote("a\\b"), '"a\\\\b"');
    eq(M.quote("a\nb"), '"a\\nb"');
    eq(M.quote(7), '"7"');
    eq(M.quote(""), '""');
});

t("rule: a regra de uma tela, pelo nome da saída", () => {
    eq(M.rule(tela("DP-1", 1920, 1080)),
        'hl.monitor({ output = "DP-1", mode = "1920x1080@60", position = "0x0", scale = 1, transform = 0, disabled = false })');
    eq(M.rule(tela("eDP-1", 2560, 1440, { x: 1920.4, y: -10.6, scale: 1.3333333, transform: 3, mode: "2560x1440@165" })),
        'hl.monitor({ output = "eDP-1", mode = "2560x1440@165", position = "1920x-11", scale = 1.3333, transform = 3, disabled = false })');
});

t("rule: desligada, espelho, automática e sem modo sabido", () => {
    eq(M.rule(tela("DP-1", 1920, 1080, { enabled: false, mirror: "X", x: 5 })), 'hl.monitor({ output = "DP-1", disabled = true })');
    eq(M.rule(tela("DP-2", 1920, 1080, { mirror: "DP-1" })),
        'hl.monitor({ output = "DP-2", mode = "1920x1080@60", position = "0x0", scale = 1, transform = 0, disabled = false, mirror = "DP-1" })');
    eq(M.rule(tela("DP-1", 1920, 1080, { auto: true, x: 300 })),
        'hl.monitor({ output = "DP-1", mode = "1920x1080@60", position = "auto", scale = 1, transform = 0, disabled = false })');
    eq(M.rule(tela("DP-1", 0, 0, { mode: "lixo", x: 300, scale: 0 })),
        'hl.monitor({ output = "DP-1", mode = "preferred", position = "auto", scale = 1, transform = 0, disabled = false })');
});

t("rule: a escala gravada é uma que o Hyprland aceita, e o transform fica de 0 a 7", () => {
    assert.ok(M.rule(tela("A", 1366, 768, { scale: 1.25 })).includes("scale = " + Math.round(M.fitScale(1366, 768, 1.25) * 10000) / 10000 + ","));
    assert.ok(M.rule(tela("A", 1920, 1080, { transform: 13 })).includes("transform = 5,"));
    assert.ok(M.rule(tela("A", 1920, 1080, { transform: "x" })).includes("transform = 0,"));
    assert.ok(M.rule(tela("A", 1920, 1080, { transform: undefined })).includes("transform = 0,"));
});

t("rule: o modeline vai inteiro no mode", () => {
    const modo = M.modeline(M.timing(1920, 1080, 75, FULLHD));
    assert.ok(M.rule(tela("A", 1920, 1080, { refresh: 75, mode: modo })).includes('mode = "' + modo + '",'));
});

t("lua: uma regra hl.monitor por saída, as espelhadas por último", () => {
    const l = [tela("C", 1920, 1080, { mirror: "A" }), tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920 })];
    const linhas = M.lua(l).split("\n");
    igual(linhas, [M.rule(l[1]), M.rule(l[2]), M.rule(l[0]), ""]);
    eq(M.lua([]), "");
    eq(M.lua(null), "");
    // Sem tela desligada não há bloco de religar.
    assert.ok(!M.lua(l).includes("get_monitors"));
});

t("lua: cada saída aparece em regra uma vez fora do bloco das desligadas", () => {
    const rnd = gerador(47);
    const p = [];
    for (let i = 0; i < 200; i++) {
        let l = desenho(rnd, rnd.int(1, 5));
        for (let k = 0; k < 4; k++)
            l = operacao(rnd, l)[1];
        const texto = M.lua(l);
        const antes = texto.split("\n-- Desligadas")[0];
        for (const m of l) {
            const n = antes.split('output = "' + m.name + '"').length - 1;
            if (n !== (m.enabled ? 1 : 0))
                p.push(m.name + (m.enabled ? " ligada" : " desligada") + " tem " + n + " regras: " + pos(l));
        }
        // Quem é espelhada vem antes de quem a espelha.
        for (const m of l.filter((x) => x.enabled && x.mirror))
            if (!(antes.indexOf('output = "' + m.mirror + '"') < antes.indexOf('output = "' + m.name + '"')))
                p.push(m.name + " vem antes de " + m.mirror + ", que ela espelha");
    }
    vazio(p, "regras");
});

t("fileText: o cabeçalho é só comentário, e a linha mk:telas é a mesma lista em JSON", () => {
    const l = [tela("A", 1920, 1080, { description: 'Dell "P2319H"\n-- x', limits: FULLHD }), tela("B", 2560, 1440, { x: 1920, enabled: false })];
    const texto = M.fileText(l);
    const linhas = texto.split("\n");
    const telas = linhas.filter((x) => x.startsWith("-- mk:telas "));
    eq(telas.length, 1);
    igual(JSON.parse(telas[0].slice("-- mk:telas ".length)), l.map((m) => M.slim(m)));
    const ate = linhas.indexOf(telas[0]);
    vazio(linhas.slice(0, ate + 1).filter((x) => !x.startsWith("-- ")), "linhas do cabeçalho que não são comentário");
    eq(linhas.slice(ate + 1).join("\n"), M.lua(l));
    assert.ok(texto.endsWith("\n"));
    // Vazio também vale.
    assert.ok(M.fileText(null).includes("-- mk:telas []\n"));
});

t("fileText: o que o painel guardou, o painel relê igual (ida e volta pelo merge)", () => {
    const rnd = gerador(53);
    const p = [];
    for (let i = 0; i < 100; i++) {
        let l = desenho(rnd, rnd.int(1, 4));
        for (let k = 0; k < 3; k++)
            l = operacao(rnd, l)[1];
        const linha = M.fileText(l).split("\n").find((x) => x.startsWith("-- mk:telas "));
        const relido = JSON.parse(linha.slice(12));
        if (!M.same(relido, l))
            p.push("releu diferente: " + pos(l));
        if (!M.same(M.merge(l, relido), l))
            p.push("o merge com o próprio arquivo mudou a lista: " + pos(l));
    }
    vazio(p, "arquivos");
});

// Nomes que tentam sair da string, do comentário ou do arquivo.
const HOSTIS = ['DP-1', 'a"b', "a'b", "a\\", "a\\\\", '\\"', '" })', '"\nos.execute("true")\n--', "]]", "--[[", "]==]", "--",
    "a\nb", "a\r\nb", "\n", "a\tb", "a\0b", "\u0001\u0002\u001f", "\u007f", "\u001b[31m", "\\u0041", "\\n", "\\065", "\\x41", "\\z",
    "\\u{41}", "ação çã", "日本語", "😀", "  ", "\ud800", "x\udfffy", "%s%d", "${x}", "$(reboot)", "`reboot`", "a".repeat(5000),
    '", disabled = true, x = "', "\"; hl.exec_cmd(\"true\") --", "\\\"; hl.exec_cmd(\"true\") --", "\v\f\b", ""];

function configuracoes() {
    const modo75 = M.modeline(M.timing(1920, 1080, 75, FULLHD));
    return {
        "uma tela": [tela("eDP-1", 1920, 1080)],
        "duas lado a lado": lado(),
        "uma em retrato (transform 1)": [tela("A", 1920, 1080), tela("B", 1920, 1080, { x: 1920, transform: 1 })],
        "uma em retrato (transform 3)": [tela("A", 1920, 1080, { transform: 3 }), tela("B", 1920, 1080, { x: 1080 })],
        "invertida (transform 5)": [tela("A", 1920, 1080, { transform: 5 })],
        "espelho": [tela("A", 1920, 1080), tela("B", 2560, 1440, { mirror: "A" })],
        "uma desligada": [tela("A", 1920, 1080), tela("B", 2560, 1440, { x: 1920, enabled: false })],
        "duas desligadas": [tela("A", 1920, 1080), tela("B", 2560, 1440, { enabled: false }), tela("C", 0, 0, { enabled: false, auto: true })],
        "todas desligadas (arquivo torto)": [tela("A", 1920, 1080, { enabled: false })],
        "modeline": [tela("A", 1920, 1080, { refresh: 75, mode: modo75, limits: FULLHD })],
        "modeline numa desligada": [tela("A", 1920, 1080), tela("B", 1920, 1080, { refresh: 75, mode: modo75, enabled: false })],
        "automática": [tela("A", 0, 0, { auto: true, mode: "preferred" })],
        "escala fracionada": [tela("A", 2560, 1440, { scale: 1.3333 }), tela("B", 3840, 2160, { x: 1920, scale: 1.5 })],
        "posição negativa e quebrada": [tela("A", 1920, 1080, { x: -1920.5, y: -0.2 }), tela("B", 1920, 1080)],
        "números tortos": [tela("A", 1920, 1080, { x: NaN, y: Infinity, scale: NaN, transform: "z" }), tela("B", NaN, undefined, { scale: -1 })],
        "nenhuma": []
    };
}

if (temLuac) {
    t("fileText: o Lua de cada configuração passa no luac", () => {
        const p = [];
        for (const [nome, l] of Object.entries(configuracoes())) {
            const erro = erroDoLuac(M.fileText(l));
            if (erro !== "")
                p.push(nome + ": " + erro);
            const so = erroDoLuac(M.lua(l));
            if (so !== "")
                p.push(nome + " (só o lua()): " + so);
        }
        vazio(p, "configurações que o Lua recusa");
    });

    t("fileText: nenhum nome, descrição, espelho ou modo hostil quebra o Lua", () => {
        const textos = [];
        for (const h of HOSTIS) {
            textos.push(M.fileText([tela(h, 1920, 1080, { description: h }), tela("B", 1920, 1080, { x: 1920, mirror: h })]));
            textos.push(M.fileText([tela("A", 1920, 1080), tela(h, 1920, 1080, { enabled: false, description: h, mode: h })]));
            textos.push(M.fileText([tela("A", 1920, 1080, { mode: h, description: h }), tela(h + "2", 0, 0, { enabled: false, mirror: h })]));
        }
        vazio(reprovadosNoLuac(textos), "arquivos que o Lua recusa");
    });

    t("quote: qualquer texto vira uma string que o Lua aceita (todos os caracteres de controle, 500 textos sorteados)", () => {
        const rnd = gerador(59);
        const pedacos = ['"', "\\", "\n", "\r", "\0", "]]", "--", "u", "x", "z", "{", "}", "0", "9", "'", "\u001f", "\u0085", "é", "\ud83d", "\ude00", " "];
        const textos = [];
        for (let c = 0; c < 256; c++)
            textos.push(String.fromCharCode(c), "a" + String.fromCharCode(c) + "1");
        for (let i = 0; i < 500; i++) {
            let s = "";
            for (let k = rnd.int(1, 12); k > 0; k--)
                s += rnd.um(pedacos);
            textos.push(s);
        }
        vazio(reprovadosNoLuac(textos.map((s) => "local x = " + M.quote(s))), "strings que o Lua recusa");
    });

    t("o Lua de 300 configurações sorteadas, depois de operações do painel, passa no luac", () => {
        const rnd = gerador(61);
        const textos = [];
        for (let i = 0; i < 300; i++) {
            let l = desenho(rnd, rnd.int(1, 5));
            for (let k = rnd.int(0, 6); k > 0; k--)
                l = operacao(rnd, l)[1];
            if (rnd() < 0.3)
                rnd.um(l).name = rnd.um(HOSTIS);
            textos.push(M.fileText(l));
        }
        vazio(reprovadosNoLuac(textos), "arquivos que o Lua recusa");
    });
}

if (temLua) {
    t("o Lua gerado pede ao Hyprland o que a lista diz", () => {
        const modo75 = M.modeline(M.timing(1920, 1080, 75, FULLHD));
        const l = [tela("C", 1920, 1080, { mirror: "A" }), tela("A", 1920, 1080, { refresh: 75, mode: modo75 }),
            tela("B", 2560, 1440, { x: 1920, y: -180, scale: 1.25, transform: 3 })];
        const r = rodaLua(M.fileText(l), ["A", "B", "C"]);
        igual(r.regras, [
            { output: "A", mode: modo75, position: "0x0", scale: 1, transform: 0, disabled: false },
            { output: "B", mode: "2560x1440@60", position: "1920x-180", scale: 1.25, transform: 3, disabled: false },
            { output: "C", mode: "1920x1080@60", position: "0x0", scale: 1, transform: 0, disabled: false, mirror: "A" }
        ]);
        igual(r.eventos, []);
        igual(r.comandos, []);
    });

    t("o Lua gerado: o nome hostil chega ao Hyprland byte a byte, e nada mais roda", () => {
        const p = [];
        for (const h of HOSTIS) {
            const l = [tela(h, 1920, 1080, { description: h, mode: h }), tela("B", 1920, 1080, { x: 1920, mirror: h }),
                tela(h + "!", 1920, 1080, { enabled: false, description: h })];
            const r = rodaLua(M.fileText(l), [h, "B"]);
            const visto = JSON.stringify(h).slice(0, 60);
            if (r.regras.length !== 3)
                p.push(visto + ": " + r.regras.length + " regras em vez de 3");
            else {
                // Um pedaço de UTF-16 sem par não existe em UTF-8: vira U+FFFD, como em qualquer gravação.
                if (hex(r.regras[0].output) !== hex(h) || hex(r.regras[0].mode) !== hex(h))
                    p.push(visto + ": a saída ou o modo chegou diferente: " + JSON.stringify(r.regras[0]));
                // Espelho de nome vazio é não espelhar.
                if (hex(r.regras[1].mirror || "") !== hex(h))
                    p.push(visto + ": o espelho chegou diferente");
                if (hex(r.regras[2].output) !== hex(h + "!") || r.regras[2].disabled !== true)
                    p.push(visto + ": a desligada chegou diferente: " + JSON.stringify(r.regras[2]));
                if (Object.keys(r.regras[0]).sort().join() !== "disabled,mode,output,position,scale,transform")
                    p.push(visto + ": a regra ganhou ou perdeu campo: " + Object.keys(r.regras[0]));
            }
            if (r.comandos.length > 0)
                p.push(visto + ": rodou " + JSON.stringify(r.comandos));
        }
        vazio(p, "nomes hostis");
    });

    t("o Lua gerado: a tela desligada fica desligada enquanto houver outra", () => {
        const l = [tela("A", 1920, 1080), tela("B", 2560, 1440, { x: 1920, scale: 1.25, transform: 1, enabled: false, mirror: "" })];
        const r = rodaLua(M.fileText(l), ["A"], ["A"]);
        igual(r.regras, [{ output: "A", mode: "1920x1080@60", position: "0x0", scale: 1, transform: 0, disabled: false },
            { output: "B", disabled: true }]);
        // Fica de olho em tela saindo do cabo; enquanto A estiver lá, nada recarrega.
        igual(r.eventos, ["monitor.removed"]);
        igual(r.comandos, []);
    });

    t("o Lua gerado: a outra tela saiu do cabo, o arquivo é relido; sozinha, a desligada religa", () => {
        const l = [tela("A", 1920, 1080), tela("B", 2560, 1440, { x: 1920, scale: 1.25, transform: 1, enabled: false })];
        // A sai do cabo com o arquivo já carregado.
        igual(rodaLua(M.fileText(l), ["A"], []).comandos, ["hyprctl reload"]);
        // O arquivo relido sem A: B religa, com o modo, a escala e a rotação que tinha, onde o Hyprland quiser.
        const r = rodaLua(M.fileText(l), [], []);
        igual(r.regras.filter((x) => x.output === "B"),
            [{ output: "B", mode: "2560x1440@60", position: "auto", scale: 1.25, transform: 1, disabled: false }]);
        igual(r.eventos, ["monitor.added"]);
        igual(r.comandos, []);
        // A volta: o arquivo é relido de novo, e B desliga.
        igual(rodaLua(M.fileText(l), [], ["A"]).comandos, ["hyprctl reload"]);
    });

    t("o Lua gerado: saída virtual, FALLBACK e a própria desligada não contam como outra tela", () => {
        const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { enabled: false }), tela("C", 1920, 1080, { enabled: false })];
        for (const ligadas of [["HEADLESS-2"], ["FALLBACK"], ["B", "C"], ["HEADLESS-1", "FALLBACK", "B"], []]) {
            const r = rodaLua(M.fileText(l), ligadas);
            igual(r.regras.filter((x) => x.output !== "A").map((x) => [x.output, x.disabled]), [["B", false], ["C", false]],
                "com " + JSON.stringify(ligadas) + " as desligadas deviam religar");
            igual(r.eventos, ["monitor.added"]);
        }
        // Uma tela de verdade com nome parecido conta.
        const r = rodaLua(M.fileText(l), ["HEADLESS", "B"]);
        igual(r.regras.filter((x) => x.output !== "A").map((x) => [x.output, x.disabled]), [["B", true], ["C", true]]);
    });

    t("o Lua gerado: se o Hyprland não souber listar as telas, a desligada religa (melhor que sem imagem)", () => {
        const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { enabled: false })];
        const r = rodaLua(M.fileText(l), null);
        igual(r.regras.filter((x) => x.output === "B").map((x) => x.disabled), [false]);
    });

    t("o Lua gerado: uma desligada que espelhava religa com lugar próprio", () => {
        const l = [tela("A", 1920, 1080), tela("B", 1920, 1080, { enabled: false, mirror: "A", x: 500 })];
        const b = rodaLua(M.fileText(l), []).regras.find((x) => x.output === "B");
        igual(b, { output: "B", mode: "1920x1080@60", position: "auto", scale: 1, transform: 0, disabled: false });
    });
}

// --- o resumo da lista ---

t("summary: uma linha sobre a tela", () => {
    eq(M.summary(tela("A", 1920, 1080)), "1920 × 1080 · 60 Hz · 100%");
    eq(M.summary(tela("A", 2560, 1440, { refresh: 143.912, scale: 1.25 })), "2560 × 1440 · 143,91 Hz · 125%");
    eq(M.summary(tela("A", 1920, 1080, { transform: 1 })), "1920 × 1080 · 60 Hz · 100% · 90°");
    eq(M.summary(tela("A", 1920, 1080, { transform: 2 })), "1920 × 1080 · 60 Hz · 100% · 180°");
    eq(M.summary(tela("A", 1920, 1080, { transform: 3 })), "1920 × 1080 · 60 Hz · 100% · 270°");
    eq(M.summary(tela("A", 1920, 1080, { transform: 4 })), "1920 × 1080 · 60 Hz · 100% · invertida");
    eq(M.summary(tela("A", 1920, 1080, { transform: 5 })), "1920 × 1080 · 60 Hz · 100% · 90° · invertida");
    eq(M.summary(tela("A", 1920, 1080, { enabled: false, mirror: "B" })), "desligada");
    eq(M.summary(tela("A", 1920, 1080, { mirror: "B" })), "espelha B");
    eq(M.summary(tela("A", 0, 0, { auto: true })), "modo automático");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
