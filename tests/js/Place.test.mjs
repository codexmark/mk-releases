// shell/mare/Place.js: o lugar que o fundo do Maré escreve na folha (o
// carimbo e as coordenadas junto ao horizonte), a partir do que o clima achou.
import { t, eq, igual } from "./harness.mjs";
import { load } from "./load.mjs";

const P = load("Place.js");

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["coords", "part", "label"];
    const funcoes = Object.keys(P).filter((k) => typeof P[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("coords: graus e minutos, com o hemisfério em português (s/n, o/l)", () => {
    eq(P.coords(-8.76194, -63.90389), "8°46′ s · 63°54′ o");
    eq(P.coords(-19.92083, -43.93778), "19°55′ s · 43°56′ o");
    eq(P.coords(48.85341, 2.3488), "48°51′ n · 2°21′ l");
    eq(P.coords(35.6895, 139.69171), "35°41′ n · 139°42′ l");
});

t("part: uma coordenada só, com a letra do lado dela", () => {
    eq(P.part(-8.5, "n", "s"), "8°30′ s");
    eq(P.part(8.5, "n", "s"), "8°30′ n");
    eq(P.part(0.004, "l", "o"), "0°00′ l");
});

t("coords: o minuto que arredonda para 60 vira o grau seguinte", () => {
    eq(P.coords(-8.9999, -63.9999), "9°00′ s · 64°00′ o");
    eq(P.coords(10.5, 20.25), "10°30′ n · 20°15′ l");
});

t("coords: o equador e o meridiano de Greenwich ficam no norte e no leste", () => {
    eq(P.coords(0, 0), "0°00′ n · 0°00′ l");
});

t("coords: sem número de verdade, ou fora do mapa, não escreve nada", () => {
    for (const [a, b] of [[null, 1], [1, undefined], ["x", 2], [NaN, 3], [91, 0], [0, 181], [-90.5, 0], [Infinity, 0], ["", ""], [{}, []]])
        eq(P.coords(a, b), "", JSON.stringify([a, b]));
    // Texto que é número vale (o JSON pode trazer assim).
    eq(P.coords("-8.5", "-63.5"), "8°30′ s · 63°30′ o");
});

t("label: a cidade em maiúsculas e o país, como no carimbo", () => {
    eq(P.label("Porto Velho", "BR"), "PORTO VELHO · BR");
    eq(P.label("São João del-Rei", "br"), "SÃO JOÃO DEL-REI · BR");
    eq(P.label("Paris", ""), "PARIS");
    eq(P.label("  Lisboa  ", " pt "), "LISBOA · PT");
});

t("label: sem cidade não há lugar (o carimbo mostra um traço)", () => {
    for (const c of ["", "   ", null, undefined, 42, {}])
        eq(P.label(c, "BR"), "", String(c));
});

t("label: um nome comprido demais é cortado para caber no carimbo", () => {
    const longo = P.label("Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch", "GB");
    eq(longo.length <= 34, true, longo);
    eq(longo.endsWith("… · GB"), true, longo);
});
