// shell/mare/PowerPanel.qml, o trecho entre "puro: início" e "puro: fim": a
// carga em porcentagem, o tempo que falta, a capacidade e a potência, do
// jeito que o painel de bateria escreve.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("PowerPanel.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["duration", "percent", "watts", "wh"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("PowerPanel.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

t("percent: o UPower dá de 0 a 100; de 0 a 1 também serve", () => {
    eq(M.percent(87.4), 87);
    eq(M.percent(99.5), 100);
    eq(M.percent(0.42), 42);
    eq(M.percent(1), 100);
    eq(M.percent(0), 0);
});

t("duration: horas e minutos", () => {
    eq(M.duration(5400), "1 h 30 min");
    eq(M.duration(3600), "1 h 0 min");
    eq(M.duration(8100), "2 h 15 min");
    eq(M.duration(45 * 60), "45 min");
    eq(M.duration(90), "2 min");
    eq(M.duration(36000 + 59 * 60), "10 h 59 min");
});

t("duration: sem tempo (0, negativo, nulo, NaN) fica vazio", () => {
    eq(M.duration(0), "");
    eq(M.duration(-5), "");
    eq(M.duration(null), "");
    eq(M.duration(undefined), "");
    eq(M.duration(NaN), "");
});

t("duration: os minutos nunca chegam a 60 (3599 s é 1 h, não '60 min')", () => {
    eq(M.duration(3599), "1 h 0 min");
    eq(M.duration(3570), "1 h 0 min");
    eq(M.duration(7199), "2 h 0 min");
    eq(M.duration(3569), "59 min");
    const ruins = [];
    for (let s = 1; s <= 6 * 3600; s += 7) {
        const m = /(\d+) min$/.exec(M.duration(s));
        if (!m || Number(m[1]) > 59)
            ruins.push(s + " s -> " + M.duration(s));
    }
    vazio(ruins, "tempos com 60 minutos");
});

t("wh: uma casa, com vírgula; sem capacidade fica vazio", () => {
    eq(M.wh(52.36), "52,4 Wh");
    eq(M.wh(60), "60,0 Wh");
    eq(M.wh(0), "");
    eq(M.wh(-1), "");
    eq(M.wh(undefined), "");
});

t("watts: mais carregando, menos (o sinal tipográfico) gastando", () => {
    eq(M.watts(18.44, true), "+18,4 W");
    eq(M.watts(7.25, false), "−7,3 W");
    eq(M.watts(0, false), "−0,0 W");
    eq(M.watts(65, true), "+65,0 W");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
