// shell/mare/Wave.js: a onda da ilha. O nível de cada faixa segue o som (sobe
// depressa, desce devagar), o rastro segue o nível com atraso, e a curva liga
// as faixas em pontos suaves, presos em zero nas pontas.
import { t, eq, igual, perto, assert } from "./harness.mjs";
import { load } from "./load.mjs";

const W = load("Wave.js");

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["clamp01", "smooth", "trail", "curve"];
    const funcoes = Object.keys(W).filter((k) => typeof W[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("clamp01: prende em 0 a 1; o que não é número vale zero", () => {
    eq(W.clamp01(0.4), 0.4);
    eq(W.clamp01(3), 1);
    eq(W.clamp01(-2), 0);
    eq(W.clamp01(NaN), 0);
    eq(W.clamp01(undefined), 0);
});

t("smooth: sobe 70% do caminho e desce 35% (movimento de líquido)", () => {
    perto(W.smooth([0], [1], 1)[0], 0.7);
    perto(W.smooth([1], [0], 1)[0], 0.65);
    perto(W.smooth([0.5], [0.5], 1)[0], 0.5);
});

t("smooth: sem quadro anterior parte do zero; valor fora de 0 a 1 é aparado; faixa que falta vale zero", () => {
    perto(W.smooth([], [1, 1], 2)[1], 0.7);
    perto(W.smooth([0], [7], 1)[0], 0.7);
    perto(W.smooth([0.4], [-3], 1)[0], 0.26);
    igual(W.smooth([0, 0], [NaN], 2), [0, 0]);
});

t("trail: segue o nível com atraso (sobe 22% do caminho, desce 9%)", () => {
    perto(W.trail([0], [1], 1)[0], 0.22);
    perto(W.trail([1], [0], 1)[0], 0.91);
});

t("trail: fica para trás na subida e demora mais a cair que o nível", () => {
    let nivel = [0], rastro = [0];
    for (let i = 0; i < 3; i++) {
        nivel = W.smooth(nivel, [1], 1);
        rastro = W.trail(rastro, nivel, 1);
    }
    assert.ok(rastro[0] < nivel[0], "na subida o rastro vem atrás");
    for (let i = 0; i < 4; i++) {
        nivel = W.smooth(nivel, [0], 1);
        rastro = W.trail(rastro, nivel, 1);
    }
    assert.ok(rastro[0] > nivel[0], "na queda o rastro ainda está no alto");
});

t("trail: em silêncio acaba em zero (não fica um fio parado)", () => {
    let rastro = [0.8, 0.3];
    for (let i = 0; i < 200; i++)
        rastro = W.trail(rastro, [0, 0], 2);
    igual(rastro, [0, 0]);
});

t("trail: sem rastro anterior parte do zero, e tem sempre o número de faixas pedido", () => {
    eq(W.trail([], [1, 1, 1], 3).length, 3);
    eq(W.trail([0.5], [], 2).length, 2);
});

t("curve: começa e termina em zero, na largura dada, com três pontos por faixa", () => {
    const pts = W.curve([0.2, 0.9, 0.4, 0.1], 4, 120, 10);
    eq(pts.length, 4 * 3 + 1);
    igual(pts[0], [0, 0]);
    igual(pts[pts.length - 1], [120, 0]);
    for (let i = 1; i < pts.length; i++)
        assert.ok(pts[i][0] > pts[i - 1][0], "x só cresce");
});

t("curve: nunca passa da altura dada nem fica negativa", () => {
    const pts = W.curve([1, 0, 1, 0, 1, 1, 0.5, 0.2], 8, 300, 9);
    for (const [, y] of pts)
        assert.ok(y >= 0 && y <= 9 + 1e-9, "y=" + y);
});

t("curve: o contraste é do quadro: a faixa mais forte sobe mais que a mais fraca", () => {
    const pts = W.curve([0.5, 0.5, 0.9, 0.5, 0.5], 5, 200, 10);
    const meio = pts[Math.floor(pts.length / 2)][1];
    assert.ok(meio > pts[2][1] && meio > pts[pts.length - 3][1]);
});

t("curve: sem altura, sem largura ou com as faixas incompletas é uma linha reta", () => {
    igual(W.curve([0.5, 0.5], 2, 100, 0), [[0, 0], [100, 0]]);
    igual(W.curve([0.5, 0.5], 2, 0, 10), [[0, 0], [0, 0]]);
    igual(W.curve([0.5], 2, 100, 10), [[0, 0], [100, 0]]);
    igual(W.curve([], 28, 100, 10), [[0, 0], [100, 0]]);
});
