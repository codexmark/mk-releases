pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland

// As telas ligadas ao computador, para a página Monitores dos Ajustes.
// `live` é o que o Hyprland tem agora (scripts/mare-monitors list); `draft` é
// o que a pessoa está montando no painel. Nada muda na tela até apply(); aí
// ~/.config/mk/monitores.lua (uma regra hl.monitor por saída, carregado
// depois do ajustes.lua) é regravado, o Hyprland recarrega e começa a
// contagem: sem um keep() em 15 s, o arquivo anterior volta. Quem devolve é
// um vigia do script, fora do Maré, para a tela voltar mesmo sem imagem.
// Uma tela desligada só fica desligada enquanto houver outra: o próprio
// monitores.lua religa quando ela fica sozinha (ver lua()).
Singleton {
    id: root

    readonly property string luaPath: Settings.dir + "/monitores.lua"
    readonly property string script: Quickshell.shellPath("scripts/mare-monitors")
    // O monitors.lua pessoal, ao lado do hyprland.lua: carrega por último e
    // vale sobre o monitores.lua do painel.
    readonly property string userRules: Settings.dir + "/hypr/monitors.lua"
    readonly property int trialSeconds: 15

    property var live: []
    property var saved: []
    property var draft: []
    // As saídas com regra no monitors.lua pessoal (procura de texto do script).
    property var ruled: []
    property string selected: ""
    property bool loaded: false

    // Contagem depois de aplicar.
    property bool pending: false
    property int remaining: 0
    property var applied: []
    property string error: ""
    // Depois de voltar, o rascunho que estava em teste reaparece para ser
    // corrigido, em vez de sumir.
    property var comeback: null
    // A releitura que fecha uma troca (a segunda, com o Hyprland já parado).
    property bool lastRead: false
    // Já deu tempo de o Hyprland trocar e ele não ficou como pedido.
    property bool settled: false
    readonly property bool differs: pending && settled && !matches(applied, live)

    readonly property bool dirty: !pending && !same(draft, live)
    readonly property var current: find(draft, selected)
    readonly property bool overlap: overlaps(draft)
    readonly property bool loose: detached(draft)

    // --- puro: início ---
    // Daqui até "puro: fim" nada toca no Maré nem no Hyprland: são funções de
    // lista para lista e de lista para texto, testadas fora do shell com node.
    // Uma tela: { name, description, w, h, refresh, mode, x, y, scale,
    // transform, enabled, mirror, auto, modes: [{ w, h, r }], limits }. w/h
    // são os pontos do modo; x/y e o tamanho lógico já contam escala e rotação.
    // limits é o que o monitor declara no EDID ({ vmin, vmax (Hz), hmin, hmax
    // (kHz), clock (MHz) }, ou null); mode é "LxA@taxa" ou, numa taxa fora da
    // lista do monitor, o "modeline ..." calculado em timing().
    // transform é o do Hyprland, 0 a 7: a rotação é transform % 4 e de 4 em
    // diante a imagem está invertida (flipped), o que o painel só preserva.

    function copy(list) {
        return JSON.parse(JSON.stringify(list || []));
    }

    function find(list, name) {
        for (const m of list || [])
            if (m.name === name)
                return m;
        return null;
    }

    function round2(n) {
        return Math.round(n * 100) / 100;
    }

    // O Hyprland só aceita escalas (múltiplos de 1/120) que dão um tamanho
    // lógico inteiro; fora disso ele mesmo troca pela mais próxima. A conta é
    // a dele, para o painel só oferecer o que vai valer.
    function fitScale(w, h, want) {
        const zero = Math.round((want > 0 ? want : 1) * 120);
        if (!(w > 0 && h > 0))
            return zero / 120;
        const ok = k => k > 0 && (w * 120) % k === 0 && (h * 120) % k === 0;
        for (let i = 0; i < 90; i++) {
            if (ok(zero + i))
                return (zero + i) / 120;
            if (ok(zero - i))
                return (zero - i) / 120;
        }
        return 1;
    }

    readonly property var wantedScales: [1, 1.25, 1.5, 2]

    function scaleOptions(m) {
        const out = [];
        const wanted = wantedScales.slice();
        if (m)
            wanted.push(m.scale);
        for (const s of wanted) {
            const fit = m ? fitScale(m.w, m.h, s) : s;
            if (!out.some(o => Math.abs(o - fit) < 0.004))
                out.push(fit);
        }
        return out.sort((a, b) => a - b);
    }

    // As escalas de sempre que esta resolução não aceita (1366x768 fica sem
    // 125% e 150%), para a página dizer por que sumiram.
    function missingScales(m) {
        if (!m || !(m.w > 0 && m.h > 0))
            return [];
        return wantedScales.filter(s => !sameScale(fitScale(m.w, m.h, s), s));
    }

    function sameScale(a, b) {
        return Math.abs(a - b) < 0.006;
    }

    function logical(m) {
        const s = m.scale > 0 ? m.scale : 1;
        const w = Math.round(m.w / s);
        const h = Math.round(m.h / s);
        return m.transform % 2 === 1 ? { w: h, h: w } : { w: w, h: h };
    }

    // Está no desenho: ligada, com lugar próprio e tamanho conhecido.
    function placed(m) {
        return m.enabled && !m.mirror && !m.auto && m.w > 0 && m.h > 0;
    }

    function modeText(w, h, r) {
        return w + "x" + h + "@" + round2(r);
    }

    // O modo da lista mais perto do que a tela está usando (o Hyprland
    // informa 60.012 e lista 60.01).
    function nearestMode(m) {
        let best = null;
        for (const mode of m.modes || []) {
            if (mode.w !== m.w || mode.h !== m.h)
                continue;
            if (!best || Math.abs(mode.r - m.refresh) < Math.abs(best.r - m.refresh))
                best = mode;
        }
        return best;
    }

    // Menos linhas de intervalo vertical que isto e a imagem falha (12 foram
    // poucas num Dell P2319H; com 25 ficou boa).
    readonly property int minVblank: 25

    // O sinal de uma taxa que o monitor não lista, dentro dos limites que ele
    // declara; null se não cabe. Horizontal como no CVT-RBv2 (80 pontos de
    // intervalo: 8 + 32 + 40). No vertical, o intervalo do CVT com blanking
    // reduzido (460 µs), nunca menos que minVblank (numa tela de 768 linhas a
    // 60 Hz os 460 µs são 22), se couber; se não, o maior que os limites
    // deixam: a frequência de linha fica 0,1 kHz abaixo do teto (o EDID o dá
    // em kHz inteiros) e o pixel clock não passa do máximo. Se com isso a
    // linha fica abaixo do piso do monitor (640x480 a 60 Hz com piso de
    // 31 kHz), o intervalo cresce até 0,1 kHz acima dele, enquanto o teto e
    // o clock deixarem. O `WxH@taxa` do Hyprland usa o CVT de blanking normal,
    // que estoura esses limites.
    function timing(w, h, r, limits) {
        const l = limits;
        if (!l || !(w > 0 && h > 0 && r > 0) || !(l.vmax > 0 && l.hmax > 0 && l.clock > 0))
            return null;
        if (r > l.vmax || r < (l.vmin || 0))
            return null;
        const htotal = w + 80;
        const floor = l.hmin > 0 ? Math.ceil((l.hmin * 1000 + 100) / r) : 0;
        const vtotal = Math.min(Math.max(Math.ceil(h / (1 - 460e-6 * r)), h + minVblank, floor),
            Math.floor((l.hmax * 1000 - 100) / r),
            Math.floor(l.clock * 1e6 / (htotal * r)));
        if (!(vtotal - h >= minVblank) || vtotal * r < (l.hmin || 0) * 1000)
            return null;
        // O Hyprland guarda o clock em kHz inteiros.
        return { w: w, h: h, r: r, htotal: htotal, vtotal: vtotal, khz: Math.floor(htotal * vtotal * r / 1000) };
    }

    // "modeline 165.75 1920 1928 1960 2000 1080 1083 1088 1105 +hsync -vsync"
    function modeline(t) {
        return ["modeline", String(t.khz / 1000), t.w, t.w + 8, t.w + 40, t.htotal,
            t.h, t.h + 3, t.h + 8, t.vtotal, "+hsync", "-vsync"].join(" ");
    }

    // O que um "modeline ..." guardado mostra: { w, h, r }, ou null.
    function modelineInfo(mode) {
        const p = String(mode || "").trim().split(/\s+/);
        if (p[0] !== "modeline" || p.length < 10)
            return null;
        const n = p.slice(1, 10).map(Number);
        if (!n.every(v => v > 0))
            return null;
        return { w: n[1], h: n[5], r: n[0] * 1e6 / (n[4] * n[8]) };
    }

    // A taxa está na lista do monitor para a resolução em uso?
    function listed(m, r) {
        return rates(m).some(x => Math.abs(x - r) < 0.006);
    }

    // As taxas inteiras acima da maior da lista, até o teto vertical do EDID,
    // que cabem nos limites (ver timing). Da mais alta para a mais baixa.
    function extraRates(m) {
        const all = rates(m);
        if (!m || !m.limits || all.length === 0)
            return [];
        const out = [];
        for (let r = Math.floor(m.limits.vmax); r > Math.round(Math.max.apply(null, all)); r--)
            if (timing(m.w, m.h, r, m.limits))
                out.push(r);
        return out;
    }

    // O modo que vai para a regra: o da lista pelo nome; fora dela, o
    // modeline calculado. Sem limites para calcular (o EDID sumiu), vale o
    // modeline que já estava guardado para esta resolução e taxa.
    function modeFor(m) {
        if (!listed(m, m.refresh)) {
            const t = timing(m.w, m.h, m.refresh, m.limits);
            if (t)
                return modeline(t);
            const had = modelineInfo(m.mode);
            if (had && had.w === m.w && had.h === m.h && Math.abs(had.r - m.refresh) < 0.5)
                return m.mode;
        }
        return modeText(m.w, m.h, m.refresh);
    }

    // A taxa que o painel mostra para o que o Hyprland informa. Num modeline
    // ele informa 74,66 para 75: vale a taxa fora da lista que o arquivo
    // guardou, ou a que o painel ofereceria, se estiver mais perto que
    // qualquer modo da lista.
    function shownRate(m, old) {
        const near = nearestMode(m);
        const gap = near ? Math.abs(near.r - m.refresh) : Infinity;
        if (old && old.w === m.w && old.h === m.h && modelineInfo(old.mode)
            && Math.abs(old.refresh - m.refresh) < Math.min(1, gap))
            return Number(old.refresh);
        let best = null;
        for (const r of extraRates(m))
            if (Math.abs(r - m.refresh) < Math.min(0.6, gap) && (best === null || Math.abs(r - m.refresh) < Math.abs(best - m.refresh)))
                best = r;
        return best !== null ? best : near ? near.r : m.refresh;
    }

    // O que o `mare-monitors list` devolve -> a tela do painel. Uma tela
    // desligada vale pelo que o arquivo guardou dela, se guardou.
    function entry(raw, saved) {
        const m = {
            name: String(raw.name), description: String(raw.description || ""),
            w: Number(raw.w) || 0, h: Number(raw.h) || 0, refresh: Number(raw.refresh) || 0,
            x: Number(raw.x) || 0, y: Number(raw.y) || 0, scale: Number(raw.scale) || 1,
            transform: (Number(raw.transform) || 0) % 8, enabled: raw.enabled !== false,
            mirror: String(raw.mirror || ""), auto: false, mode: "preferred",
            modes: raw.modes || [], limits: raw.limits || null
        };
        const old = find(saved, m.name);
        if (!m.enabled && old && old.w > 0) {
            for (const k of ["w", "h", "refresh", "x", "y", "scale", "transform"])
                m[k] = Number(old[k]) || 0;
            m.scale = m.scale || 1;
        }
        if (m.w > 0 && m.h > 0) {
            m.refresh = shownRate(m, old);
            if (old && modelineInfo(old.mode))
                m.mode = old.mode;
            m.mode = modeFor(m);
            m.scale = fitScale(m.w, m.h, m.scale);
        } else {
            m.auto = true;
        }
        return m;
    }

    // Resoluções diferentes da tela, da maior para a menor.
    function resolutions(m) {
        const out = [];
        for (const mode of (m && m.modes) || [])
            if (!out.some(o => o.w === mode.w && o.h === mode.h))
                out.push({ w: mode.w, h: mode.h });
        return out;
    }

    // Taxas da resolução em uso, da mais alta para a mais baixa.
    function rates(m) {
        return ((m && m.modes) || []).filter(mode => mode.w === m.w && mode.h === m.h).map(mode => mode.r);
    }

    // Fecha os buracos do desenho num eixo: uma tela que ficou longe das
    // outras encosta de novo. O cursor não atravessa um buraco entre telas,
    // então uma tela solta ficaria fora do alcance do mouse. Só olha um eixo
    // (duas telas pelo canto passam por ele): quem arruma o desenho é o
    // arrange.
    function closeGaps(list, axis, side) {
        const shown = list.filter(placed).sort((a, b) => a[axis] - b[axis]);
        let edge = -Infinity, shift = 0;
        for (const m of shown) {
            const start = m[axis] - shift;
            if (edge !== -Infinity && start > edge + 1) {
                shift += start - edge;
                m[axis] = edge;
            } else {
                m[axis] = start;
            }
            edge = Math.max(edge, m[axis] + logical(m)[side]);
        }
    }

    // A primeira tela de `others` que está por cima de `m`, ou null.
    function clash(m, others) {
        const a = logical(m);
        for (const o of others) {
            const b = logical(o);
            if (o !== m && m.x < o.x + b.w && o.x < m.x + a.w && m.y < o.y + b.h && o.y < m.y + a.h)
                return o;
        }
        return null;
    }

    // Os grupos de telas do desenho que encostam umas nas outras por um lado
    // (só pelo canto não é encostar: o mouse não passa). Aqui encostar é sem
    // folga nenhuma: um ponto de distância conta como solta, para o arrange
    // fechar (o relationTo e o detached perdoam esse ponto).
    function islands(list) {
        const shown = (list || []).filter(placed);
        const touch = (m, o) => {
            const a = logical(m), b = logical(o);
            return ((m.x + a.w === o.x || o.x + b.w === m.x) && m.y < o.y + b.h && o.y < m.y + a.h)
                || ((m.y + a.h === o.y || o.y + b.h === m.y) && m.x < o.x + b.w && o.x < m.x + a.w);
        };
        const groups = [];
        for (const first of shown) {
            if (groups.some(g => g.indexOf(first) >= 0))
                continue;
            const g = [first];
            for (let i = 0; i < g.length; i++)
                for (const o of shown)
                    if (g.indexOf(o) < 0 && touch(g[i], o))
                        g.push(o);
            groups.push(shown.filter(m => g.indexOf(m) >= 0));
        }
        return groups;
    }

    // O arranjador: deixa o desenho sem tela por cima de outra e sem tela
    // solta, mexendo o mínimo. As telas de `keep` (a que a pessoa moveu e a
    // referência dela) não saem do lugar; as outras é que cedem.
    //
    // 1. Quem está por cima de outra é empurrada para além dela, em cadeia:
    //    as de `keep` primeiro, e depois cada tela, na ordem em que aparecem
    //    nesse sentido, vai para depois de todas as já assentadas em que
    //    bate. Com `side` (o lado de um place) é sempre nesse eixo e nesse
    //    sentido; sem ele, para a direita ou para baixo, o que for menos.
    // 2. Cada grupo solto anda inteiro até encostar no desenho por um lado,
    //    pelo caminho mais curto que não o põe por cima de ninguém: fica com
    //    o desalinho que tinha se assim encosta, ou alinha pela borda. O
    //    desenho é o grupo das telas de `keep` (sem `keep`, o maior grupo).
    //
    // Arrumar de novo o que já está arrumado não muda nada. Mexe na lista
    // recebida e a devolve; não leva o desenho para 0x0 (ver normalize).
    function arrange(list, keep, side) {
        const shown = (list || []).filter(placed);
        for (const m of shown) {
            m.x = Math.round(m.x);
            m.y = Math.round(m.y);
        }
        const fixed = shown.filter(m => (keep || []).indexOf(m.name) >= 0);

        const axis = side === "acima" || side === "abaixo" ? "y" : "x";
        const span = axis === "x" ? "w" : "h";
        const back = side === "esquerda" || side === "acima";
        const front = m => back ? -(m[axis] + logical(m)[span]) : m[axis];
        const settled = fixed.slice();
        const rest = shown.filter(m => fixed.indexOf(m) < 0)
            .sort((a, b) => front(a) - front(b) || shown.indexOf(a) - shown.indexOf(b));
        for (const m of rest) {
            for (let hit = clash(m, settled); hit; hit = clash(m, settled)) {
                const down = !side && hit.y + logical(hit).h - m.y < hit.x + logical(hit).w - m.x;
                if (down)
                    m.y = hit.y + logical(hit).h;
                else
                    m[axis] = back ? hit[axis] - logical(m)[span] : hit[axis] + logical(hit)[span];
            }
            settled.push(m);
        }

        const groups = islands(shown);
        let big = null;
        for (const g of groups)
            if (!big || g.length > big.length)
                big = g;
        let main = [];
        const loose = [];
        for (const g of groups) {
            if (fixed.length > 0 ? g.some(m => fixed.indexOf(m) >= 0) : g === big)
                main = main.concat(g);
            else
                loose.push(g);
        }
        while (loose.length > 0) {
            let best = null;
            for (const g of loose)
                for (const c of g)
                    for (const t of main)
                        for (const rel of ["direita", "esquerda", "abaixo", "acima"]) {
                            const a = logical(c), b = logical(t);
                            const beside = rel === "direita" || rel === "esquerda";
                            const along = rel === "direita" ? t.x + b.w - c.x : rel === "esquerda" ? t.x - a.w - c.x
                                : rel === "abaixo" ? t.y + b.h - c.y : t.y - a.h - c.y;
                            const p = beside ? "y" : "x", q = beside ? "h" : "w";
                            const slides = [t[p] - c[p], t[p] + b[q] - c[p] - a[q]];
                            if (c[p] < t[p] + b[q] && t[p] < c[p] + a[q])
                                slides.unshift(0);
                            for (const across of slides) {
                                const dx = beside ? along : across, dy = beside ? across : along;
                                const cost = Math.abs(dx) + Math.abs(dy);
                                if (best && cost >= best.cost)
                                    continue;
                                if (g.some(m => clash(Object.assign({}, m, { x: m.x + dx, y: m.y + dy }), main)))
                                    continue;
                                best = { group: g, dx: dx, dy: dy, cost: cost };
                            }
                        }
            if (!best)
                break;
            for (const m of best.group) {
                m.x += best.dx;
                m.y += best.dy;
            }
            main = main.concat(best.group);
            loose.splice(loose.indexOf(best.group), 1);
        }
        return list;
    }

    // Põe `m` à direita do desenho, na altura da tela que vai mais à direita,
    // para encostar nela. É por onde entra uma tela religada ou uma que
    // deixou de espelhar: a posição que ela guardava era de outro desenho.
    function enter(list, m) {
        let right = -Infinity, top = 0;
        for (const o of list)
            if (o !== m && placed(o) && o.x + logical(o).w > right) {
                right = o.x + logical(o).w;
                top = o.y;
            }
        m.x = right === -Infinity ? 0 : right;
        m.y = top;
    }

    // Arruma o desenho (ver arrange) e leva o canto de cima à esquerda para
    // 0x0: sem tela por cima de outra, sem tela solta, sem buracos.
    function normalize(list, keep, side) {
        arrange(list, keep, side);
        let minX = Infinity, minY = Infinity;
        for (const m of list)
            if (placed(m)) {
                minX = Math.min(minX, m.x);
                minY = Math.min(minY, m.y);
            }
        if (minX === Infinity)
            return list;
        for (const m of list)
            if (placed(m)) {
                m.x = Math.round(m.x - minX);
                m.y = Math.round(m.y - minY);
            }
        return list;
    }

    // Muda modo, escala ou rotação de uma tela. Se ela cresce ou encolhe, as
    // que estavam à direita ou abaixo acompanham a borda, sem abrir buraco
    // nem ficar por cima. Num desenho em L, puxar quem está além da borda que
    // recuou poria uma tela por cima de outra: aí a borda só empurra, e quem
    // ficou longe é o normalize que encosta.
    function resize(list, name, patch) {
        const out = copy(list);
        const m = find(out, name);
        if (!m)
            return out;
        const before = logical(m);
        Object.assign(m, patch);
        if (m.w > 0 && m.h > 0) {
            m.scale = fitScale(m.w, m.h, m.scale);
            m.mode = modeFor(m);
        }
        const after = logical(m);
        if (!placed(m))
            return normalize(out);
        const follow = pull => {
            const next = copy(out);
            for (const o of next) {
                if (o.name === name || !placed(o))
                    continue;
                if (o.x >= m.x + before.w - 1 && (pull || after.w > before.w))
                    o.x += after.w - before.w;
                if (o.y >= m.y + before.h - 1 && (pull || after.h > before.h))
                    o.y += after.h - before.h;
            }
            return next;
        };
        const pulled = follow(true);
        return normalize(overlaps(pulled) ? follow(false) : pulled, [name]);
    }

    function pickMode(list, name, w, h, r) {
        const m = find(list, name);
        if (!m)
            return copy(list);
        // Trocar a resolução fica na taxa mais perto da atual.
        let rate = r;
        if (rate === undefined) {
            const probe = { w: w, h: h, refresh: m.refresh, modes: m.modes };
            const near = nearestMode(probe);
            rate = near ? near.r : m.refresh;
        }
        return resize(list, name, { w: w, h: h, refresh: rate });
    }

    // rel: "esquerda" | "direita" | "acima" | "abaixo" | "espelhar", sempre
    // em relação à tela `other`. É o pedido que manda: a tela fica desse lado
    // e as outras cedem. Se o lado já tinha uma tela, ela vai para além da
    // que chegou, no mesmo eixo, em cadeia; e quem ficaria solta encosta de
    // novo (ver arrange). Quem espelhava a tela que virou espelho ganha lugar
    // próprio (ver enter).
    function place(list, name, rel, other) {
        const out = copy(list);
        const m = find(out, name);
        const o = find(out, other);
        if (!m || !o || m === o || !m.enabled || !o.enabled)
            return out;
        // A referência tem de ter lugar próprio: uma tela que espelha outra
        // não serve (a página nem a oferece).
        if (o.mirror)
            return out;
        if (rel === "espelhar") {
            m.mirror = other;
            for (const x of out)
                if (x.mirror === name) {
                    x.mirror = "";
                    enter(out, x);
                }
            return normalize(out, [other]);
        }
        if (!placed(o) || !(m.w > 0))
            return out;
        const a = logical(m), b = logical(o);
        m.mirror = "";
        m.auto = false;
        m.x = rel === "esquerda" ? o.x - a.w : rel === "direita" ? o.x + b.w : o.x;
        m.y = rel === "acima" ? o.y - a.h : rel === "abaixo" ? o.y + b.h : o.y;
        return normalize(out, [other, name], rel);
    }

    // Quem está espelhando esta tela.
    function mirrorsOf(list, name) {
        return (list || []).filter(m => m.enabled && m.mirror === name).map(m => m.name);
    }

    // Desfaz o espelho de `name`: ela ganha lugar próprio à direita do
    // desenho (com duas telas, à direita de quem espelhava), sem tirar
    // ninguém do lugar.
    function unmirror(list, name) {
        const out = copy(list);
        const m = find(out, name);
        if (!m || !m.mirror)
            return out;
        m.mirror = "";
        if (!m.enabled || !(m.w > 0 && m.h > 0)) {
            // Sem modo sabido não há onde encostar: o Hyprland escolhe.
            m.auto = true;
            return out;
        }
        m.auto = false;
        enter(out, m);
        return normalize(out);
    }

    // De que lado de `other` a tela está, ou "" se não encosta nela.
    function relationTo(list, name, other) {
        const m = find(list, name);
        const o = find(list, other);
        if (!m || !o || m === o)
            return "";
        if (m.mirror)
            return m.mirror === other ? "espelhar" : "";
        if (!placed(m) || !placed(o))
            return "";
        const a = logical(m), b = logical(o);
        const sideBySide = m.y < o.y + b.h && o.y < m.y + a.h;
        const stacked = m.x < o.x + b.w && o.x < m.x + a.w;
        if (sideBySide && Math.abs(m.x + a.w - o.x) <= 1)
            return "esquerda";
        if (sideBySide && Math.abs(o.x + b.w - m.x) <= 1)
            return "direita";
        if (stacked && Math.abs(m.y + a.h - o.y) <= 1)
            return "acima";
        if (stacked && Math.abs(o.y + b.h - m.y) <= 1)
            return "abaixo";
        return "";
    }

    // As telas que servem de referência para a posição de `name`: ligadas e
    // com lugar próprio (uma que espelha outra não tem lugar).
    function references(list, name) {
        return (list || []).filter(o => o.name !== name && o.enabled && !o.mirror);
    }

    // A tela de referência natural: a que ela espelha ou em que encosta.
    function neighbour(list, name) {
        const m = find(list, name);
        const refs = references(list, name);
        if (m && m.mirror && refs.some(o => o.name === m.mirror))
            return m.mirror;
        for (const o of refs)
            if (relationTo(list, name, o.name) !== "")
                return o.name;
        return refs.length > 0 ? refs[0].name : "";
    }

    function enabledCount(list) {
        return list.filter(m => m.enabled).length;
    }

    // Liga ou desliga. A última tela ligada não desliga; uma tela religada
    // entra à direita das outras, na altura da que vai mais à direita, para
    // encostar nela (ou onde o Hyprland quiser, sem modo sabido). Quem
    // espelhava a tela desligada entra do mesmo jeito, e as que ficariam
    // soltas sem ela encostam de novo.
    function setEnabled(list, name, on) {
        const out = copy(list);
        const m = find(out, name);
        if (!m || m.enabled === on)
            return out;
        if (!on) {
            if (enabledCount(out) <= 1)
                return out;
            m.enabled = false;
            m.mirror = "";
            for (const o of out)
                if (o.mirror === name) {
                    o.mirror = "";
                    enter(out, o);
                }
            return normalize(out);
        }
        m.enabled = true;
        m.mirror = "";
        if (!(m.w > 0 && m.h > 0)) {
            m.auto = true;
            m.mode = "preferred";
            return out;
        }
        m.auto = false;
        enter(out, m);
        return normalize(out);
    }

    function overlaps(list) {
        const shown = (list || []).filter(placed);
        for (let i = 0; i < shown.length; i++)
            for (let j = i + 1; j < shown.length; j++) {
                const a = shown[i], b = shown[j];
                const la = logical(a), lb = logical(b);
                if (a.x < b.x + lb.w && b.x < a.x + la.w && a.y < b.y + lb.h && b.y < a.y + la.h)
                    return true;
            }
        return false;
    }

    // Alguma tela não encosta em nenhuma outra por um lado inteiro (só pelo
    // canto, por exemplo): o mouse não chega nela.
    function detached(list) {
        const shown = (list || []).filter(placed);
        if (shown.length < 2)
            return false;
        const seen = [shown[0].name];
        for (let i = 0; i < seen.length; i++)
            for (const o of shown)
                if (seen.indexOf(o.name) < 0 && relationTo(shown, seen[i], o.name) !== "")
                    seen.push(o.name);
        return seen.length < shown.length;
    }

    // O retângulo que contém o desenho inteiro.
    function bounds(list) {
        let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
        for (const m of list || []) {
            if (!placed(m))
                continue;
            const l = logical(m);
            x0 = Math.min(x0, m.x);
            y0 = Math.min(y0, m.y);
            x1 = Math.max(x1, m.x + l.w);
            y1 = Math.max(y1, m.y + l.h);
        }
        return x0 === Infinity ? { x: 0, y: 0, w: 1, h: 1 } : { x: x0, y: y0, w: Math.max(1, x1 - x0), h: Math.max(1, y1 - y0) };
    }

    // Só o que vira regra: é o que se compara e o que se guarda.
    function slim(m) {
        return {
            name: m.name, description: m.description || "", w: m.w, h: m.h, refresh: m.refresh,
            mode: m.mode, x: Math.round(m.x), y: Math.round(m.y), scale: Math.round(m.scale * 10000) / 10000,
            transform: m.transform, enabled: m.enabled, mirror: m.mirror || "", auto: m.auto === true
        };
    }

    function same(a, b) {
        return JSON.stringify((a || []).map(slim)) === JSON.stringify((b || []).map(slim));
    }

    // O Hyprland ficou como o pedido? Compara só as telas no cabo e só o que
    // se vê (a taxa ele arredonda do jeito dele). Se não bate, outra regra
    // está valendo sobre a nossa (o hypr/monitors.lua pessoal carrega depois).
    function matches(want, got) {
        const on = (got || []).filter(m => m.enabled).length;
        for (const w of want || []) {
            const g = find(got, w.name);
            if (!g)
                continue;
            // Desligada e sozinha, a tela religa: é o esperado.
            if (!w.enabled && g.enabled && on <= 1)
                continue;
            if (w.enabled !== g.enabled)
                return false;
            if (!w.enabled)
                continue;
            if ((w.mirror || "") !== (g.mirror || ""))
                return false;
            if (w.mirror || !(w.w > 0))
                continue;
            // Um modeline recusado deixa a tela na taxa de antes.
            if (Math.abs(w.refresh - g.refresh) > 1)
                return false;
            if (w.w !== g.w || w.h !== g.h || w.transform !== g.transform || !sameScale(w.scale, g.scale))
                return false;
            if (!w.auto && (Math.abs(w.x - g.x) > 1 || Math.abs(w.y - g.y) > 1))
                return false;
        }
        return true;
    }

    // Texto -> string Lua entre aspas. O escape do JSON é aceito pelo Lua,
    // menos o \uXXXX (no Lua é \u{XXXX}, e nem todo Lua tem): um caractere de
    // controle vai em decimal (\027) e um pedaço de UTF-16 sem par, que não
    // existe em UTF-8, vira U+FFFD.
    function quote(s) {
        return JSON.stringify(String(s)).replace(/\\(?:u([0-9a-fA-F]{4})|.)/g, (all, hex) => {
            if (hex === undefined)
                return all;
            const code = parseInt(hex, 16);
            return code < 256 ? "\\" + ("00" + code).slice(-3) : "\\239\\191\\189";
        });
    }

    // A regra de uma tela. Vai pelo nome da saída: o Hyprland procura primeiro
    // uma regra com o nome e só depois a geral (output = ""), então estas
    // valem sobre a escala geral do ajustes.lua, em qualquer ordem.
    function rule(m) {
        if (!m.enabled)
            return "hl.monitor({ output = " + quote(m.name) + ", disabled = true })";
        const known = m.w > 0 && m.h > 0;
        const parts = [
            "output = " + quote(m.name),
            "mode = " + quote(known ? m.mode : "preferred"),
            "position = " + quote(m.auto || !known ? "auto" : Math.round(m.x) + "x" + Math.round(m.y)),
            "scale = " + String(Math.round((known ? fitScale(m.w, m.h, m.scale) : m.scale || 1) * 10000) / 10000),
            "transform = " + String((Number(m.transform) || 0) % 8),
            "disabled = false"
        ];
        if (m.mirror)
            parts.push("mirror = " + quote(m.mirror));
        return "hl.monitor({ " + parts.join(", ") + " })";
    }

    // O Lua que aplica a lista: as espelhadas por último, depois de quem elas
    // espelham.
    //
    // As desligadas vão num bloco à parte: `disabled = true` gravado e a outra
    // tela fora do cabo deixariam o computador sem imagem nenhuma, também no
    // próximo login. Então a regra só desliga enquanto houver outra tela
    // (hl.get_monitors só lista as que estão mostrando imagem); sozinha, a
    // tela religa, e o arquivo é relido quando a situação muda. Uma saída
    // virtual (HEADLESS-n) não conta como outra tela: ninguém olha para ela.
    function lua(list) {
        const on = (list || []).filter(m => m.enabled);
        const off = (list || []).filter(m => !m.enabled);
        const sorted = on.filter(m => !m.mirror).concat(on.filter(m => m.mirror));
        let text = sorted.map(rule).join("\n") + (sorted.length > 0 ? "\n" : "");
        if (off.length === 0)
            return text;
        const wake = m => rule(Object.assign({}, m, { enabled: true, auto: true, mirror: "" }));
        text += "\n-- Desligadas só enquanto houver outra tela: sozinhas, religam.\n"
            + "local off = { " + off.map(m => "[" + quote(m.name) + "] = true").join(", ") + " }\n"
            + "local function others()\n"
            + "    local ok, list = pcall(hl.get_monitors)\n"
            + "    local n = 0\n"
            + "    if ok and type(list) == \"table\" then\n"
            + "        for _, m in ipairs(list) do\n"
            + "            local found, name = pcall(function() return m.name end)\n"
            + "            if found and type(name) == \"string\" and name ~= \"FALLBACK\"\n"
            + "                and not name:match(\"^HEADLESS%-\") and not off[name] then\n"
            + "                n = n + 1\n"
            + "            end\n"
            + "        end\n"
            + "    end\n"
            + "    return n\n"
            + "end\n"
            + "local alone = others() == 0\n"
            + "if alone then\n"
            + off.map(m => "    " + wake(m) + "\n").join("")
            + "else\n"
            + off.map(m => "    " + rule(m) + "\n").join("")
            + "end\n"
            + "hl.on(alone and \"monitor.added\" or \"monitor.removed\", function()\n"
            + "    if (others() == 0) ~= alone then\n"
            + "        hl.exec_cmd(\"hyprctl reload\")\n"
            + "    end\n"
            + "end)\n";
        return text;
    }

    // As telas ligadas agora mais as guardadas que não estão ligadas (a tela
    // da mesa do escritório não some do arquivo quando o notebook viaja).
    //
    // A posição guardada de uma tela fora do cabo valia para o desenho de
    // antes. Ela volta para o lado da vizinha que tinha, no lugar em que a
    // vizinha está agora; sem vizinha no desenho, o Hyprland escolhe. No
    // fim o arranjador garante que nenhuma fica solta nem por cima de outra,
    // sem mexer nas telas ligadas.
    function merge(connected, saved) {
        const out = (connected || []).map(slim);
        const old = (saved || []).filter(m => m && typeof m.name === "string").map(slim);
        const away = old.filter(m => !find(out, m.name)).map(m => Object.assign({}, m));
        const drawn = out.slice();
        const wait = away.filter(placed);
        for (let again = true; again;) {
            again = false;
            for (const m of wait.slice()) {
                const spot = seat(old, drawn, m);
                if (!spot)
                    continue;
                m.x = spot.x;
                m.y = spot.y;
                drawn.push(m);
                wait.splice(wait.indexOf(m), 1);
                again = true;
            }
        }
        // As que sobraram vão juntas para a direita do desenho, com o arranjo
        // que tinham entre si; sem desenho ou sem lugar, o Hyprland escolhe.
        const b = bounds(drawn), g = bounds(wait), room = drawn.some(placed);
        for (const m of wait) {
            if (room) {
                m.x += b.x + b.w - g.x;
                m.y += b.y - g.y;
            } else {
                m.auto = true;
            }
        }
        // As ligadas ficam onde estão; as de fora do cabo é que se ajeitam.
        if (room)
            arrange(out.concat(away), out.filter(placed).map(m => m.name));
        return out.concat(away);
    }

    // O lugar de `m` (como estava em `old`) no desenho `drawn`: do mesmo lado
    // de uma tela em que encostava. null se não há onde.
    function seat(old, drawn, m) {
        const a = logical(m);
        for (const o of drawn) {
            const rel = placed(o) ? relationTo(old, m.name, o.name) : "";
            if (rel === "")
                continue;
            const was = find(old, o.name);
            const b = logical(o);
            const side = rel === "esquerda" || rel === "direita";
            const spot = Object.assign({}, m, {
                x: rel === "esquerda" ? o.x - a.w : rel === "direita" ? o.x + b.w : o.x + m.x - was.x,
                y: rel === "acima" ? o.y - a.h : rel === "abaixo" ? o.y + b.h : o.y + m.y - was.y
            });
            // A vizinha encolheu e o desalinho de antes não encosta mais.
            if (relationTo([spot, o], m.name, o.name) === "") {
                if (side)
                    spot.y = o.y;
                else
                    spot.x = o.x;
            }
            if (!overlaps(drawn.concat([spot])))
                return spot;
        }
        return null;
    }

    // O monitores.lua inteiro. A linha "mk:telas" é a mesma lista em JSON,
    // para o painel reler o que guardou.
    function fileText(list) {
        return "-- Gerado pelos Ajustes do Mk (Super + ,), página Monitores. Não edite: o painel reescreve.\n"
            + "-- Uma regra por tela, pelo nome da saída; valem sobre a regra geral do ajustes.lua.\n"
            + "-- Um mode \"modeline ...\" é uma taxa fora da lista do monitor, calculada pelos limites do EDID.\n"
            + "-- A regra troca a da saída inteira: o que o painel não tem (vrr, bitdepth, cm...) vai\n"
            + "-- no seu hypr/monitors.lua, que carrega depois e vale sobre este arquivo.\n"
            + "-- mk:telas " + JSON.stringify((list || []).map(slim)) + "\n"
            + lua(list);
    }

    function hz(r) {
        return String(round2(r)).replace(".", ",") + " Hz";
    }

    function percent(s) {
        return Math.round(s * 100) + "%";
    }

    // Uma linha sobre a tela, para a lista: "1920 × 1080 · 60 Hz · 100%".
    function summary(m) {
        if (!m.enabled)
            return "desligada";
        if (m.mirror)
            return "espelha " + m.mirror;
        if (!(m.w > 0))
            return "modo automático";
        return m.w + " × " + m.h + " · " + hz(m.refresh) + " · " + percent(m.scale)
            + (m.transform % 4 ? " · " + (m.transform % 4) * 90 + "°" : "")
            + (m.transform >= 4 ? " · invertida" : "");
    }
    // --- puro: fim ---

    // O que o painel chama. Durante a contagem nada se edita.
    function select(name) {
        if (find(draft, name))
            selected = name;
    }

    // Toda mudança do rascunho passa por aqui.
    function edit(next) {
        if (pending)
            return;
        comeback = null;
        error = "";
        draft = next;
    }

    function setMode(w, h, r) {
        edit(pickMode(draft, selected, w, h, r));
    }

    function setScale(s) {
        edit(resize(draft, selected, { scale: s }));
    }

    // t é a rotação (0 a 3); a inversão que a tela já tinha continua.
    function setTransform(t) {
        const flip = current && current.transform >= 4 ? 4 : 0;
        edit(resize(draft, selected, { transform: flip + t % 4 }));
    }

    function setPlace(rel, other) {
        edit(place(draft, selected, rel, other));
    }

    // Desfaz o espelho de quem espelha a tela escolhida.
    function splitMirror(name) {
        edit(unmirror(draft, name));
    }

    function toggle(on) {
        edit(setEnabled(draft, selected, on));
    }

    function discard() {
        edit(copy(live));
    }

    function refresh() {
        if (lister.running)
            again = true;
        else
            lister.running = true;
    }
    property bool again: false

    function apply() {
        if (pending || !dirty || trial.running || starter.running)
            return;
        error = "";
        comeback = null;
        settled = false;
        lastRead = false;
        applied = normalize(copy(draft));
        pending = true;
        remaining = trialSeconds;
        // O vigia do script espera um pouco mais que o painel: quem decide é
        // o painel, e o vigia só age sozinho se o Maré não estiver mais lá.
        trial.command = ["bash", script, "trial", String(trialSeconds + 4), luaPath, fileText(merge(applied, saved))];
        starter.restart();
    }

    function keep() {
        if (!pending || trial.running || starter.running)
            return;
        countdown.stop();
        pending = false;
        Quickshell.execDetached(["bash", script, "keep"]);
        settle.restart();
    }

    function revert() {
        if (!pending || trial.running || starter.running)
            return;
        countdown.stop();
        pending = false;
        // O que estava em teste volta como rascunho, para corrigir só o que
        // estava errado.
        comeback = copy(applied);
        Quickshell.execDetached(["bash", script, "revert"]);
        settle.restart();
    }

    function received(data, last) {
        const list = (data.monitors || []).map(m => entry(m, data.saved || []));
        const names = l => l.map(m => m.name).sort().join("\n");
        const keepDraft = dirty && names(list) === names(live);
        saved = data.saved || [];
        ruled = (data.monitors || []).filter(m => m.ruled === true).map(m => String(m.name));
        live = list;
        if (comeback && !pending) {
            // Só enquanto as telas forem as mesmas; e a última releitura
            // depois de voltar encerra o assunto.
            if (names(comeback) === names(list))
                draft = copy(comeback);
            else
                draft = copy(list);
            if (last || names(comeback) !== names(list))
                comeback = null;
        } else if (!keepDraft) {
            draft = copy(list);
        }
        if (!find(draft, selected)) {
            const focused = (data.monitors || []).find(m => m.focused);
            selected = focused ? focused.name : draft.length > 0 ? draft[0].name : "";
        }
        if (last && pending)
            settled = true;
        loaded = true;
    }

    Process {
        id: lister
        command: ["bash", root.script, "list", root.luaPath, root.userRules]
        stdout: StdioCollector {
            onStreamFinished: {
                // Com outra leitura na fila, a que fecha é a próxima.
                const last = root.lastRead && !root.again;
                if (last)
                    root.lastRead = false;
                try {
                    root.received(JSON.parse(text), last);
                } catch (e) {
                    console.warn("monitores: lista ilegível: " + e);
                }
                if (root.again) {
                    root.again = false;
                    lister.running = true;
                }
            }
        }
    }

    Process {
        id: trial
        stdout: StdioCollector {
            onStreamFinished: {
                if (text.trim() === "ok") {
                    root.remaining = root.trialSeconds;
                    countdown.restart();
                } else {
                    root.pending = false;
                    root.error = "não deu para gravar o monitores.lua";
                    console.warn("monitores: mare-monitors trial: " + text.trim());
                }
                settle.restart();
            }
        }
    }

    // Um instante antes de gravar: o Settings grava o ajustes.lua 400 ms
    // depois do último toque, e o reload tem de ler o novo, não o antigo.
    Timer {
        id: starter
        interval: 450
        onTriggered: trial.running = true
    }

    Timer {
        id: countdown
        interval: 1000
        repeat: true
        onTriggered: {
            root.remaining -= 1;
            if (root.remaining <= 0)
                root.revert();
        }
    }

    // O Hyprland leva um instante para trocar de modo: relê duas vezes.
    Timer {
        id: settle
        interval: 700
        onTriggered: {
            root.refresh();
            settleAgain.restart();
        }
    }
    Timer {
        id: settleAgain
        interval: 2000
        onTriggered: {
            root.lastRead = true;
            root.refresh();
        }
    }

    // Tela ligada ou tirada do cabo.
    Connections {
        target: Hyprland
        function onRawEvent(event) {
            if (event.name.startsWith("monitoradded") || event.name.startsWith("monitorremoved"))
                settle.restart();
        }
    }
}
