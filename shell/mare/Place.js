.pragma library

// O lugar que o fundo do Maré escreve na folha: a cidade no carimbo e as
// coordenadas junto ao horizonte. Vêm do que o clima achou (Weather.data, do
// scripts/mare-weather): a cidade de ~/.config/mare/city ou do fuso do
// sistema. Sem clima (sem rede, cidade desconhecida), nada é escrito.

// "8°46′ s · 63°54′ o": graus e minutos, hemisfério em português. "" se os
// números não forem uma posição no mapa.
function coords(lat, lon) {
    if (lat === null || lon === null || lat === "" || lon === "" || typeof lat === "object" || typeof lon === "object")
        return "";
    var a = Number(lat), b = Number(lon);
    if (!isFinite(a) || !isFinite(b) || Math.abs(a) > 90 || Math.abs(b) > 180)
        return "";
    return part(a, "n", "s") + " · " + part(b, "l", "o");
}

function part(v, pos, neg) {
    var total = Math.round(Math.abs(v) * 60);
    var deg = Math.floor(total / 60), min = total % 60;
    return deg + "°" + (min < 10 ? "0" : "") + min + "′ " + (v < 0 ? neg : pos);
}

// "PORTO VELHO · BR": a cidade em maiúsculas e o país. "" sem cidade.
function label(city, country) {
    if (typeof city !== "string" || city.trim() === "")
        return "";
    var name = city.trim().toUpperCase();
    var cc = typeof country === "string" ? country.trim().toUpperCase() : "";
    var room = 34 - (cc !== "" ? cc.length + 3 : 0);
    if (name.length > room)
        name = name.slice(0, room - 1) + "…";
    return cc !== "" ? name + " · " + cc : name;
}
