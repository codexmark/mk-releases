.pragma library

// A lógica da janela do MkPulso que não depende do QML: escrever tamanhos,
// taxas e tempos do jeito que a pessoa lê, escolher o valor da janela de tempo
// (agora, 1 min, 3 min), procurar, ordenar e separar os grupos em seções. Os
// dados vêm do coletor (../CONTRATO.md); o que ele não soube ler chega como
// null e aqui vira um traço, nunca zero.

var energias = ["nenhuma", "muito baixa", "baixa", "moderada", "alta", "muito alta"];

function valido(v) {
    return typeof v === "number" && isFinite(v) && v >= 0;
}

// Uma casa decimal abaixo de 10, nenhuma dali em diante, com vírgula.
function num(v) {
    var s = v.toFixed(1);
    return Number(s) < 10 ? s.replace(".", ",") : String(Math.round(v));
}

// kB -> "839 MB", "5,4 GB".
function tamanho(kb) {
    if (!valido(kb))
        return "—";
    var unidades = ["KB", "MB", "GB", "TB"], i = 0, v = kb;
    while (v >= 1024 && i < unidades.length - 1) {
        v /= 1024;
        i++;
    }
    return (i === 0 ? String(Math.round(v)) : num(v)) + " " + unidades[i];
}

// bytes por segundo -> "8,0 KB/s".
function taxa(bps) {
    if (!valido(bps))
        return "—";
    var unidades = ["B/s", "KB/s", "MB/s", "GB/s"], i = 0, v = bps;
    while (v >= 1024 && i < unidades.length - 1) {
        v /= 1024;
        i++;
    }
    return (i === 0 ? String(Math.round(v)) : num(v)) + " " + unidades[i];
}

function pct(v) {
    return valido(v) ? num(v) + "%" : "—";
}

// Segundos em palavras curtas: as duas maiores unidades.
function duracao(s) {
    if (!valido(s))
        return "—";
    s = Math.floor(s);
    if (s === 0)
        return "agora";
    if (s < 60)
        return s + " s";
    var d = Math.floor(s / 86400), h = Math.floor(s % 86400 / 3600), m = Math.floor(s % 3600 / 60);
    if (d > 0)
        return d + " d" + (h > 0 ? " " + h + " h" : "");
    if (h > 0)
        return h + " h" + (m > 0 ? " " + m + " min" : "");
    return m + " min";
}

// Há quanto tempo, de um instante (segundos da época) até o agora dado.
function desde(inicio, agora) {
    if (!valido(inicio) || !valido(agora))
        return "—";
    return duracao(Math.max(0, agora - inicio));
}

// Minúsculas e sem acento.
function fold(s) {
    if (s === null || s === undefined)
        return "";
    return String(s).toLowerCase().normalize("NFD").replace(/[̀-ͯ]/g, "");
}

// O valor de uma coluna na janela de tempo: "agora", "1m" ou "3m". Só CPU,
// memória e disco têm média; o resto vale sempre o de agora.
function valor(grupo, coluna, janela) {
    if (!grupo)
        return null;
    if (janela !== "agora" && grupo.media) {
        var m = grupo.media[coluna + "_" + janela];
        if (m !== undefined && m !== null)
            return m;
    }
    var v = grupo[coluna];
    return v === undefined ? null : v;
}

// O grupo casa com o que foi digitado? Todas as palavras têm de aparecer em
// algum lugar: nome, id, unidade, título de janela, pid, nome ou comando de
// um processo.
function casa(grupo, texto) {
    var palavras = fold(texto).split(/\s+/).filter(function (w) { return w.length > 0; });
    if (palavras.length === 0)
        return true;
    var onde = [grupo.nome, grupo.id, grupo.unidade, grupo.pid];
    var i;
    for (i = 0; i < (grupo.janelas || []).length; i++)
        onde.push(grupo.janelas[i].titulo);
    for (i = 0; i < (grupo.processos || []).length; i++)
        onde.push(grupo.processos[i].pid, grupo.processos[i].nome, grupo.processos[i].cmd);
    var palheiro = fold(onde.join("\n"));
    for (i = 0; i < palavras.length; i++)
        if (palheiro.indexOf(palavras[i]) < 0)
            return false;
    return true;
}

function filtrar(grupos, texto) {
    return (grupos || []).filter(function (g) { return casa(g, texto); });
}

function nivelEnergia(e) {
    var i = energias.indexOf(e);
    return i < 0 ? 0 : i;
}

// Uma cópia ordenada pela coluna. O que não tem valor (null) vai para o fim
// nos dois sentidos; empate, pelo nome.
function ordenar(grupos, coluna, decrescente, janela) {
    var chave = function (g) {
        if (coluna === "nome")
            return fold(g.nome);
        if (coluna === "energia")
            return nivelEnergia(g.energia);
        return valor(g, coluna, janela);
    };
    return (grupos || []).slice().sort(function (a, b) {
        var x = chave(a), y = chave(b);
        var semX = x === null || x === undefined, semY = y === null || y === undefined;
        if (semX !== semY)
            return semX ? 1 : -1;
        if (!semX && x !== y) {
            var menor = x < y ? -1 : 1;
            return decrescente ? -menor : menor;
        }
        var nx = fold(a.nome), ny = fold(b.nome);
        return nx < ny ? -1 : nx > ny ? 1 : 0;
    });
}

// Os grupos em seções, na ordem em que a pessoa procura: o que ela abriu, o
// que roda por trás, o que é do sistema. Seção vazia não aparece.
function secoes(grupos) {
    var tipos = [["app", "Apps"], ["servico", "Serviços"], ["sistema", "Sistema"]], out = [];
    for (var i = 0; i < tipos.length; i++) {
        var lista = (grupos || []).filter(function (g) { return g.tipo === tipos[i][0]; });
        if (lista.length > 0)
            out.push({ tipo: tipos[i][0], titulo: tipos[i][1], grupos: lista });
    }
    return out;
}

// Quanto pintar uma célula (0 a 1) para um valor diante do teto da coluna.
function calor(v, teto) {
    if (!valido(v) || !valido(teto) || teto === 0)
        return 0;
    return Math.min(1, v / teto);
}

// A série de um gráfico com mais um ponto, guardando no máximo `max`.
function empurra(serie, v, max) {
    var s = (serie || []).concat([valido(v) ? v : 0]);
    return s.length > max ? s.slice(s.length - max) : s;
}

// A tabela de processos achatada, para uma lista só: o título de cada seção,
// os grupos dela e, nos grupos abertos, os processos. `abertos` é { chave: true }
// e `recolhidas` é { tipo: true }. Um grupo de um processo só não abre: a
// linha dele já diz tudo.
// A chave de um grupo na janela. O id sozinho não basta: a mesma unidade pode
// existir no sistema e na sessão (dbus-broker.service), e só o escopo as separa.
function chave(grupo) {
    return grupo.id + "|" + (grupo.escopo || "");
}

function linhas(secoes, abertos, recolhidas) {
    var out = [];
    for (var i = 0; i < (secoes || []).length; i++) {
        var s = secoes[i];
        out.push({ tipo: "secao", secao: s });
        if (recolhidas && recolhidas[s.tipo])
            continue;
        for (var j = 0; j < s.grupos.length; j++) {
            var g = s.grupos[j], ps = g.processos || [];
            out.push({ tipo: "grupo", grupo: g });
            if (abertos && abertos[chave(g)] && ps.length > 1)
                for (var k = 0; k < ps.length; k++)
                    out.push({ tipo: "processo", grupo: g, processo: ps[k] });
        }
    }
    return out;
}

// O maior valor de uma coluna entre os grupos (a escala do calor).
function teto(grupos, coluna, janela) {
    var m = 0;
    for (var i = 0; i < (grupos || []).length; i++) {
        var v = valor(grupos[i], coluna, janela);
        if (valido(v) && v > m)
            m = v;
    }
    return m;
}
