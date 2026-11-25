.pragma library

// A lógica da janela do MkSom que não depende do QML: os passos e os limites
// dos controles, a curva resultante, o aviso de distorção, o ritmo das
// aplicações ao arrastar, as predefinições, o medidor e o diagnóstico do
// microfone, os textos e os atalhos. Testado no node
// (tests/apps/mksom-Som.test.mjs). O contrato com o motor está em
// ../CONTRATO.md.

// --- o equalizador ---------------------------------------------------------------

var BANDAS = [31, 62, 125, 250, 500, 1000, 2000, 4000, 8000, 16000];
var GANHO_MIN = -12;
var GANHO_MAX = 12;
var PRE_MIN = -12;
var PRE_MAX = 0;
var PASSO = 0.5;
var PASSO_GRANDE = 3;
var Q = 1;
// A taxa em que a curva é calculada (a do PipeWire costuma ser esta).
var TAXA = 48000;
// A curva vai de 20 Hz a 20 kHz e mostra de −ESCALA a +ESCALA dB.
var CURVA_DE = 20;
var CURVA_ATE = 20000;
var ESCALA = 15;

function prender(v, min, max) {
    v = Number(v);
    if (!isFinite(v))
        return min > 0 ? min : (max < 0 ? max : 0);
    return Math.max(min, Math.min(max, v));
}

// O valor no passo de meio dB mais próximo (sem −0).
function noPasso(v) {
    const r = Math.round(Number(v) / PASSO) * PASSO;
    return isFinite(r) && r !== 0 ? r : 0;
}

// Um ganho de banda válido: no passo e entre −12 e +12.
function ganho(v) {
    return noPasso(prender(v, GANHO_MIN, GANHO_MAX));
}

// Um pré-ganho válido: no passo e entre −12 e 0.
function pre(v) {
    return noPasso(prender(v, PRE_MIN, PRE_MAX));
}

// O que uma tecla faz com o ganho de uma banda (ou com o pré-ganho, se
// `ehPre`): "cima"/"baixo" meio dB, "pagina_cima"/"pagina_baixo" 3 dB,
// "zero" zera.
function mover(valor, tecla, ehPre) {
    const passos = { cima: PASSO, baixo: -PASSO, pagina_cima: PASSO_GRANDE, pagina_baixo: -PASSO_GRANDE };
    const novo = tecla === "zero" ? 0 : Number(valor) + (passos[tecla] || 0);
    return ehPre ? pre(novo) : ganho(novo);
}

// De uma posição no trilho (0 em cima, `tamanho` embaixo) para o valor, e de
// volta. O trilho vai de `max` (em cima) a `min` (embaixo).
function daPosicao(pos, tamanho, min, max) {
    if (!(tamanho > 0))
        return noPasso(prender(0, min, max));
    const fr = prender(pos / tamanho, 0, 1);
    return noPasso(prender(max - fr * (max - min), min, max));
}

function paraPosicao(valor, tamanho, min, max) {
    return (max - prender(valor, min, max)) / (max - min) * tamanho;
}

// Para os controles deitados (o pré-ganho, o volume): da posição no trilho
// (0 à esquerda) para o valor, no passo dado.
function noTrilho(pos, tamanho, min, max, passo) {
    if (!(tamanho > 0))
        return min;
    const fr = prender(pos / tamanho, 0, 1);
    return andar(min + fr * (max - min), 0, min, max, passo);
}

// O valor mais `delta`, no passo e dentro dos limites (sem −0).
function andar(valor, delta, min, max, passo) {
    const v = prender(Math.round((Number(valor) + delta) / passo) * passo, min, max);
    return v === 0 ? 0 : v;
}

// Os ganhos com a banda `i` trocada (uma lista nova: o QML vê a mudança).
function trocar(ganhos, i, v) {
    const novo = ganhos.slice();
    if (i >= 0 && i < novo.length)
        novo[i] = ganho(v);
    return novo;
}

function plano() {
    return BANDAS.map(() => 0);
}

function mesmos(a, b) {
    if (!Array.isArray(a) || !Array.isArray(b) || a.length !== b.length)
        return false;
    for (let i = 0; i < a.length; i++)
        if (Number(a[i]) !== Number(b[i]))
            return false;
    return true;
}

// O rótulo de uma frequência: "31 Hz", "1 kHz"; curto: "31", "1k".
function rotulo(hz, curto) {
    hz = Number(hz);
    if (hz >= 1000) {
        const k = String(Math.round(hz / 100) / 10).replace(".", ",");
        return curto ? k + "k" : k + " kHz";
    }
    return curto ? String(hz) : hz + " Hz";
}

// Um número de dB como se lê: "+3,5", "0", "−2" (com o sinal de menos de verdade).
function textoDoGanho(db) {
    db = Math.round(Number(db) * 10) / 10;
    if (!isFinite(db) || db === 0)
        return "0";
    const s = String(Math.abs(db)).replace(".", ",");
    return (db > 0 ? "+" : "−") + s;
}

function textoDb(db) {
    return textoDoGanho(db) + " dB";
}

// O filtro de cada banda, como no motor: prateleira nas pontas, sino no meio.
function tipoDaBanda(i) {
    if (i === 0)
        return "lowshelf";
    if (i === BANDAS.length - 1)
        return "highshelf";
    return "peaking";
}

// A resposta, em dB, de um filtro biquad (as fórmulas do Audio EQ Cookbook,
// com Q para o sino e inclinação 1 para as prateleiras) na frequência f.
function resposta(tipo, f0, g, f) {
    g = Number(g);
    if (!g)
        return 0;
    const A = Math.pow(10, g / 40);
    const w0 = 2 * Math.PI * f0 / TAXA;
    const c = Math.cos(w0);
    const s = Math.sin(w0);
    let b0, b1, b2, a0, a1, a2;
    if (tipo === "peaking") {
        const alfa = s / (2 * Q);
        b0 = 1 + alfa * A; b1 = -2 * c; b2 = 1 - alfa * A;
        a0 = 1 + alfa / A; a1 = -2 * c; a2 = 1 - alfa / A;
    } else {
        const k = 2 * Math.sqrt(A) * s / Math.SQRT2;
        if (tipo === "lowshelf") {
            b0 = A * ((A + 1) - (A - 1) * c + k); b1 = 2 * A * ((A - 1) - (A + 1) * c); b2 = A * ((A + 1) - (A - 1) * c - k);
            a0 = (A + 1) + (A - 1) * c + k; a1 = -2 * ((A - 1) + (A + 1) * c); a2 = (A + 1) + (A - 1) * c - k;
        } else {
            b0 = A * ((A + 1) + (A - 1) * c + k); b1 = -2 * A * ((A - 1) + (A + 1) * c); b2 = A * ((A + 1) + (A - 1) * c - k);
            a0 = (A + 1) - (A - 1) * c + k; a1 = 2 * ((A - 1) - (A + 1) * c); a2 = (A + 1) - (A - 1) * c - k;
        }
    }
    const w = 2 * Math.PI * f / TAXA;
    const c1 = Math.cos(w);
    const c2 = Math.cos(2 * w);
    const num = b0 * b0 + b1 * b1 + b2 * b2 + 2 * (b0 * b1 + b1 * b2) * c1 + 2 * b0 * b2 * c2;
    const den = a0 * a0 + a1 * a1 + a2 * a2 + 2 * (a0 * a1 + a1 * a2) * c1 + 2 * a0 * a2 * c2;
    if (!(num > 0) || !(den > 0))
        return 0;
    return 10 * Math.log10(num / den);
}

// A soma, em dB, de todas as bandas na frequência f (sem o pré-ganho).
function somaEm(ganhos, f) {
    let total = 0;
    for (let i = 0; i < BANDAS.length; i++)
        total += resposta(tipoDaBanda(i), BANDAS[i], ganhos ? ganhos[i] : 0, f);
    return total;
}

// A curva resultante: n pontos { f, db } de 20 Hz a 20 kHz, em escala
// logarítmica, já com o pré-ganho.
function curva(ganhos, preGanho, n) {
    n = Math.max(2, Math.round(n) || 64);
    const out = [];
    for (let i = 0; i < n; i++) {
        const f = CURVA_DE * Math.pow(CURVA_ATE / CURVA_DE, i / (n - 1));
        out.push({ f: f, db: somaEm(ganhos, f) + (Number(preGanho) || 0) });
    }
    return out;
}

// O ponto mais alto da soma das bandas (sem o pré-ganho), em dB. Nunca
// abaixo de 0: um corte não pede pré-ganho.
function pico(ganhos) {
    let maior = 0;
    const pontos = curva(ganhos, 0, 121);
    for (let i = 0; i < pontos.length; i++)
        if (pontos[i].db > maior)
            maior = pontos[i].db;
    return maior;
}

// Quanto os ganhos positivos passam do pré-ganho, em dB (0 ou menos: cabe).
function sobra(ganhos, preGanho) {
    return pico(ganhos) + (Number(preGanho) || 0);
}

function podeDistorcer(ganhos, preGanho) {
    return sobra(ganhos, preGanho) > 0.05;
}

// O pré-ganho que cobre o pico (no passo, e não abaixo do limite).
function preSugerido(ganhos) {
    const p = pico(ganhos);
    if (p <= 0.05)
        return 0;
    return pre(-Math.ceil((p - 0.05) / PASSO) * PASSO);
}

// A frase do aviso, ou "".
function avisoDeDistorcao(ganhos, preGanho) {
    if (!podeDistorcer(ganhos, preGanho))
        return "";
    const quer = preSugerido(ganhos);
    if (quer >= Number(preGanho))
        return "pode distorcer: os ganhos passam do limite do pré-ganho";
    return "pode distorcer: baixe o pré-ganho para " + textoDb(quer);
}

// Os pontos da curva no desenho: [{ x, y }], com 0 dB no meio da altura.
function pontos(ganhos, preGanho, largura, altura, n) {
    const c = curva(ganhos, preGanho, n);
    return c.map((p, i) => ({
        x: largura * i / (c.length - 1),
        y: altura / 2 - prender(p.db, -ESCALA, ESCALA) / ESCALA * (altura / 2)
    }));
}

// Onde cai, no desenho, a frequência f (de 0 a `largura`).
function xDaFrequencia(f, largura) {
    return largura * Math.log(prender(f, CURVA_DE, CURVA_ATE) / CURVA_DE) / Math.log(CURVA_ATE / CURVA_DE);
}

// O que vai para o `eq ajustar`.
function pedido(ganhos, preGanho) {
    return { ganhos: BANDAS.map((_, i) => ganho(ganhos ? ganhos[i] : 0)), pre: pre(preGanho) };
}

// --- o ritmo das aplicações ------------------------------------------------------

// Durante o arraste, no máximo uma aplicação a cada INTERVALO_MS, e nunca
// duas ao mesmo tempo.
var INTERVALO_MS = 100;

function devoAplicar(agora, ultima, emVoo) {
    return !emVoo && agora - ultima >= INTERVALO_MS;
}

// Quanto falta para a próxima aplicação poder sair (0: já pode).
function esperaPara(agora, ultima) {
    return Math.max(0, Math.min(INTERVALO_MS, INTERVALO_MS - (agora - ultima)));
}

// --- predefinições ---------------------------------------------------------------

var NOME_MAX = 40;

// O nome da predefinição que tem exatamente estes valores, ou "".
function predefinicaoDe(lista, ganhos, preGanho) {
    for (let i = 0; i < (lista || []).length; i++)
        if (mesmos(lista[i].ganhos, ganhos) && Number(lista[i].pre) === Number(preGanho))
            return lista[i].nome;
    return "";
}

function acha(lista, nome) {
    const quer = String(nome).toLowerCase();
    for (let i = 0; i < (lista || []).length; i++)
        if (String(lista[i].nome).toLowerCase() === quer)
            return lista[i];
    return null;
}

function daPessoa(lista) {
    return (lista || []).filter(p => !p.fabrica);
}

// O nome sem os espaços das pontas (só espaços: é o que o motor tira).
function aparado(nome) {
    return String(nome).replace(/^ +| +$/g, "");
}

// Por que este nome não serve para guardar ("" = serve). As mesmas regras do
// motor (ValidaNome), mais: não pode ser o de uma predefinição de fábrica.
function erroDoNome(nome, lista) {
    nome = String(nome === null || nome === undefined ? "" : nome);
    if (/[\u0000-\u001f\u007f-\u009f]/.test(nome))
        return "o nome não pode ter caractere de controle";
    nome = aparado(nome);
    if (nome === "")
        return "dê um nome à predefinição";
    if (Array.from(nome).length > NOME_MAX)
        return "o nome pode ter até 40 caracteres";
    if (/[\/\\]/.test(nome))
        return "o nome não pode ter barra";
    if (nome.charAt(0) === "-" || nome === "." || nome === "..")
        return "o nome não pode começar com - nem ser só pontos";
    const ja = acha(lista, nome);
    if (ja && ja.fabrica)
        return "“" + nome + "” é uma predefinição de fábrica: escolha outro nome";
    return "";
}

// --- o microfone -----------------------------------------------------------------

var PISO = -90;
// O medidor mostra de MEDIDOR_DE dBFS (vazio) a 0 (cheio).
var MEDIDOR_DE = -60;
// O pico da barra cai devagar: tantos dB por segundo.
var QUEDA_POR_SEGUNDO = 18;
// Quantos níveis a janela guarda (uns 2 s, a 15 por segundo) e quantos
// precisa para julgar.
var JANELA = 30;
var MINIMO_PARA_JULGAR = 12;
var SATURA_EM = -1;
var BAIXO_ATE = -40;
var VOLUME_MAX = 150;

// De dBFS para a fração da barra (0 a 1).
function fracao(db) {
    db = Number(db);
    if (!isFinite(db))
        return 0;
    return prender((db - MEDIDOR_DE) / (0 - MEDIDOR_DE), 0, 1);
}

// O pico que a barra segura: sobe na hora, cai devagar.
function picoQueCai(anterior, novo, dtMs) {
    anterior = isFinite(Number(anterior)) ? Number(anterior) : PISO;
    novo = isFinite(Number(novo)) ? Number(novo) : PISO;
    const caido = anterior - QUEDA_POR_SEGUNDO * Math.max(0, Number(dtMs) || 0) / 1000;
    return Math.max(PISO, novo, caido);
}

// Junta um nível aos últimos (uma lista nova, com no máximo JANELA itens).
function junta(niveis, nivel) {
    if (!nivel || typeof nivel.pico !== "number" || typeof nivel.rms !== "number")
        return niveis || [];
    return (niveis || []).concat([{ pico: nivel.pico, rms: nivel.rms }]).slice(-JANELA);
}

// O maior pico dos últimos níveis (o piso, se não há nenhum).
function maiorPico(niveis) {
    let maior = PISO;
    for (let i = 0; i < (niveis || []).length; i++)
        if (niveis[i].pico > maior)
            maior = niveis[i].pico;
    return maior;
}

function entradaDe(entradas, id) {
    for (let i = 0; i < (entradas || []).length; i++)
        if (entradas[i].id === id)
            return entradas[i];
    return null;
}

// A entrada que a janela mostra ao abrir: a padrão, ou a primeira.
function entradaInicial(entradas) {
    for (let i = 0; i < (entradas || []).length; i++)
        if (entradas[i].padrao)
            return entradas[i].id;
    return (entradas || []).length > 0 ? entradas[0].id : -1;
}

// A lista com uma entrada alterada (lista e item novos: o QML vê a mudança).
function comEntrada(entradas, id, campos) {
    return (entradas || []).map(e => e.id === id ? Object.assign({}, e, campos) : e);
}

// A lista com esta entrada como a padrão (e nenhuma outra).
function comPadrao(entradas, id) {
    return (entradas || []).map(e => Object.assign({}, e, { padrao: e.id === id }));
}

// Um volume válido para o motor: inteiro de 0 a 150.
function volume(v) {
    return Math.round(prender(v, 0, VOLUME_MAX));
}

function textoDoVolume(v) {
    return v === null || v === undefined ? "—" : volume(v) + "%";
}

// O diagnóstico em uma frase: { texto, problema }. `niveis` são os últimos
// níveis do medidor da entrada escolhida; `ouvindo`, se o medidor roda.
function diagnostico(entradas, entrada, niveis, ouvindo) {
    if (!entradas || entradas.length === 0)
        return { texto: "nenhuma entrada", problema: true };
    if (!entrada)
        return { texto: "escolha uma entrada", problema: false };
    if (entrada.mudo)
        return { texto: "a entrada está muda", problema: true };
    const n = (niveis || []).length;
    const maior = maiorPico(niveis);
    if (n > 0 && maior >= SATURA_EM)
        return { texto: "está saturando (baixe o volume)", problema: true };
    if (n < MINIMO_PARA_JULGAR)
        return { texto: ouvindo ? "ouvindo…" : "o medidor está parado", problema: false };
    if (maior < BAIXO_ATE)
        return { texto: "o nível está muito baixo (fale perto / suba o volume)", problema: true };
    return { texto: "o nível está bom", problema: false };
}

// O texto do botão de gravar e ouvir, conforme a etapa.
function textoDaGravacao(etapa, falta) {
    if (etapa === "gravando")
        return "gravando… " + Math.max(0, Math.round(Number(falta) || 0)) + " s";
    if (etapa === "ouvindo")
        return "tocando o que gravou…";
    return "gravar e ouvir";
}

// --- a janela --------------------------------------------------------------------

// Abaixo desta largura os controles estreitam e os rótulos encurtam; abaixo
// desta altura a curva some.
var LARGURA_ESTREITA = 480;
var ALTURA_BAIXA = 470;

function estreita(largura) {
    return largura < LARGURA_ESTREITA;
}

function baixa(altura) {
    return altura < ALTURA_BAIXA;
}

function area(nome) {
    return nome === "microfone" ? "microfone" : "equalizador";
}

// Por onde o som sai, numa frase.
function textoDaSaida(estado) {
    if (!estado || !estado.som || !estado.som.responde)
        return (estado && estado.som && estado.som.erro) || "o PipeWire não respondeu";
    const eq = estado.eq || {};
    if (eq.ligado && eq.na_saida) {
        const a = eq.saida_anterior;
        if (a && a.existe)
            return "equalizando " + (a.descricao || a.nome);
        return "equalizando a saída";
    }
    if (eq.ligado)
        return "ligado, mas a saída padrão é outra: " + (estado.saida_padrao ? estado.saida_padrao.descricao : "nenhuma");
    return estado.saida_padrao ? "saída: " + estado.saida_padrao.descricao : "nenhuma saída de som";
}

// O arquivo de exemplo que responde a um pedido (MKSOM_EXEMPLO): os nomes do
// subcomando, sem o que a pessoa escolheu.
function arquivoDeExemplo(args) {
    const partes = [];
    for (let i = 0; i < (args || []).length; i++) {
        if (!/^[a-z]+$/.test(args[i]))
            break;
        partes.push(args[i]);
        if (partes.length === (partes[0] === "eq" && partes[1] === "predefinicao" ? 3 : 2))
            break;
        if (partes[0] === "estado")
            break;
    }
    return partes.join("-") + ".json";
}

var ATALHOS = [
    ["Tab", "anda entre as bandas"],
    ["↑ ↓", "meio dB a mais ou a menos"],
    ["Page Up / Down", "3 dB a mais ou a menos"],
    ["0", "zera a banda"],
    ["Espaço", "liga e desliga o equalizador"],
    ["Ctrl+0", "deixa tudo plano"],
    ["Ctrl+S", "guarda como predefinição"],
    ["Ctrl+1", "Equalizador"],
    ["Ctrl+2", "Microfone"],
    ["F1", "mostra estes atalhos"],
    ["Esc", "fecha a folha aberta"]
];
