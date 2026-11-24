// shell/mare/Keybinds.js: os atalhos do Mk como dados, e as funções que a
// página Atalhos usa para ler, conferir e gravar uma troca. A comparação com
// config/hypr/mare/binds.lua é do bin/check-keybinds, não daqui.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { load } from "./load.mjs";

const K = load("Keybinds.js");
const acao = (id) => {
    const a = K.byId(id);
    assert.ok(a, "não existe a ação " + id);
    return a;
};
const ids = (lista) => lista.map((a) => a.id);

// Os modificadores e teclas do Qt usados nos casos de fromQt.
const QT = { SHIFT: 0x02000000, CTRL: 0x04000000, ALT: 0x08000000, SUPER: 0x10000000, KEYPAD: 0x20000000 };

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["normalizeKey", "byId", "parse", "format", "same", "canonical", "clean", "effective",
        "expand", "conflicts", "conflict", "validate", "keyLabel", "pretty", "fromQt", "modsFromQt", "groups",
        "entries", "luaTable", "fromLua", "diverged", "lua", "evalCode", "suspendCode", "resumeCode"];
    const funcoes = Object.keys(K).filter((k) => typeof K[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)), [], "função sem teste: acrescente o caso e ponha na lista");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

// --- os dados: ações ---

t("ações: ids únicos, em minúsculas e com ponto (grupo.nome)", () => {
    const visto = {};
    const ruins = [];
    for (const a of K.actions) {
        if (!/^[a-z]+(\.[a-z]+)+$/.test(a.id))
            ruins.push("formato: " + a.id);
        if (visto[a.id])
            ruins.push("repetido: " + a.id);
        visto[a.id] = true;
    }
    vazio(ruins, "ids ruins");
});

t("ações: toda ação tem grupo e descrição, e a descrição não se repete", () => {
    // O diverged() casa o bind do Hyprland com a ação pela descrição.
    const visto = {};
    const ruins = [];
    for (const a of K.actions) {
        if (typeof a.group !== "string" || a.group.trim() === "" || a.group !== a.group.toUpperCase())
            ruins.push(a.id + ": grupo " + JSON.stringify(a.group));
        if (typeof a.desc !== "string" || a.desc.trim() === "" || a.desc !== a.desc.trim())
            ruins.push(a.id + ": descrição " + JSON.stringify(a.desc));
        if (visto[a.desc])
            ruins.push(a.id + ": descrição repetida de " + visto[a.desc]);
        visto[a.desc] = a.id;
    }
    vazio(ruins, "ações ruins");
});

t("ações: só os campos conhecidos (id, group, desc, combo, required, fixed, family)", () => {
    const campos = ["id", "group", "desc", "combo", "required", "fixed", "family"];
    const ruins = [];
    for (const a of K.actions) {
        for (const k of Object.keys(a))
            if (!campos.includes(k))
                ruins.push(a.id + ": campo " + k);
        if (a.required !== undefined && a.required !== true)
            ruins.push(a.id + ": required");
        if (a.fixed !== undefined && a.fixed !== true)
            ruins.push(a.id + ": fixed");
    }
    vazio(ruins, "campos ruins");
});

t("combos de fábrica: modificadores conhecidos, na ordem, separados por ' + '", () => {
    const ruins = [];
    for (const a of K.actions) {
        const p = K.parse(a.combo);
        if (typeof a.combo !== "string" || a.combo === "" || K.format(p.mods, p.key) !== a.combo)
            ruins.push(a.id + ": " + JSON.stringify(a.combo) + " devia ser " + JSON.stringify(K.format(p.mods, p.key)));
        if (!/^[A-Za-z0-9_:]+( \+ [A-Za-z0-9_:]+)*$/.test(a.combo))
            ruins.push(a.id + ": escrita " + JSON.stringify(a.combo));
    }
    vazio(ruins, "combos ruins");
});

t("combos de fábrica: família só com modificadores; o resto com uma tecla da lista", () => {
    const ruins = [];
    for (const a of K.actions) {
        const p = K.parse(a.combo);
        if (a.family) {
            if (p.key !== "" || p.mods.length === 0)
                ruins.push(a.id + ": família com combo " + a.combo);
            if (!Array.isArray(K.families[a.family]) || typeof K.familyLabels[a.family] !== "string")
                ruins.push(a.id + ": família desconhecida " + a.family);
        } else if (K.keys.indexOf(p.key) < 0 && !(a.fixed && /^(mouse:\d+|mouse_up|mouse_down)$/.test(p.key))) {
            ruins.push(a.id + ": tecla " + JSON.stringify(p.key));
        }
    }
    vazio(ruins, "combos ruins");
});

t("combos de fábrica: o de toda ação que se troca passa no próprio validate", () => {
    const ruins = [];
    for (const a of K.actions)
        if (!a.fixed && K.canonical(a, a.combo) !== a.combo)
            ruins.push(a.id + ": " + a.combo + " -> " + K.canonical(a, a.combo));
    vazio(ruins, "combos que o editor recusaria");
});

t("combos de fábrica: nenhuma ação bate em outra", () => {
    const ruins = [];
    for (const a of K.actions)
        for (const outro of K.conflicts(a.id, a.combo, {}))
            ruins.push(a.id + " (" + a.combo + ") x " + outro.id + " (" + outro.combo + ")");
    vazio(ruins, "conflitos de fábrica");
});

t("emergência: as ações required existem, não são fixas nem de família", () => {
    const req = K.actions.filter((a) => a.required);
    assert.ok(req.length >= 2);
    igual(ids(req).filter((id) => !/^emergency\./.test(id)), []);
    igual(ids(req.filter((a) => a.fixed || a.family)), []);
});

// --- os dados: teclas ---

t("teclas: a lista não repete nome (o Hyprland não liga para a caixa)", () => {
    const visto = {};
    const ruins = [];
    for (const k of K.keys) {
        if (visto[k.toLowerCase()])
            ruins.push(k);
        visto[k.toLowerCase()] = true;
    }
    vazio(ruins, "teclas repetidas");
});

t("teclas: famílias, apelidos, códigos, pontuação com Shift e emergência apontam para teclas da lista", () => {
    const ruins = [];
    for (const f of ["arrows", "digits"])
        for (const k of K.families[f])
            if (K.keys.indexOf(k) < 0)
                ruins.push("família " + f + ": " + k);
    for (const [de, para] of Object.entries(K.aliases)) {
        if (K.keys.indexOf(para) < 0)
            ruins.push("apelido " + de + " -> " + para);
        // normalizeKey procura em minúsculas e sem espaço nas pontas.
        if (de !== de.trim().toLowerCase())
            ruins.push("apelido que nunca casa: " + JSON.stringify(de));
    }
    for (const [n, nome] of Object.entries(K.codes)) {
        if (K.keys.indexOf(nome) < 0)
            ruins.push("código " + n + " -> " + nome);
        if (!/^\d{1,3}$/.test(n) || Number(n) <= 8)
            ruins.push("código fora da faixa: " + n);
    }
    for (const [de, para] of Object.entries(K.shifted)) {
        if (K.keys.indexOf(para) < 0)
            ruins.push("shift " + de + " -> " + para);
        if (!/^.@\d+$/.test(de))
            ruins.push("shift mal escrito: " + de);
    }
    for (const e of K.everywhere)
        if (e !== e.toLowerCase() || K.normalizeKey(e) === "")
            ruins.push("emergência: " + e);
    vazio(ruins, "referências ruins");
});

t("teclas: um código físico não tem dois nomes, nem um nome dois códigos", () => {
    const nomes = Object.values(K.codes);
    igual(nomes.filter((n, i) => nomes.indexOf(n) !== i), []);
});

t("teclas: os rótulos (keyLabels) são de teclas que existem, em minúsculas", () => {
    const ruins = [];
    const lower = K.keys.map((k) => k.toLowerCase());
    for (const k of Object.keys(K.keyLabels))
        if (k !== k.toLowerCase() || (lower.indexOf(k) < 0 && !/^mouse/.test(k)))
            ruins.push(k);
    vazio(ruins, "rótulos órfãos");
});

t("modificadores: mods, modLabels e masks falam dos mesmos quatro", () => {
    igual(K.mods, ["SUPER", "CTRL", "ALT", "SHIFT"]);
    igual(Object.keys(K.modLabels).sort(), K.mods.slice().sort());
    igual(Object.keys(K.masks).sort(), K.mods.slice().sort());
    // Os bits do modmask do Hyprland.
    igual(K.masks, { SHIFT: 1, CTRL: 4, ALT: 8, SUPER: 64 });
});

// --- normalizeKey ---

t("normalizeKey: toda tecla da lista volta igual, em qualquer caixa", () => {
    const ruins = [];
    for (const k of K.keys)
        for (const escrita of [k, k.toLowerCase(), k.toUpperCase(), "  " + k + " "])
            if (K.normalizeKey(escrita) !== k)
                ruins.push(JSON.stringify(escrita) + " -> " + JSON.stringify(K.normalizeKey(escrita)));
    vazio(ruins, "teclas que não voltam");
});

t("normalizeKey: o que alguém digitaria vira o nome do xkb", () => {
    eq(K.normalizeKey("enter"), "RETURN");
    eq(K.normalizeKey("Enter"), "RETURN");
    eq(K.normalizeKey("ESC"), "ESCAPE");
    eq(K.normalizeKey("espaço"), "SPACE");
    eq(K.normalizeKey("Barra de Espaço"), "SPACE");
    eq(K.normalizeKey("pgdn"), "NEXT");
    eq(K.normalizeKey("PageUp"), "PRIOR");
    eq(K.normalizeKey("←"), "Left");
    eq(K.normalizeKey("cima"), "Up");
    eq(K.normalizeKey(","), "COMMA");
    eq(K.normalizeKey("."), "PERIOD");
    eq(K.normalizeKey("ç"), "CCEDILLA");
    eq(K.normalizeKey("Ç"), "CCEDILLA");
    eq(K.normalizeKey("prtsc"), "PRINT");
});

t("normalizeKey: code:<n> vira o nome quando a tecla tem lugar fixo, senão fica code:<n>", () => {
    eq(K.normalizeKey("code:36"), "RETURN");
    eq(K.normalizeKey("CODE:9"), "ESCAPE");
    eq(K.normalizeKey("code:65"), "SPACE");
    eq(K.normalizeKey("code:38"), "code:38");
    eq(K.normalizeKey("code:038"), "code:38");
    for (const [n, nome] of Object.entries(K.codes))
        eq(K.normalizeKey("code:" + n), nome);
});

t("normalizeKey: o que não existe devolve ''", () => {
    for (const ruim of ["", "   ", null, undefined, "banana", "code:", "code:abc", "code:1234", "code:-1",
        "F13x", "SUPER", "a b", "mouse:272", 42.5, {}, []])
        eq(K.normalizeKey(ruim), "", JSON.stringify(ruim));
});

t("normalizeKey: nome de propriedade do JavaScript não é tecla", () => {
    for (const ruim of ["constructor", "__proto__", "toString", "tostring", "valueOf", "hasOwnProperty",
        "hasownproperty", "__defineGetter__", "isPrototypeOf", "length"])
        eq(K.normalizeKey(ruim), "", ruim);
});

// --- byId, parse, format, same ---

t("byId: acha a ação; null se não existe", () => {
    eq(K.byId("win.close").combo, "SUPER + Q");
    eq(K.byId("win.close"), K.actions.find((a) => a.id === "win.close"));
    eq(K.byId("nao.existe"), null);
    eq(K.byId(""), null);
    eq(K.byId(undefined), null);
});

t("parse: o exemplo do cabeçalho, SUPER + SHIFT + K", () => {
    igual(K.parse("SUPER + SHIFT + K"), { mods: ["SUPER", "SHIFT"], key: "K" });
});

t("parse: aceita sem espaço e em minúsculas, e põe os modificadores em ordem", () => {
    igual(K.parse("shift+super+k"), { mods: ["SUPER", "SHIFT"], key: "k" });
    igual(K.parse("  ALT +  ctrl + Tab "), { mods: ["CTRL", "ALT"], key: "Tab" });
    igual(K.parse("SHIFT + SHIFT + A"), { mods: ["SHIFT"], key: "A" });
    igual(K.parse("PRINT"), { mods: [], key: "PRINT" });
    igual(K.parse("SUPER + mouse:272"), { mods: ["SUPER"], key: "mouse:272" });
});

t("parse: numa família o combo é só de modificadores e a tecla fica vazia", () => {
    igual(K.parse("SUPER + SHIFT"), { mods: ["SUPER", "SHIFT"], key: "" });
    igual(K.parse("SUPER"), { mods: ["SUPER"], key: "" });
});

t("parse: vazio, null e undefined dão um combo vazio", () => {
    for (const nada of ["", null, undefined, " + "])
        igual(K.parse(nada), { mods: [], key: "" });
});

t("format: modificadores na ordem SUPER, CTRL, ALT, SHIFT e a tecla no fim", () => {
    eq(K.format(["SHIFT", "SUPER"], "K"), "SUPER + SHIFT + K");
    eq(K.format(["SHIFT", "ALT", "CTRL", "SUPER"], "F1"), "SUPER + CTRL + ALT + SHIFT + F1");
    eq(K.format([], "PRINT"), "PRINT");
    eq(K.format(["SUPER"], ""), "SUPER");
    eq(K.format([], ""), "");
    eq(K.format(["HYPER", "SUPER"], "K"), "SUPER + K");
});

t("same: o mesmo combo escrito de dois jeitos", () => {
    assert.ok(K.same("super+k", "SUPER + K"));
    assert.ok(K.same("SHIFT + SUPER + K", "SUPER + SHIFT + k"));
    assert.ok(K.same("", ""));
    assert.ok(!K.same("SUPER + K", "SUPER + J"));
    assert.ok(!K.same("SUPER + K", "SUPER + SHIFT + K"));
    assert.ok(!K.same("SUPER + K", ""));
});

// --- validate ---

t("validate: devolve o combo pronto e nenhum erro", () => {
    igual(K.validate(acao("win.close"), ["SHIFT", "SUPER"], "w"), { combo: "SUPER + SHIFT + W", error: "" });
    igual(K.validate(acao("win.close"), ["CTRL"], "enter"), { combo: "CTRL + RETURN", error: "" });
    igual(K.validate(acao("win.close"), ["ALT"], "code:38"), { combo: "ALT + code:38", error: "" });
});

t("validate: sem tecla, 'falta a tecla'", () => {
    for (const nada of ["", "  ", null, undefined])
        igual(K.validate(acao("win.close"), ["SUPER"], nada), { combo: "", error: "falta a tecla" });
});

t("validate: tecla desconhecida diz qual", () => {
    igual(K.validate(acao("win.close"), ["SUPER"], " banana "), { combo: "", error: "tecla desconhecida: banana" });
});

t("validate: sem Super, Ctrl ou Alt, só tecla que não escreve nada", () => {
    const a = acao("win.close");
    for (const tecla of ["A", "1", "RETURN", "SPACE", "COMMA", "KP_1", "TAB", "DELETE"])
        for (const m of [[], ["SHIFT"]]) {
            const r = K.validate(a, m, tecla);
            eq(r.combo, "", tecla);
            assert.match(r.error, /sozinha pararia de digitar/, tecla);
        }
    igual(K.validate(a, [], "PRINT"), { combo: "PRINT", error: "" });
    igual(K.validate(a, ["SHIFT"], "F5"), { combo: "SHIFT + F5", error: "" });
    igual(K.validate(a, [], "XF86Calculator"), { combo: "XF86Calculator", error: "" });
    igual(K.validate(a, [], "pause"), { combo: "PAUSE", error: "" });
});

t("validate: família pede Super, Ctrl ou Alt e não leva tecla", () => {
    const a = acao("win.focus");
    igual(K.validate(a, ["ALT", "SHIFT"], ""), { combo: "ALT + SHIFT", error: "" });
    igual(K.validate(a, ["CTRL"], "ignorada"), { combo: "CTRL", error: "" });
    igual(K.validate(a, ["SHIFT"], ""), { combo: "", error: "escolha Super, Ctrl ou Alt" });
    igual(K.validate(a, [], ""), { combo: "", error: "escolha Super, Ctrl ou Alt" });
});

t("validate: emergência só vai para onde qualquer teclado alcança", () => {
    const a = acao("emergency.terminal");
    for (const tecla of ["A", "z", "0", "9", "F1", "F12", "RETURN", "ESCAPE", "SPACE", "TAB", "BACKSPACE", "DELETE"])
        eq(K.validate(a, ["SUPER", "SHIFT"], tecla).error, "", tecla);
    for (const tecla of ["PERIOD", "COMMA", "PRINT", "Left", "KP_1", "CCEDILLA", "code:38", "XF86Calculator",
        "HOME", "INSERT"]) {
        const r = K.validate(a, ["SUPER"], tecla);
        eq(r.combo, "", tecla);
        assert.match(r.error, /^emergência:/, tecla);
    }
    // E nunca sem Super, Ctrl ou Alt, nem numa tecla de função.
    assert.match(K.validate(a, [], "F5").error, /^emergência:/);
    assert.match(K.validate(a, ["SHIFT"], "F5").error, /^emergência:/);
});

t("validate: toda tecla de `everywhere` serve para a emergência", () => {
    for (const e of K.everywhere)
        eq(K.validate(acao("emergency.exit"), ["CTRL", "ALT"], e).error, "", e);
});

// --- canonical ---

t("canonical: escreve do nosso jeito (modificadores em ordem, tecla pelo nome)", () => {
    eq(K.canonical(acao("win.close"), "shift+super+enter"), "SUPER + SHIFT + RETURN");
    eq(K.canonical(acao("win.close"), "super + code:36"), "SUPER + RETURN");
    eq(K.canonical(acao("win.close"), "ALT + left"), "ALT + Left");
    eq(K.canonical(acao("win.focus"), "alt"), "ALT");
    eq(K.canonical(acao("shot.area"), "print"), "PRINT");
});

t("canonical: '' (sem atalho) fica ''", () => eq(K.canonical(acao("win.close"), ""), ""));

t("canonical: null quando o combo não serve para a ação", () => {
    eq(K.canonical(acao("win.close"), "SUPER + banana"), null);
    eq(K.canonical(acao("win.close"), "SUPER"), null);
    eq(K.canonical(acao("win.close"), "K"), null);
    eq(K.canonical(acao("win.focus"), "SUPER + K"), null);
    eq(K.canonical(acao("win.focus"), "SHIFT"), null);
    eq(K.canonical(acao("emergency.exit"), "SUPER + PERIOD"), null);
});

// --- clean ---

t("clean: o que não é um objeto de trocas dá {}", () => {
    for (const nada of [null, undefined, "", "SUPER + K", 42, true, []])
        igual(K.clean(nada), {});
});

t("clean: guarda a troca válida, escrita do nosso jeito", () => {
    igual(K.clean({ "win.close": "shift + super + w", "app.browser": " ALT + F2 " }),
        { "app.browser": "ALT + F2", "win.close": "SUPER + SHIFT + W" });
});

t("clean: sai na ordem da lista de ações, não na do ajustes.json", () => {
    igual(Object.keys(K.clean({ "win.close": "SUPER + F9", "mare.project": "SUPER + F8" })),
        ["mare.project", "win.close"]);
});

t("clean: cai fora id que não existe, ação fixa, valor que não é texto e combo inválido", () => {
    igual(K.clean({
        "nao.existe": "SUPER + K",
        "mare.switcher.next": "SUPER + J",
        "media.play": "SUPER + J",
        "win.close": 42,
        "win.float": null,
        "win.fullscreen": ["SUPER + J"],
        "app.browser": "SUPER + banana",
        "app.files": "K",
        "win.focus": "SUPER + K"
    }), {});
});

t("clean: combo igual ao de fábrica não é troca, em qualquer escrita", () => {
    igual(K.clean({ "win.close": "SUPER + Q", "win.float": "super+v", "win.move": "shift + super",
        "app.terminal": "SUPER + enter" }), {});
});

t("clean: '' tira o atalho, menos de emergência e de família", () => {
    igual(K.clean({ "win.close": "", "emergency.exit": "", "win.focus": "", "ws.move": "   " }), { "win.close": "" });
});

t("clean: nome de propriedade do JavaScript não passa por tecla e não vai para o Hyprland", () => {
    igual(K.clean({ "win.close": "SUPER + constructor", "win.float": "SUPER + __proto__",
        "app.files": "SUPER + toString", "app.browser": "SUPER + hasOwnProperty" }), {});
    for (const v of Object.values(K.effective({ "win.close": "SUPER + constructor" })))
        eq(typeof v, "string");
});

t("clean: um id que é nome de propriedade do JavaScript não entra", () => {
    igual(K.clean(JSON.parse('{"__proto__": "SUPER + K", "constructor": "SUPER + J", "toString": "SUPER + L"}')), {});
});

t("clean: a emergência não fica com o combo de fábrica de quem não foi mexido", () => {
    igual(K.clean({ "emergency.terminal": "SUPER + Q" }), {});
    igual(K.clean({ "emergency.terminal": "super + shift + escape" }), {});
    // Uma família ocupa várias teclas: Super + 1 é do ws.focus.
    igual(K.clean({ "emergency.terminal": "SUPER + 1" }), {});
});

t("clean: a emergência pode ficar com o combo de quem saiu dele", () => {
    igual(K.clean({ "emergency.terminal": "SUPER + Q", "win.close": "SUPER + F9" }),
        { "emergency.terminal": "SUPER + Q", "win.close": "SUPER + F9" });
    igual(K.clean({ "emergency.terminal": "SUPER + Q", "win.close": "" }),
        { "emergency.terminal": "SUPER + Q", "win.close": "" });
    // As duas de emergência trocando de lugar.
    igual(K.clean({ "emergency.terminal": "SUPER + SHIFT + ESCAPE", "emergency.exit": "SUPER + SHIFT + RETURN" }),
        { "emergency.terminal": "SUPER + SHIFT + ESCAPE", "emergency.exit": "SUPER + SHIFT + RETURN" });
});

t("clean: uma emergência que cai devolve o combo de fábrica dela, e derruba quem ia para ele", () => {
    // exit vai para Super + Q (do win.close, que não saiu): cai e volta para
    // Super + Shift + Esc. Aí o terminal, que ia para Super + Shift + Esc, cai também.
    igual(K.clean({ "emergency.terminal": "SUPER + SHIFT + ESCAPE", "emergency.exit": "SUPER + Q" }), {});
});

t("clean: uma emergência num combo livre fica", () => {
    igual(K.clean({ "emergency.terminal": "CTRL + ALT + T", "emergency.exit": "SUPER + BACKSPACE" }),
        { "emergency.terminal": "CTRL + ALT + T", "emergency.exit": "SUPER + BACKSPACE" });
});

t("clean: não muda o objeto que recebeu", () => {
    const trocas = { "win.close": "super+w", "nao.existe": "x" };
    K.clean(trocas);
    igual(trocas, { "win.close": "super+w", "nao.existe": "x" });
});

t("clean: limpar duas vezes dá o mesmo", () => {
    const uma = K.clean({ "win.close": "super+w", "win.float": "", "emergency.exit": "ctrl+alt+delete" });
    igual(K.clean(uma), uma);
});

// --- effective, expand ---

t("effective: sem trocas, o combo de fábrica de toda ação", () => {
    const e = K.effective({});
    igual(Object.keys(e), ids(K.actions));
    for (const a of K.actions)
        eq(e[a.id], a.combo);
    igual(K.effective(null), e);
    igual(K.effective(undefined), e);
});

t("effective: a troca vale, a inválida não, e '' é sem atalho", () => {
    const e = K.effective({ "win.close": "super+w", "win.float": "", "app.browser": "SUPER + banana" });
    eq(e["win.close"], "SUPER + W");
    eq(e["win.float"], "");
    eq(e["app.browser"], "SUPER + B");
    eq(e["win.fullscreen"], "SUPER + F");
});

t("expand: as combinações que a ação ocupa, em minúsculas", () => {
    igual(K.expand(acao("win.close"), "SUPER + Q"), ["super + q"]);
    igual(K.expand(acao("win.close"), "shift+super+Q"), ["super + shift + q"]);
    igual(K.expand(acao("win.close"), ""), []);
});

t("expand: uma família ocupa uma combinação por tecla", () => {
    igual(K.expand(acao("win.focus"), "SUPER"), ["super + left", "super + right", "super + up", "super + down"]);
    igual(K.expand(acao("ws.move"), "SUPER + SHIFT"), [1, 2, 3, 4, 5, 6, 7, 8, 9].map((n) => "super + shift + " + n));
    igual(K.expand(acao("mare.launcher.tap"), "SUPER"), ["super + super_l", "super + super_r"]);
    igual(K.expand(acao("win.focus"), ""), []);
});

// --- conflicts, conflict ---

t("conflicts: quem já usa o combo", () => {
    igual(ids(K.conflicts("win.close", "SUPER + F", {})), ["win.fullscreen"]);
    igual(ids(K.conflicts("win.close", "super+f", {})), ["win.fullscreen"]);
    // Com as ações fixas também.
    igual(ids(K.conflicts("win.close", "ALT + TAB", {})), ["mare.switcher.next"]);
});

t("conflicts: [] para combo livre, o próprio combo, sem atalho e id desconhecido", () => {
    igual(K.conflicts("win.close", "SUPER + F9", {}), []);
    igual(K.conflicts("win.close", "SUPER + Q", {}), []);
    igual(K.conflicts("win.close", "", {}), []);
    igual(K.conflicts("nao.existe", "SUPER + F", {}), []);
    igual(K.conflicts("win.close", "SUPER + F9"), []);
});

t("conflicts: olha o combo que vale agora, não o de fábrica", () => {
    igual(K.conflicts("win.close", "SUPER + F", { "win.fullscreen": "SUPER + F9" }), []);
    igual(ids(K.conflicts("win.close", "SUPER + F9", { "win.fullscreen": "SUPER + F9" })), ["win.fullscreen"]);
    // Quem ficou sem atalho não bate em ninguém.
    igual(K.conflicts("win.close", "SUPER + F", { "win.fullscreen": "" }), []);
});

t("conflicts: uma tecla solta bate numa família", () => {
    igual(ids(K.conflicts("mare.project", "SUPER + 3", {})), ["ws.focus"]);
    igual(ids(K.conflicts("mare.project", "SUPER + SHIFT + Left", {})), ["win.move"]);
    igual(ids(K.conflicts("mare.project", "super + left", {})), ["win.focus"]);
});

t("conflicts: uma família ocupa várias teclas e pode bater em mais de um", () => {
    igual(ids(K.conflicts("ws.focus", "SUPER + SHIFT", {})), ["ws.move"]);
    const trocas = { "mare.project": "ALT + 1", "mare.calendar": "ALT + 9" };
    igual(ids(K.conflicts("ws.focus", "ALT", trocas)), ["mare.project", "mare.calendar"]);
    // Setas e números não se cruzam.
    igual(K.conflicts("win.focus", "ALT", trocas), []);
});

t("conflict: o primeiro deles; null se ninguém", () => {
    eq(K.conflict("win.close", "SUPER + F", {}), acao("win.fullscreen"));
    eq(K.conflict("ws.focus", "ALT", { "mare.project": "ALT + 1", "mare.calendar": "ALT + 9" }), acao("mare.project"));
    eq(K.conflict("win.close", "SUPER + F9", {}), null);
    eq(K.conflict("win.close", "", {}), null);
});

// --- keyLabel, pretty ---

t("keyLabel: como cada tecla aparece na página", () => {
    eq(K.keyLabel("RETURN"), "Enter");
    eq(K.keyLabel("Left"), "←");
    eq(K.keyLabel("PERIOD"), ".");
    eq(K.keyLabel("CCEDILLA"), "Ç");
    eq(K.keyLabel("q"), "Q");
    eq(K.keyLabel("7"), "7");
    eq(K.keyLabel("F5"), "F5");
    eq(K.keyLabel("KP_Add"), "KP_Add");
    eq(K.keyLabel("XF86AudioRaiseVolume"), "Audio Raise Volume");
    eq(K.keyLabel("XF86WLAN"), "WLAN");
    eq(K.keyLabel("code:38"), "tecla 38");
    eq(K.keyLabel("mouse:272"), "arrastar");
});

t("keyLabel: toda tecla da lista tem um rótulo em texto, não vazio", () => {
    const ruins = K.keys.filter((k) => typeof K.keyLabel(k) !== "string" || K.keyLabel(k).trim() === "");
    vazio(ruins, "teclas sem rótulo");
});

t("pretty: os exemplos do cabeçalho", () => {
    eq(K.pretty(acao("emergency.terminal"), "SUPER + SHIFT + RETURN"), "Super + Shift + Enter");
    eq(K.pretty(acao("win.focus"), "SUPER"), "Super + setas");
    eq(K.pretty(acao("win.close"), ""), "sem atalho");
});

t("pretty: família mostra o nome dela no lugar da tecla; o toque no Super é 'Super (toque)'", () => {
    eq(K.pretty(acao("ws.move"), "SUPER + SHIFT"), "Super + Shift + 1…9");
    eq(K.pretty(acao("win.resize"), "CTRL + ALT"), "Ctrl + Alt + setas");
    eq(K.pretty(acao("mare.launcher.tap"), "SUPER"), "Super (toque)");
});

t("pretty: tecla sozinha, mouse, mídia e sem ação", () => {
    eq(K.pretty(acao("shot.area"), "PRINT"), "Print");
    eq(K.pretty(acao("win.drag"), "SUPER + mouse:272"), "Super + arrastar");
    eq(K.pretty(acao("ws.next.wheel"), "SUPER + mouse_down"), "Super + roda para baixo");
    eq(K.pretty(acao("media.brightness.up"), "XF86MonBrightnessUp"), "Mon Brightness Up");
    eq(K.pretty(null, "SUPER + COMMA"), "Super + ,");
    eq(K.pretty(undefined, "super + alt + k"), "Super + Alt + K");
});

t("pretty: o combo de fábrica de toda ação tem um texto para a página", () => {
    const ruins = [];
    for (const a of K.actions) {
        const p = K.pretty(a, a.combo);
        if (typeof p !== "string" || p === "" || p === "sem atalho" || /undefined|null|\+\s*$|^\s*\+/.test(p))
            ruins.push(a.id + ": " + JSON.stringify(p));
    }
    vazio(ruins, "textos ruins");
});

// --- fromQt, modsFromQt ---

t("fromQt: modificador sozinho dá ''", () => {
    // Shift, Ctrl, Meta, Alt, CapsLock, NumLock, Super_L, Super_R, AltGr.
    for (const key of [0x01000020, 0x01000021, 0x01000022, 0x01000023, 0x01000024, 0x01000025,
        0x01000053, 0x01000054, 0x01001103])
        eq(K.fromQt(key, 50, 0), "", key.toString(16));
});

t("fromQt: letras, F1 a F12 e as teclas com nome", () => {
    eq(K.fromQt(0x41, 38, 0), "A");
    eq(K.fromQt(0x5a, 52, QT.SUPER), "Z");
    eq(K.fromQt(0x01000030, 67, 0), "F1");
    eq(K.fromQt(0x0100003b, 96, 0), "F12");
    eq(K.fromQt(0x01000000, 9, 0), "ESCAPE");
    eq(K.fromQt(0x01000004, 36, 0), "RETURN");
    eq(K.fromQt(0x01000001, 23, 0), "TAB");
    eq(K.fromQt(0x01000012, 113, 0), "Left");
    eq(K.fromQt(0x01000016, 112, 0), "PRIOR");
    eq(K.fromQt(0x20, 65, 0), "SPACE");
    eq(K.fromQt(0x2c, 59, 0), "COMMA");
    eq(K.fromQt(0xc7, 47, 0), "CCEDILLA");
    eq(K.fromQt(0x01000009, 107, 0), "PRINT");
});

t("fromQt: Shift + Tab (Backtab) é TAB", () => eq(K.fromQt(0x01000002, 23, QT.SHIFT), "TAB"));

t("fromQt: a fileira de números vale pela posição, mesmo com Shift ('!' e '@')", () => {
    eq(K.fromQt(0x31, 10, 0), "1");
    eq(K.fromQt(0x21, 10, QT.SHIFT), "1");
    eq(K.fromQt(0x40, 11, QT.SHIFT), "2");
    eq(K.fromQt(0x28, 18, QT.SHIFT), "9");
    eq(K.fromQt(0x29, 19, QT.SHIFT), "0");
    for (let n = 1; n <= 9; n++)
        eq(K.fromQt(0x30 + n, 9 + n, 0), String(n));
});

t("fromQt: pontuação com Shift vira o nome da tecla (americano e ABNT2)", () => {
    eq(K.fromQt(0x3c, 59, QT.SHIFT), "COMMA");
    eq(K.fromQt(0x3e, 60, QT.SHIFT), "PERIOD");
    eq(K.fromQt(0x3f, 61, QT.SHIFT), "SLASH");
    eq(K.fromQt(0x3f, 97, QT.SHIFT), "SLASH");
    eq(K.fromQt(0x3a, 47, QT.SHIFT), "SEMICOLON");
    eq(K.fromQt(0x3a, 61, QT.SHIFT), "SEMICOLON");
    // Um símbolo num lugar que não é de nenhum dos dois layouts: o número físico.
    eq(K.fromQt(0x3c, 94, QT.SHIFT), "code:94");
});

t("fromQt: o teclado numérico é KP_*, não a fileira de números", () => {
    eq(K.fromQt(0x31, 87, QT.KEYPAD), "KP_1");
    eq(K.fromQt(0x30, 90, QT.KEYPAD), "KP_0");
    eq(K.fromQt(0x2b, 86, QT.KEYPAD), "KP_Add");
    eq(K.fromQt(0x2d, 82, QT.KEYPAD), "KP_Subtract");
    eq(K.fromQt(0x2a, 63, QT.KEYPAD), "KP_Multiply");
    eq(K.fromQt(0x2f, 106, QT.KEYPAD), "KP_Divide");
    eq(K.fromQt(0x2e, 91, QT.KEYPAD), "KP_Decimal");
    eq(K.fromQt(0x01000005, 104, QT.KEYPAD), "KP_Enter");
    eq(K.fromQt(0x01000005, 104, 0), "KP_Enter");
    // Uma tecla do bloco numérico sem nome (Home com NumLock desligado).
    eq(K.fromQt(0x01000010, 79, QT.KEYPAD), "code:79");
});

t("fromQt: o que não tem nome vira code:<n>, ou o nome se o lugar é fixo", () => {
    eq(K.fromQt(0x0, 38, 0), "code:38");
    eq(K.fromQt(0xe7, 200, 0), "code:200");
    eq(K.fromQt(0x0, 127, 0), "PAUSE");
    eq(K.fromQt(0x0, 0, 0), "");
    eq(K.fromQt(0x0, 8, 0), "");
});

t("fromQt: tudo que ele devolve o normalizeKey aceita igual", () => {
    const ruins = [];
    for (let scan = 9; scan <= 135; scan++)
        for (const key of [0, 0x41, 0x31, 0x3c, 0x2c, 0x01000004, 0x01000012, 0x01000030, 0xc7, 0x20])
            for (const m of [0, QT.SHIFT, QT.KEYPAD]) {
                // A marca de keypad só vem nas teclas do bloco numérico (79 a 91).
                if (m === QT.KEYPAD && (scan < 79 || scan > 91))
                    continue;
                const k = K.fromQt(key, scan, m);
                if (k !== "" && K.normalizeKey(k) !== k)
                    ruins.push([key.toString(16), scan, m].join("/") + " -> " + k + " -> " + K.normalizeKey(k));
            }
    vazio(ruins, "teclas que o editor capturaria e recusaria");
});

t("modsFromQt: os modificadores segurados, na nossa ordem", () => {
    igual(K.modsFromQt(0), []);
    igual(K.modsFromQt(QT.SUPER), ["SUPER"]);
    igual(K.modsFromQt(QT.SHIFT | QT.SUPER), ["SUPER", "SHIFT"]);
    igual(K.modsFromQt(QT.SHIFT | QT.ALT | QT.CTRL | QT.SUPER), ["SUPER", "CTRL", "ALT", "SHIFT"]);
    // A marca de keypad não é modificador.
    igual(K.modsFromQt(QT.KEYPAD | QT.CTRL), ["CTRL"]);
    igual(K.modsFromQt(QT.SHIFT | QT.ALT | QT.CTRL | QT.SUPER), K.mods);
});

// --- groups, entries ---

t("groups: os grupos na ordem da lista, cada um uma vez, com todas as ações", () => {
    const g = K.groups();
    const nomes = g.map((x) => x.name);
    igual(nomes.filter((n, i) => nomes.indexOf(n) !== i), [], "grupo partido em dois: as ações dele não estão juntas");
    eq(nomes[0], "EMERGÊNCIA");
    igual(ids([].concat(...g.map((x) => x.items))), ids(K.actions));
    for (const grupo of g)
        for (const a of grupo.items)
            eq(a.group, grupo.name);
});

t("entries: uma linha por ação para a busca, com o combo que vale agora", () => {
    const e = K.entries({});
    eq(e.length, K.actions.length);
    igual(e.find((x) => x.id === "app.lazygit"), { id: "app.lazygit", label: "lazygit", combo: "Super + G", group: "APPS" });
    igual(e.find((x) => x.id === "win.focus").combo, "Super + setas");
    const trocado = K.entries({ "app.lazygit": "super+shift+g", "win.float": "" });
    eq(trocado.find((x) => x.id === "app.lazygit").combo, "Super + Shift + G");
    eq(trocado.find((x) => x.id === "win.float").combo, "sem atalho");
    igual(K.entries(undefined), e);
});

// --- luaTable, lua, fromLua, evalCode ---

t("luaTable: o exemplo do cabeçalho", () => {
    eq(K.luaTable({ "win.close": "SUPER + W" }), '{ ["win.close"] = "SUPER + W" }');
});

t("luaTable: sem trocas é {}; só entra o que passa no clean", () => {
    eq(K.luaTable({}), "{}");
    eq(K.luaTable(null), "{}");
    eq(K.luaTable({ "nao.existe": "SUPER + K", "win.close": "SUPER + Q" }, true), "{}");
    eq(K.luaTable({ "win.close": "super+w", "win.float": "", "x\"] = os.execute(\"rm\") --": "SUPER + K" }),
        '{ ["win.close"] = "SUPER + W", ["win.float"] = "" }');
});

t("luaTable: com indent, uma troca por linha", () => {
    eq(K.luaTable({ "win.close": "SUPER + W", "win.float": "" }, true),
        '{\n    ["win.close"] = "SUPER + W",\n    ["win.float"] = "",\n}');
});

t("lua: o atalhos.lua tem o aviso de gerado e devolve a tabela", () => {
    const texto = K.lua({ "win.close": "SUPER + W" });
    assert.match(texto, /^-- Gerado pelos Ajustes do Mk/);
    assert.ok(texto.endsWith('return {\n    ["win.close"] = "SUPER + W",\n}\n'));
    assert.ok(K.lua({}).endsWith("return {}\n"));
    // Fora dos comentários do começo, só o return.
    igual(texto.split("\n").filter((l) => l !== "" && !/^--/.test(l))[0], "return {");
});

t("fromLua: lê de volta o que lua() gera", () => {
    const trocas = { "win.close": "SUPER + W", "win.float": "", "emergency.terminal": "CTRL + ALT + T",
        "win.focus": "ALT", "app.browser": "SUPER + code:38" };
    igual(K.fromLua(K.lua(trocas)), K.clean(trocas));
    igual(K.fromLua(K.lua({})), {});
});

t("fromLua: lê a mesma tabela escrita à mão, e passa pelo clean", () => {
    const texto = 'return {\n'
        + '  ["win.close"]="super+w",\n'
        + '  -- ["win.float"] = "SUPER + Z",\n'
        + '  [ "app.browser" ] = "SUPER + banana",\n'
        + '  ["nao.existe"] = "SUPER + K",\n'
        + '  ["mare.lock"] = "SUPER + F9", ["win.fullscreen"] = "",\n'
        + '}\n';
    igual(K.fromLua(texto), { "mare.lock": "SUPER + F9", "win.close": "SUPER + W", "win.fullscreen": "" });
});

t("fromLua: texto vazio, null ou que não é a tabela dá {}", () => {
    for (const nada of ["", null, undefined, "return {}", "os.execute('rm -rf ~')", "win.close = SUPER + W"])
        igual(K.fromLua(nada), {});
});

t("evalCode: o Lua que troca os atalhos na sessão", () => {
    eq(K.evalCode({}), 'require("mare.binds").apply({})');
    eq(K.evalCode({ "win.close": "super+w" }), 'require("mare.binds").apply({ ["win.close"] = "SUPER + W" })');
    assert.ok(!K.evalCode({ "win.close": "SUPER + W", "win.float": "" }).includes("\n"));
});

t("suspendCode e resumeCode: os atalhos dormem e voltam", () => {
    eq(K.suspendCode(20), 'require("mare.binds").suspend(20)');
    eq(K.resumeCode(), 'require("mare.binds").resume()');
});

// --- diverged ---

// O que o `hyprctl -j binds` responderia se o Hyprland tivesse exatamente as trocas.
function bindsDe(trocas) {
    const todos = K.effective(trocas);
    const out = [];
    for (const a of K.actions) {
        if (todos[a.id] === "")
            continue;
        const p = K.parse(todos[a.id]);
        const modmask = p.mods.reduce((s, m) => s + K.masks[m], 0);
        for (const key of a.family ? K.families[a.family] : [p.key])
            out.push({ description: a.desc, modmask: modmask, key: key, keycode: 0 });
    }
    return out;
}

t("diverged: o Hyprland com os atalhos de fábrica não acusa ninguém", () => {
    igual(K.diverged(bindsDe({}), {}), []);
});

t("diverged: o Hyprland com as trocas aplicadas não acusa ninguém", () => {
    const trocas = { "win.close": "SUPER + W", "win.float": "", "win.focus": "ALT", "emergency.terminal": "CTRL + ALT + T" };
    igual(K.diverged(bindsDe(trocas), trocas), []);
});

t("diverged: a troca que o Hyprland não aceitou aparece", () => {
    igual(K.diverged(bindsDe({}), { "win.close": "SUPER + F9" }), ["win.close"]);
    igual(K.diverged(bindsDe({}), { "win.close": "SUPER + F9", "ws.focus": "ALT" }), ["win.close", "ws.focus"]);
    // Pediu sem atalho e o bind continua lá.
    igual(K.diverged(bindsDe({}), { "win.float": "" }), ["win.float"]);
    // Só os modificadores mudaram.
    igual(K.diverged(bindsDe({ "win.close": "SUPER + SHIFT + Q" }), {}), ["win.close"]);
});

t("diverged: uma família com tecla faltando aparece", () => {
    const binds = bindsDe({}).filter((b) => !(b.description === K.byId("win.focus").desc && b.key === "Down"));
    igual(K.diverged(binds, {}), ["win.focus"]);
});

t("diverged: a caixa da tecla não importa, e code:<n> casa pelo keycode", () => {
    const binds = bindsDe({ "win.close": "SUPER + code:38" }).map((b) => {
        if (b.description === K.byId("win.close").desc)
            return { description: b.description, modmask: b.modmask, key: "", keycode: 38 };
        return { description: b.description, modmask: b.modmask, key: String(b.key).toLowerCase(), keycode: 0 };
    });
    igual(K.diverged(binds, { "win.close": "SUPER + code:38" }), []);
});

t("diverged: as ações fixas não são conferidas", () => {
    const binds = bindsDe({}).filter((b) => b.description !== K.byId("media.play").desc
        && b.description !== K.byId("mare.switcher.next").desc);
    igual(K.diverged(binds, {}), []);
});

t("diverged: sem lista, ou sem nenhum bind com descrição nossa, não acusa ninguém", () => {
    igual(K.diverged(null, { "win.close": "SUPER + F9" }), []);
    igual(K.diverged(undefined, {}), []);
    igual(K.diverged({}, {}), []);
    igual(K.diverged([], { "win.close": "SUPER + F9" }), []);
    // Uma sessão com o binds.lua antigo: binds sem descrição.
    igual(K.diverged([{ description: "", modmask: 64, key: "Q" }, { modmask: 64, key: "F" }], { "win.close": "SUPER + F9" }), []);
});
