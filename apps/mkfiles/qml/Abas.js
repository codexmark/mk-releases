.pragma library
.import "Arquivos.js" as Arquivos

// O estado das abas e dos painéis: cada aba tem a pasta em que está e o
// histórico de por onde passou; cada painel, as suas abas e qual está à vista.
// Toda função devolve um valor novo.

// O "caminho" da lixeira, que é um lugar e não uma pasta do disco.
var LIXEIRA = "lixeira:";
var MAXIMO_DE_ABAS = 30;

var proximoId = 0;

function eLixeira(caminho) {
    return caminho === LIXEIRA;
}

function nova(caminho) {
    proximoId += 1;
    return { id: "aba" + proximoId, caminho: caminho, tras: [], frente: [] };
}

// Vai para outra pasta: a de agora entra no histórico e o "avançar" some.
function ir(aba, caminho) {
    if (caminho === aba.caminho)
        return aba;
    return { id: aba.id, caminho: caminho, tras: aba.tras.concat([aba.caminho]).slice(-100), frente: [] };
}

function podeVoltar(aba) { return aba.tras.length > 0; }
function podeAvancar(aba) { return aba.frente.length > 0; }
function podeSubir(aba) { return !eLixeira(aba.caminho) && aba.caminho !== "/"; }
// Dentro de um compactado ("arquivo//interno") a aba não é uma pasta do disco.
function eCompactado(caminho) { return Arquivos.partesDoCompactado(caminho) !== null; }

function voltar(aba) {
    if (!podeVoltar(aba))
        return aba;
    return { id: aba.id, caminho: aba.tras[aba.tras.length - 1], tras: aba.tras.slice(0, -1), frente: [aba.caminho].concat(aba.frente) };
}

function avancar(aba) {
    if (!podeAvancar(aba))
        return aba;
    return { id: aba.id, caminho: aba.frente[0], tras: aba.tras.concat([aba.caminho]), frente: aba.frente.slice(1) };
}

function subir(aba) {
    return podeSubir(aba) ? ir(aba, Arquivos.pai(aba.caminho)) : aba;
}

// O nome que a aba mostra.
function titulo(aba, home) {
    if (eLixeira(aba.caminho))
        return "Lixeira";
    if (aba.caminho === home)
        return "Pasta pessoal";
    return Arquivos.nomeDe(aba.caminho);
}

// --- o painel: { abas, atual } ------------------------------------------------

function painel(caminhos, atual) {
    var abas = (caminhos || []).slice(0, MAXIMO_DE_ABAS).map(nova);
    return { abas: abas, atual: Math.max(0, Math.min(abas.length - 1, atual || 0)) };
}

function abaAtual(p) {
    return p.abas[p.atual];
}

// Abre uma aba logo depois da atual e vai para ela.
function abrir(p, caminho) {
    if (p.abas.length >= MAXIMO_DE_ABAS)
        return p;
    var abas = p.abas.slice();
    abas.splice(p.atual + 1, 0, nova(caminho));
    return { abas: abas, atual: p.atual + 1 };
}

function duplicar(p, i) {
    if (i < 0 || i >= p.abas.length || p.abas.length >= MAXIMO_DE_ABAS)
        return p;
    var abas = p.abas.slice();
    abas.splice(i + 1, 0, nova(p.abas[i].caminho));
    return { abas: abas, atual: i + 1 };
}

// Fecha uma aba. A última não fecha (quem decide fechar a janela é a janela).
function fechar(p, i) {
    if (p.abas.length <= 1 || i < 0 || i >= p.abas.length)
        return p;
    var abas = p.abas.slice();
    abas.splice(i, 1);
    var atual = p.atual > i || p.atual >= abas.length ? p.atual - 1 : p.atual;
    return { abas: abas, atual: Math.max(0, atual) };
}

function escolher(p, i) {
    return i >= 0 && i < p.abas.length ? { abas: p.abas, atual: i } : p;
}

// Ctrl+Tab: a próxima (ou a anterior), dando a volta.
function trocar(p, delta) {
    var n = p.abas.length;
    return { abas: p.abas, atual: ((p.atual + delta) % n + n) % n };
}

// Arrastar uma aba para outra posição; a que está à vista continua à vista.
function mover(p, de, para) {
    if (de < 0 || de >= p.abas.length || para < 0 || para >= p.abas.length || de === para)
        return p;
    var abas = p.abas.slice();
    var vista = p.abas[p.atual].id;
    var aba = abas.splice(de, 1)[0];
    abas.splice(para, 0, aba);
    var atual = 0;
    for (var i = 0; i < abas.length; i++)
        if (abas[i].id === vista)
            atual = i;
    return { abas: abas, atual: atual };
}

// Troca a aba atual por outra versão dela (depois de ir, voltar, subir…).
function comAba(p, aba) {
    var abas = p.abas.slice();
    abas[p.atual] = aba;
    return { abas: abas, atual: p.atual };
}

// --- guardar e retomar ----------------------------------------------------------

// O "estado" do contrato, para gravar ao fechar.
function paraEstado(paineis, ativo, duplo, lateral) {
    return {
        paineis: paineis.map(function (p) {
            // Uma aba dentro de um compactado volta na pasta do arquivo.
            return { abas: p.abas.map(function (a) { return Arquivos.pastaReal(a.caminho); }), atual: p.atual };
        }),
        ativo: ativo === 1 ? 1 : 0,
        duplo: !!duplo,
        lateral_px: Math.max(120, Math.min(480, Math.round(lateral || 200)))
    };
}

function caminhoValido(c) {
    return typeof c === "string" && (c === LIXEIRA || (c.charAt(0) === "/" && c.indexOf("\u0000") < 0 && !eCompactado(c)));
}

// Os dois painéis da janela, a partir do estado gravado. O que não serve vira
// a pasta pessoal: nunca volta um painel sem aba.
function doEstado(estado, home) {
    var paineis = [];
    var gravados = estado && estado.paineis instanceof Array ? estado.paineis : [];
    for (var i = 0; i < 2; i++) {
        var g = gravados[i];
        var caminhos = g && g.abas instanceof Array ? g.abas.filter(caminhoValido) : [];
        if (caminhos.length === 0)
            caminhos = [home];
        paineis.push(painel(caminhos, g ? g.atual : 0));
    }
    return paineis;
}

// O que a janela abre: com pedidos (`mkfiles pasta arquivo…`, já vistos pelo
// motor: [{ caminho, pasta }]) cada pasta vira uma aba do primeiro painel e
// o arquivo abre a pasta dele, selecionado; sem pedidos, o estado gravado.
// Devolve { paineis, selecionar: { caminhoDaPasta: [caminhos] } }.
function inicial(pedidos, estado, home) {
    var paineis = doEstado(estado, home);
    var selecionar = {};
    var pastas = [];
    for (var i = 0; i < (pedidos || []).length; i++) {
        var p = pedidos[i];
        var pasta = p.pasta ? p.caminho : Arquivos.pai(p.caminho);
        if (pastas.indexOf(pasta) < 0)
            pastas.push(pasta);
        if (!p.pasta)
            selecionar[pasta] = (selecionar[pasta] || []).concat([p.caminho]);
    }
    if (pastas.length > 0)
        paineis[0] = painel(pastas, 0);
    return { paineis: paineis, selecionar: selecionar };
}

// As linhas de MKFILES_ABRIR: um caminho absoluto por linha.
function pedidosDe(texto) {
    return String(texto || "").split("\n").filter(function (l) {
        return l.charAt(0) === "/" && l.indexOf("\u0000") < 0;
    });
}
