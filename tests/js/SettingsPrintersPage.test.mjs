// shell/mare/SettingsPrintersPage.qml, o trecho entre "puro: início" e "puro:
// fim": o estado que o mare-printers contou, o recado de uma ação (e a linha
// a que ele se refere), a impressora achada que já tem fila e o texto da fila.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("SettingsPrintersPage.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["addedIn", "noteOf", "queueText", "stateOf"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("SettingsPrintersPage.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

// O que o `mare-printers status` responde.
const STATUS = {
    installed: true, running: true, enabled: true, "default": "HP_LaserJet_M15w", pending: "",
    printers: [
        { name: "HP_LaserJet_M15w", info: "HP LaserJet M15w", uri: "ipp://HP-M15w.local:631/ipp/print", state: "idle", jobs: [] },
        { name: "Epson_L3250", info: "EPSON L3250 Series", uri: "ipps://EPSON-L3250.local:631/ipp/print", state: "printing", jobs: [{ id: 12 }] }
    ],
    last: null
};

t("stateOf: fica com os seis campos do estado, na mesma ordem (o texto é comparado)", () => {
    const s = M.stateOf(STATUS);
    igual(Object.keys(s), ["installed", "running", "enabled", "default", "pending", "printers"]);
    eq(s.installed, true);
    eq(s["default"], "HP_LaserJet_M15w");
    eq(s.printers.length, 2);
    // O resultado da última ação não faz parte do estado.
    assert.ok(!("last" in s));
});

t("stateOf: resposta vazia vira tudo falso, vazio e sem impressoras", () => {
    igual(M.stateOf({}), { installed: false, running: false, enabled: false, "default": "", pending: "", printers: [] });
});

t("stateOf: só true vale como verdadeiro; nulo vira vazio", () => {
    igual(M.stateOf({ installed: "true", running: 1, enabled: null, "default": null, pending: null, printers: null }),
        { installed: false, running: false, enabled: false, "default": "", pending: "", printers: [] });
});

t("stateOf: a mesma resposta dá o mesmo texto; uma fila que muda dá outro", () => {
    eq(JSON.stringify(M.stateOf(STATUS)), JSON.stringify(M.stateOf(JSON.parse(JSON.stringify(STATUS)))));
    // Só o resultado da última ação mudou: o estado é o mesmo, a lista não é refeita.
    eq(JSON.stringify(M.stateOf(Object.assign({}, STATUS, { last: { key: "test:x", ok: true, at: 5 } }))), JSON.stringify(M.stateOf(STATUS)));
    const outro = JSON.parse(JSON.stringify(STATUS));
    outro.printers[0].state = "paused";
    assert.notEqual(JSON.stringify(M.stateOf(outro)), JSON.stringify(M.stateOf(STATUS)));
    assert.notEqual(JSON.stringify(M.stateOf(Object.assign({}, STATUS, { pending: "add:ipp://x" }))), JSON.stringify(M.stateOf(STATUS)));
});

t("noteOf: ação que deu certo numa fila aponta para a linha dela", () => {
    igual(M.noteOf({ key: "test:HP_LaserJet_M15w", ok: true, message: "página de teste enviada", at: 1 }),
        { error: false, text: "página de teste enviada", target: "HP_LaserJet_M15w" });
    igual(M.noteOf({ key: "default:Epson_L3250", ok: true, message: "", at: 1 }), { error: false, text: "", target: "Epson_L3250" });
});

t("noteOf: remover e adicionar que deram certo não têm linha (o recado vai no alto)", () => {
    igual(M.noteOf({ key: "remove:HP_LaserJet_M15w", ok: true, message: "removida", at: 1 }), { error: false, text: "removida", target: "" });
    igual(M.noteOf({ key: "add:ipp://HP-M15w.local:631/ipp/print", ok: true, message: "adicionada", at: 1 }),
        { error: false, text: "adicionada", target: "" });
});

t("noteOf: remover e adicionar que falharam apontam para a linha, com o endereço inteiro", () => {
    igual(M.noteOf({ key: "add:ipp://HP-M15w.local:631/ipp/print", ok: false, message: "a impressora não respondeu", at: 1 }),
        { error: true, text: "a impressora não respondeu", target: "ipp://HP-M15w.local:631/ipp/print" });
    igual(M.noteOf({ key: "remove:Fila:com:dois-pontos", ok: false, message: "x", at: 1 }), { error: true, text: "x", target: "Fila:com:dois-pontos" });
});

t("noteOf: ação sem alvo (ligar o serviço) não aponta para linha nenhuma", () => {
    igual(M.noteOf({ key: "enable", ok: true, message: "serviço ligado", at: 1 }), { error: false, text: "serviço ligado", target: "" });
    igual(M.noteOf({ key: "install", ok: false, message: "", at: 1 }), { error: true, text: "não deu certo", target: "" });
});

t("noteOf: falha sem mensagem diz 'não deu certo'; só ok === true é sucesso", () => {
    eq(M.noteOf({ key: "test:X", ok: false }).text, "não deu certo");
    eq(M.noteOf({ key: "test:X" }).error, true);
    eq(M.noteOf({ key: "test:X", ok: "true", message: "m" }).error, true);
    eq(M.noteOf({ key: "test:X", ok: true }).text, "");
});

t("noteOf: a mensagem vai como veio, com aspas, barra e acento", () => {
    const m = "lpadmin: não conectou a \"X.local:631\" (C:\\x)";
    eq(M.noteOf({ key: "add:ipp://X.local:631/", ok: false, message: m }).text, m);
});

t("addedIn: a impressora achada já tem fila, em ipp ou em ipps", () => {
    const filas = STATUS.printers;
    eq(M.addedIn("ipp://HP-M15w.local:631/ipp/print", filas), true);
    eq(M.addedIn("ipps://HP-M15w.local:631/ipp/print", filas), true);
    eq(M.addedIn("ipp://EPSON-L3250.local:631/ipp/print", filas), true);
    eq(M.addedIn("ipp://Brother-HL1212W.local:631/ipp/print", filas), false);
    eq(M.addedIn("ipp://HP-M15w.local:631/ipp/print", []), false);
});

t("addedIn: outro protocolo, outra porta ou outro caminho é outra impressora", () => {
    const filas = STATUS.printers;
    eq(M.addedIn("socket://HP-M15w.local:631/ipp/print", filas), false);
    eq(M.addedIn("ipp://HP-M15w.local:8631/ipp/print", filas), false);
    eq(M.addedIn("ipp://HP-M15w.local:631/ipp/print2", filas), false);
    // O "ipp:" só sai do começo.
    eq(M.addedIn("dnssd://ipp://HP-M15w.local:631/ipp/print", filas), false);
    // Fila USB: compara o endereço inteiro.
    eq(M.addedIn("usb://HP/LaserJet?serial=A:1", [{ uri: "usb://HP/LaserJet?serial=A:1" }]), true);
});

t("queueText: vazia, uma e várias", () => {
    eq(M.queueText(0), "fila vazia");
    eq(M.queueText(1), "1 na fila");
    eq(M.queueText(2), "2 na fila");
    eq(M.queueText(37), "37 na fila");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
