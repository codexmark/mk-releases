// shell/mare/SettingsAppsPage.qml, o trecho entre "puro: início" e "puro:
// fim": os papéis que o mare-default-apps lista, por id; o aviso de app de
// terminal como padrão; e o padrão que não está entre as escolhas.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("SettingsAppsPage.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["rolesById", "strayIn", "terminalAppIn"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("SettingsAppsPage.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

// O que o `mare-default-apps list` responde em .roles.
const PAPEIS = [
    { id: "navegador", current: "firefox.desktop", currentName: "Firefox", apps: [
        { id: "firefox.desktop", name: "Firefox", terminal: false }, { id: "chromium.desktop", name: "Chromium", terminal: false }] },
    { id: "arquivos", current: "org.gnome.Nautilus.desktop", currentName: "Arquivos", apps: [
        { id: "org.gnome.Nautilus.desktop", name: "Arquivos", terminal: false }, { id: "yazi.desktop", name: "Yazi", terminal: true }] },
    { id: "editor", current: "nvim.desktop", currentName: "Neovim", apps: [
        { id: "nvim.desktop", name: "Neovim", terminal: true }, { id: "org.gnome.TextEditor.desktop", name: "Editor de texto", terminal: false }] },
    { id: "pdf", current: "", currentName: "", apps: [] },
    { id: "imagens", current: "gimp.desktop", currentName: "GIMP: \"edição\"", apps: [{ id: "org.gnome.Loupe.desktop", name: "Visualizador", terminal: false }] }
];
const com = (id, current) => PAPEIS.map((r) => (r.id === id ? Object.assign({}, r, { current: current }) : r));

t("rolesById: a lista vira um mapa pelo id, com o papel inteiro", () => {
    const mapa = M.rolesById(PAPEIS);
    igual(Object.keys(mapa), ["navegador", "arquivos", "editor", "pdf", "imagens"]);
    assert.ok(mapa.editor === PAPEIS[2]);
    igual(M.rolesById([]), {});
});

t("rolesById: id repetido, vale o último", () => {
    eq(M.rolesById([{ id: "pdf", current: "a" }, { id: "pdf", current: "b" }]).pdf.current, "b");
});

t("rolesById: resposta sem a lista de papéis lança (quem chama mostra a falha de leitura)", () => {
    assert.throws(() => M.rolesById(undefined));
    assert.throws(() => M.rolesById(null));
});

t("terminalAppIn: algum papel com app de terminal como padrão", () => {
    eq(M.terminalAppIn(M.rolesById(PAPEIS)), true);
    // Com o editor gráfico, não: o Yazi está na lista dos arquivos, mas não é o padrão.
    eq(M.terminalAppIn(M.rolesById(com("editor", "org.gnome.TextEditor.desktop"))), false);
    eq(M.terminalAppIn(M.rolesById(com("editor", "org.gnome.TextEditor.desktop").map(
        (r) => (r.id === "arquivos" ? Object.assign({}, r, { current: "yazi.desktop" }) : r)))), true);
});

t("terminalAppIn: sem papéis, sem padrão ou com padrão de fora da lista, não", () => {
    eq(M.terminalAppIn({}), false);
    eq(M.terminalAppIn(M.rolesById([PAPEIS[3]])), false);
    eq(M.terminalAppIn(M.rolesById([PAPEIS[4]])), false);
});

t("strayIn: o padrão que está entre as escolhas não é mostrado à parte", () => {
    const mapa = M.rolesById(PAPEIS);
    eq(M.strayIn("navegador", mapa.navegador), "");
    eq(M.strayIn("editor", mapa.editor), "");
});

t("strayIn: o padrão de fora das escolhas aparece pelo nome, como veio", () => {
    eq(M.strayIn("imagens", M.rolesById(PAPEIS).imagens), "GIMP: \"edição\"");
});

t("strayIn: sem padrão, sem o papel ou na linha do terminal, vazio", () => {
    const mapa = M.rolesById(PAPEIS);
    eq(M.strayIn("pdf", mapa.pdf), "");
    eq(M.strayIn("video", mapa.video), "");
    eq(M.strayIn("video", undefined), "");
    // A linha do terminal não vem dos papéis.
    eq(M.strayIn("terminal", mapa.terminal), "");
    eq(M.strayIn("terminal", PAPEIS[4]), "");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
