.pragma library

// Os textos curtos das leituras da ilha, de largura previsível. A ilha reserva
// o lugar do mais largo de cada um (`reserve`, usado pelo IslandItem), então
// não estica nem encolhe quando um número ganha ou perde um dígito: só muda de
// tamanho quando algo entra ou sai dela. Testado no node
// (tests/js/Island.test.mjs).

// O texto mais largo que cada leitura mostra.
var reserve = { rate: "999K/s", percent: "100%", degrees: "99°" };

// Bytes por segundo com três algarismos no máximo: "999B/s", "2,4K/s",
// "803K/s", "16M/s". Perto da virada (1000 a 1023) já passa para a unidade de
// cima, para nunca haver quatro algarismos.
function rate(bps) {
    var v = Number(bps);
    if (!(v > 0))
        v = 0;
    var units = ["B", "K", "M", "G", "T"];
    var u = 0;
    while (u < units.length - 1 && Math.round(v) >= 1000) {
        v /= 1024;
        u++;
    }
    var text;
    if (u === 0)
        text = String(Math.round(v));
    else if (v < 9.95)
        text = Math.max(1, v).toFixed(1).replace(".", ",");
    else
        text = String(Math.round(v));
    return text + units[u] + "/s";
}

// Uma fração (0 a 1) como "31%".
function percent(fraction) {
    var v = Number(fraction);
    if (!(v > 0))
        v = 0;
    return Math.round(Math.min(1, v) * 100) + "%";
}

function degrees(celsius) {
    return Math.round(Number(celsius) || 0) + "°";
}
