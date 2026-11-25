.pragma library

// Os modos do Maré: apresentação ("present") e jogo ("game"). Aqui fica só a
// conta, sem tocar em nada: o Modes.qml lê a máquina, pergunta o que fazer e
// aplica a resposta. Testado no node (tests/js/Modes.test.mjs).
//
// Um modo quer certos interruptores num certo valor (`wants`). Ao ligar, cada
// um que não estava assim é anotado ({ before, set }) e mudado; o que já
// estava do jeito que o modo quer não é anotado, e por isso continua ligado
// depois. Ao desligar, volta só o que foi anotado e ainda está como o modo
// deixou. Se a pessoa mexe num deles à mão durante o modo, a anotação some
// (`reconcile`): a escolha passa a ser dela.
//
// Os interruptores:
//   dnd      o não perturbe (Notifs.dnd)
//   idle     a ociosidade segurada (Caffeine.on)
//   profile  o perfil de energia (PowerProfiles.profile), só no jogo e só se
//            a máquina oferece "desempenho" (caps.performance)
// `now` é o valor de cada um agora; null ou undefined quer dizer "não sei"
// (sem power-profiles-daemon), e o modo deixa em paz.
//
// O estado é { mode: "" | "present" | "game", saved: { chave: { before, set } } }.
// Nenhuma função mexe no estado recebido: devolvem um novo.

var labels = { present: "apresentação", game: "jogo" };

function known(mode) {
    return Object.prototype.hasOwnProperty.call(labels, mode);
}

function empty() {
    return { mode: "", saved: {} };
}

function label(mode) {
    return known(mode) ? labels[mode] : "";
}

// O que o modo quer: { chave: valor }. null para um modo que não existe.
function wants(mode, caps) {
    if (!known(mode))
        return null;
    var out = { dnd: true, idle: true };
    if (mode === "game" && caps && caps.performance !== null && caps.performance !== undefined)
        out.profile = caps.performance;
    return out;
}

function copy(saved) {
    var out = {};
    for (var k in saved)
        out[k] = { before: saved[k].before, set: saved[k].set };
    return out;
}

// Liga o modo (ou troca de um para o outro). Devolve { state, apply }: o
// estado novo e o que mudar na máquina, como { chave: valor }. Na troca, o
// que os dois modos querem continua anotado com o "antes" original; o que só
// o modo antigo queria volta já.
function enter(state, mode, now, caps) {
    var want = wants(mode, caps);
    if (want === null || state.mode === mode)
        return { state: state, apply: {} };
    var saved = copy(state.saved);
    var apply = {};
    for (var old in saved) {
        if (Object.prototype.hasOwnProperty.call(want, old) && want[old] === saved[old].set)
            continue;
        if (now[old] === saved[old].set)
            apply[old] = saved[old].before;
        delete saved[old];
    }
    for (var k in want) {
        if (saved[k] !== undefined || now[k] === null || now[k] === undefined || now[k] === want[k])
            continue;
        saved[k] = { before: now[k], set: want[k] };
        apply[k] = want[k];
    }
    return { state: { mode: mode, saved: saved }, apply: apply };
}

// Desliga o modo: volta o que ele mudou e que ainda está como ele deixou.
function leave(state, now) {
    var apply = {};
    for (var k in state.saved)
        if (now[k] === state.saved[k].set)
            apply[k] = state.saved[k].before;
    return { state: empty(), apply: apply };
}

// O atalho e o botão: desliga se é este o modo ligado; senão, liga este.
function toggle(state, mode, now, caps) {
    if (!known(mode))
        return { state: state, apply: {} };
    return state.mode === mode ? leave(state, now) : enter(state, mode, now, caps);
}

// Um interruptor mudou fora do modo (a pessoa mexeu, o inibidor caiu): o que
// não está mais como o modo deixou sai das anotações. Sem mudança, devolve o
// mesmo estado.
function reconcile(state, now) {
    var saved = {};
    var dropped = false;
    for (var k in state.saved) {
        if (now[k] === state.saved[k].set)
            saved[k] = { before: state.saved[k].before, set: state.saved[k].set };
        else
            dropped = true;
    }
    return dropped ? { mode: state.mode, saved: saved } : state;
}

// --- Hyprland (só o modo jogo) ---
// O mesmo hl.config dos Ajustes, mandado por `hyprctl eval`. `was` diz o que
// estava ligado no Hyprland quando o modo começou ({ blur: true, shadow:
// true }): o Mk não tem ajuste de desfoque nem de sombra (de fábrica são
// desligados), então só se desliga, e depois se religa, o que foi lido como
// ligado. As animações têm ajuste (Velocidade) e voltam ao que ele diz.

function decoration(was, value) {
    var parts = [];
    if (was && was.blur === true)
        parts.push("blur = { enabled = " + value + " }");
    if (was && was.shadow === true)
        parts.push("shadow = { enabled = " + value + " }");
    return parts.length > 0 ? ", decoration = { " + parts.join(", ") + " }" : "";
}

function hyprOff(was) {
    return "hl.config({ animations = { enabled = false }" + decoration(was, "false") + " })\n";
}

// `animations`: os Ajustes querem animação (Velocidade diferente de "sem")?
function hyprBack(was, animations) {
    return "hl.config({ animations = { enabled = " + (animations ? "true" : "false") + " }"
        + decoration(was, "true") + " })\n";
}

function truth(v) {
    if (typeof v === "boolean")
        return v;
    if (typeof v === "number" && isFinite(v))
        return v !== 0;
    if (v === "true" || v === "1")
        return true;
    if (v === "false" || v === "0")
        return false;
    return null;
}

// A resposta do `hyprctl -j getoption <opção>` para uma opção de ligar e
// desligar: true, false, ou null se não deu para entender (aí o modo não
// mexe na opção). Aceita o valor em "int", "bool" ou "value", e a resposta
// em texto ("int: 1").
function optionOn(text) {
    var s = String(text === null || text === undefined ? "" : text);
    var o = null;
    try {
        o = JSON.parse(s);
    } catch (e) {
        var m = /^\s*(?:int|bool|value):\s*(\S+)\s*$/m.exec(s);
        return m ? truth(m[1]) : null;
    }
    if (o === null || typeof o !== "object" || Array.isArray(o))
        return null;
    var fields = ["bool", "int", "value"];
    for (var i = 0; i < fields.length; i++)
        if (Object.prototype.hasOwnProperty.call(o, fields[i]))
            return truth(o[fields[i]]);
    return null;
}
