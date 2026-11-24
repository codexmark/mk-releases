.pragma library

// A lógica da janela do MkCorte que não depende do QML: o modelo da edição
// (uma lista ordenada de clipes, cada um um trecho de um vídeo de origem), as
// operações sobre ela, o desfazer, a conversão entre o tempo da edição e o
// tempo de cada vídeo, a linha do tempo, os atalhos e os argumentos do motor.
// Nada aqui muda o que recebe: toda operação devolve uma edição nova (ou a
// mesma, se não havia o que fazer). Testado em
// tests/apps/mkcorte-Corte.test.mjs.
//
//   edição: { fontes: [{ arquivo, duracao, fps, largura, altura, temAudio, tamanho }],
//             clipes: [{ fonte, inicio, fim }] }   (fonte é o índice em fontes)

// O menor clipe que a janela deixa existir, em segundos.
var MINIMO = 0.05;
// Dois tempos a menos disto são o mesmo ponto.
var JUNTO = 0.002;

function valido(v) {
    return typeof v === "number" && isFinite(v) && v >= 0;
}

function arred(t) {
    return Math.round(t * 1000) / 1000;
}

function entre(v, min, max) {
    return Math.min(Math.max(v, min), Math.max(min, max));
}

// ── a edição ───────────────────────────────────────────────────────────

function vazia() {
    return { fontes: [], clipes: [] };
}

// A resposta do `sondar` no formato que a edição guarda.
function fonteDe(midia) {
    var m = midia || {};
    return {
        arquivo: String(m.arquivo || ""),
        duracao: valido(m.duracao_s) ? m.duracao_s : 0,
        fps: valido(m.fps) && m.fps > 0 ? m.fps : 30,
        largura: valido(m.largura) ? m.largura : 0,
        altura: valido(m.altura) ? m.altura : 0,
        temAudio: m.tem_audio === true,
        tamanho: valido(m.tamanho_bytes) ? m.tamanho_bytes : 0
    };
}

// Abrir um vídeo: uma edição com um clipe do tamanho dele todo.
function abrirFonte(midia) {
    var f = fonteDe(midia);
    if (f.arquivo === "" || f.duracao < MINIMO)
        return vazia();
    return { fontes: [f], clipes: [{ fonte: 0, inicio: 0, fim: f.duracao }] };
}

function duracaoDoClipe(c) {
    return c ? Math.max(0, c.fim - c.inicio) : 0;
}

// A duração da edição: a soma dos clipes.
function duracao(ed) {
    var total = 0, i;
    for (i = 0; i < ed.clipes.length; i++)
        total += duracaoDoClipe(ed.clipes[i]);
    return arred(total);
}

// Em que instante da edição o clipe i começa.
function inicioDoClipe(ed, i) {
    var t = 0, k;
    for (k = 0; k < i && k < ed.clipes.length; k++)
        t += duracaoDoClipe(ed.clipes[k]);
    return arred(t);
}

// Tempo da edição → { clipe, tempo (na fonte), dentro (do clipe) }. Na emenda
// de dois clipes, o instante é do que começa. Depois do fim, é o fim do
// último. null se a edição está vazia.
function localizar(ed, t) {
    var n = ed.clipes.length, acumulado = 0, i, d;
    if (n === 0)
        return null;
    t = valido(t) ? t : 0;
    for (i = 0; i < n; i++) {
        d = duracaoDoClipe(ed.clipes[i]);
        if (t < acumulado + d - JUNTO / 2)
            return { clipe: i, tempo: arred(ed.clipes[i].inicio + Math.max(0, t - acumulado)), dentro: arred(Math.max(0, t - acumulado)) };
        acumulado += d;
    }
    return { clipe: n - 1, tempo: ed.clipes[n - 1].fim, dentro: arred(duracaoDoClipe(ed.clipes[n - 1])) };
}

// (clipe, tempo na fonte) → tempo da edição. Um tempo fora do clipe cai na
// ponta mais próxima dele.
function tempoNaEdicao(ed, i, tempoNaFonte) {
    var c = ed.clipes[i];
    if (!c)
        return 0;
    return arred(inicioDoClipe(ed, i) + entre(tempoNaFonte - c.inicio, 0, duracaoDoClipe(c)));
}

function copia(ed, clipes) {
    return { fontes: ed.fontes, clipes: clipes };
}

// Dividir no instante t da edição: o clipe sob o cursor vira dois. Perto
// demais de uma ponta, não há o que dividir.
function dividir(ed, t) {
    var onde = localizar(ed, t), c, clipes;
    if (!onde)
        return ed;
    c = ed.clipes[onde.clipe];
    if (onde.tempo - c.inicio < MINIMO || c.fim - onde.tempo < MINIMO)
        return ed;
    clipes = ed.clipes.slice();
    clipes.splice(onde.clipe, 1, { fonte: c.fonte, inicio: c.inicio, fim: onde.tempo }, { fonte: c.fonte, inicio: onde.tempo, fim: c.fim });
    return copia(ed, clipes);
}

// Até onde as pontas do clipe i podem ir: os limites do vídeo de origem, mas
// sem invadir o vizinho que é o pedaço anterior (ou seguinte) do mesmo vídeo.
function limites(ed, i) {
    var c = ed.clipes[i], antes = ed.clipes[i - 1], depois = ed.clipes[i + 1], min = 0, max;
    if (!c)
        return { min: 0, max: 0 };
    max = ed.fontes[c.fonte].duracao;
    if (antes && antes.fonte === c.fonte && antes.fim <= c.inicio + JUNTO)
        min = Math.min(antes.fim, c.inicio);
    if (depois && depois.fonte === c.fonte && depois.inicio >= c.fim - JUNTO)
        max = Math.max(depois.inicio, c.fim);
    return { min: min, max: max };
}

// Aparar o início do clipe i para o instante dado do vídeo de origem. Pode
// encolher ou crescer de volta, dentro dos limites.
function apararInicio(ed, i, tempoNaFonte) {
    var c = ed.clipes[i], novo, clipes;
    if (!c || !valido(tempoNaFonte) && !(tempoNaFonte < 0))
        return ed;
    novo = arred(entre(tempoNaFonte, limites(ed, i).min, c.fim - MINIMO));
    if (novo === c.inicio)
        return ed;
    clipes = ed.clipes.slice();
    clipes[i] = { fonte: c.fonte, inicio: novo, fim: c.fim };
    return copia(ed, clipes);
}

function apararFim(ed, i, tempoNaFonte) {
    var c = ed.clipes[i], novo, clipes;
    if (!c || !valido(tempoNaFonte) && !(tempoNaFonte < 0))
        return ed;
    novo = arred(Math.max(c.inicio + MINIMO, Math.min(tempoNaFonte, limites(ed, i).max)));
    if (novo === c.fim)
        return ed;
    clipes = ed.clipes.slice();
    clipes[i] = { fonte: c.fonte, inicio: c.inicio, fim: novo };
    return copia(ed, clipes);
}

// Trazer o início (ou o fim) do clipe sob o cursor para o cursor.
function trazerInicio(ed, t) {
    var onde = localizar(ed, t);
    return onde ? apararInicio(ed, onde.clipe, onde.tempo) : ed;
}

function trazerFim(ed, t) {
    var onde = localizar(ed, t);
    return onde ? apararFim(ed, onde.clipe, onde.tempo) : ed;
}

function remover(ed, i) {
    var clipes;
    if (i < 0 || i >= ed.clipes.length)
        return ed;
    clipes = ed.clipes.slice();
    clipes.splice(i, 1);
    return copia(ed, clipes);
}

// O clipe i e o seguinte são pedaços vizinhos e contíguos do mesmo vídeo?
function podeJuntar(ed, i) {
    var a = ed.clipes[i], b = ed.clipes[i + 1];
    return !!a && !!b && a.fonte === b.fonte && Math.abs(a.fim - b.inicio) <= JUNTO;
}

// Juntar o clipe i com o seguinte (o contrário de dividir).
function juntar(ed, i) {
    var clipes;
    if (!podeJuntar(ed, i))
        return ed;
    clipes = ed.clipes.slice();
    clipes.splice(i, 2, { fonte: clipes[i].fonte, inicio: clipes[i].inicio, fim: clipes[i + 1].fim });
    return copia(ed, clipes);
}

// Mover o clipe da posição `de` para a posição `para` (as duas contadas na
// lista como ela fica depois).
function mover(ed, de, para) {
    var n = ed.clipes.length, clipes, c;
    if (de < 0 || de >= n || n < 2)
        return ed;
    para = entre(para, 0, n - 1);
    if (para === de)
        return ed;
    clipes = ed.clipes.slice();
    c = clipes.splice(de, 1)[0];
    clipes.splice(para, 0, c);
    return copia(ed, clipes);
}

function indiceDaFonte(ed, arquivo) {
    for (var i = 0; i < ed.fontes.length; i++)
        if (ed.fontes[i].arquivo === arquivo)
            return i;
    return -1;
}

// Juntar outro vídeo: entra inteiro, depois do clipe sob o cursor (no fim, se
// a edição está vazia). O mesmo arquivo de novo reaproveita a fonte.
function inserirFonte(ed, midia, t) {
    var f = fonteDe(midia), fontes = ed.fontes, k, onde, clipes;
    if (f.arquivo === "" || f.duracao < MINIMO)
        return ed;
    k = indiceDaFonte(ed, f.arquivo);
    if (k < 0) {
        fontes = ed.fontes.concat([f]);
        k = fontes.length - 1;
    }
    onde = localizar(ed, t);
    clipes = ed.clipes.slice();
    clipes.splice(onde ? onde.clipe + 1 : clipes.length, 0, { fonte: k, inicio: 0, fim: fontes[k].duracao });
    return { fontes: fontes, clipes: clipes };
}

// ── andar pela edição ──────────────────────────────────────────────────

// As teclas 0–9: o instante a n décimos da edição.
function pular(ed, n) {
    return arred(duracao(ed) * entre(Math.floor(n), 0, 9) / 10);
}

// Andar delta segundos a partir de t, sem sair da edição.
function passo(ed, t, delta) {
    return arred(entre((valido(t) ? t : 0) + delta, 0, duracao(ed)));
}

// A duração de um quadro do vídeo que está sob o instante t.
function quadro(ed, t) {
    var onde = localizar(ed, t);
    return onde ? 1 / ed.fontes[ed.clipes[onde.clipe].fonte].fps : 1 / 30;
}

// As bordas dos clipes, em tempo da edição: 0, cada emenda e o fim.
function bordas(ed) {
    var out = [0], t = 0, i;
    for (i = 0; i < ed.clipes.length; i++) {
        t += duracaoDoClipe(ed.clipes[i]);
        out.push(arred(t));
    }
    return out;
}

function bordaAnterior(ed, t) {
    var b = bordas(ed), i;
    for (i = b.length - 1; i >= 0; i--)
        if (b[i] < t - JUNTO)
            return b[i];
    return 0;
}

function bordaSeguinte(ed, t) {
    var b = bordas(ed), i;
    for (i = 0; i < b.length; i++)
        if (b[i] > t + JUNTO)
            return b[i];
    return b[b.length - 1];
}

// O encaixe: o alvo mais próximo do valor, se está a menos que a tolerância;
// senão, o próprio valor.
function encaixar(valor, alvos, tolerancia) {
    var melhor = valor, perto = tolerancia, i, d;
    for (i = 0; i < (alvos || []).length; i++) {
        d = Math.abs(alvos[i] - valor);
        if (d <= perto) {
            perto = d;
            melhor = alvos[i];
        }
    }
    return melhor;
}

// Onde a alça do clipe i gruda (em tempo do vídeo de origem): na borda do
// vizinho que é outro pedaço do mesmo vídeo, nas pontas do vídeo e no
// cursor, se ele está em cima deste clipe.
function alvosDeEncaixe(ed, i, cursor) {
    var c = ed.clipes[i], lim, alvos, onde;
    if (!c)
        return [];
    lim = limites(ed, i);
    alvos = [lim.min, lim.max];
    onde = localizar(ed, cursor);
    if (onde && onde.clipe === i && onde.tempo > c.inicio + JUNTO && onde.tempo < c.fim - JUNTO)
        alvos.push(onde.tempo);
    return alvos;
}

// A pré-visualização chegou ao instante `tempoNaFonte` tocando o clipe i: o
// que fazer. { clipe, tempo, salta, trocaFonte } ou { fim: true }. Enquanto
// está dentro do clipe (ou a emenda com o seguinte é contígua), só segue.
function avancar(ed, i, tempoNaFonte) {
    var c = ed.clipes[i], prox = ed.clipes[i + 1];
    if (!c)
        return { fim: true };
    if (tempoNaFonte < c.fim - 0.03)
        return { clipe: i, tempo: tempoNaFonte, salta: false, trocaFonte: false };
    if (!prox)
        return { fim: true };
    if (prox.fonte === c.fonte && Math.abs(prox.inicio - c.fim) <= 0.03)
        return { clipe: i + 1, tempo: Math.max(tempoNaFonte, prox.inicio), salta: false, trocaFonte: false };
    return { clipe: i + 1, tempo: prox.inicio, salta: true, trocaFonte: prox.fonte !== c.fonte };
}

// ── desfazer e refazer ─────────────────────────────────────────────────

// Quantos passos o desfazer guarda.
var MEMORIA = 200;

// O histórico de uma edição recém-aberta: nada a desfazer, nada a exportar.
function historia(ed) {
    return { passado: [], atual: { id: 1, ed: ed }, futuro: [], proximo: 2, limpa: 1 };
}

// Uma edição nova entra no histórico (a mesma, não: operação sem efeito não
// vira passo de desfazer). O que estava para refazer se perde.
function aplicar(h, ed) {
    if (ed === h.atual.ed)
        return h;
    var passado = h.passado.concat([h.atual]);
    if (passado.length > MEMORIA)
        passado = passado.slice(passado.length - MEMORIA);
    return { passado: passado, atual: { id: h.proximo, ed: ed }, futuro: [], proximo: h.proximo + 1, limpa: h.limpa };
}

function podeDesfazer(h) {
    return !!h && h.passado.length > 0;
}

function podeRefazer(h) {
    return !!h && h.futuro.length > 0;
}

function desfazer(h) {
    if (!podeDesfazer(h))
        return h;
    return { passado: h.passado.slice(0, -1), atual: h.passado[h.passado.length - 1], futuro: [h.atual].concat(h.futuro), proximo: h.proximo, limpa: h.limpa };
}

function refazer(h) {
    if (!podeRefazer(h))
        return h;
    return { passado: h.passado.concat([h.atual]), atual: h.futuro[0], futuro: h.futuro.slice(1), proximo: h.proximo, limpa: h.limpa };
}

// Depois de exportar: o estado de agora é o que está salvo.
function marcarExportado(h) {
    return { passado: h.passado, atual: h.atual, futuro: h.futuro, proximo: h.proximo, limpa: h.atual.id };
}

// Há mudanças que ainda não foram exportadas? (Desfazer até o estado
// exportado conta como "não há".)
function haMudancas(h) {
    return !!h && h.atual.id !== h.limpa;
}

// ── a linha do tempo ───────────────────────────────────────────────────

// Como a edição se espalha numa linha de `largura` pixels: { escala (px por
// segundo), desloca (px escondidos à esquerda) }. Com zoom, o clipe sob o
// cursor ocupa 80% da linha.
function vista(ed, cursor, largura, zoom) {
    var total = duracao(ed), onde, d, escala, desloca;
    if (total <= 0 || !(largura > 0))
        return { escala: 1, desloca: 0 };
    escala = largura / total;
    onde = zoom ? localizar(ed, cursor) : null;
    if (!onde)
        return { escala: escala, desloca: 0 };
    d = duracaoDoClipe(ed.clipes[onde.clipe]);
    escala = Math.max(escala, largura * 0.8 / d);
    desloca = entre(inicioDoClipe(ed, onde.clipe) * escala - largura * 0.1, 0, total * escala - largura);
    return { escala: escala, desloca: desloca };
}

function xDoTempo(v, t) {
    return t * v.escala - v.desloca;
}

function tempoDoX(v, x) {
    return (x + v.desloca) / v.escala;
}

// O quadro da tira mais próximo do instante dado (o caminho do arquivo), ou "".
function miniaturaEm(quadros, tempo) {
    var melhor = "", perto = Infinity, i, d;
    for (i = 0; i < (quadros || []).length; i++) {
        d = Math.abs(quadros[i].tempo_s - tempo);
        if (d < perto) {
            perto = d;
            melhor = quadros[i].arquivo;
        }
    }
    return melhor;
}

// A tira de um clipe desenhado com `larguraPx`: fatias de mais ou menos
// `larguraDaFatia`, cada uma com o quadro do meio dela.
function fatias(clipe, larguraPx, larguraDaFatia, quadros) {
    var n = Math.max(1, Math.min(60, Math.round(larguraPx / Math.max(8, larguraDaFatia)))), out = [], i, d = duracaoDoClipe(clipe);
    if (!(larguraPx > 0) || !quadros || quadros.length === 0)
        return out;
    for (i = 0; i < n; i++)
        out.push({ x: larguraPx * i / n, largura: larguraPx / n, arquivo: miniaturaEm(quadros, clipe.inicio + (i + 0.5) * d / n) });
    return out;
}

// Soltaram o clipe i no instante t da edição: a posição nova dele (quantos
// dos outros clipes têm o meio antes de t).
function destinoDoArrasto(ed, i, t) {
    var pos = 0, acumulado = 0, k, d;
    for (k = 0; k < ed.clipes.length; k++) {
        d = duracaoDoClipe(ed.clipes[k]);
        if (k !== i && acumulado + d / 2 < t)
            pos++;
        acumulado += d;
    }
    return pos;
}

// ── o motor ────────────────────────────────────────────────────────────

// A edição como o motor a lê (CONTRATO.md): só os vídeos que algum clipe usa.
function paraOMotor(ed, opcoes) {
    var o = opcoes || {}, fontes = [], mapa = {}, clipes = [], i, c;
    for (i = 0; i < ed.clipes.length; i++) {
        c = ed.clipes[i];
        if (mapa[c.fonte] === undefined) {
            mapa[c.fonte] = fontes.length;
            fontes.push(ed.fontes[c.fonte].arquivo);
        }
        clipes.push({ fonte: mapa[c.fonte], inicio: arred(c.inicio), fim: arred(c.fim) });
    }
    return { fontes: fontes, clipes: clipes, qualidade: ["1080", "720"].indexOf(o.qualidade) >= 0 ? o.qualidade : "original",
        modo: o.modo === "rapido" ? "rapido" : "exato" };
}

function argsSondar(arquivo) {
    return ["sondar", "--", arquivo];
}

function argsMiniaturas(arquivo) {
    return ["miniaturas", "--quantas", "16", "--", arquivo];
}

function argsPlano(ed, opcoes) {
    return ["plano", "--json", JSON.stringify(paraOMotor(ed, opcoes))];
}

function argsExportar(ed, opcoes) {
    return ["exportar", "--json", JSON.stringify(paraOMotor(ed, opcoes))];
}

// Salvar o quadro sob o cursor: do vídeo e do instante em que ele está.
function argsQuadro(ed, t) {
    var onde = localizar(ed, t), c, f;
    if (!onde)
        return null;
    c = ed.clipes[onde.clipe];
    f = ed.fontes[c.fonte];
    // O último instante do vídeo não tem quadro: recua um.
    return ["quadro", "--", f.arquivo, String(arred(Math.max(0, Math.min(onde.tempo, f.duracao - 1 / f.fps))))];
}

// O arquivo de exemplo (MKCORTE_EXEMPLO) que responde a um pedido.
function arquivoDeExemplo(args) {
    return String((args || [])[0] || "") + ".json";
}

// O modo que vale: o rápido só quando o plano diz que dá.
function modoPossivel(plano, modo) {
    return modo === "rapido" && !!plano && !!plano.sem_recodificar && plano.sem_recodificar.pode === true ? "rapido" : "exato";
}

// ── textos ─────────────────────────────────────────────────────────────

// 7.25 → "0:07.2"; 3725 → "1:02:05.0".
function tempo(s) {
    if (!valido(s))
        return "–";
    var d = Math.floor(s * 10 + 1e-6), h = Math.floor(d / 36000), m = Math.floor(d / 600) % 60, seg = Math.floor(d / 10) % 60;
    var fim = (seg < 10 ? "0" : "") + seg + "." + (d % 10);
    return h > 0 ? h + ":" + (m < 10 ? "0" : "") + m + ":" + fim : m + ":" + fim;
}

// O que falta: "12 s", "3 min", "1 h 05 min".
function resta(s) {
    if (!valido(s))
        return "";
    if (s < 60)
        return Math.ceil(s) + " s";
    var min = Math.ceil(s / 60);
    return min < 60 ? min + " min" : Math.floor(min / 60) + " h " + (min % 60 < 10 ? "0" : "") + (min % 60) + " min";
}

function tamanho(bytes) {
    if (!valido(bytes))
        return "–";
    var u = ["B", "KB", "MB", "GB", "TB"], i = 0, v = bytes;
    while (v >= 1024 && i < u.length - 1) {
        v /= 1024;
        i++;
    }
    return (i === 0 || v >= 100 ? Math.round(v) : v.toFixed(1).replace(".", ",")) + " " + u[i];
}

function nome(caminho) {
    var s = String(caminho || "");
    return s.slice(s.lastIndexOf("/") + 1);
}

function pasta(caminho) {
    var s = String(caminho || ""), i = s.lastIndexOf("/");
    return i <= 0 ? "/" : s.slice(0, i);
}

// "file:///a%20b.mp4" → "/a b.mp4". O que não é arquivo local vira "".
function caminhoDeUrl(url) {
    var s = String(url === null || url === undefined ? "" : url);
    if (s.charAt(0) === "/")
        return s;
    var m = /^file:\/\/(localhost)?(\/.*)$/.exec(s);
    if (!m)
        return "";
    try {
        return decodeURIComponent(m[2]);
    } catch (e) {
        return "";
    }
}

// "/a b#1.mp4" → "file:///a%20b%231.mp4": o que a pré-visualização e as
// imagens recebem (um "#" ou "?" solto no nome cortaria o caminho).
function urlDeCaminho(caminho) {
    var s = String(caminho || "");
    if (s.charAt(0) !== "/")
        return "";
    return "file://" + s.split("/").map(encodeURIComponent).join("/");
}

// A lista de arquivos que o lançador passou (MKCORTE_ARQUIVOS, uma lista JSON).
function lerArquivos(texto) {
    var lista;
    try {
        lista = JSON.parse(texto || "[]");
    } catch (e) {
        return [];
    }
    if (!Array.isArray(lista))
        return [];
    return lista.filter(function (c) { return typeof c === "string" && c.charAt(0) === "/"; });
}

// "1920×1080 · 30 qps · 1:02.4 · com áudio".
function resumoDaFonte(f) {
    if (!f)
        return "";
    return f.largura + "×" + f.altura + "  ·  " + (Math.round(f.fps * 100) / 100).toString().replace(".", ",") + " qps  ·  " + tempo(f.duracao) +
        "  ·  " + (f.temAudio ? "com áudio" : "sem áudio");
}

// O que o plano respondeu, numa linha.
function textoDoPlano(plano) {
    if (!plano || plano.erro)
        return "";
    return tempo(plano.duracao_s) + "  ·  ≈ " + tamanho(plano.tamanho_estimado_bytes) + "  ·  " + plano.largura + "×" + plano.altura +
        (plano.tem_audio ? "" : "  ·  sem áudio");
}

function textoDoProgresso(fase) {
    if (!fase || !valido(fase.pct))
        return "começando…";
    var partes = [Math.floor(fase.pct) + "%"];
    if (valido(fase.resta_s))
        partes.push("resta " + resta(fase.resta_s));
    if (valido(fase.velocidade))
        partes.push(String(fase.velocidade).replace(".", ",") + "×");
    return partes.join("  ·  ");
}

// ── o teclado ──────────────────────────────────────────────────────────

// Os modificadores e as teclas do Qt (Qt.ShiftModifier, Qt.Key_Left…): aqui
// por número, para a tabela de atalhos ser testada fora do QML.
var SHIFT = 0x02000000;
var CTRL = 0x04000000;
var ALT = 0x08000000;
var TECLA = {
    espaco: 0x20, esquerda: 0x01000012, direita: 0x01000014, cima: 0x01000013, baixo: 0x01000015, abre: 0x5b, fecha: 0x5d,
    apagar: 0x01000007, esc: 0x01000000, enter: 0x01000005, retorno: 0x01000004, tab: 0x01000001, tabVolta: 0x01000002, interrogacao: 0x3f, f1: 0x01000030,
    F: 0x46, J: 0x4a, O: 0x4f, Q: 0x51, S: 0x53, X: 0x58, Z: 0x5a
};

// A lista que o "?" mostra: [teclas, o que faz].
var ATALHOS = [
    ["Espaço", "toca ou pausa"],
    ["0 – 9", "pula para 0% – 90% da edição"],
    ["← →", "anda 1 segundo"],
    ["Shift + ← →", "anda 5 segundos"],
    ["Alt + ← →", "anda um quadro"],
    ["[  ]", "vai para a borda anterior / seguinte"],
    ["S", "divide o clipe no cursor"],
    ["X ou Delete", "remove o clipe sob o cursor"],
    ["J", "junta o clipe com o seguinte (desfaz uma divisão)"],
    ["Alt + [  ]", "move o clipe para antes / depois"],
    ["Ctrl + Espaço", "traz o início do clipe para o cursor"],
    ["Ctrl + Shift + Espaço", "traz o fim do clipe para o cursor"],
    ["Ctrl + Z", "desfaz"],
    ["Ctrl + Shift + Z", "refaz"],
    ["Z", "zoom no clipe sob o cursor"],
    ["F", "salva o quadro como PNG"],
    ["Ctrl + O", "abre um vídeo"],
    ["Ctrl + Shift + O", "adiciona outro vídeo"],
    ["Ctrl + S", "exporta"],
    ["Q", "sai (pergunta antes, se há edição não exportada)"],
    ["?", "mostra estes atalhos"]
];

// A tecla apertada na janela principal → a ação: { acao, n?, delta? } ou
// null. `mods` são os modificadores do Qt (só Shift, Ctrl e Alt contam).
function acaoDaTecla(tecla, mods) {
    var shift = (mods & SHIFT) !== 0, ctrl = (mods & CTRL) !== 0, alt = (mods & ALT) !== 0;
    var limpo = !shift && !ctrl && !alt;
    if (tecla === TECLA.interrogacao || tecla === TECLA.f1)
        return { acao: "atalhos" };
    if (tecla === TECLA.espaco && !alt)
        return { acao: ctrl ? (shift ? "trazerFim" : "trazerInicio") : "tocar" };
    if (tecla >= 0x30 && tecla <= 0x39 && limpo)
        return { acao: "pular", n: tecla - 0x30 };
    if ((tecla === TECLA.esquerda || tecla === TECLA.direita) && !ctrl) {
        var sinal = tecla === TECLA.esquerda ? -1 : 1;
        if (alt && !shift)
            return { acao: "quadro", delta: sinal };
        if (!alt)
            return { acao: "andar", delta: sinal * (shift ? 5 : 1) };
        return null;
    }
    if ((tecla === TECLA.abre || tecla === TECLA.fecha) && !ctrl && !shift) {
        var antes = tecla === TECLA.abre;
        return { acao: alt ? (antes ? "moverAntes" : "moverDepois") : (antes ? "bordaAnterior" : "bordaSeguinte") };
    }
    if (tecla === TECLA.apagar && limpo)
        return { acao: "remover" };
    if (ctrl && !alt) {
        if (tecla === TECLA.Z)
            return { acao: shift ? "refazer" : "desfazer" };
        if (tecla === TECLA.O)
            return { acao: shift ? "adicionar" : "abrir" };
        if (tecla === TECLA.S && !shift)
            return { acao: "exportar" };
        return null;
    }
    if (!limpo)
        return null;
    switch (tecla) {
    case TECLA.S: return { acao: "dividir" };
    case TECLA.X: return { acao: "remover" };
    case TECLA.J: return { acao: "juntar" };
    case TECLA.Z: return { acao: "zoom" };
    case TECLA.F: return { acao: "salvarQuadro" };
    case TECLA.Q: return { acao: "sair" };
    }
    return null;
}

// A tecla apertada com um diálogo aberto: "confirmar" (Enter), "cancelar"
// (Esc), "proximo" / "anterior" (Tab, setas para os lados), "cima" / "baixo".
function acaoDoDialogo(tecla, mods) {
    var shift = (mods & SHIFT) !== 0;
    switch (tecla) {
    case TECLA.enter:
    case TECLA.retorno:
    case TECLA.espaco: return "confirmar";
    case TECLA.esc: return "cancelar";
    case TECLA.tab: return shift ? "anterior" : "proximo";
    case TECLA.tabVolta: return "anterior";
    case TECLA.direita: return "proximo";
    case TECLA.esquerda: return "anterior";
    case TECLA.cima: return "cima";
    case TECLA.baixo: return "baixo";
    }
    return null;
}

// Andar numa lista de n opções em roda.
function girar(i, n, delta) {
    return n > 0 ? ((i + delta) % n + n) % n : 0;
}

// ── o ponteiro na linha do tempo ───────────────────────────────────────

// A largura da alça de cada ponta de um clipe, em pixels.
var ALCA = 8;

// O que há sob o ponteiro em (x, y) na linha do tempo desenhada com a vista
// `v`: { tipo, clipe }. tipo: "inicio" / "fim" (as alças), "remover" (o × no
// canto de cima à direita do clipe), "clipe" (o corpo) ou "nada".
function toque(ed, v, x, y) {
    var t = 0, n = ed.clipes.length, i, d, x0, x1, largura, alca;
    for (i = 0; i < n; i++) {
        d = duracaoDoClipe(ed.clipes[i]);
        x0 = xDoTempo(v, t);
        x1 = xDoTempo(v, t + d);
        if (x >= x0 && (x < x1 || (i === n - 1 && x <= x1))) {
            largura = x1 - x0;
            alca = Math.min(ALCA, largura / 3);
            if (largura >= 44 && y >= 2 && y <= 20 && x >= x1 - 28 && x <= x1 - 10)
                return { tipo: "remover", clipe: i };
            if (x - x0 <= alca)
                return { tipo: "inicio", clipe: i };
            if (x1 - x <= alca)
                return { tipo: "fim", clipe: i };
            return { tipo: "clipe", clipe: i };
        }
        t += d;
    }
    return { tipo: "nada", clipe: -1 };
}

// Onde aparece a marca de "vai cair aqui" quando o clipe i é arrastado até o
// instante t: o x da emenda em que ele entra.
function marcaDoArrasto(ed, v, i, t) {
    var d = destinoDoArrasto(ed, i, t);
    return xDoTempo(v, bordas(ed)[d <= i ? d : d + 1]);
}
