// shell/mare/Island.js: os textos curtos das leituras da ilha, de largura
// previsível. A ilha reserva o lugar do texto mais largo de cada leitura
// (`reserve`) e por isso não estica nem encolhe quando um número ganha ou perde
// um dígito; só muda de tamanho quando algo entra ou sai dela.
import { t, eq, igual, assert } from "./harness.mjs";
import { load } from "./load.mjs";

const I = load("Island.js");

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["rate", "percent", "degrees"];
    const funcoes = Object.keys(I).filter((k) => typeof I[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("rate: três algarismos no máximo, sem espaço", () => {
    eq(I.rate(0), "0B/s");
    eq(I.rate(999), "999B/s");
    eq(I.rate(1024), "1,0K/s");
    eq(I.rate(2.4 * 1024), "2,4K/s");
    eq(I.rate(9.96 * 1024), "10K/s");
    eq(I.rate(803 * 1024), "803K/s");
    eq(I.rate(1.4 * 1024 * 1024), "1,4M/s");
    eq(I.rate(16.4 * 1024 * 1024), "16M/s");
    eq(I.rate(120 * 1024 * 1024), "120M/s");
});

t("rate: perto da virada de unidade não aparece 1000 nem 1023 (passa para a unidade de cima)", () => {
    eq(I.rate(1000), "1,0K/s");
    eq(I.rate(1023), "1,0K/s");
    eq(I.rate(999.6 * 1024), "1,0M/s");
    eq(I.rate(1023 * 1024), "1,0M/s");
    eq(I.rate(1000 * 1024 * 1024), "1,0G/s");
});

t("rate: nenhum valor passa da largura reservada", () => {
    for (let e = 0; e < 34; e += 0.07) {
        const s = I.rate(Math.pow(2, e));
        assert.ok(s.length <= I.reserve.rate.length, s + " é mais largo que " + I.reserve.rate);
    }
});

t("rate: valor que não é número, ou negativo, vira zero", () => {
    eq(I.rate(NaN), "0B/s");
    eq(I.rate(undefined), "0B/s");
    eq(I.rate(-5), "0B/s");
});

t("percent: de 0 a 100, inteiro, e cabe no reservado", () => {
    eq(I.percent(0), "0%");
    eq(I.percent(0.094), "9%");
    eq(I.percent(0.305), "31%");
    eq(I.percent(1), "100%");
    eq(I.percent(1.7), "100%");
    eq(I.percent(-1), "0%");
    eq(I.percent(NaN), "0%");
    assert.ok("100%".length <= I.reserve.percent.length);
});

t("degrees: inteiro com o sinal de grau, e cabe no reservado até 99", () => {
    eq(I.degrees(46.4), "46°");
    eq(I.degrees(99.4), "99°");
    assert.ok(I.degrees(99).length <= I.reserve.degrees.length);
});

t("reserve: o texto mais largo de cada leitura", () => {
    igual(I.reserve, { rate: "999K/s", percent: "100%", degrees: "99°" });
});
