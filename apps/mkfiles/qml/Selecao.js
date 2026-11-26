.pragma library
.import "Arquivos.js" as Arquivos

// A seleção de um painel: quais caminhos estão marcados, onde está o cursor do
// teclado e de onde parte o intervalo do Shift. Toda função devolve um estado
// novo: { sel: { caminho: true }, n, ancora, cursor, onde }.

function nova() {
    return { sel: {}, n: 0, ancora: -1, cursor: -1, onde: "" };
}

function feita(sel, n, ancora, cursor, itens) {
    return { sel: sel, n: n, ancora: ancora, cursor: cursor, onde: cursor >= 0 && cursor < itens.length ? itens[cursor].caminho : "" };
}

function copia(sel) {
    var c = {};
    for (var k in sel)
        c[k] = true;
    return c;
}

function tem(est, item) {
    return !!item && est.sel[item.caminho] === true;
}

// Só este item.
function so(itens, i) {
    if (i < 0 || i >= itens.length)
        return nova();
    var sel = {};
    sel[itens[i].caminho] = true;
    return feita(sel, 1, i, i, itens);
}

function comIntervalo(sel, n, itens, a, b) {
    var de = Math.max(0, Math.min(a, b)), ate = Math.min(itens.length - 1, Math.max(a, b));
    for (var i = de; i <= ate; i++)
        if (sel[itens[i].caminho] !== true) {
            sel[itens[i].caminho] = true;
            n++;
        }
    return n;
}

// Um clique: simples troca a seleção; Ctrl liga e desliga o item; Shift
// estende da âncora até ele (com Ctrl, somando ao que havia).
function clicar(est, itens, i, mods) {
    if (i < 0 || i >= itens.length)
        return mods && (mods.ctrl || mods.shift) ? est : nova();
    if (mods && mods.shift) {
        var ancora = est.ancora >= 0 && est.ancora < itens.length ? est.ancora : 0;
        var sel = mods.ctrl ? copia(est.sel) : {};
        var n = comIntervalo(sel, mods.ctrl ? est.n : 0, itens, ancora, i);
        return feita(sel, n, ancora, i, itens);
    }
    if (mods && mods.ctrl)
        return alternar(est, itens, i);
    return so(itens, i);
}

// Liga ou desliga um item, sem mexer nos outros.
function alternar(est, itens, i) {
    if (i < 0 || i >= itens.length)
        return est;
    var sel = copia(est.sel);
    var n = est.n;
    var c = itens[i].caminho;
    if (sel[c] === true) {
        delete sel[c];
        n--;
    } else {
        sel[c] = true;
        n++;
    }
    return feita(sel, n, i, i, itens);
}

// O cursor do teclado foi para i: sem modificador seleciona só ele; com Shift
// estende; com Ctrl só anda.
function andar(est, itens, i, mods) {
    if (i < 0 || i >= itens.length)
        return est;
    if (mods && mods.shift)
        return clicar(est, itens, i, { shift: true });
    if (mods && mods.ctrl)
        return feita(est.sel, est.n, est.ancora, i, itens);
    return so(itens, i);
}

function todos(itens) {
    var sel = {};
    for (var i = 0; i < itens.length; i++)
        sel[itens[i].caminho] = true;
    return feita(sel, itens.length, 0, itens.length > 0 ? itens.length - 1 : -1, itens);
}

function inverter(est, itens) {
    var sel = {};
    var n = 0;
    for (var i = 0; i < itens.length; i++)
        if (est.sel[itens[i].caminho] !== true) {
            sel[itens[i].caminho] = true;
            n++;
        }
    return feita(sel, n, est.ancora, est.cursor, itens);
}

// Soma à seleção os itens cujo nome casa com o padrão ("*.jpg").
function porPadrao(est, itens, padrao) {
    var sel = copia(est.sel);
    var n = est.n;
    var primeiro = -1;
    for (var i = 0; i < itens.length; i++)
        if (Arquivos.casaPadrao(itens[i].nome, padrao)) {
            if (primeiro < 0)
                primeiro = i;
            if (sel[itens[i].caminho] !== true) {
                sel[itens[i].caminho] = true;
                n++;
            }
        }
    return feita(sel, n, primeiro >= 0 ? primeiro : est.ancora, primeiro >= 0 ? primeiro : est.cursor, itens);
}

// A seleção de `base` mais os itens destes índices (o retângulo do mouse).
function comIndices(base, itens, indices) {
    var sel = copia(base.sel);
    var n = base.n;
    var ultimo = base.cursor;
    for (var k = 0; k < indices.length; k++) {
        var i = indices[k];
        if (i < 0 || i >= itens.length)
            continue;
        ultimo = i;
        if (sel[itens[i].caminho] !== true) {
            sel[itens[i].caminho] = true;
            n++;
        }
    }
    return feita(sel, n, base.ancora, ultimo, itens);
}

// Seleciona os itens com estes caminhos (o que acabou de ser criado ou colado).
function porCaminhos(itens, caminhos) {
    var quero = {};
    for (var k = 0; k < caminhos.length; k++)
        quero[caminhos[k]] = true;
    var sel = {};
    var n = 0, primeiro = -1;
    for (var i = 0; i < itens.length; i++)
        if (quero[itens[i].caminho] === true) {
            sel[itens[i].caminho] = true;
            n++;
            if (primeiro < 0)
                primeiro = i;
        }
    return feita(sel, n, primeiro, primeiro, itens);
}

// Os itens selecionados, na ordem em que aparecem.
function lista(est, itens) {
    if (est.n === 0)
        return [];
    var saida = [];
    for (var i = 0; i < itens.length && saida.length < est.n; i++)
        if (est.sel[itens[i].caminho] === true)
            saida.push(itens[i]);
    return saida;
}

function caminhos(est, itens) {
    return lista(est, itens).map(function (it) { return it.caminho; });
}

// A listagem mudou (ordem nova, itens que saíram): fica selecionado só o que
// ainda existe, e o cursor volta para o item em que estava.
function reancorar(est, itens) {
    var sel = {};
    var n = 0, cursor = -1;
    for (var i = 0; i < itens.length; i++) {
        var c = itens[i].caminho;
        if (est.sel[c] === true) {
            sel[c] = true;
            n++;
        }
        if (c === est.onde)
            cursor = i;
    }
    if (cursor < 0 && est.cursor >= 0)
        cursor = Math.min(est.cursor, itens.length - 1);
    return feita(sel, n, cursor, cursor, itens);
}

// Para onde o cursor vai com uma tecla de navegação. colunas = 1 nas listas.
function passo(tecla, cursor, n, colunas, porPagina) {
    if (n <= 0)
        return -1;
    var c = Math.max(1, colunas || 1);
    var pagina = Math.max(1, porPagina || 10) * c;
    if (cursor < 0)
        return tecla === "fim" ? n - 1 : 0;
    var i = cursor;
    switch (tecla) {
    case "esquerda": i = cursor - 1; break;
    case "direita": i = cursor + 1; break;
    case "cima": i = cursor - c >= 0 ? cursor - c : cursor; break;
    case "baixo": i = cursor + c < n ? cursor + c : (Math.floor(cursor / c) < Math.floor((n - 1) / c) ? n - 1 : cursor); break;
    case "inicio": i = 0; break;
    case "fim": i = n - 1; break;
    case "pagina-cima": i = cursor - pagina; break;
    case "pagina-baixo": i = cursor + pagina; break;
    }
    return Math.max(0, Math.min(n - 1, i));
}

// Os índices que um retângulo (em coordenadas do conteúdo) toca. g descreve a
// visão: { n, colunas, larguraDaCelula, alturaDaCelula } (nas listas, colunas
// = 1 e a largura é a da linha).
function noRetangulo(g, r) {
    var indices = [];
    if (g.n <= 0 || g.alturaDaCelula <= 0 || g.larguraDaCelula <= 0)
        return indices;
    var x0 = Math.min(r.x, r.x + r.l), x1 = Math.max(r.x, r.x + r.l);
    var y0 = Math.min(r.y, r.y + r.a), y1 = Math.max(r.y, r.y + r.a);
    var colunas = Math.max(1, g.colunas);
    var l0 = Math.max(0, Math.floor(y0 / g.alturaDaCelula));
    var l1 = Math.min(Math.ceil(g.n / colunas) - 1, Math.floor(y1 / g.alturaDaCelula));
    var c0 = Math.max(0, Math.floor(x0 / g.larguraDaCelula));
    var c1 = Math.min(colunas - 1, Math.floor(x1 / g.larguraDaCelula));
    if (x1 < 0 || y1 < 0)
        return indices;
    for (var l = l0; l <= l1; l++)
        for (var c = c0; c <= c1; c++) {
            var i = l * colunas + c;
            if (i < g.n)
                indices.push(i);
        }
    return indices;
}

// Para onde o cursor vai com uma tecla de navegação, numa visão. g descreve a
// visão de agora: { visao: "icones"|"lista"|"compacta", colunas (da grade),
// linhas (da compacta, que enche de cima para baixo), porPagina (linhas, ou
// colunas na compacta, que cabem na tela) }. Sem cursor (-1), a primeira tecla
// põe o cursor no primeiro item (o End, no último); a seguinte já anda.
// Devolve -1 só quando não há itens.
function destino(tecla, cursor, n, g) {
    var visao = g && g.visao || "icones";
    if (visao === "lista")
        return passo(tecla, cursor, n, 1, g && g.porPagina);
    if (visao === "compacta") {
        // As setas trocam de papel: para baixo é o vizinho, para o lado é a
        // coluna seguinte.
        var troca = { cima: "esquerda", baixo: "direita", esquerda: "cima", direita: "baixo" };
        return passo(troca[tecla] || tecla, cursor, n, g && g.linhas, g && g.porPagina);
    }
    // Numa grade de uma linha só não há "embaixo": para a tecla não morrer,
    // baixo e cima andam para o vizinho.
    var colunas = Math.max(1, g && g.colunas || 1);
    if (n <= colunas && (tecla === "baixo" || tecla === "cima"))
        tecla = tecla === "baixo" ? "direita" : "esquerda";
    return passo(tecla, cursor, n, g && g.colunas, g && g.porPagina);
}

// A tecla de navegação aplicada à seleção: sozinha, seleciona só o destino;
// com Shift, estende da âncora até ele; com Ctrl, só leva o cursor. O estado
// que volta é o mesmo objeto quando não há para onde ir.
function mover(est, itens, tecla, mods, g) {
    var i = destino(tecla, est.cursor, itens.length, g);
    if (i < 0)
        return est;
    return andar(est, itens, i, mods);
}
