// shell/mare/BatteryWatch.qml, o trecho entre "puro: início" e "puro: fim":
// os dois degraus do aviso de bateria fraca (20% e 8%), um aviso por descarga
// em cada um, e o rearme na tomada ou quando a carga passa do degrau + 3.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("BatteryWatch.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["percent", "step"];
const CONSTANTES = ["critical", "low", "slack"];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("BatteryWatch.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

// Uma descarga inteira: cada leitura é [carga, "bat" | "tomada" | "parada"].
// Devolve os avisos dados, na ordem: ["low@20", "critical@8", ...].
function roda(leituras, ligado) {
    let low = false, critical = false;
    const avisos = [];
    for (const [p, como] of leituras) {
        const r = M.step(p, como === "tomada", como === "bat", ligado !== false, low, critical);
        low = r.low;
        critical = r.critical;
        if (r.warn !== "")
            avisos.push(r.warn + "@" + p);
    }
    return avisos;
}
const descendo = (de, ate) => Array.from({ length: de - ate + 1 }, (_, i) => [de - i, "bat"]);

t("os degraus são 20% e 8%, com folga de 3 para rearmar", () => {
    eq(M.low, 20);
    eq(M.critical, 8);
    eq(M.slack, 3);
});

t("percent: o UPower dá de 0 a 100; de 0 a 1 também serve", () => {
    eq(M.percent(87), 87);
    eq(M.percent(19.6), 20);
    eq(M.percent(100), 100);
    eq(M.percent(0.5), 50);
    eq(M.percent(0), 0);
    eq(M.percent(1), 100);
});

t("step: acima de 20% não avisa", () => {
    igual(M.step(21, false, true, true, false, false), { low: false, critical: false, warn: "" });
    igual(M.step(100, false, true, true, false, false), { low: false, critical: false, warn: "" });
});

t("step: em 20% avisa o degrau normal, uma vez", () => {
    igual(M.step(20, false, true, true, false, false), { low: true, critical: false, warn: "low" });
    igual(M.step(20, false, true, true, true, false), { low: true, critical: false, warn: "" });
    igual(M.step(15, false, true, true, true, false), { low: true, critical: false, warn: "" });
});

t("step: em 8% avisa o crítico, que também marca o normal", () => {
    igual(M.step(8, false, true, true, true, false), { low: true, critical: true, warn: "critical" });
    igual(M.step(9, false, true, true, true, false), { low: true, critical: false, warn: "" });
    igual(M.step(3, false, true, true, true, true), { low: true, critical: true, warn: "" });
});

t("step: ligar o notebook já em 5% dá só o aviso crítico", () => {
    igual(M.step(5, false, true, true, false, false), { low: true, critical: true, warn: "critical" });
    igual(roda([[5, "bat"], [5, "bat"], [4, "bat"]]), ["critical@5"]);
});

t("uma descarga de 100 a 1 avisa duas vezes: em 20 e em 8", () => {
    igual(roda(descendo(100, 1)), ["low@20", "critical@8"]);
});

t("a carga oscilando em volta do degrau não repete o aviso", () => {
    igual(roda([[21, "bat"], [20, "bat"], [21, "bat"], [20, "bat"], [22, "bat"], [23, "bat"], [19, "bat"]]), ["low@20"]);
    igual(roda([[8, "bat"], [9, "bat"], [11, "bat"], [8, "bat"], [7, "bat"]]), ["critical@8"]);
});

t("passar do degrau + 3 rearma: 24% o normal, 12% o crítico", () => {
    // 23 ainda não rearma; 24 rearma.
    igual(M.step(23, false, true, true, true, false).low, true);
    igual(M.step(24, false, true, true, true, false).low, false);
    igual(M.step(11, false, true, true, true, true).critical, true);
    igual(M.step(12, false, true, true, true, true), { low: true, critical: false, warn: "" });
    igual(roda([[20, "bat"], [24, "parada"], [20, "bat"]]), ["low@20", "low@20"]);
    igual(roda([[8, "bat"], [12, "parada"], [8, "bat"]]), ["critical@8", "critical@8"]);
});

t("na tomada rearma os dois degraus e não avisa, seja qual for a carga", () => {
    igual(M.step(5, true, false, true, true, true), { low: false, critical: false, warn: "" });
    igual(M.step(5, true, false, true, false, false), { low: false, critical: false, warn: "" });
    igual(roda(descendo(25, 5).concat([[6, "tomada"]], [[6, "bat"]])), ["low@20", "critical@8", "critical@6"]);
    igual(roda([[18, "bat"], [18, "tomada"], [19, "tomada"], [18, "bat"]]), ["low@18", "low@18"]);
});

t("estado que não é descarga (desconhecido, parada) não avisa nem marca", () => {
    igual(M.step(5, false, false, true, false, false), { low: false, critical: false, warn: "" });
    igual(roda([[15, "parada"], [7, "parada"]]), []);
    // Mas rearma pela carga.
    igual(M.step(50, false, false, true, true, true), { low: false, critical: false, warn: "" });
});

t("com o aviso desligado nos Ajustes não avisa nem marca; ao religar, avisa", () => {
    igual(roda(descendo(30, 1), false), []);
    igual(M.step(15, false, true, false, false, false), { low: false, critical: false, warn: "" });
    igual(M.step(15, false, true, true, false, false).warn, "low");
});

t("step não mexe no que recebeu e sempre devolve os três campos", () => {
    for (let p = 0; p <= 100; p++)
        for (const [c, d] of [[true, false], [false, true], [false, false]])
            for (const l of [true, false])
                for (const k of [true, false]) {
                    const r = M.step(p, c, d, true, l, k);
                    igual(Object.keys(r).sort(), ["critical", "low", "warn"]);
                    assert.ok(["", "low", "critical"].includes(r.warn));
                    // Quem avisa fica marcado: o mesmo passo de novo não avisa.
                    if (r.warn !== "")
                        eq(M.step(p, c, d, true, r.low, r.critical).warn, "");
                }
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
