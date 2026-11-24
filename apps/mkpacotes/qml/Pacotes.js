.pragma library

// A lógica da janela do MkPacotes que não depende do QML: escrever tamanhos
// e datas do jeito que a pessoa lê, filtrar, procurar e ordenar os
// instalados, marcar vários, montar os argumentos de uma ação, resumir o
// plano e entender o que o `agir` transmite. Os dados vêm do coletor
// (../CONTRATO.md); o que ele não soube chega como null e aqui vira um traço,
// nunca zero.

var acoesSemPacote = ["orfaos", "cache", "atualizar"];

function valido(v) {
    return typeof v === "number" && isFinite(v) && v >= 0;
}

// Uma casa decimal abaixo de 10, nenhuma dali em diante, com vírgula.
function num(v) {
    var s = v.toFixed(1);
    return Number(s) < 10 ? s.replace(".", ",") : String(Math.round(v));
}

// bytes -> "9,6 MB", "19 GB".
function tamanho(bytes) {
    if (!valido(bytes))
        return "—";
    var unidades = ["B", "KB", "MB", "GB", "TB"], i = 0, v = bytes;
    while (v >= 1024 && i < unidades.length - 1) {
        v /= 1024;
        i++;
    }
    return (i === 0 ? String(Math.round(v)) : num(v)) + " " + unidades[i];
}

function plural(n, um, varios) {
    return n + " " + (n === 1 ? um : varios);
}

// Segundos da época -> "20/11/2026".
function data(s) {
    if (!valido(s))
        return "—";
    var d = new Date(s * 1000);
    var dois = function (n) { return (n < 10 ? "0" : "") + n; };
    return dois(d.getDate()) + "/" + dois(d.getMonth() + 1) + "/" + d.getFullYear();
}

// Há quanto tempo, em dias do calendário (a virada é à meia-noite): "hoje",
// "ontem", "há 3 dias"; passou de um mês, a data.
function quando(s, agora) {
    if (!valido(s) || !valido(agora))
        return "—";
    var meiaNoite = function (t) {
        var d = new Date(t * 1000);
        return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
    };
    var dias = Math.round((meiaNoite(agora) - meiaNoite(s)) / 86400000);
    if (dias <= 0)
        return "hoje";
    if (dias === 1)
        return "ontem";
    if (dias <= 30)
        return "há " + dias + " dias";
    return data(s);
}

// Minúsculas e sem acento.
function fold(s) {
    if (s === null || s === undefined)
        return "";
    return String(s).toLowerCase().normalize("NFD").replace(/[̀-ͯ]/g, "");
}

// O pacote casa com o que foi digitado? Todas as palavras têm de aparecer no
// nome ou na descrição.
function casa(pacote, texto) {
    var palavras = fold(texto).split(/\s+/).filter(function (w) { return w.length > 0; });
    if (palavras.length === 0)
        return true;
    var palheiro = fold(pacote.nome) + "\n" + fold(pacote.descricao);
    for (var i = 0; i < palavras.length; i++)
        if (palheiro.indexOf(palavras[i]) < 0)
            return false;
    return true;
}

// O pacote está no filtro? Filtro desconhecido deixa passar tudo.
function noFiltro(p, filtro) {
    switch (filtro) {
    case "pedidos":
        return p.motivo === "pedido";
    case "aur":
        return p.origem === "aur";
    case "orfaos":
        return p.orfao === true;
    case "atualizacoes":
        return p.atualizacao !== null && p.atualizacao !== undefined;
    }
    return true;
}

// Os instalados de um filtro (todos, pedidos, aur, orfaos, atualizacoes) que
// casam com a busca.
function filtrar(pacotes, filtro, texto) {
    return (pacotes || []).filter(function (p) { return noFiltro(p, filtro) && casa(p, texto); });
}

function contagens(pacotes) {
    var c = { todos: 0, pedidos: 0, aur: 0, orfaos: 0, atualizacoes: 0 };
    (pacotes || []).forEach(function (p) {
        for (var f in c)
            if (noFiltro(p, f))
                c[f]++;
    });
    return c;
}

// Uma cópia ordenada pela coluna (nome, tamanho_bytes, instalado_em_s). O que
// não tem valor (null) vai para o fim nos dois sentidos; empate, pelo nome.
function ordenar(pacotes, coluna, decrescente) {
    return (pacotes || []).slice().sort(function (a, b) {
        var x = coluna === "nome" ? a.nome : a[coluna], y = coluna === "nome" ? b.nome : b[coluna];
        var semX = x === null || x === undefined, semY = y === null || y === undefined;
        if (semX !== semY)
            return semX ? 1 : -1;
        if (!semX && x !== y) {
            var menor = x < y ? -1 : 1;
            return decrescente ? -menor : menor;
        }
        return a.nome < b.nome ? -1 : a.nome > b.nome ? 1 : 0;
    });
}

function origem(o) {
    if (o === null || o === undefined || o === "")
        return "—";
    return o === "aur" ? "AUR" : o;
}

function motivo(m) {
    return m === "pedido" ? "pedido por você" : m === "dependencia" ? "dependência" : "—";
}

// A regra do coletor para nome de pacote: o que não passa nem é mandado.
function nomeValido(nome) {
    return typeof nome === "string" && /^[a-z0-9@_+][a-z0-9@._+-]*$/.test(nome);
}

// As marcas da tabela ({ nome: true }): devolve um mapa novo com o nome
// marcado ou desmarcado.
function alterna(marcas, nome) {
    var novo = {};
    for (var k in (marcas || {}))
        if (k !== nome)
            novo[k] = true;
    if (!(marcas || {})[nome])
        novo[nome] = true;
    return novo;
}

function marcados(marcas) {
    return Object.keys(marcas || {}).sort();
}

// Os argumentos de `plano` e de `agir` depois do nome do subcomando: a ação,
// --mesmo-assim se pedido e, depois de --, só o que é nome de pacote.
function argsDeAcao(acao, nomes, mesmoAssim) {
    var args = [acao];
    if (mesmoAssim)
        args.push("--mesmo-assim");
    var bons = acoesSemPacote.indexOf(acao) >= 0 ? [] : (nomes || []).filter(nomeValido);
    return bons.length > 0 ? args.concat(["--"], bons) : args;
}

function tituloDoPlano(plano) {
    if (!plano)
        return "";
    var n = (plano.pedidos || []).length;
    switch (plano.acao) {
    case "instalar":
        return n === 1 ? "Instalar " + plano.pedidos[0] + "?" : "Instalar " + n + " pacotes?";
    case "remover":
        return n === 1 ? "Remover " + plano.pedidos[0] + "?" : "Remover " + n + " pacotes?";
    case "orfaos":
        return n === 1 ? "Remover o órfão " + plano.pedidos[0] + "?" : "Remover os " + n + " órfãos?";
    case "cache":
        return "Limpar o cache de pacotes?";
    case "atualizar":
        return "Atualizar tudo?";
    }
    return "";
}

// Uma frase com o que o plano faz e quanto pesa.
function resumoDoPlano(plano) {
    if (!plano)
        return "";
    if (plano.nada_a_fazer)
        return "Nada a fazer.";
    if (plano.acao === "atualizar")
        return "Abre o mk-update num terminal: ele atualiza o Mk, o Arch e o AUR.";
    if (plano.acao === "cache") {
        var c = plano.cache || { arquivos: 0, sobram: 0 };
        return "Saem do cache " + plural(c.arquivos - c.sobram, "arquivo", "arquivos")
            + " de pacotes que não estão mais instalados. Libera " + tamanho(plano.liberar_bytes) + "; ficam " + c.sobram + ".";
    }
    var lista = plano.acao === "instalar" ? (plano.entra || []) : (plano.sai || []);
    var carona = lista.filter(function (x) { return !x.pedido; }).length;
    var frase;
    if (plano.acao === "instalar") {
        frase = (lista.length === 1 ? "Entra " : "Entram ") + plural(lista.length, "pacote", "pacotes");
        if (carona > 0)
            frase += ": o que você pediu e " + plural(carona, "dependência", "dependências");
        frase += ".";
        if (valido(plano.baixar_bytes))
            frase += " Baixa " + tamanho(plano.baixar_bytes) + ".";
        return frase;
    }
    frase = (lista.length === 1 ? "Sai " : "Saem ") + plural(lista.length, "pacote", "pacotes");
    if (carona > 0)
        frase += ": o que você pediu e " + plural(carona, "dependência que ficaria", "dependências que ficariam") + " sem uso";
    frase += ".";
    if (valido(plano.liberar_bytes))
        frase += " Libera " + tamanho(plano.liberar_bytes) + ".";
    return frase;
}

// Dá para confirmar? Com o pacman travado o coletor recusaria (menos a
// atualização: o mk-update sabe lidar com a trava).
function podeConfirmar(plano) {
    if (!plano || plano.nada_a_fazer)
        return false;
    return !plano.travado || plano.acao === "atualizar";
}

// Uma linha do que o `agir` transmite: { tipo: "linha" | "ok" | "erro" | "nada" }.
function interpreta(linha) {
    var r;
    try {
        r = JSON.parse(linha);
    } catch (e) {
        return { tipo: "nada" };
    }
    if (!r || typeof r !== "object")
        return { tipo: "nada" };
    if (typeof r.linha === "string")
        return { tipo: "linha", texto: r.linha };
    if (typeof r.erro === "string")
        return { tipo: "erro", texto: r.erro };
    if (r.ok === true)
        return { tipo: "ok", acao: r.acao, pacotes: r.pacotes || [] };
    return { tipo: "nada" };
}

// O registro com mais uma linha, guardando no máximo `max`.
function empurra(linhas, linha, max) {
    var l = (linhas || []).concat([linha]);
    return l.length > max ? l.slice(l.length - max) : l;
}

// As versões de um evento do histórico: "de → para", ou a única que há.
function versoes(evento) {
    if (!evento)
        return "";
    if (evento.de && evento.para && evento.de !== evento.para)
        return evento.de + " → " + evento.para;
    return evento.para || evento.de || "";
}

// Um comando do plano numa linha, com aspas no que tem espaço.
function linhaDeComando(args) {
    return (args || []).map(function (a) {
        return /^[A-Za-z0-9@%+=:,.\/_-]+$/.test(a) ? a : "'" + String(a).replace(/'/g, "'\\''") + "'";
    }).join(" ");
}
