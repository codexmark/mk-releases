// shell/mare/SettingsSchema.js: os Ajustes como dados. As invariantes do
// esquema (toda linha bem formada, nada apontando para o que não existe) e
// cada função: o que aparece, a busca, o Lua do Hyprland, o hypridle.conf e
// como a máquina dorme.
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { load, REPO } from "./load.mjs";

const S = load("SettingsSchema.js");
const K = load("Keybinds.js");
const linha = (key) => {
    const r = S.rowOf(key);
    assert.ok(r, "não existe a linha " + key);
    return r;
};
const paginas = [].concat(...S.groups.map((g) => g.items));
const CTLS = ["switch", "slider", "choice"];
const PRECISA = ["touchpad", "bateria", "brilho"];
// Os "@..." que o hyprCode sabe pôr no Lua por um caminho próprio.
const ARROBAS = ["@animations", "@monitor.scale", "@kb_options"];
const DIR = "/home/ana/.config/mk/hypr";

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["defaults", "shown", "optionShown", "options", "pageRows", "rowOf", "touchesHypr",
        "pageName", "groupOf", "fold", "search", "kbOptions", "ours", "animationCode", "luaValue", "hyprTable",
        "valueOf", "hyprCode", "hyprLua", "sleepMode", "sleepCommand", "sleepStep", "hypridleConf", "orphan"];
    const funcoes = Object.keys(S).filter((k) => typeof S[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)), [], "função sem teste: acrescente o caso e ponha na lista");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

// --- invariantes: páginas ---

t("páginas: chaves únicas, nome e número em sequência (01, 02...)", () => {
    const ruins = [];
    const visto = {};
    paginas.forEach((p, i) => {
        if (!/^[a-z]+$/.test(p.key) || visto[p.key])
            ruins.push("chave: " + p.key);
        visto[p.key] = true;
        if (typeof p.name !== "string" || p.name.trim() === "")
            ruins.push(p.key + ": sem nome");
        if (p.num !== String(i + 1).padStart(2, "0"))
            ruins.push(p.key + ": num " + p.num + ", devia ser " + String(i + 1).padStart(2, "0"));
    });
    for (const g of S.groups)
        if (typeof g.name !== "string" || g.name === "" || g.name !== g.name.toUpperCase() || g.items.length === 0)
            ruins.push("grupo: " + g.name);
    const nomes = S.groups.map((g) => g.name);
    if (nomes.some((n, i) => nomes.indexOf(n) !== i))
        ruins.push("grupo repetido");
    vazio(ruins, "páginas ruins");
});

t("páginas: toda página tem a frase de abertura (blurbs), e nenhuma frase sobra", () => {
    igual(paginas.filter((p) => typeof S.blurbs[p.key] !== "string" || S.blurbs[p.key].trim() === "").map((p) => p.key), []);
    igual(Object.keys(S.blurbs).filter((k) => !paginas.some((p) => p.key === k)), []);
});

// --- invariantes: linhas ---

t("linhas: toda linha tem page, group, key, label, desc e ctl válidos", () => {
    const ruins = [];
    for (const r of S.rows) {
        const quem = r.key || JSON.stringify(r);
        if (typeof r.key !== "string" || !/^[a-z]+\.[A-Za-z]+$/.test(r.key))
            ruins.push(quem + ": key");
        if (!paginas.some((p) => p.key === r.page))
            ruins.push(quem + ": page " + r.page + " não está na lista de páginas");
        if (typeof r.label !== "string" || r.label.trim() === "")
            ruins.push(quem + ": label");
        // A busca junta desc e group ao texto: têm de ser texto.
        if (typeof r.desc !== "string")
            ruins.push(quem + ": desc");
        if (typeof r.group !== "string" || r.group === "" || r.group !== r.group.toUpperCase())
            ruins.push(quem + ": group");
        if (!CTLS.includes(r.ctl))
            ruins.push(quem + ": ctl " + r.ctl);
        if (r.def === undefined)
            ruins.push(quem + ": sem def");
        if (r.kw !== undefined && (typeof r.kw !== "string" || r.kw.trim() === ""))
            ruins.push(quem + ": kw");
    }
    vazio(ruins, "linhas ruins");
});

t("linhas: só os campos que o cabeçalho descreve", () => {
    const campos = ["page", "group", "key", "label", "desc", "ctl", "def", "min", "max", "step", "unit",
        "options", "hypr", "kw", "quando", "precisa"];
    const ruins = [];
    for (const r of S.rows)
        for (const k of Object.keys(r))
            if (!campos.includes(k))
                ruins.push(r.key + ": " + k);
    vazio(ruins, "campos desconhecidos");
});

t("linhas: chaves únicas, e nenhuma repete uma chave escondida", () => {
    const chaves = S.rows.map((r) => r.key);
    igual(chaves.filter((k, i) => chaves.indexOf(k) !== i), []);
    igual(Object.keys(S.hidden).filter((k) => chaves.includes(k)), []);
});

t("linhas: as de uma página ficam juntas, na ordem das páginas", () => {
    const ordem = [];
    for (const r of S.rows)
        if (ordem[ordem.length - 1] !== r.page)
            ordem.push(r.page);
    igual(ordem.filter((p, i) => ordem.indexOf(p) !== i), [], "página partida em dois pedaços");
});

t("slider: min < max, step > 0, unit, e o def dentro da faixa e num passo", () => {
    const ruins = [];
    for (const r of S.rows.filter((x) => x.ctl === "slider")) {
        if (![r.min, r.max, r.step, r.def].every((n) => typeof n === "number" && isFinite(n))) {
            ruins.push(r.key + ": min/max/step/def têm de ser números");
            continue;
        }
        if (!(r.min < r.max) || !(r.step > 0))
            ruins.push(r.key + ": faixa " + r.min + ".." + r.max + " passo " + r.step);
        if (r.def < r.min || r.def > r.max)
            ruins.push(r.key + ": def " + r.def + " fora de " + r.min + ".." + r.max);
        const passos = (r.def - r.min) / r.step;
        if (Math.abs(passos - Math.round(passos)) > 1e-6)
            ruins.push(r.key + ": def " + r.def + " não cai num passo de " + r.step);
        const total = (r.max - r.min) / r.step;
        if (Math.abs(total - Math.round(total)) > 1e-6)
            ruins.push(r.key + ": o max " + r.max + " não cai num passo de " + r.step);
        if (typeof r.unit !== "string")
            ruins.push(r.key + ": unit");
        if (r.options !== undefined)
            ruins.push(r.key + ": slider com options");
    }
    vazio(ruins, "sliders ruins");
});

t("switch: o def é true ou false", () => {
    igual(S.rows.filter((r) => r.ctl === "switch" && typeof r.def !== "boolean").map((r) => r.key), []);
});

t("choice: pelo menos duas opções, valores únicos, rótulos, e o def entre elas", () => {
    const ruins = [];
    for (const r of S.rows.filter((x) => x.ctl === "choice")) {
        if (!Array.isArray(r.options) || r.options.length < 2) {
            ruins.push(r.key + ": options");
            continue;
        }
        const vs = r.options.map((o) => o.v);
        if (vs.some((v, i) => vs.indexOf(v) !== i))
            ruins.push(r.key + ": valor repetido");
        if (!vs.includes(r.def))
            ruins.push(r.key + ": def " + JSON.stringify(r.def) + " não está nas options");
        for (const o of r.options) {
            if (typeof o.label !== "string" || o.label.trim() === "")
                ruins.push(r.key + ": opção sem label");
            if (!["string", "number"].includes(typeof o.v) || typeof o.v !== typeof r.def)
                ruins.push(r.key + ": opção " + JSON.stringify(o.v) + " de tipo diferente do def");
            if (o.precisa !== undefined && !PRECISA.includes(o.precisa))
                ruins.push(r.key + ": opção precisa de " + o.precisa);
            for (const k of Object.keys(o))
                if (!["v", "label", "precisa"].includes(k))
                    ruins.push(r.key + ": campo de opção " + k);
        }
    }
    igual(S.rows.filter((r) => r.ctl !== "choice" && r.options !== undefined).map((r) => r.key), []);
    vazio(ruins, "choices ruins");
});

t("precisa: só touchpad, bateria ou brilho (o que o mare-caps responde)", () => {
    igual(S.rows.filter((r) => r.precisa !== undefined && !PRECISA.includes(r.precisa)).map((r) => r.key), []);
});

t("quando: aponta para outra chave que existe, com valores que ela pode ter", () => {
    const ruins = [];
    for (const r of S.rows) {
        if (r.quando === undefined)
            continue;
        if (typeof r.quando !== "object" || Object.keys(r.quando).length === 0)
            ruins.push(r.key + ": quando vazio");
        for (const [k, quer] of Object.entries(r.quando)) {
            const alvo = S.rowOf(k);
            if (!alvo) {
                ruins.push(r.key + ": quando aponta para " + k + ", que não existe");
                continue;
            }
            if (k === r.key)
                ruins.push(r.key + ": quando aponta para a própria linha");
            const lista = Array.isArray(quer) ? quer : [quer];
            if (lista.length === 0)
                ruins.push(r.key + ": quando com lista vazia (nunca apareceria)");
            for (const v of lista) {
                const serve = alvo.ctl === "choice" ? alvo.options.some((o) => o.v === v)
                    : alvo.ctl === "switch" ? typeof v === "boolean"
                    : typeof v === "number" && v >= alvo.min && v <= alvo.max;
                if (!serve)
                    ruins.push(r.key + ": " + k + " nunca vale " + JSON.stringify(v));
            }
        }
    }
    vazio(ruins, "quando ruins");
});

t("quando: com tudo de fábrica, nenhuma linha depende de outra que está escondida", () => {
    // Senão a linha só apareceria mexendo num ajuste que não está na tela.
    const ruins = [];
    for (const r of S.rows)
        for (const k of Object.keys(r.quando || {}))
            if (S.rowOf(k) && S.rowOf(k).quando)
                ruins.push(r.key + " depende de " + k + ", que também tem quando");
    vazio(ruins, "quando em cadeia");
});

t("hypr: um caminho do Hyprland (a.b ou a.b.c) ou um dos @ que o hyprCode conhece, sem repetir", () => {
    const ruins = [];
    const visto = {};
    for (const r of S.rows) {
        if (r.hypr === undefined)
            continue;
        if (r.hypr[0] === "@" ? !ARROBAS.includes(r.hypr) : !/^[a-z_]+(\.[a-z_]+){1,2}$/.test(r.hypr))
            ruins.push(r.key + ": " + r.hypr);
        if (visto[r.hypr])
            ruins.push(r.key + ": mesmo campo de " + visto[r.hypr]);
        visto[r.hypr] = r.key;
    }
    vazio(ruins, "hypr ruins");
});

t("hibernação: nenhuma opção 'hibernar', e nada no esquema fala em hibernar", () => {
    const ruins = [];
    for (const r of S.rows) {
        for (const o of r.options || [])
            if (/hibern/i.test(String(o.v) + " " + o.label))
                ruins.push(r.key + ": opção " + o.v);
        if (/hibern/i.test([r.label, r.desc, r.kw || "", String(r.def)].join(" ")))
            ruins.push(r.key + ": texto");
    }
    for (const [k, texto] of Object.entries(S.blurbs))
        if (/hibern/i.test(texto))
            ruins.push("blurb " + k);
    for (const [k, cmd] of Object.entries(S.sleepCommands))
        if (/hibern/i.test(k + " " + cmd))
            ruins.push("sleepCommands." + k);
    vazio(ruins, "restos da hibernação");
    igual(linha("power.sleep").options.map((o) => o.v), ["suspender", "nada"]);
    eq(linha("power.sleep").def, "suspender");
});

// --- defaults, valueOf, rowOf ---

t("defaults: o def de toda linha mais as chaves escondidas", () => {
    const d = S.defaults();
    igual(Object.keys(d).sort(), S.rows.map((r) => r.key).concat(Object.keys(S.hidden)).sort());
    for (const r of S.rows)
        eq(d[r.key], r.def, r.key);
    eq(d["apps.terminal"], "ghostty");
    eq(d["emoji.tone"], 0);
    igual(d["emoji.recent"], []);
    eq(d["power.sleep"], "suspender");
});

t("defaults: cada chamada devolve um objeto novo", () => {
    const a = S.defaults();
    a["hypr.gapsIn"] = 99;
    eq(S.defaults()["hypr.gapsIn"], 6);
    eq(linha("hypr.gapsIn").def, 6);
});

t("defaults: cabem no JSON do ajustes.json sem perder nada", () => {
    igual(JSON.parse(JSON.stringify(S.defaults())), S.defaults());
});

t("valueOf: o valor guardado; sem ele, o de fábrica", () => {
    eq(S.valueOf({ "hypr.gapsIn": 10 }, "hypr.gapsIn"), 10);
    eq(S.valueOf({}, "hypr.gapsIn"), 6);
    eq(S.valueOf({ "hypr.gapsIn": 0 }, "hypr.gapsIn"), 0);
    eq(S.valueOf({ "island.sound": false }, "island.sound"), false);
    eq(S.valueOf({ "hypr.kbLayout": "" }, "hypr.kbLayout"), "");
    for (const r of S.rows)
        eq(S.valueOf({}, r.key), r.def, r.key);
});

t("valueOf: chave que não é de nenhuma linha dá undefined", () => {
    eq(S.valueOf({ "nao.existe": 1 }, "nao.existe"), undefined);
    eq(S.valueOf({}, "nao.existe"), undefined);
});

t("rowOf: a linha da chave; null se não existe", () => {
    eq(S.rowOf("hypr.gapsIn").label, "Entre janelas");
    eq(S.rowOf("hypr.gapsIn"), S.rows.find((r) => r.key === "hypr.gapsIn"));
    eq(S.rowOf("nao.existe"), null);
    eq(S.rowOf("apps.terminal"), null);
    eq(S.rowOf(undefined), null);
});

t("touchesHypr: só as chaves cuja linha tem hypr", () => {
    eq(S.touchesHypr("hypr.gapsIn"), true);
    eq(S.touchesHypr("motion.speed"), true);
    eq(S.touchesHypr("input.caps"), true);
    eq(S.touchesHypr("hypr.scale"), true);
    eq(S.touchesHypr("island.sound"), false);
    eq(S.touchesHypr("idle.lock"), false);
    eq(S.touchesHypr("apps.terminal"), false);
    eq(S.touchesHypr("nao.existe"), false);
});

t("pageName e groupOf: a página pela chave e o grupo dela", () => {
    igual(S.pageName("ilha"), { key: "ilha", name: "Ilha", num: "01" });
    eq(S.pageName("entrada").name, "Teclado e mouse");
    eq(S.pageName("nao.existe"), null);
    eq(S.groupOf("ilha"), "APARÊNCIA");
    eq(S.groupOf("energia"), "DISPOSITIVOS");
    eq(S.groupOf("sobre"), "SISTEMA");
    eq(S.groupOf("nao.existe"), "");
    for (const p of paginas) {
        eq(S.pageName(p.key), p);
        assert.ok(S.groupOf(p.key) !== "");
    }
});

// --- shown, optionShown, options, pageRows ---

t("shown: sem values e sem caps, toda linha aparece", () => {
    igual(S.rows.filter((r) => !S.shown(r)).map((r) => r.key), []);
});

t("shown: a linha com `precisa` some quando a máquina não tem", () => {
    const r = linha("hypr.naturalScroll");
    eq(S.shown(r, {}, { touchpad: false }), false);
    eq(S.shown(r, {}, { touchpad: true }), true);
    // Sem caps (o mare-caps ainda não respondeu), a máquina tem tudo.
    eq(S.shown(r, {}, undefined), true);
    eq(S.shown(r, {}, null), true);
    eq(S.shown(r, {}, {}), true);
    eq(S.shown(r, {}, { bateria: false, brilho: false }), true);
    eq(S.shown(linha("power.lowWarn"), {}, { bateria: false }), false);
    eq(S.shown(linha("idle.dim"), {}, { brilho: false }), false);
});

t("shown: a linha com `quando` segue o valor da outra chave (o de fábrica também conta)", () => {
    const r = linha("night.from");
    eq(S.shown(r, { "night.auto": true }), true);
    eq(S.shown(r, { "night.auto": false }), false);
    eq(S.shown(r, {}), false);
    // Sem values não se olha `quando`.
    eq(S.shown(r), true);
    eq(S.shown(r, null, { touchpad: false }), true);
});

t("shown: `quando` com lista: 'Dormir' só aparece com power.sleep = suspender", () => {
    const r = linha("idle.suspend");
    eq(S.shown(r, {}), true);
    eq(S.shown(r, { "power.sleep": "suspender" }), true);
    eq(S.shown(r, { "power.sleep": "nada" }), false);
    eq(S.shown(r, { "power.sleep": "hibernar" }), false);
});

t("shown: `quando` e `precisa` juntos: os dois têm de valer", () => {
    const r = { key: "x.y", precisa: "bateria", quando: { "night.auto": true, "power.sleep": ["nada"] } };
    eq(S.shown(r, { "night.auto": true, "power.sleep": "nada" }, { bateria: true }), true);
    eq(S.shown(r, { "night.auto": true, "power.sleep": "nada" }, { bateria: false }), false);
    eq(S.shown(r, { "night.auto": true }, { bateria: true }), false);
    eq(S.shown(r, { "power.sleep": "nada" }, { bateria: true }), false);
});

t("optionShown: a opção com `precisa` some como a linha some", () => {
    const r = linha("power.sleep");
    const o = { v: "nada", label: "x", precisa: "bateria" };
    eq(S.optionShown(r, o, { "power.sleep": "suspender" }, { bateria: false }), false);
    eq(S.optionShown(r, o, { "power.sleep": "suspender" }, { bateria: true }), true);
    eq(S.optionShown(r, o, { "power.sleep": "suspender" }, {}), true);
    eq(S.optionShown(r, o, { "power.sleep": "suspender" }, undefined), true);
    eq(S.optionShown(r, o, undefined, { bateria: false }), false);
    eq(S.optionShown(r, o, null, { bateria: false }), false);
});

t("optionShown: a opção escolhida fica, mesmo sem a máquina ter", () => {
    const r = linha("power.sleep");
    eq(S.optionShown(r, { v: "nada", label: "x", precisa: "bateria" }, { "power.sleep": "nada" }, { bateria: false }), true);
    // A de fábrica conta como escolhida quando não há valor guardado.
    eq(S.optionShown(r, { v: "suspender", label: "x", precisa: "bateria" }, {}, { bateria: false }), true);
});

t("optionShown: opção sem `precisa` aparece sempre", () => {
    const r = linha("power.sleep");
    for (const o of r.options)
        eq(S.optionShown(r, o, {}, { touchpad: false, bateria: false, brilho: false }), true);
});

t("options: as opções da linha que valem nesta máquina", () => {
    const r = { key: "power.sleep", options: [{ v: "suspender", label: "a" }, { v: "nada", label: "b", precisa: "bateria" }] };
    igual(S.options(r, {}, { bateria: false }).map((o) => o.v), ["suspender"]);
    igual(S.options(r, { "power.sleep": "nada" }, { bateria: false }).map((o) => o.v), ["suspender", "nada"]);
    igual(S.options(r, {}, { bateria: true }).map((o) => o.v), ["suspender", "nada"]);
    igual(S.options(linha("power.sleep")), linha("power.sleep").options);
    // Uma linha que não é choice não tem opções.
    igual(S.options(linha("hypr.gapsIn"), {}, {}), []);
});

t("pageRows: sem values e caps, todas as linhas da página, as escondidas também", () => {
    igual(S.pageRows("tela").map((r) => r.key),
        ["ui.scale", "hypr.scale", "night.temperature", "night.auto", "night.from", "night.to"]);
    igual(S.pageRows("sobre"), []);
    igual(S.pageRows("nao.existe"), []);
    let total = 0;
    for (const p of paginas)
        total += S.pageRows(p.key).length;
    eq(total, S.rows.length);
});

t("pageRows: com values e caps, só as que aparecem", () => {
    igual(S.pageRows("tela", {}).map((r) => r.key), ["ui.scale", "hypr.scale", "night.temperature", "night.auto"]);
    eq(S.pageRows("tela", { "night.auto": true }).length, 6);
    igual(S.pageRows("energia", { "power.sleep": "nada" }, { brilho: false, bateria: false }).map((r) => r.key),
        ["idle.lock", "idle.off", "power.sleep"]);
    igual(S.pageRows("entrada", {}, { touchpad: false }).map((r) => r.key),
        ["hypr.kbLayout", "input.caps", "hypr.repeatRate", "hypr.repeatDelay", "hypr.sensitivity"]);
});

// --- fold, search ---

t("fold: minúsculas e sem acento", () => {
    eq(S.fold("Atualizações"), "atualizacoes");
    eq(S.fold("ÁÉÍÓÚ àèìòù ÂÊÔ ãõ Ç ü"), "aeiou aeiou aeo ao c u");
    eq(S.fold("α β"), "α β");
    eq(S.fold(42), "42");
    eq(S.fold(""), "");
});

t("search: busca vazia não acha nada", () => {
    igual(S.search(""), []);
    igual(S.search("   "), []);
    igual(S.search("xyzzy quux"), []);
});

t("search: acha a linha pelo rótulo, e diz a página e o grupo", () => {
    const r = S.search("opacidade em repouso");
    igual(r, [{ page: "ilha", key: "island.restOpacity", label: "Opacidade em repouso", where: "Ilha › forma" }]);
});

t("search: sem acento e sem caixa", () => {
    const esperado = S.search("Notificações");
    assert.ok(esperado.length > 0);
    igual(S.search("notificacoes"), esperado);
    igual(S.search("NOTIFICAÇÕES"), esperado);
});

t("search: acha a página pelo nome, pelo grupo e pela frase de abertura", () => {
    igual(S.search("impressoras")[0], { page: "impressoras", key: "", label: "Impressoras", where: "página" });
    assert.ok(S.search("wireguard").some((x) => x.page === "vpn" && x.where === "página"));
    assert.ok(S.search("dispositivos").some((x) => x.page === "monitores" && x.where === "página"));
});

t("search: acha pelas palavras de busca (kw) e pela descrição", () => {
    assert.ok(S.search("wallpaper").some((x) => x.key === "backdrop.dim"));
    assert.ok(S.search("hidpi").some((x) => x.key === "hypr.scale"));
    assert.ok(S.search("vim").some((x) => x.key === "input.caps"));
    assert.ok(S.search("senha para voltar").some((x) => x.key === "idle.lock"));
});

t("search: cada palavra precisa aparecer em algum lugar", () => {
    igual(S.search("cantos retos").map((x) => x.key), ["hypr.rounding"]);
    igual(S.search("retos cantos").map((x) => x.key), ["hypr.rounding"]);
    igual(S.search("cantos bateria"), []);
});

t("search: toda linha se acha pelo próprio rótulo", () => {
    const ruins = [];
    for (const r of S.rows) {
        const achou = S.search(r.label + " " + S.pageName(r.page).name + " " + r.group);
        if (!achou.some((x) => x.key === r.key && x.page === r.page && x.label === r.label))
            ruins.push(r.key);
    }
    vazio(ruins, "linhas que a busca não acha");
});

t("search: páginas primeiro, depois as linhas, depois os atalhos", () => {
    const tipos = S.search("notificações").map((x) => x.where === "página" ? 0 : x.page === "atalhos" && x.key.indexOf("atalho:") === 0 ? 2 : 1);
    igual(tipos, tipos.slice().sort());
    assert.ok(tipos.includes(0) && tipos.includes(1) && tipos.includes(2), "esperava os três tipos: " + tipos);
});

t("search: no máximo 9 resultados", () => {
    eq(S.search("a").length, 9);
    eq(S.search("e").length, 9);
});

t("search: a linha escondida não aparece (quando e precisa)", () => {
    assert.ok(S.search("começa às").some((x) => x.key === "night.from"));
    assert.ok(!S.search("começa às", null, {}).some((x) => x.key === "night.from"));
    assert.ok(S.search("começa às", null, { "night.auto": true }).some((x) => x.key === "night.from"));
    assert.ok(S.search("rolagem natural", null, {}, {}).some((x) => x.key === "hypr.naturalScroll"));
    assert.ok(!S.search("rolagem natural", null, {}, { touchpad: false }).some((x) => x.key === "hypr.naturalScroll"));
});

t("search: acha um atalho, com o combo que vale agora", () => {
    igual(S.search("lazygit"), [{ page: "atalhos", key: "atalho:app.lazygit", label: "lazygit", where: "Atalhos › Super + G" }]);
    igual(S.search("lazygit", { "app.lazygit": "super + shift + g" }),
        [{ page: "atalhos", key: "atalho:app.lazygit", label: "lazygit", where: "Atalhos › Super + Shift + G" }]);
    igual(S.search("lazygit", { "app.lazygit": "" })[0].where, "Atalhos › sem atalho");
    // Pelo combo e pela palavra "atalho".
    assert.ok(S.search("super shift enter").some((x) => x.key === "atalho:emergency.terminal"));
    assert.ok(S.search("atalho fechar").some((x) => x.key === "atalho:win.close"));
});

t("search: todo resultado leva a uma página que existe", () => {
    for (const q of ["a", "e", "o", "tela", "super", "som", "mouse"])
        for (const x of S.search(q)) {
            assert.ok(S.pageName(x.page), q + ": página " + x.page);
            igual(Object.keys(x), ["page", "key", "label", "where"]);
        }
});

t("search: 'hibernar' não acha nada", () => {
    igual(S.search("hibernar"), []);
    igual(S.search("hibernação"), []);
});

// --- kbOptions ---

t("kbOptions: de fábrica, nenhuma opção do xkb", () => eq(S.kbOptions({}), ""));

t("kbOptions: Alt+Shift só existe com dois layouts", () => {
    eq(S.kbOptions({ "hypr.kbLayout": "br,us" }), "grp:alt_shift_toggle");
    eq(S.kbOptions({ "hypr.kbLayout": "us" }), "");
    eq(S.kbOptions({ "hypr.kbLayout": "br" }), "");
});

t("kbOptions: o Caps Lock vira o que a pessoa escolheu", () => {
    eq(S.kbOptions({ "input.caps": "esc" }), "caps:escape");
    eq(S.kbOptions({ "input.caps": "ctrl" }), "ctrl:nocaps");
    eq(S.kbOptions({ "input.caps": "normal" }), "");
    eq(S.kbOptions({ "hypr.kbLayout": "br,us", "input.caps": "ctrl" }), "grp:alt_shift_toggle,ctrl:nocaps");
});

t("kbOptions: toda opção de Caps Lock que não é 'normal' tem a sua do xkb", () => {
    for (const o of linha("input.caps").options)
        eq(S.kbOptions({ "input.caps": o.v }) !== "", o.v !== "normal", o.v);
    igual(Object.keys(S.capsOptions).filter((k) => !linha("input.caps").options.some((o) => o.v === k)), []);
});

t("kbOptions: um valor estranho no ajustes.json não vira opção do xkb", () => {
    for (const ruim of ["banana", "constructor", "toString", "__proto__", "valueOf", 42, null, true])
        eq(S.kbOptions({ "input.caps": ruim }), "", String(ruim));
});

// --- ours ---

t("ours: falso com tudo de fábrica e nada mexido", () => {
    eq(S.ours({}, undefined, ["motion.speed"]), false);
    eq(S.ours({}, {}, ["hypr.kbLayout", "input.caps"]), false);
    eq(S.ours({ "motion.speed": 1, "hypr.gapsIn": 20 }, { "hypr.gapsIn": true }, ["motion.speed"]), false);
    eq(S.ours({}, null, []), false);
});

t("ours: verdadeiro se uma das chaves saiu da fábrica ou foi mexida nesta sessão", () => {
    eq(S.ours({ "motion.speed": 0.7 }, undefined, ["motion.speed"]), true);
    eq(S.ours({ "input.caps": "esc" }, {}, ["hypr.kbLayout", "input.caps"]), true);
    eq(S.ours({ "hypr.kbLayout": "us" }, {}, ["hypr.kbLayout", "input.caps"]), true);
    // Voltou ao de fábrica nesta sessão: ainda é assunto nosso (tem de desfazer).
    eq(S.ours({}, { "motion.speed": true }, ["motion.speed"]), true);
    eq(S.ours({ "input.caps": "normal" }, { "input.caps": true }, ["hypr.kbLayout", "input.caps"]), true);
});

// --- animationCode ---

t("animações: as folhas, curvas e velocidades são as do config/hypr/mare/motion.lua", () => {
    const lua = readFileSync(resolve(REPO, "config/hypr/mare/motion.lua"), "utf8");
    const doLua = [...lua.matchAll(/^hl\.animation\(\{ leaf = "([^"]+)", enabled = true, speed = ([0-9.]+), bezier = "([^"]+)"(?:, style = "([^"]+)")? \}\)$/gm)]
        .map((m) => [m[1], Number(m[2]), m[3], m[4] || ""]);
    eq((lua.match(/^hl\.animation\(/gm) || []).length, doLua.length, "linha de hl.animation num formato que o teste não lê");
    assert.ok(doLua.length > 0);
    igual(S.animations, doLua);
});

t("animationCode: de fábrica não emite nada (as animações da pessoa ficam em paz)", () => {
    eq(S.animationCode({}), "");
    eq(S.animationCode({ "motion.speed": 1 }, {}), "");
    eq(S.animationCode({ "hypr.gapsIn": 20 }, { "hypr.gapsIn": true }), "");
});

t("animationCode: 'sem' (0) não emite nada", () => {
    eq(S.animationCode({ "motion.speed": 0 }), "");
    eq(S.animationCode({ "motion.speed": 0 }, { "motion.speed": true }), "");
});

t("animationCode: calmo (1.4) multiplica o speed de cada folha: número maior", () => {
    igual(S.animationCode({ "motion.speed": 1.4 }).split("\n"), [
        'hl.animation({ leaf = "global", enabled = true, speed = 3.36, bezier = "mare" })',
        'hl.animation({ leaf = "windows", enabled = true, speed = 3.36, bezier = "mare" })',
        'hl.animation({ leaf = "windowsIn", enabled = true, speed = 2.8, bezier = "mare", style = "popin 95%" })',
        'hl.animation({ leaf = "windowsOut", enabled = true, speed = 2.24, bezier = "mareOut", style = "popin 94%" })',
        'hl.animation({ leaf = "border", enabled = true, speed = 2.8, bezier = "mare" })',
        'hl.animation({ leaf = "fade", enabled = true, speed = 2.38, bezier = "mare" })',
        'hl.animation({ leaf = "layers", enabled = true, speed = 2.8, bezier = "mare", style = "fade" })',
        'hl.animation({ leaf = "workspaces", enabled = true, speed = 3.92, bezier = "mare", style = "slide" })',
        'hl.animation({ leaf = "specialWorkspace", enabled = true, speed = 3.36, bezier = "mare", style = "slidevert" })',
        ""
    ]);
});

// Os speeds que um animationCode emitiu, por folha.
function speeds(code) {
    const out = {};
    for (const m of code.matchAll(/leaf = "([^"]+)", enabled = true, speed = ([0-9.]+),/g))
        out[m[1]] = Number(m[2]);
    return out;
}

t("animationCode: rápido (0.7) dá números menores, com duas casas", () => {
    igual(speeds(S.animationCode({ "motion.speed": 0.7 })), { global: 1.68, windows: 1.68, windowsIn: 1.4,
        windowsOut: 1.12, border: 1.4, fade: 1.19, layers: 1.4, workspaces: 1.96, specialWorkspace: 1.68 });
});

t("animationCode: normal mexido nesta sessão repete os números de fábrica (desfaz a troca)", () => {
    const fabrica = {};
    for (const a of S.animations)
        fabrica[a[0]] = a[1];
    igual(speeds(S.animationCode({ "motion.speed": 1 }, { "motion.speed": true })), fabrica);
    igual(speeds(S.animationCode({}, { "motion.speed": true })), fabrica);
});

t("animationCode: toda opção de Velocidade maior que zero emite todas as folhas, na escala dela", () => {
    for (const o of linha("motion.speed").options) {
        const code = S.animationCode({ "motion.speed": o.v }, { "motion.speed": true });
        if (o.v === 0) {
            eq(code, "");
            continue;
        }
        const s = speeds(code);
        igual(Object.keys(s), S.animations.map((a) => a[0]));
        for (const a of S.animations)
            eq(s[a[0]], Math.round(a[1] * o.v * 100) / 100, o.label + " " + a[0]);
    }
});

// --- luaValue ---

t("luaValue: booleano e número", () => {
    eq(S.luaValue(true), "true");
    eq(S.luaValue(false), "false");
    eq(S.luaValue(6), "6");
    eq(S.luaValue(0), "0");
    eq(S.luaValue(-1), "-1");
    eq(S.luaValue(0.82), "0.82");
    eq(S.luaValue(2.0), "2");
});

t("luaValue: o número sai com no máximo três casas, sem lixo de ponto flutuante", () => {
    eq(S.luaValue(1 / 3), "0.333");
    eq(S.luaValue(0.1 + 0.2), "0.3");
    eq(S.luaValue(0.7 * 3), "2.1");
    eq(S.luaValue(-0.30000000000000004), "-0.3");
});

t("luaValue: texto entre aspas, com aspas e barras escapadas", () => {
    eq(S.luaValue("br"), '"br"');
    eq(S.luaValue("br,us"), '"br,us"');
    eq(S.luaValue(""), '""');
    eq(S.luaValue('a"b'), '"a\\"b"');
    eq(S.luaValue("a\\b"), '"a\\\\b"');
    eq(S.luaValue('\\"'), '"\\\\\\""');
});

t("luaValue: um texto não consegue fechar a string e virar código Lua", () => {
    const ataque = '") os.execute("rm -rf ~") --';
    const lua = S.luaValue(ataque);
    // Tirando os pares escapados, sobram só as duas aspas das pontas.
    eq(lua.replace(/\\[\\"]/g, "").replace(/[^"]/g, ""), '""');
    eq(S.luaValue('x\\'), '"x\\\\"');
});

t("luaValue: quebra de linha no texto não quebra a string do Lua", () => {
    eq(S.luaValue("a\nb"), '"a\\nb"');
    eq(S.luaValue("a\r\nb"), '"a\\r\\nb"');
    assert.ok(!/[\n\r]/.test(S.luaValue("br\n\") os.exit() --")));
});

// --- hyprTable, hyprCode, hyprLua ---

const TABELA_DE_FABRICA = '{ general = { gaps_in = 6, gaps_out = 12, border_size = 1 }, '
    + 'decoration = { rounding = 0, inactive_opacity = 1 }, '
    + 'input = { kb_layout = "br", repeat_rate = 25, repeat_delay = 600, sensitivity = 0, '
    + 'touchpad = { natural_scroll = true, tap_to_click = true } } }';

t("hyprTable: de fábrica, só as linhas com hypr; nada de kb_options nem de animations", () => {
    eq(S.hyprTable({}), TABELA_DE_FABRICA);
    eq(S.hyprTable({}, {}), TABELA_DE_FABRICA);
    eq(S.hyprTable(S.defaults()), TABELA_DE_FABRICA);
});

t("hyprTable: toda linha com um caminho do Hyprland entra, e as outras não", () => {
    const tabela = S.hyprTable({});
    for (const r of S.rows.filter((x) => x.hypr && x.hypr[0] !== "@"))
        assert.ok(tabela.includes(r.hypr.split(".").pop() + " = " + S.luaValue(r.def)), r.hypr);
    assert.ok(!/island|idle|night|backdrop|scale|@/.test(tabela));
});

t("hyprTable: o valor guardado troca o de fábrica", () => {
    const tabela = S.hyprTable({ "hypr.gapsIn": 10, "hypr.inactiveOpacity": 0.86, "hypr.naturalScroll": false, "hypr.sensitivity": -0.3 });
    assert.ok(tabela.includes("general = { gaps_in = 10, gaps_out = 12, border_size = 1 }"));
    assert.ok(tabela.includes("inactive_opacity = 0.86"));
    assert.ok(tabela.includes("sensitivity = -0.3"));
    assert.ok(tabela.includes("touchpad = { natural_scroll = false, tap_to_click = true }"));
});

t("hyprTable: o kb_options só entra quando o layout ou o Caps Lock saíram da fábrica", () => {
    assert.ok(S.hyprTable({ "input.caps": "esc" }).includes('kb_options = "caps:escape"'));
    assert.ok(S.hyprTable({ "hypr.kbLayout": "br,us" }).includes('kb_layout = "br,us"'));
    assert.ok(S.hyprTable({ "hypr.kbLayout": "br,us" }).includes('kb_options = "grp:alt_shift_toggle"'));
    // Só um layout diferente, sem opção nenhuma: limpa o kb_options.
    assert.ok(S.hyprTable({ "hypr.kbLayout": "us" }).includes('kb_options = ""'));
    assert.ok(!S.hyprTable({ "hypr.gapsIn": 10 }).includes("kb_options"));
});

t("hyprTable: voltar ao de fábrica nesta sessão limpa o kb_options (changed)", () => {
    assert.ok(S.hyprTable({}, { "input.caps": true }).includes('kb_options = ""'));
    assert.ok(S.hyprTable({ "input.caps": "normal" }, { "hypr.kbLayout": true }).includes('kb_options = ""'));
});

t("hyprTable: a Velocidade liga ou desliga as animações, e de fábrica não fala delas", () => {
    assert.ok(S.hyprTable({ "motion.speed": 0 }).endsWith(", animations = { enabled = false } }"));
    assert.ok(S.hyprTable({ "motion.speed": 0.7 }).endsWith(", animations = { enabled = true } }"));
    assert.ok(S.hyprTable({}, { "motion.speed": true }).endsWith(", animations = { enabled = true } }"));
    assert.ok(!S.hyprTable({}).includes("animations"));
});

t("hyprTable: não muda os values que recebeu", () => {
    const values = { "hypr.gapsIn": 10, "input.caps": "esc" };
    S.hyprTable(values, { "motion.speed": true });
    igual(values, { "hypr.gapsIn": 10, "input.caps": "esc" });
});

t("hyprCode: hl.config com a tabela, as animações e a escala dos monitores", () => {
    eq(S.hyprCode({}), "hl.config(" + TABELA_DE_FABRICA + ")\n"
        + 'hl.monitor({ output = "", mode = "preferred", position = "auto", scale = 1 })\n');
    const calmo = S.hyprCode({ "motion.speed": 1.4, "hypr.scale": 1.25 });
    igual(calmo.split("\n").map((l) => l.split("(")[0]),
        ["hl.config"].concat(S.animations.map(() => "hl.animation"), ["hl.monitor", ""]));
    assert.ok(calmo.endsWith('scale = 1.25 })\n'));
});

t("hyprCode: a escala de toda opção de 'Escala da tela' vai para o hl.monitor", () => {
    for (const o of linha("hypr.scale").options)
        assert.ok(S.hyprCode({ "hypr.scale": o.v }).endsWith("scale = " + o.v + " })\n"), o.label);
});

t("hyprCode: `changed` (as chaves mexidas nesta sessão) entra no eval", () => {
    const code = S.hyprCode({}, { "motion.speed": true, "input.caps": true });
    assert.ok(code.includes('kb_options = ""'));
    assert.ok(code.includes("animations = { enabled = true }"));
    eq((code.match(/^hl\.animation\(/gm) || []).length, S.animations.length);
});

t("hyprLua: o ajustes.lua é o aviso de gerado mais o hyprCode sem `changed`", () => {
    for (const values of [{}, { "motion.speed": 0.7, "input.caps": "ctrl", "hypr.gapsIn": 3 }]) {
        const lua = S.hyprLua(values);
        const linhas = lua.split("\n");
        assert.match(linhas[0], /^-- Gerado pelos Ajustes do Mk .*Não edite/);
        eq(linhas.slice(1).join("\n"), S.hyprCode(values));
    }
});

t("hyprLua: com tudo de fábrica não toca em kb_options nem em animações", () => {
    const lua = S.hyprLua({});
    assert.ok(!lua.includes("kb_options"));
    assert.ok(!lua.includes("animation"));
});

t("hyprLua: o que sai é Lua que compila, com qualquer opção e com texto torto no ajustes.json", () => {
    const luac = spawnSync("luac", ["-v"], { encoding: "utf8" });
    if (luac.error || luac.status !== 0) {
        console.log("    # sem luac: sintaxe do Lua não conferida");
        return;
    }
    const casos = [{}, S.defaults(), { "motion.speed": 0 }, { "motion.speed": 1.4, "hypr.kbLayout": "br,us", "input.caps": "esc" },
        { "hypr.kbLayout": 'br"\n) os.exit(1) --', "input.caps": "x\\" }, { "hypr.kbLayout": "a\r\nb\\", "hypr.sensitivity": -0.35 }];
    for (const r of S.rows.filter((x) => x.ctl === "choice"))
        for (const o of r.options)
            casos.push({ [r.key]: o.v });
    for (const values of casos)
        for (const texto of [S.hyprLua(values), S.hyprCode(values, { "motion.speed": true, "input.caps": true })]) {
            const r = spawnSync("luac", ["-p", "-"], { input: texto, encoding: "utf8" });
            eq(r.status, 0, JSON.stringify(values) + "\n" + r.stderr + "\n" + texto);
        }
});

// --- sleepMode, sleepCommand, sleepStep ---

t("sleepMode: vale a escolha de power.sleep; de fábrica, suspender", () => {
    eq(S.sleepMode({}), "suspender");
    eq(S.sleepMode({ "power.sleep": "suspender" }), "suspender");
    eq(S.sleepMode({ "power.sleep": "nada" }), "nada");
});

t("sleepMode: um 'hibernar' antigo vira 'nada', não suspender, possa a máquina o que puder", () => {
    eq(S.sleepMode({ "power.sleep": "hibernar" }), "nada");
    eq(S.sleepMode({ "power.sleep": "hibernar" }, { sono: { canSuspend: true, canHibernate: true } }), "nada");
    eq(S.sleepMode({ "power.sleep": "hibernar" }, { sono: { canSuspend: false } }), "nada");
});

t("sleepMode: suspender com o suspend proibido vira 'nada'", () => {
    eq(S.sleepMode({}, { sono: { canSuspend: false, risky: [] } }), "nada");
    eq(S.sleepMode({ "power.sleep": "suspender" }, { sono: { canSuspend: false } }), "nada");
    eq(S.sleepMode({}, { sono: { canSuspend: true, risky: ["AMD Navi 10 (1002:731f)"] } }), "suspender");
});

t("sleepMode: sem caps, ou sem a parte do sono, vale a escolha", () => {
    eq(S.sleepMode({}, undefined), "suspender");
    eq(S.sleepMode({}, null), "suspender");
    eq(S.sleepMode({}, {}), "suspender");
    eq(S.sleepMode({}, { sono: {} }), "suspender");
    eq(S.sleepMode({ "power.sleep": "nada" }, { sono: { canSuspend: true } }), "nada");
});

t("sleepMode: qualquer outro valor é 'nada'; só existem dois modos", () => {
    for (const ruim of ["banana", "", "hibernate", "suspend", "constructor", null, 0, true, "SUSPENDER"])
        eq(S.sleepMode({ "power.sleep": ruim }), "nada", String(ruim));
});

t("sleepCommand: suspender é 'systemctl suspend'; o resto não dorme ('')", () => {
    eq(S.sleepCommand({}), "systemctl suspend");
    eq(S.sleepCommand({ "power.sleep": "nada" }), "");
    eq(S.sleepCommand({ "power.sleep": "hibernar" }), "");
    eq(S.sleepCommand({ "power.sleep": "hibernar" }, { sono: { canSuspend: true } }), "");
    eq(S.sleepCommand({}, { sono: { canSuspend: false } }), "");
    eq(S.sleepCommand({ "power.sleep": "constructor" }), "");
});

t("sleepCommand: nenhum jeito de dormir chama a hibernação", () => {
    igual(S.sleepCommands, { suspender: "systemctl suspend" });
});

t("sleepStep: o passo de dormir que um hypridle.conf tem", () => {
    eq(S.sleepStep("listener {\n    timeout = 1800\n    on-timeout = systemctl suspend\n}\n"), "systemctl suspend");
    eq(S.sleepStep("listener {\n    timeout = 300\n    on-timeout = loginctl lock-session\n}\n"), "");
    eq(S.sleepStep(""), "");
});

t("sleepStep: acha o hibernate de um arquivo antigo, para o syncIdle reescrever sem ele", () => {
    const antigo = "listener {\n    timeout = 1800\n    on-timeout = systemctl hibernate\n}\n";
    eq(S.sleepStep(antigo), "systemctl hibernate");
    // Diferente do passo de qualquer arquivo que os Ajustes geram hoje.
    for (const values of [{}, { "power.sleep": "nada" }, { "power.sleep": "hibernar" }])
        assert.notEqual(S.sleepStep(S.hypridleConf(values, DIR)), S.sleepStep(antigo));
});

t("sleepStep: lê de volta o passo do hypridle.conf que os Ajustes geram", () => {
    eq(S.sleepStep(S.hypridleConf({}, DIR)), "systemctl suspend");
    eq(S.sleepStep(S.hypridleConf({ "power.sleep": "nada" }, DIR)), "");
    eq(S.sleepStep(S.hypridleConf({ "idle.suspend": 0 }, DIR)), "");
    eq(S.sleepStep(S.hypridleConf({}, DIR, { sono: { canSuspend: false } })), "");
});

// --- hypridleConf ---

// Os listeners de um hypridle.conf: [{ timeout, on, resume }].
function listeners(conf) {
    return [...conf.matchAll(/\nlistener \{\n    timeout = (\S+)\n    on-timeout = ([^\n]+)\n(?:    on-resume = ([^\n]+)\n)?\}\n/g)]
        .map((m) => ({ timeout: m[1], on: m[2], resume: m[3] || "" }));
}

t("hypridleConf: de fábrica, escurece em 4 min, bloqueia em 5, apaga em 6 e dorme em 30", () => {
    const conf = S.hypridleConf({}, DIR);
    igual(listeners(conf), [
        { timeout: "240", on: "brightnessctl -s set 15%", resume: "brightnessctl -r" },
        { timeout: "300", on: "loginctl lock-session", resume: "" },
        { timeout: "360", on: "hyprctl dispatch 'hl.dsp.dpms({ action = \"off\" })'", resume: "hyprctl dispatch 'hl.dsp.dpms({ action = \"on\" })'" },
        { timeout: "1800", on: "systemctl suspend", resume: "" }
    ]);
    eq((conf.match(/^listener \{$/gm) || []).length, 4);
});

t("hypridleConf: o cabeçalho avisa que é gerado, e o general bloqueia antes de dormir", () => {
    const conf = S.hypridleConf({}, DIR);
    assert.match(conf, /^# Gerado pelos Ajustes do Mk .*Não edite\.\n\ngeneral \{\n/);
    assert.ok(conf.includes("    before_sleep_cmd = loginctl lock-session\n"));
    assert.ok(conf.includes("    after_sleep_cmd = hyprctl dispatch 'hl.dsp.dpms({ action = \"on\" })'\n"));
});

t("hypridleConf: o lock_cmd usa o shell e o hyprlock.conf da pasta dada", () => {
    const conf = S.hypridleConf({}, "/opt/mk/hypr");
    assert.ok(conf.includes("    lock_cmd = qs -p /opt/mk/shell ipc call mare lock || pidof hyprlock || hyprlock -c /opt/mk/hypr/hyprlock.conf\n"));
    // Só o /hypr do fim vira /shell.
    assert.ok(S.hypridleConf({}, "/home/hypr/x/hypr").includes("qs -p /home/hypr/x/shell ipc"));
});

t("hypridleConf: minutos viram segundos", () => {
    const l = listeners(S.hypridleConf({ "idle.dim": 1, "idle.lock": 10, "idle.off": 60, "idle.suspend": 120 }, DIR));
    igual(l.map((x) => x.timeout), ["60", "600", "3600", "7200"]);
    eq(listeners(S.hypridleConf({ "idle.lock": 0.5 }, DIR))[1].timeout, "30");
});

t("hypridleConf: zero quer dizer nunca: aquele passo não acontece", () => {
    igual(listeners(S.hypridleConf({ "idle.dim": 0 }, DIR)).map((x) => x.timeout), ["300", "360", "1800"]);
    igual(listeners(S.hypridleConf({ "idle.lock": 0 }, DIR)).map((x) => x.timeout), ["240", "360", "1800"]);
    igual(listeners(S.hypridleConf({ "idle.off": 0 }, DIR)).map((x) => x.timeout), ["240", "300", "1800"]);
    igual(listeners(S.hypridleConf({ "idle.suspend": 0 }, DIR)).map((x) => x.timeout), ["240", "300", "360"]);
    const nada = S.hypridleConf({ "idle.dim": 0, "idle.lock": 0, "idle.off": 0, "idle.suspend": 0 }, DIR);
    igual(listeners(nada), []);
    assert.ok(!nada.includes("listener"));
});

t("hypridleConf: sem brilho para mexer, o passo de escurecer não entra", () => {
    const conf = S.hypridleConf({}, DIR, { brilho: false });
    assert.ok(!conf.includes("brightnessctl"));
    igual(listeners(conf).map((x) => x.timeout), ["300", "360", "1800"]);
    assert.ok(S.hypridleConf({}, DIR, { brilho: true }).includes("brightnessctl"));
    assert.ok(S.hypridleConf({}, DIR, {}).includes("brightnessctl"));
});

t("hypridleConf: o passo de dormir segue power.sleep e o que a máquina pode", () => {
    assert.ok(S.hypridleConf({}, DIR).includes("on-timeout = systemctl suspend\n"));
    for (const [values, caps] of [[{ "power.sleep": "nada" }, undefined], [{ "power.sleep": "hibernar" }, undefined],
        [{}, { sono: { canSuspend: false } }], [{ "power.sleep": "hibernar" }, { sono: { canSuspend: true } }]]) {
        const conf = S.hypridleConf(values, DIR, caps);
        assert.ok(!conf.includes("systemctl"), JSON.stringify(values));
        igual(listeners(conf).map((x) => x.timeout), ["240", "300", "360"]);
    }
});

t("hypridleConf: nunca hiberna, escolha o que escolher", () => {
    for (const modo of ["suspender", "nada", "hibernar", "hibernate", "suspend-then-hibernate"])
        assert.ok(!/hibern/i.test(S.hypridleConf({ "power.sleep": modo }, DIR, { sono: { canSuspend: true } })), modo);
});

t("hypridleConf: os passos são os das linhas de Energia, e respeitam a faixa de cada uma", () => {
    for (const key of ["idle.dim", "idle.lock", "idle.off", "idle.suspend"]) {
        const r = linha(key);
        eq(r.page, "energia");
        eq(r.unit, "min");
        // O zero ("nunca", como diz a página) tem de caber no slider.
        eq(r.min, 0, key);
        const conf = S.hypridleConf({ "idle.dim": 0, "idle.lock": 0, "idle.off": 0, "idle.suspend": 0, [key]: r.max }, DIR);
        igual(listeners(conf).map((x) => x.timeout), [String(r.max * 60)], key);
    }
});

// --- orphan: uma escolha salva que deixou de existir ---

t("orphan: 'hibernar' salvo em power.sleep é uma escolha que não existe mais", () => {
    eq(S.orphan({ "power.sleep": "hibernar" }, "power.sleep"), true);
    eq(S.orphan({ "power.sleep": "constructor" }, "power.sleep"), true);
});

t("orphan: uma escolha que existe, ou nenhuma (vale a de fábrica), não é órfã", () => {
    eq(S.orphan({ "power.sleep": "suspender" }, "power.sleep"), false);
    eq(S.orphan({ "power.sleep": "nada" }, "power.sleep"), false);
    eq(S.orphan({}, "power.sleep"), false);
});

t("orphan: só vale para linhas de escolha; chave desconhecida ou de outro tipo dá false", () => {
    eq(S.orphan({ "idle.lock": 999 }, "idle.lock"), false);
    eq(S.orphan({}, "nao.existe"), false);
});
