// shell/mare/SwitcherPanel.qml, o trecho entre "puro: início" e "puro: fim":
// a ordem de uso das janelas, o índice que dá a volta e a janela de sete
// cartões que acompanha o escolhido.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("SwitcherPanel.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["firstCard", "focusOrder", "wrapped"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("SwitcherPanel.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

t("focusOrder: a posição que o Hyprland informa; 0 é a janela de agora", () => {
    eq(M.focusOrder({ lastIpcObject: { focusHistoryID: 0 } }), 0);
    eq(M.focusOrder({ lastIpcObject: { focusHistoryID: 5 } }), 5);
});

t("focusOrder: sem o dado do Hyprland, a janela vai para o fim", () => {
    eq(M.focusOrder({ lastIpcObject: {} }), 999);
    eq(M.focusOrder({ lastIpcObject: null }), 999);
    eq(M.focusOrder({}), 999);
});

t("focusOrder: ordena as janelas da mais recente para a mais antiga", () => {
    const janelas = [
        { address: "0xc", lastIpcObject: { focusHistoryID: 2 } }, { address: "0xnova" },
        { address: "0xa", lastIpcObject: { focusHistoryID: 0 } }, { address: "0xb", lastIpcObject: { focusHistoryID: 1 } }
    ];
    igual(janelas.slice().sort((a, b) => M.focusOrder(a) - M.focusOrder(b)).map((j) => j.address), ["0xa", "0xb", "0xc", "0xnova"]);
});

t("wrapped: dentro da fila é o próprio índice", () => {
    eq(M.wrapped(0, 5), 0);
    eq(M.wrapped(4, 5), 4);
});

t("wrapped: passa do fim e volta ao começo; para trás, vai ao fim", () => {
    eq(M.wrapped(5, 5), 0);
    eq(M.wrapped(6, 5), 1);
    eq(M.wrapped(-1, 5), 4);
    eq(M.wrapped(-5, 5), 0);
    eq(M.wrapped(-6, 5), 4);
    eq(M.wrapped(23, 5), 3);
    eq(M.wrapped(-23, 5), 2);
});

t("wrapped: sem janela nenhuma é 0, sem dividir por zero", () => {
    eq(M.wrapped(0, 0), 0);
    eq(M.wrapped(3, 0), 0);
    eq(M.wrapped(-2, 0), 0);
    eq(M.wrapped(7, 1), 0);
});

t("wrapped: Tab e Shift+Tab seguidos passam por todas as janelas", () => {
    for (const n of [1, 2, 7, 8, 13]) {
        const ida = new Set(), volta = new Set();
        for (let i = 1; i <= n; i++) {
            ida.add(M.wrapped(i, n));
            volta.add(M.wrapped(-i, n));
        }
        eq(ida.size, n);
        eq(volta.size, n);
    }
});

t("firstCard: até sete janelas, todas à mostra desde a primeira", () => {
    for (let n = 0; n <= 7; n++)
        for (let s = 0; s < Math.max(1, n); s++)
            eq(M.firstCard(s, n, 7), 0, n + " janelas, escolhida " + s);
});

t("firstCard: com mais de sete, o escolhido fica no quarto lugar enquanto dá", () => {
    eq(M.firstCard(0, 12, 7), 0);
    eq(M.firstCard(3, 12, 7), 0);
    eq(M.firstCard(4, 12, 7), 1);
    eq(M.firstCard(7, 12, 7), 4);
    // No fim da fila a janela para: não sobra cartão vazio à direita.
    eq(M.firstCard(8, 12, 7), 5);
    eq(M.firstCard(11, 12, 7), 5);
});

t("firstCard: o escolhido está sempre entre os cartões à mostra, e eles nunca passam do fim", () => {
    const ruins = [];
    for (let n = 1; n <= 30; n++) {
        for (let s = 0; s < n; s++) {
            const de = M.firstCard(s, n, 7);
            const ate = Math.min(n, de + 7);
            if (de < 0 || s < de || s >= ate || ate - de !== Math.min(n, 7))
                ruins.push(n + " janelas, escolhida " + s + ": mostra de " + de + " a " + (ate - 1));
        }
    }
    vazio(ruins, "filas erradas");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
