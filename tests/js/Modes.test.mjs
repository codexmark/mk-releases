// shell/mare/Modes.js: o modo apresentação e o modo jogo. A conta de "guardar
// o estado de antes e devolver só o que o modo mudou", o Lua que cala o
// Hyprland no modo jogo e a leitura do `hyprctl -j getoption`.
import { t, eq, igual } from "./harness.mjs";
import { load } from "./load.mjs";

const M = load("Modes.js");

// A máquina de mentira: o que o Modes.qml faria com cada resposta.
const PERF = 2;
const BAL = 1;
const caps = { performance: PERF };
function maquina(inicio) {
    const m = { now: Object.assign({ dnd: false, idle: false, profile: BAL }, inicio), s: M.empty(), applied: [] };
    const commit = (step) => {
        m.s = step.state;
        m.applied.push(step.apply);
        Object.assign(m.now, step.apply);
    };
    m.toggle = (mode, c) => commit(M.toggle(m.s, mode, m.now, c || caps));
    m.leave = () => commit(M.leave(m.s, m.now));
    // A pessoa mexe à mão: o valor muda e o QML avisa (reconcile).
    m.hand = (key, value) => {
        m.now[key] = value;
        m.s = M.reconcile(m.s, m.now);
    };
    return m;
}

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["empty", "label", "wants", "enter", "leave", "toggle", "reconcile", "hyprOff", "hyprBack", "optionOn",
        "known", "copy", "decoration", "truth"];
    const funcoes = Object.keys(M).filter((k) => typeof M[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("empty: nenhum modo, nada guardado, e cada chamada é um objeto novo", () => {
    igual(M.empty(), { mode: "", saved: {} });
    const a = M.empty();
    a.saved.dnd = 1;
    igual(M.empty().saved, {});
});

t("label: o nome do modo em português; sem modo, vazio", () => {
    eq(M.label("present"), "apresentação");
    eq(M.label("game"), "jogo");
    eq(M.label(""), "");
    eq(M.label("constructor"), "");
});

t("wants: apresentação pede não perturbe e sem ociosidade; jogo pede isso e o desempenho", () => {
    igual(M.wants("present", caps), { dnd: true, idle: true });
    igual(M.wants("game", caps), { dnd: true, idle: true, profile: PERF });
});

t("wants: sem o perfil de desempenho na máquina, o jogo não fala de energia", () => {
    igual(M.wants("game", { performance: null }), { dnd: true, idle: true });
    igual(M.wants("game", {}), { dnd: true, idle: true });
    igual(M.wants("game", undefined), { dnd: true, idle: true });
});

t("wants: um modo que não existe não pede nada", () => {
    eq(M.wants("festa", caps), null);
    eq(M.wants("", caps), null);
    eq(M.wants("toString", caps), null);
});

t("apresentação: liga o não perturbe e segura a ociosidade; ao desligar, os dois voltam", () => {
    const m = maquina();
    m.toggle("present");
    eq(m.s.mode, "present");
    igual(m.applied[0], { dnd: true, idle: true });
    igual(m.now, { dnd: true, idle: true, profile: BAL });
    m.toggle("present");
    eq(m.s.mode, "");
    igual(m.applied[1], { dnd: false, idle: false });
    igual(m.now, { dnd: false, idle: false, profile: BAL });
    igual(m.s, M.empty());
});

t("já estava ligado: o não perturbe de antes continua ligado depois do modo", () => {
    const m = maquina({ dnd: true });
    m.toggle("present");
    igual(m.applied[0], { idle: true }, "só mexe no que não estava como o modo quer");
    m.toggle("present");
    igual(m.applied[1], { idle: false });
    igual(m.now, { dnd: true, idle: false, profile: BAL });
});

t("já estava ligada: a cafeína de antes continua ligada depois do modo", () => {
    const m = maquina({ idle: true });
    m.toggle("present");
    m.toggle("present");
    igual(m.now, { dnd: false, idle: true, profile: BAL });
});

t("tudo já estava como o modo quer: o modo liga, não mexe em nada e não desfaz nada", () => {
    const m = maquina({ dnd: true, idle: true });
    m.toggle("present");
    eq(m.s.mode, "present");
    igual(m.applied[0], {});
    m.toggle("present");
    igual(m.applied[1], {});
    igual(m.now, { dnd: true, idle: true, profile: BAL });
});

t("mexeu à mão no meio: o que a pessoa desligou fica desligado, e o resto volta", () => {
    const m = maquina({ dnd: false, idle: false });
    m.toggle("present");
    m.hand("dnd", false);
    eq(m.s.mode, "present", "o modo continua ligado");
    m.toggle("present");
    igual(m.applied[1], { idle: false });
    igual(m.now, { dnd: false, idle: false, profile: BAL });
});

t("mexeu à mão e voltou atrás: a escolha é dela, o modo não desfaz na saída", () => {
    const m = maquina();
    m.toggle("present");
    m.hand("dnd", false);
    m.hand("dnd", true);
    m.toggle("present");
    igual(m.applied[1], { idle: false });
    eq(m.now.dnd, true);
});

t("mexeu à mão no que já estava ligado antes: fica como ela deixou", () => {
    const m = maquina({ dnd: true });
    m.toggle("present");
    m.hand("dnd", false);
    m.toggle("present");
    eq(m.now.dnd, false);
});

t("leave: sem o aviso da mudança, um valor diferente do que o modo pôs também não é desfeito", () => {
    const m = maquina();
    m.toggle("present");
    m.now.idle = false; // o inibidor caiu e ninguém avisou
    m.leave();
    igual(m.applied[1], { dnd: false });
});

t("jogo: tudo da apresentação, mais o perfil de desempenho; ao desligar, o perfil de antes volta", () => {
    const m = maquina({ profile: 0 });
    m.toggle("game");
    eq(m.s.mode, "game");
    igual(m.applied[0], { dnd: true, idle: true, profile: PERF });
    m.toggle("game");
    igual(m.applied[1], { dnd: false, idle: false, profile: 0 });
    igual(m.now, { dnd: false, idle: false, profile: 0 });
});

t("jogo: o perfil 0 (economia) é um valor, não 'sem perfil'", () => {
    const m = maquina({ profile: 0 });
    m.toggle("game");
    eq(m.s.saved.profile.before, 0);
});

t("jogo: sem power-profiles-daemon (perfil desconhecido) ou sem 'desempenho', a energia fica em paz", () => {
    const sem = maquina({ profile: null });
    sem.toggle("game");
    igual(sem.applied[0], { dnd: true, idle: true });
    sem.toggle("game");
    igual(sem.applied[1], { dnd: false, idle: false });
    const fraco = maquina();
    fraco.toggle("game", { performance: null });
    igual(fraco.applied[0], { dnd: true, idle: true });
    eq(fraco.now.profile, BAL);
});

t("jogo: a pessoa trocou o perfil à mão durante o modo, e ele fica", () => {
    const m = maquina();
    m.toggle("game");
    m.hand("profile", 0);
    m.toggle("game");
    igual(m.now, { dnd: false, idle: false, profile: 0 });
});

t("os modos são exclusivos: ligar o jogo com a apresentação ligada troca de um para o outro", () => {
    const m = maquina();
    m.toggle("present");
    m.toggle("game");
    eq(m.s.mode, "game");
    igual(m.applied[1], { profile: PERF }, "o que já estava posto não é posto de novo");
    igual(m.s.saved.dnd, { before: false, set: true }, "o estado de antes continua o original");
    igual(m.s.saved.idle, { before: false, set: true });
});

t("trocar de um modo para o outro e desligar: volta ao estado de antes do primeiro", () => {
    const m = maquina({ dnd: false, idle: true, profile: 0 });
    m.toggle("present");
    m.toggle("game");
    m.toggle("game");
    eq(m.s.mode, "");
    igual(m.now, { dnd: false, idle: true, profile: 0 });
    igual(m.s, M.empty());
});

t("do jogo para a apresentação: o perfil volta já; o resto espera o modo desligar", () => {
    const m = maquina();
    m.toggle("game");
    m.toggle("present");
    eq(m.s.mode, "present");
    igual(m.applied[1], { profile: BAL });
    igual(Object.keys(m.s.saved).sort(), ["dnd", "idle"]);
    m.toggle("present");
    igual(m.now, { dnd: false, idle: false, profile: BAL });
});

t("do jogo para a apresentação com o perfil mexido à mão: ele fica como a pessoa deixou", () => {
    const m = maquina();
    m.toggle("game");
    m.hand("profile", 0);
    m.toggle("present");
    igual(m.applied[1], {});
    eq(m.now.profile, 0);
});

t("ida e volta entre os modos várias vezes não perde o estado de antes", () => {
    const m = maquina({ dnd: true, idle: false, profile: 0 });
    for (const mode of ["present", "game", "present", "game", "present"])
        m.toggle(mode);
    m.toggle("present");
    igual(m.now, { dnd: true, idle: false, profile: 0 });
});

t("enter: o mesmo modo de novo não muda nada; um modo que não existe também não", () => {
    const m = maquina();
    m.toggle("present");
    const again = M.enter(m.s, "present", m.now, caps);
    igual(again.apply, {});
    igual(again.state, m.s);
    const nada = M.enter(m.s, "festa", m.now, caps);
    igual(nada.apply, {});
    igual(nada.state, m.s);
    const t0 = M.toggle(M.empty(), "festa", m.now, caps);
    igual(t0.state, M.empty());
});

t("leave: sem modo ligado não faz nada", () => {
    igual(M.leave(M.empty(), { dnd: true, idle: true, profile: PERF }), { state: M.empty(), apply: {} });
});

t("as funções não mexem no estado recebido (o QML troca a propriedade inteira)", () => {
    const now = { dnd: false, idle: false, profile: BAL };
    const antes = M.empty();
    const dentro = M.enter(antes, "game", now, caps).state;
    igual(antes, M.empty());
    const copia = JSON.parse(JSON.stringify(dentro));
    M.leave(dentro, { dnd: true, idle: true, profile: PERF });
    M.reconcile(dentro, { dnd: false, idle: false, profile: 0 });
    M.enter(dentro, "present", { dnd: true, idle: true, profile: PERF }, caps);
    igual(dentro, copia);
});

t("reconcile: só larga o que saiu do valor que o modo pôs; sem mudança, devolve o mesmo estado", () => {
    const now = { dnd: false, idle: false, profile: BAL };
    const s = M.enter(M.empty(), "game", now, caps).state;
    const igualzinho = { dnd: true, idle: true, profile: PERF };
    eq(M.reconcile(s, igualzinho), s);
    const r = M.reconcile(s, { dnd: true, idle: false, profile: PERF });
    eq(r.mode, "game");
    igual(Object.keys(r.saved).sort(), ["dnd", "profile"]);
});

// --- Hyprland ---

t("hyprOff: o modo jogo desliga as animações, com as mesmas chaves do hl.config dos Ajustes", () => {
    eq(M.hyprOff({}), "hl.config({ animations = { enabled = false } })\n");
    eq(M.hyprOff(undefined), "hl.config({ animations = { enabled = false } })\n");
});

t("hyprOff: desfoque e sombra só entram se estavam ligados quando o modo começou", () => {
    eq(M.hyprOff({ blur: true }),
        "hl.config({ animations = { enabled = false }, decoration = { blur = { enabled = false } } })\n");
    eq(M.hyprOff({ shadow: true }),
        "hl.config({ animations = { enabled = false }, decoration = { shadow = { enabled = false } } })\n");
    eq(M.hyprOff({ blur: true, shadow: true }),
        "hl.config({ animations = { enabled = false }, decoration = { blur = { enabled = false }, shadow = { enabled = false } } })\n");
    eq(M.hyprOff({ blur: false, shadow: null }), "hl.config({ animations = { enabled = false } })\n");
});

t("hyprBack: as animações voltam ao que os Ajustes dizem (Velocidade 'sem' = desligadas)", () => {
    eq(M.hyprBack({}, true), "hl.config({ animations = { enabled = true } })\n");
    eq(M.hyprBack({}, false), "hl.config({ animations = { enabled = false } })\n");
});

t("hyprBack: religa só o desfoque e a sombra que o modo desligou", () => {
    eq(M.hyprBack({ blur: true }, true),
        "hl.config({ animations = { enabled = true }, decoration = { blur = { enabled = true } } })\n");
    eq(M.hyprBack({ blur: true, shadow: true }, false),
        "hl.config({ animations = { enabled = false }, decoration = { blur = { enabled = true }, shadow = { enabled = true } } })\n");
    eq(M.hyprBack({ blur: false }, true), "hl.config({ animations = { enabled = true } })\n");
});

t("optionOn: lê o `hyprctl -j getoption` (int, bool ou value)", () => {
    eq(M.optionOn('{"option": "decoration:blur:enabled", "int": 1, "set": true}'), true);
    eq(M.optionOn('{"option": "decoration:blur:enabled", "int": 0, "set": true}'), false);
    eq(M.optionOn('{"option": "decoration:shadow:enabled", "bool": true, "set": false}'), true);
    eq(M.optionOn('{"option": "decoration:shadow:enabled", "bool": false}'), false);
    eq(M.optionOn('{"value": true}'), true);
    eq(M.optionOn('{"value": "false"}'), false);
    eq(M.optionOn('{"value": 1}'), true);
});

t("optionOn: o `set` (se a pessoa escreveu a opção) não é o valor", () => {
    eq(M.optionOn('{"option": "x", "set": true}'), null);
});

t("optionOn: lê também a resposta em texto", () => {
    eq(M.optionOn("int: 1\nset: true\n"), true);
    eq(M.optionOn("int: 0\nset: false\n"), false);
    eq(M.optionOn("bool: true\nset: true"), true);
});

t("optionOn: o que não deu para ler é 'não sei' (null), e o modo não mexe", () => {
    for (const ruim of ["", "no such option", "{", "null", "[]", '{"erro": "x"}', '{"int": "abc"}', undefined, null])
        eq(M.optionOn(ruim), null, String(ruim));
});

// --- as peças de dentro ---

t("known: só os dois modos, e nenhum nome herdado de Object", () => {
    eq(M.known("present"), true);
    eq(M.known("game"), true);
    for (const n of ["", "festa", "constructor", "toString", "__proto__"])
        eq(M.known(n), false, n);
});

t("copy: as anotações copiadas por dentro (mexer na cópia não mexe na original)", () => {
    const a = { dnd: { before: false, set: true } };
    const b = M.copy(a);
    igual(b, a);
    b.dnd.before = true;
    delete b.dnd;
    igual(a, { dnd: { before: false, set: true } });
});

t("decoration: o pedaço do hl.config para o que estava ligado; nada ligado, nada escrito", () => {
    eq(M.decoration({}, "false"), "");
    eq(M.decoration(null, "false"), "");
    eq(M.decoration({ shadow: true }, "true"), ", decoration = { shadow = { enabled = true } }");
});

t("truth: booleano, número e os textos 'true'/'false'/'1'/'0'; o resto é null", () => {
    igual([true, false, 1, 0, 2, "true", "false", "1", "0"].map(M.truth),
        [true, false, true, false, true, true, false, true, false]);
    igual(["", "sim", null, undefined, NaN, {}, []].map(M.truth), [null, null, null, null, null, null, null]);
});
