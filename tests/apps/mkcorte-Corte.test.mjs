// apps/mkcorte/qml/Corte.js: o modelo da edição do MkCorte (clipes, dividir,
// aparar, remover, juntar, mover, inserir), o desfazer, a conversão entre o
// tempo da edição e o de cada vídeo, a linha do tempo, os atalhos de teclado
// e os argumentos do motor.
import { readFileSync } from "node:fs";
import { t, eq, igual, perto, assert } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const C = load(REPO + "/apps/mkcorte/qml/Corte.js");
const contrato = readFileSync(REPO + "/apps/mkcorte/CONTRATO.md", "utf8");
const exemplo = (nome) => JSON.parse(readFileSync(REPO + "/tests/apps/mkcorte-exemplo/" + nome + ".json", "utf8"));

const midia = (extra) => Object.assign({
    arquivo: "/v/passeio.mp4", tamanho_bytes: 48000000, duracao_s: 60, largura: 1920, altura: 1080, fps: 30, tem_audio: true
}, extra || {});
const outra = (extra) => midia(Object.assign({ arquivo: "/v/em pé.mov", duracao_s: 10, largura: 1080, altura: 1920, fps: 60, tem_audio: false }, extra || {}));
// Os clipes como [fonte, início, fim], para comparar numa linha.
const clipes = (ed) => ed.clipes.map((c) => [c.fonte, c.inicio, c.fim]);
const um = () => C.abrirFonte(midia());
// passeio dividido em 20 e 40: [0–20][20–40][40–60].
const tres = () => C.dividir(C.dividir(um(), 20), 40);

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["valido", "arred", "entre", "vazia", "fonteDe", "abrirFonte", "duracaoDoClipe", "duracao", "inicioDoClipe", "localizar",
        "tempoNaEdicao", "copia", "dividir", "limites", "apararInicio", "apararFim", "trazerInicio", "trazerFim", "remover", "podeJuntar", "juntar",
        "mover", "indiceDaFonte", "inserirFonte", "pular", "passo", "quadro", "bordas", "bordaAnterior", "bordaSeguinte", "encaixar", "alvosDeEncaixe",
        "avancar", "historia", "aplicar", "podeDesfazer", "podeRefazer", "desfazer", "refazer", "marcarExportado", "haMudancas", "vista", "xDoTempo",
        "tempoDoX", "miniaturaEm", "fatias", "destinoDoArrasto", "paraOMotor", "argsSondar", "argsMiniaturas", "argsPlano", "argsExportar",
        "argsQuadro", "arquivoDeExemplo", "modoPossivel", "tempo", "resta", "tamanho", "nome", "pasta", "caminhoDeUrl", "urlDeCaminho", "lerArquivos",
        "resumoDaFonte", "textoDoPlano", "textoDoProgresso", "acaoDaTecla", "acaoDoDialogo", "girar", "toque", "marcaDoArrasto"];
    const funcoes = Object.keys(C).filter((k) => typeof C[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("valido, arred, entre: os pequenos", () => {
    for (const v of [0, 1, 0.5])
        eq(C.valido(v), true);
    for (const v of [null, undefined, NaN, Infinity, -1, "3"])
        eq(C.valido(v), false);
    eq(C.arred(1.23456), 1.235);
    eq(C.arred(0.1 + 0.2), 0.3);
    eq(C.entre(5, 0, 3), 3);
    eq(C.entre(-1, 0, 3), 0);
    eq(C.entre(2, 0, 3), 2);
    eq(C.entre(5, 4, 1), 4, "um máximo abaixo do mínimo não inverte");
});

// ── abrir ──────────────────────────────────────────────────────────────

t("abrirFonte: um clipe do tamanho do vídeo todo", () => {
    const ed = um();
    igual(clipes(ed), [[0, 0, 60]]);
    igual(ed.fontes, [{ arquivo: "/v/passeio.mp4", duracao: 60, fps: 30, largura: 1920, altura: 1080, temAudio: true, tamanho: 48000000 }]);
    eq(C.duracao(ed), 60);
    igual(C.vazia(), { fontes: [], clipes: [] });
    // O que o motor não soube (null) não quebra; sem duração não há o que abrir.
    igual(C.fonteDe({ arquivo: "/a.mp4", duracao_s: 5, fps: null, largura: null }), { arquivo: "/a.mp4", duracao: 5, fps: 30, largura: 0, altura: 0, temAudio: false, tamanho: 0 });
    igual(C.fonteDe(null).arquivo, "");
    igual(C.abrirFonte(midia({ duracao_s: null })), C.vazia());
    igual(C.abrirFonte({ erro: "não achei" }), C.vazia());
});

// ── o tempo da edição e o tempo do vídeo ───────────────────────────────

t("localizar e tempoNaEdicao: ida e volta, com clipes fora de ordem e de fontes diferentes", () => {
    // [40–60 do passeio][0–10 do em pé][0–20 do passeio]: 20 + 10 + 20 = 50 s.
    let ed = C.inserirFonte(tres(), outra(), 50);
    ed = C.remover(C.mover(ed, 2, 0), 2);
    igual(clipes(ed), [[0, 40, 60], [0, 0, 20], [1, 0, 10]]);
    ed = C.mover(ed, 2, 1);
    igual(clipes(ed), [[0, 40, 60], [1, 0, 10], [0, 0, 20]]);
    eq(C.duracao(ed), 50);
    igual([0, 1, 2, 3].map((i) => C.inicioDoClipe(ed, i)), [0, 20, 30, 50]);
    igual(C.localizar(ed, 0), { clipe: 0, tempo: 40, dentro: 0 });
    igual(C.localizar(ed, 12.5), { clipe: 0, tempo: 52.5, dentro: 12.5 });
    // Na emenda, o instante é do clipe que começa.
    igual(C.localizar(ed, 20), { clipe: 1, tempo: 0, dentro: 0 });
    igual(C.localizar(ed, 29.99), { clipe: 1, tempo: 9.99, dentro: 9.99 });
    igual(C.localizar(ed, 35), { clipe: 2, tempo: 5, dentro: 5 });
    // No fim e além dele: o fim do último clipe.
    igual(C.localizar(ed, 50), { clipe: 2, tempo: 20, dentro: 20 });
    igual(C.localizar(ed, 999), { clipe: 2, tempo: 20, dentro: 20 });
    igual(C.localizar(ed, -3), { clipe: 0, tempo: 40, dentro: 0 });
    igual(C.localizar(ed, NaN), { clipe: 0, tempo: 40, dentro: 0 });
    eq(C.localizar(C.vazia(), 0), null);
    for (const tEd of [0, 7.3, 19.99, 20, 25, 30, 44.444, 49.9]) {
        const onde = C.localizar(ed, tEd);
        perto(C.tempoNaEdicao(ed, onde.clipe, onde.tempo), tEd, "ida e volta de " + tEd);
    }
    // Um tempo do vídeo fora do clipe cai na ponta mais próxima.
    eq(C.tempoNaEdicao(ed, 0, 10), 0);
    eq(C.tempoNaEdicao(ed, 0, 99), 20);
    eq(C.tempoNaEdicao(ed, 2, 5), 35);
    eq(C.tempoNaEdicao(ed, 9, 5), 0);
    eq(C.duracaoDoClipe(null), 0);
});

// ── dividir, juntar ────────────────────────────────────────────────────

t("dividir: o clipe sob o cursor vira dois, e nada mais muda", () => {
    const ed = um();
    const d = C.dividir(ed, 20);
    igual(clipes(d), [[0, 0, 20], [0, 20, 60]]);
    eq(C.duracao(d), 60, "dividir não muda a duração");
    igual(clipes(ed), [[0, 0, 60]], "a edição de antes não foi mexida");
    eq(d.fontes, ed.fontes, "as fontes são as mesmas");
    igual(clipes(tres()), [[0, 0, 20], [0, 20, 40], [0, 40, 60]]);
    // Num clipe aparado e fora de ordem, o corte cai no tempo certo do vídeo.
    const movido = C.mover(C.apararInicio(tres(), 2, 45), 2, 0);
    igual(clipes(movido), [[0, 45, 60], [0, 0, 20], [0, 20, 40]]);
    igual(clipes(C.dividir(movido, 5)), [[0, 45, 50], [0, 50, 60], [0, 0, 20], [0, 20, 40]]);
});

t("dividir: em cima de uma borda (ou colado nela) não faz nada, e devolve a mesma edição", () => {
    const ed = tres();
    for (const tEd of [0, 20, 40, 60, 0.02, 19.98, 20.03, 59.99, 100, -1])
        eq(C.dividir(ed, tEd), ed, "em " + tEd);
    eq(C.dividir(C.vazia(), 3).clipes.length, 0);
    igual(clipes(C.dividir(ed, 20.05)), [[0, 0, 20], [0, 20, 20.05], [0, 20.05, 40], [0, 40, 60]]);
});

t("juntar: só dois pedaços vizinhos e contíguos do mesmo vídeo", () => {
    const ed = tres();
    eq(C.podeJuntar(ed, 0), true);
    igual(clipes(C.juntar(ed, 0)), [[0, 0, 40], [0, 40, 60]]);
    igual(clipes(C.juntar(C.juntar(ed, 1), 0)), [[0, 0, 60]], "juntar tudo desfaz as divisões");
    // Aparado: já não é contíguo.
    const aparado = C.apararFim(ed, 0, 18);
    eq(C.podeJuntar(aparado, 0), false);
    eq(C.juntar(aparado, 0), aparado);
    // Fora de ordem: 20–40 seguido de 0–20 não é a continuação.
    const trocado = C.mover(ed, 0, 1);
    igual(clipes(trocado), [[0, 20, 40], [0, 0, 20], [0, 40, 60]]);
    eq(C.podeJuntar(trocado, 0), false);
    eq(C.podeJuntar(trocado, 1), false);
    // Vídeos diferentes, mesmo com os tempos batendo.
    const dois = C.inserirFonte(C.apararFim(um(), 0, 10), outra({ duracao_s: 30 }), 0);
    const comInicio = { fontes: dois.fontes, clipes: [{ fonte: 0, inicio: 0, fim: 10 }, { fonte: 1, inicio: 10, fim: 30 }] };
    eq(C.podeJuntar(comInicio, 0), false);
    // O último clipe não tem seguinte; índice torto não quebra.
    eq(C.podeJuntar(ed, 2), false);
    eq(C.juntar(ed, 2), ed);
    eq(C.juntar(ed, -1), ed);
});

// ── aparar ─────────────────────────────────────────────────────────────

t("aparar: encolhe, e cresce de volta até os limites do vídeo", () => {
    const ed = um();
    let a = C.apararInicio(ed, 0, 12.3456);
    igual(clipes(a), [[0, 12.346, 60]]);
    a = C.apararFim(a, 0, 50);
    igual(clipes(a), [[0, 12.346, 50]]);
    eq(C.duracao(a), 37.654);
    // De volta, e além: para nos limites do vídeo.
    igual(clipes(C.apararInicio(a, 0, 3)), [[0, 3, 50]]);
    igual(clipes(C.apararInicio(a, 0, -5)), [[0, 0, 50]]);
    igual(clipes(C.apararFim(a, 0, 58)), [[0, 12.346, 58]]);
    igual(clipes(C.apararFim(a, 0, 999)), [[0, 12.346, 60]]);
    igual(C.limites(a, 0), { min: 0, max: 60 });
});

t("aparar: as pontas nunca se cruzam (sobra o menor clipe)", () => {
    const ed = C.apararFim(C.apararInicio(um(), 0, 10), 0, 20);
    igual(clipes(C.apararInicio(ed, 0, 25)), [[0, 19.95, 20]]);
    igual(clipes(C.apararFim(ed, 0, 5)), [[0, 10, 10.05]]);
    eq(C.MINIMO, 0.05);
});

t("aparar: não invade o vizinho que é outro pedaço do mesmo vídeo", () => {
    // [0–20][20–40][40–60], com o do meio aparado para 25–35.
    let ed = C.apararFim(C.apararInicio(tres(), 1, 25), 1, 35);
    igual(clipes(ed), [[0, 0, 20], [0, 25, 35], [0, 40, 60]]);
    igual(C.limites(ed, 1), { min: 20, max: 40 });
    // O do meio cresce de volta só até os vizinhos.
    igual(clipes(C.apararInicio(ed, 1, 5)), [[0, 0, 20], [0, 20, 35], [0, 40, 60]]);
    igual(clipes(C.apararFim(ed, 1, 55)), [[0, 0, 20], [0, 25, 40], [0, 40, 60]]);
    // E os vizinhos crescem só até ele.
    igual(C.limites(ed, 0), { min: 0, max: 25 });
    igual(clipes(C.apararFim(ed, 0, 30)), [[0, 0, 25], [0, 25, 35], [0, 40, 60]]);
    igual(C.limites(ed, 2), { min: 35, max: 60 });
    igual(clipes(C.apararInicio(ed, 2, 0)), [[0, 0, 20], [0, 25, 35], [0, 35, 60]]);
    // Recém-divididos (colados): nenhum dos dois cresce para dentro do outro.
    ed = tres();
    eq(C.apararFim(ed, 0, 30), ed);
    eq(C.apararInicio(ed, 1, 10), ed);
});

t("aparar: o vizinho de outro vídeo, ou o mesmo vídeo fora de ordem, não limita", () => {
    const misto = C.inserirFonte(tres(), outra(), 0);
    igual(clipes(misto), [[0, 0, 20], [1, 0, 10], [0, 20, 40], [0, 40, 60]]);
    igual(C.limites(misto, 0), { min: 0, max: 60 }, "o vizinho é de outro vídeo");
    igual(C.limites(misto, 1), { min: 0, max: 10 });
    igual(C.limites(misto, 2), { min: 0, max: 40 }, "antes dele está o outro vídeo; depois, a continuação");
    // 20–40 antes de 0–20: o seguinte não é a continuação, então não limita.
    const trocado = C.mover(tres(), 0, 1);
    igual(C.limites(trocado, 0), { min: 0, max: 60 });
    igual(C.limites(trocado, 1), { min: 0, max: 40 }, "0–20 seguido de 40–60: cresce até 40");
    igual(C.limites(C.vazia(), 0), { min: 0, max: 0 });
});

t("aparar: sem mudança devolve a mesma edição; índice e tempo tortos não quebram", () => {
    const ed = um();
    eq(C.apararInicio(ed, 0, 0), ed);
    eq(C.apararFim(ed, 0, 60), ed);
    eq(C.apararFim(ed, 0, 75), ed);
    eq(C.apararInicio(ed, 3, 1), ed);
    eq(C.apararFim(ed, -1, 1), ed);
    eq(C.apararInicio(ed, 0, NaN), ed);
    eq(C.apararFim(ed, 0, "20"), ed);
});

t("trazerInicio / trazerFim: a ponta do clipe sob o cursor vem para o cursor", () => {
    const ed = tres();
    // Cursor em 25 s (5 s dentro do segundo clipe).
    igual(clipes(C.trazerInicio(ed, 25)), [[0, 0, 20], [0, 25, 40], [0, 40, 60]]);
    igual(clipes(C.trazerFim(ed, 25)), [[0, 0, 20], [0, 20, 25], [0, 40, 60]]);
    // Com o cursor na ponta não há o que trazer.
    eq(C.trazerInicio(ed, 20), ed);
    eq(C.trazerInicio(C.vazia(), 0).clipes.length, 0);
    eq(C.trazerFim(C.vazia(), 0).clipes.length, 0);
    // No fim da edição, o fim do último clipe já está lá.
    eq(C.trazerFim(ed, 60), ed);
});

// ── remover, mover, inserir ────────────────────────────────────────────

t("remover: tira o clipe e os outros fecham o espaço", () => {
    const ed = tres();
    const r = C.remover(ed, 1);
    igual(clipes(r), [[0, 0, 20], [0, 40, 60]]);
    eq(C.duracao(r), 40);
    igual(C.localizar(r, 20), { clipe: 1, tempo: 40, dentro: 0 }, "o que vinha depois do trecho removido toca em seguida");
    eq(C.remover(ed, 3), ed);
    eq(C.remover(ed, -1), ed);
    igual(clipes(C.remover(um(), 0)), []);
    eq(C.duracao(C.remover(um(), 0)), 0);
});

t("mover: para antes, para depois, e parando nas pontas", () => {
    const ed = tres();
    igual(clipes(C.mover(ed, 1, 0)), [[0, 20, 40], [0, 0, 20], [0, 40, 60]]);
    igual(clipes(C.mover(ed, 1, 2)), [[0, 0, 20], [0, 40, 60], [0, 20, 40]]);
    igual(clipes(C.mover(ed, 0, 2)), [[0, 20, 40], [0, 40, 60], [0, 0, 20]]);
    igual(clipes(C.mover(ed, 2, 0)), [[0, 40, 60], [0, 0, 20], [0, 20, 40]]);
    // O primeiro não vai para antes, o último não vai para depois.
    eq(C.mover(ed, 0, -1), ed);
    eq(C.mover(ed, 2, 3), ed);
    eq(C.mover(ed, 1, 1), ed);
    eq(C.mover(ed, 7, 0), ed);
    igual(clipes(C.mover(um(), 0, 1)), [[0, 0, 60]]);
    eq(C.duracao(C.mover(ed, 0, 2)), 60);
});

t("inserirFonte: o outro vídeo entra inteiro depois do clipe sob o cursor", () => {
    const ed = tres();
    const i = C.inserirFonte(ed, outra(), 25);
    igual(clipes(i), [[0, 0, 20], [0, 20, 40], [1, 0, 10], [0, 40, 60]]);
    eq(i.fontes.length, 2);
    eq(i.fontes[1].arquivo, "/v/em pé.mov");
    eq(C.duracao(i), 70);
    eq(ed.fontes.length, 1, "a edição de antes não ganhou a fonte");
    // No fim da edição: depois do último. Na edição vazia: é o primeiro.
    igual(clipes(C.inserirFonte(ed, outra(), 60)), [[0, 0, 20], [0, 20, 40], [0, 40, 60], [1, 0, 10]]);
    igual(clipes(C.inserirFonte(C.vazia(), outra(), 0)), [[0, 0, 10]]);
    // O mesmo arquivo de novo não repete a fonte.
    const dois = C.inserirFonte(i, outra(), 0);
    eq(dois.fontes.length, 2);
    igual(clipes(dois), [[0, 0, 20], [1, 0, 10], [0, 20, 40], [1, 0, 10], [0, 40, 60]]);
    const mesmo = C.inserirFonte(ed, midia(), 0);
    eq(mesmo.fontes.length, 1);
    igual(clipes(mesmo), [[0, 0, 20], [0, 0, 60], [0, 20, 40], [0, 40, 60]]);
    eq(C.indiceDaFonte(i, "/v/em pé.mov"), 1);
    eq(C.indiceDaFonte(i, "/v/nao.mp4"), -1);
    // O que o motor recusou não entra.
    eq(C.inserirFonte(ed, { erro: "não parece um vídeo" }, 0), ed);
});

// ── andar ──────────────────────────────────────────────────────────────

t("pular: as teclas 0–9 levam a 0%–90% da edição", () => {
    const ed = C.remover(tres(), 1); // 40 s
    igual([0, 1, 5, 9].map((n) => C.pular(ed, n)), [0, 4, 20, 36]);
    eq(C.pular(ed, 12), 36);
    eq(C.pular(ed, -1), 0);
    eq(C.pular(C.vazia(), 5), 0);
});

t("passo e quadro: andar sem sair da edição, e o tamanho de um quadro", () => {
    const ed = C.inserirFonte(um(), outra(), 60); // 60 s a 30 qps + 10 s a 60 qps
    eq(C.passo(ed, 10, 1), 11);
    eq(C.passo(ed, 10, -5), 5);
    eq(C.passo(ed, 0.4, -1), 0);
    eq(C.passo(ed, 69.5, 5), 70);
    eq(C.passo(ed, NaN, 1), 1);
    perto(C.quadro(ed, 10), 1 / 30);
    perto(C.quadro(ed, 65), 1 / 60);
    perto(C.quadro(C.vazia(), 0), 1 / 30);
    perto(C.passo(ed, 10, C.quadro(ed, 10)), 10.033);
});

t("bordas: a anterior e a seguinte ao cursor", () => {
    const ed = C.remover(tres(), 1); // bordas em 0, 20, 40
    igual(C.bordas(ed), [0, 20, 40]);
    igual(C.bordas(C.vazia()), [0]);
    eq(C.bordaSeguinte(ed, 0), 20);
    eq(C.bordaSeguinte(ed, 19.9), 20);
    eq(C.bordaSeguinte(ed, 20), 40, "em cima de uma borda, vai para a outra");
    eq(C.bordaSeguinte(ed, 40), 40);
    eq(C.bordaAnterior(ed, 40), 20);
    eq(C.bordaAnterior(ed, 20.5), 20);
    eq(C.bordaAnterior(ed, 20), 0);
    eq(C.bordaAnterior(ed, 0), 0);
    eq(C.bordaAnterior(C.vazia(), 5), 0);
    eq(C.bordaSeguinte(C.vazia(), 5), 0);
});

t("encaixar: gruda no alvo mais próximo dentro da tolerância", () => {
    eq(C.encaixar(19.8, [20, 40], 0.5), 20);
    eq(C.encaixar(19.4, [20, 40], 0.5), 19.4);
    eq(C.encaixar(30, [29.9, 30.05], 0.5), 30.05);
    eq(C.encaixar(30, [], 0.5), 30);
    eq(C.encaixar(30, null, 0.5), 30);
    eq(C.encaixar(30, [30.2], 0), 30);
});

t("alvosDeEncaixe: a borda do vizinho, as pontas do vídeo e o cursor (se está neste clipe)", () => {
    const ed = C.apararFim(C.apararInicio(tres(), 1, 25), 1, 35); // [0–20][25–35][40–60]
    // O cursor a 3 s do começo do segundo clipe (em 23 s da edição) está em 28 s do vídeo.
    igual(C.alvosDeEncaixe(ed, 1, 23), [20, 40, 28]);
    // Cursor em outro clipe, ou na ponta deste: só os limites.
    igual(C.alvosDeEncaixe(ed, 1, 5), [20, 40]);
    igual(C.alvosDeEncaixe(ed, 1, 20), [20, 40]);
    igual(C.alvosDeEncaixe(ed, 0, 5), [0, 25, 5]);
    igual(C.alvosDeEncaixe(ed, 9, 5), []);
    // Arrastando a alça do início do segundo clipe para perto do cursor ou do vizinho.
    eq(C.encaixar(27.8, C.alvosDeEncaixe(ed, 1, 23), 0.4), 28);
    eq(C.encaixar(20.3, C.alvosDeEncaixe(ed, 1, 23), 0.4), 20);
});

t("avancar: a pré-visualização pula o que não está em clipe e troca de vídeo", () => {
    // [0–20 do passeio][0–10 do em pé][40–60 do passeio].
    const ed = C.inserirFonte(C.remover(tres(), 1), outra(), 0);
    igual(clipes(ed), [[0, 0, 20], [1, 0, 10], [0, 40, 60]]);
    igual(C.avancar(ed, 0, 12), { clipe: 0, tempo: 12, salta: false, trocaFonte: false });
    igual(C.avancar(ed, 0, 19.99), { clipe: 1, tempo: 0, salta: true, trocaFonte: true });
    igual(C.avancar(ed, 1, 10.2), { clipe: 2, tempo: 40, salta: true, trocaFonte: true });
    igual(C.avancar(ed, 2, 59.98), { fim: true });
    igual(C.avancar(ed, 5, 0), { fim: true });
    // Um trecho removido do mesmo vídeo: salta sem trocar de vídeo.
    const buraco = C.remover(tres(), 1);
    igual(C.avancar(buraco, 0, 20.01), { clipe: 1, tempo: 40, salta: true, trocaFonte: false });
    // Dois pedaços contíguos (só divididos): segue tocando, sem saltar.
    igual(C.avancar(tres(), 0, 19.99), { clipe: 1, tempo: 20, salta: false, trocaFonte: false });
    igual(C.avancar(tres(), 0, 20.04), { clipe: 1, tempo: 20.04, salta: false, trocaFonte: false });
});

// ── desfazer e refazer ─────────────────────────────────────────────────

t("desfazer e refazer: uma pilha de estados", () => {
    const e0 = um(), e1 = C.dividir(e0, 20), e2 = C.remover(e1, 0);
    let h = C.historia(e0);
    eq(C.podeDesfazer(h), false);
    eq(C.podeRefazer(h), false);
    eq(C.desfazer(h), h);
    eq(C.refazer(h), h);
    h = C.aplicar(C.aplicar(h, e1), e2);
    eq(h.atual.ed, e2);
    eq(C.podeDesfazer(h), true);
    h = C.desfazer(h);
    eq(h.atual.ed, e1);
    eq(C.podeRefazer(h), true);
    h = C.desfazer(h);
    eq(h.atual.ed, e0);
    eq(C.podeDesfazer(h), false);
    h = C.refazer(C.refazer(h));
    eq(h.atual.ed, e2);
    eq(C.podeRefazer(h), false);
    // Uma mudança nova depois de desfazer apaga o que havia para refazer.
    h = C.desfazer(h);
    const e3 = C.apararFim(e1, 1, 50);
    h = C.aplicar(h, e3);
    eq(C.podeRefazer(h), false);
    eq(h.atual.ed, e3);
    eq(C.desfazer(h).atual.ed, e1);
    eq(C.podeDesfazer(null), false);
    eq(C.podeRefazer(null), false);
});

t("desfazer: operação sem efeito não vira passo, e a memória tem limite", () => {
    const e0 = tres();
    let h = C.historia(e0);
    eq(C.aplicar(h, C.dividir(e0, 20)), h, "dividir em cima da borda devolve a mesma edição");
    eq(C.aplicar(h, C.mover(e0, 0, -1)), h);
    let ed = e0;
    for (let i = 0; i < C.MEMORIA + 50; i++) {
        ed = C.apararFim(ed, 2, 59 - i * 0.05);
        h = C.aplicar(h, ed);
    }
    eq(h.passado.length, C.MEMORIA);
    eq(h.atual.ed, ed);
});

t("haMudancas: só o que ainda não foi exportado", () => {
    const e0 = um();
    let h = C.historia(e0);
    eq(C.haMudancas(h), false, "um vídeo recém-aberto não tem o que perder");
    eq(C.haMudancas(null), false);
    h = C.aplicar(h, C.dividir(e0, 20));
    eq(C.haMudancas(h), true);
    eq(C.haMudancas(C.desfazer(h)), false, "desfazer tudo volta ao estado limpo");
    h = C.marcarExportado(h);
    eq(C.haMudancas(h), false);
    const depois = C.aplicar(h, C.remover(h.atual.ed, 0));
    eq(C.haMudancas(depois), true);
    eq(C.haMudancas(C.desfazer(depois)), false);
    eq(C.haMudancas(C.desfazer(h)), true, "desfazer para antes do que foi exportado é uma mudança");
    eq(C.haMudancas(C.refazer(C.desfazer(h))), false);
});

// ── a linha do tempo ───────────────────────────────────────────────────

t("vista: a edição inteira cabe na linha; com zoom, o clipe sob o cursor ocupa 80%", () => {
    const ed = tres();
    igual(C.vista(ed, 0, 600, false), { escala: 10, desloca: 0 });
    // Zoom no segundo clipe (20 s): 480 px para ele, 60 px de cada lado.
    const v = C.vista(ed, 25, 600, true);
    eq(v.escala, 24);
    eq(v.desloca, 420);
    eq(C.xDoTempo(v, 20), 60);
    eq(C.xDoTempo(v, 40), 540);
    perto(C.tempoDoX(v, 60), 20);
    perto(C.tempoDoX(C.vista(ed, 0, 600, false), 300), 30);
    // No primeiro e no último clipe a linha não sai da edição.
    eq(C.vista(ed, 5, 600, true).desloca, 0);
    eq(C.vista(ed, 55, 600, true).desloca, 60 * 24 - 600);
    // Um clipe só já ocupa tudo: o zoom não encolhe.
    igual(C.vista(um(), 5, 600, true), { escala: 10, desloca: 0 });
    igual(C.vista(C.vazia(), 0, 600, true), { escala: 1, desloca: 0 });
    igual(C.vista(ed, 0, 0, false), { escala: 1, desloca: 0 });
});

t("miniaturaEm e fatias: cada pedaço da tira mostra o quadro mais próximo do meio dele", () => {
    const quadros = [{ tempo_s: 5, arquivo: "/c/q01.jpg" }, { tempo_s: 15, arquivo: "/c/q02.jpg" }, { tempo_s: 25, arquivo: "/c/q03.jpg" }];
    eq(C.miniaturaEm(quadros, 0), "/c/q01.jpg");
    eq(C.miniaturaEm(quadros, 14), "/c/q02.jpg");
    eq(C.miniaturaEm(quadros, 99), "/c/q03.jpg");
    eq(C.miniaturaEm([], 3), "");
    eq(C.miniaturaEm(null, 3), "");
    // Um clipe de 10 a 30 s com 300 px e fatias de 100: meios em 13,3, 20 e 26,7 s.
    igual(C.fatias({ fonte: 0, inicio: 10, fim: 30 }, 300, 100, quadros),
        [{ x: 0, largura: 100, arquivo: "/c/q02.jpg" }, { x: 100, largura: 100, arquivo: "/c/q02.jpg" }, { x: 200, largura: 100, arquivo: "/c/q03.jpg" }]);
    eq(C.fatias({ fonte: 0, inicio: 0, fim: 30 }, 20, 100, quadros).length, 1, "um clipe estreito tem uma fatia");
    igual(C.fatias({ fonte: 0, inicio: 0, fim: 30 }, 300, 100, []), [], "sem tira, sem fatias");
    igual(C.fatias({ fonte: 0, inicio: 0, fim: 30 }, 0, 100, quadros), []);
    eq(C.fatias({ fonte: 0, inicio: 0, fim: 30 }, 99999, 10, quadros).length, 60, "nunca mais que 60 imagens por clipe");
});

t("destinoDoArrasto: onde o clipe arrastado cai", () => {
    const ed = tres(); // meios em 10, 30 e 50
    eq(C.destinoDoArrasto(ed, 0, 5), 0);
    eq(C.destinoDoArrasto(ed, 0, 35), 1);
    eq(C.destinoDoArrasto(ed, 0, 55), 2);
    eq(C.destinoDoArrasto(ed, 2, 5), 0);
    eq(C.destinoDoArrasto(ed, 2, 25), 1);
    eq(C.destinoDoArrasto(ed, 1, 31), 1, "soltar em cima de si mesmo não move");
    igual(clipes(C.mover(ed, 0, C.destinoDoArrasto(ed, 0, 35))), [[0, 20, 40], [0, 0, 20], [0, 40, 60]]);
});

// ── o motor ────────────────────────────────────────────────────────────

t("paraOMotor: só os vídeos usados, na ordem em que aparecem, com os índices refeitos", () => {
    let ed = C.inserirFonte(tres(), outra(), 0);
    ed = C.inserirFonte(ed, midia({ arquivo: "/v/terceiro.mkv", duracao_s: 5 }), 0);
    // Tira os clipes do em pé e põe o terceiro vídeo na frente.
    ed = C.mover(C.remover(ed, 2), 1, 0);
    igual(clipes(ed), [[2, 0, 5], [0, 0, 20], [0, 20, 40], [0, 40, 60]]);
    igual(C.paraOMotor(ed, { qualidade: "720", modo: "rapido" }), {
        fontes: ["/v/terceiro.mkv", "/v/passeio.mp4"],
        clipes: [{ fonte: 0, inicio: 0, fim: 5 }, { fonte: 1, inicio: 0, fim: 20 }, { fonte: 1, inicio: 20, fim: 40 }, { fonte: 1, inicio: 40, fim: 60 }],
        qualidade: "720", modo: "rapido"
    });
    const padrao = C.paraOMotor(um());
    eq(padrao.qualidade, "original");
    eq(padrao.modo, "exato");
    eq(C.paraOMotor(um(), { qualidade: "4k", modo: "turbo" }).qualidade, "original");
    eq(C.paraOMotor(um(), { qualidade: "4k", modo: "turbo" }).modo, "exato");
    igual(C.paraOMotor(C.vazia()), { fontes: [], clipes: [], qualidade: "original", modo: "exato" });
});

t("os argumentos do motor: subcomandos e opções do contrato, o caminho depois de --", () => {
    const ruim = "/v/-rf 'a' \"b\": c.mp4";
    igual(C.argsSondar(ruim), ["sondar", "--", ruim]);
    igual(C.argsMiniaturas(ruim), ["miniaturas", "--quantas", "16", "--", ruim]);
    const ed = C.dividir(um(), 20);
    const plano = C.argsPlano(ed, { qualidade: "1080" });
    igual(plano.slice(0, 2), ["plano", "--json"]);
    igual(JSON.parse(plano[2]), C.paraOMotor(ed, { qualidade: "1080" }));
    const exportar = C.argsExportar(ed, { modo: "rapido" });
    igual(exportar.slice(0, 2), ["exportar", "--json"]);
    eq(JSON.parse(exportar[2]).modo, "rapido");
    eq(exportar.length, 3);
    for (const args of [C.argsSondar(ruim), C.argsMiniaturas(ruim), plano, exportar, C.argsQuadro(ed, 1)]) {
        assert.ok(contrato.includes("`mkcorte-motor " + args[0]), "subcomando fora do contrato: " + args[0]);
        for (const a of args.filter((x) => /^--[a-z]/.test(x)))
            assert.ok(contrato.includes(a), "opção fora do contrato: " + a);
    }
});

t("argsQuadro: o vídeo e o instante sob o cursor", () => {
    const ed = C.inserirFonte(C.remover(tres(), 0), outra(), 0); // [20–40][em pé 0–10][40–60]
    igual(C.argsQuadro(ed, 5), ["quadro", "--", "/v/passeio.mp4", "25"]);
    igual(C.argsQuadro(ed, 22.5), ["quadro", "--", "/v/em pé.mov", "2.5"]);
    // No fim do vídeo não há quadro: recua um.
    igual(C.argsQuadro(ed, 50), ["quadro", "--", "/v/passeio.mp4", "59.967"]);
    eq(C.argsQuadro(C.vazia(), 0), null);
});

t("arquivoDeExemplo e modoPossivel", () => {
    eq(C.arquivoDeExemplo(["sondar", "--", "/a.mp4"]), "sondar.json");
    eq(C.arquivoDeExemplo(["exportar", "--json", "{}"]), "exportar.json");
    eq(C.arquivoDeExemplo([]), ".json");
    eq(C.modoPossivel({ sem_recodificar: { pode: true } }, "rapido"), "rapido");
    eq(C.modoPossivel({ sem_recodificar: { pode: false } }, "rapido"), "exato");
    eq(C.modoPossivel({ sem_recodificar: { pode: true } }, "exato"), "exato");
    eq(C.modoPossivel(null, "rapido"), "exato");
    eq(C.modoPossivel({ erro: "x" }, "rapido"), "exato");
});

t("os dados de exemplo passam pela lógica da janela como os do motor", () => {
    const ed = C.abrirFonte(exemplo("sondar"));
    eq(ed.clipes.length, 1);
    assert.ok(C.duracao(ed) > 0);
    assert.ok(C.resumoDaFonte(ed.fontes[0]).includes("×"));
    const tira = exemplo("miniaturas");
    eq(typeof C.miniaturaEm(tira.quadros, 3), "string");
    const plano = exemplo("plano");
    assert.ok(C.textoDoPlano(plano).includes("≈"));
    eq(C.modoPossivel(plano, "rapido"), plano.sem_recodificar.pode ? "rapido" : "exato");
    const fases = exemplo("exportar");
    for (const f of fases.filter((x) => x.fase === "exportando"))
        assert.ok(/^\d+%/.test(C.textoDoProgresso(f)));
});

// ── textos ─────────────────────────────────────────────────────────────

t("tempo, resta e tamanho", () => {
    eq(C.tempo(0), "0:00.0");
    eq(C.tempo(7.25), "0:07.2");
    eq(C.tempo(62.4), "1:02.4");
    eq(C.tempo(59.96), "0:59.9");
    eq(C.tempo(3725), "1:02:05.0");
    eq(C.tempo(null), "–");
    eq(C.tempo(NaN), "–");
    eq(C.resta(0.2), "1 s");
    eq(C.resta(59), "59 s");
    eq(C.resta(61), "2 min");
    eq(C.resta(3900), "1 h 05 min");
    eq(C.resta(null), "");
    eq(C.tamanho(0), "0 B");
    eq(C.tamanho(1536), "1,5 KB");
    eq(C.tamanho(48000000), "45,8 MB");
    eq(C.tamanho(500 * 1024 * 1024), "500 MB");
    eq(C.tamanho(null), "–");
});

t("nome, pasta e as URLs: nomes com espaço, #, ?, % e acento vão e voltam inteiros", () => {
    eq(C.nome("/v/a b.mp4"), "a b.mp4");
    eq(C.nome("solto.mp4"), "solto.mp4");
    eq(C.nome(null), "");
    eq(C.pasta("/v/sub/a.mp4"), "/v/sub");
    eq(C.pasta("/a.mp4"), "/");
    eq(C.caminhoDeUrl("file:///v/a%20b.mp4"), "/v/a b.mp4");
    eq(C.caminhoDeUrl("file://localhost/v/a.mp4"), "/v/a.mp4");
    eq(C.caminhoDeUrl("/v/já é caminho.mp4"), "/v/já é caminho.mp4");
    eq(C.caminhoDeUrl("https://exemplo.org/a.mp4"), "");
    eq(C.caminhoDeUrl("file:///v/%E0%A4%A.mp4"), "");
    eq(C.caminhoDeUrl(null), "");
    eq(C.urlDeCaminho("/v/a b#1?x%.mp4"), "file:///v/a%20b%231%3Fx%25.mp4");
    eq(C.urlDeCaminho("relativo.mp4"), "");
    eq(C.urlDeCaminho(null), "");
    for (const caminho of ["/v/-rf 'a' \"b\": c.mp4", "/v/férias #2 (100%)?.mkv", "/v/日本 語.webm"])
        eq(C.caminhoDeUrl(C.urlDeCaminho(caminho)), caminho);
});

t("lerArquivos: a lista do lançador, só caminhos absolutos", () => {
    igual(C.lerArquivos('["/a b.mp4","relativo.mp4",3,"/c.mkv"]'), ["/a b.mp4", "/c.mkv"]);
    igual(C.lerArquivos(""), []);
    igual(C.lerArquivos("{"), []);
    igual(C.lerArquivos('"/a.mp4"'), []);
});

t("resumoDaFonte, textoDoPlano e textoDoProgresso", () => {
    eq(C.resumoDaFonte(um().fontes[0]), "1920×1080  ·  30 qps  ·  1:00.0  ·  com áudio");
    eq(C.resumoDaFonte(C.fonteDe(outra({ fps: 29.97 }))), "1080×1920  ·  29,97 qps  ·  0:10.0  ·  sem áudio");
    eq(C.resumoDaFonte(null), "");
    eq(C.textoDoPlano({ duracao_s: 38, tamanho_estimado_bytes: 12 * 1024 * 1024, largura: 1280, altura: 720, tem_audio: true }), "0:38.0  ·  ≈ 12,0 MB  ·  1280×720");
    eq(C.textoDoPlano({ duracao_s: 5, tamanho_estimado_bytes: 1024, largura: 640, altura: 360, tem_audio: false }), "0:05.0  ·  ≈ 1,0 KB  ·  640×360  ·  sem áudio");
    eq(C.textoDoPlano({ erro: "não achei" }), "");
    eq(C.textoDoPlano(null), "");
    eq(C.textoDoProgresso({ pct: 42.7, resta_s: 75, velocidade: 2.5 }), "42%  ·  resta 2 min  ·  2,5×");
    eq(C.textoDoProgresso({ pct: 3, resta_s: null, velocidade: null }), "3%");
    eq(C.textoDoProgresso(null), "começando…");
});

// ── o teclado ──────────────────────────────────────────────────────────

const K = C.TECLA, S = C.SHIFT, CT = C.CTRL, A = C.ALT;
const acao = (tecla, mods) => C.acaoDaTecla(tecla, mods || 0);

t("os números das teclas e dos modificadores são os do Qt", () => {
    igual([C.SHIFT, C.CTRL, C.ALT], [0x02000000, 0x04000000, 0x08000000]);
    igual([K.espaco, K.esquerda, K.direita, K.cima, K.baixo], [0x20, 0x01000012, 0x01000014, 0x01000013, 0x01000015]);
    igual([K.abre, K.fecha, K.apagar, K.esc, K.enter, K.retorno, K.tab, K.tabVolta], [0x5b, 0x5d, 0x01000007, 0x01000000, 0x01000005, 0x01000004, 0x01000001, 0x01000002]);
    igual([K.F, K.J, K.O, K.Q, K.S, K.X, K.Z, K.interrogacao, K.f1], [0x46, 0x4a, 0x4f, 0x51, 0x53, 0x58, 0x5a, 0x3f, 0x01000030]);
});

t("acaoDaTecla: todos os atalhos da janela", () => {
    igual(acao(K.espaco), { acao: "tocar" });
    igual(acao(K.espaco, CT), { acao: "trazerInicio" });
    igual(acao(K.espaco, CT | S), { acao: "trazerFim" });
    for (let n = 0; n <= 9; n++)
        igual(acao(0x30 + n), { acao: "pular", n: n });
    // Com o teclado numérico (o Qt marca com outro modificador) também.
    igual(acao(0x35, 0x20000000), { acao: "pular", n: 5 });
    igual(acao(K.esquerda), { acao: "andar", delta: -1 });
    igual(acao(K.direita), { acao: "andar", delta: 1 });
    igual(acao(K.esquerda, S), { acao: "andar", delta: -5 });
    igual(acao(K.direita, S), { acao: "andar", delta: 5 });
    igual(acao(K.esquerda, A), { acao: "quadro", delta: -1 });
    igual(acao(K.direita, A), { acao: "quadro", delta: 1 });
    igual(acao(K.abre), { acao: "bordaAnterior" });
    igual(acao(K.fecha), { acao: "bordaSeguinte" });
    igual(acao(K.abre, A), { acao: "moverAntes" });
    igual(acao(K.fecha, A), { acao: "moverDepois" });
    igual(acao(K.S), { acao: "dividir" });
    igual(acao(K.X), { acao: "remover" });
    igual(acao(K.apagar), { acao: "remover" });
    igual(acao(K.J), { acao: "juntar" });
    igual(acao(K.Z, CT), { acao: "desfazer" });
    igual(acao(K.Z, CT | S), { acao: "refazer" });
    igual(acao(K.Z), { acao: "zoom" });
    igual(acao(K.F), { acao: "salvarQuadro" });
    igual(acao(K.O, CT), { acao: "abrir" });
    igual(acao(K.O, CT | S), { acao: "adicionar" });
    igual(acao(K.S, CT), { acao: "exportar" });
    igual(acao(K.Q), { acao: "sair" });
    igual(acao(K.interrogacao, S), { acao: "atalhos" });
    igual(acao(K.interrogacao), { acao: "atalhos" });
    igual(acao(K.f1), { acao: "atalhos" });
});

t("acaoDaTecla: o que não é atalho não faz nada (nem as combinações que sobram)", () => {
    for (const [tecla, mods] of [[0x41, 0], [K.Q, CT], [K.X, CT], [K.F, A], [K.S, S], [K.S, CT | S], [K.O, 0], [K.Z, A], [K.Z, CT | A], [0x35, CT],
        [K.esquerda, CT], [K.esquerda, A | S], [K.abre, CT], [K.abre, S], [K.apagar, S], [K.espaco, A], [K.enter, 0], [K.esc, 0], [K.J, CT]])
        eq(acao(tecla, mods), null, "tecla 0x" + tecla.toString(16) + " com " + mods.toString(16));
});

t("ATALHOS: a lista que o ? mostra cobre todas as ações", () => {
    eq(C.ATALHOS.length, 21);
    for (const [teclas, texto] of C.ATALHOS) {
        assert.ok(teclas.length > 0 && texto.length > 3, teclas);
        eq(texto.charAt(0), texto.charAt(0).toLowerCase(), "minúscula no começo: " + texto);
    }
    const lista = C.ATALHOS.map((a) => a[0]).join(" | ");
    for (const tecla of ["Espaço", "0 – 9", "Shift + ← →", "Alt + ← →", "[  ]", "S", "X ou Delete", "Alt + [  ]", "Ctrl + Espaço", "Ctrl + Shift + Espaço",
        "Ctrl + Z", "Ctrl + Shift + Z", "Z", "F", "Ctrl + O", "Ctrl + Shift + O", "Ctrl + S", "Q", "?"])
        assert.ok(lista.includes(tecla), "falta na lista: " + tecla);
});

t("acaoDoDialogo e girar: as confirmações respondem a Enter, Esc, Tab e setas", () => {
    eq(C.acaoDoDialogo(K.enter, 0), "confirmar");
    eq(C.acaoDoDialogo(K.retorno, 0), "confirmar");
    eq(C.acaoDoDialogo(K.espaco, 0), "confirmar");
    eq(C.acaoDoDialogo(K.esc, 0), "cancelar");
    eq(C.acaoDoDialogo(K.tab, 0), "proximo");
    eq(C.acaoDoDialogo(K.tab, S), "anterior");
    eq(C.acaoDoDialogo(K.tabVolta, S), "anterior");
    eq(C.acaoDoDialogo(K.direita, 0), "proximo");
    eq(C.acaoDoDialogo(K.esquerda, 0), "anterior");
    eq(C.acaoDoDialogo(K.cima, 0), "cima");
    eq(C.acaoDoDialogo(K.baixo, 0), "baixo");
    eq(C.acaoDoDialogo(K.Q, 0), null);
    igual([C.girar(0, 3, 1), C.girar(2, 3, 1), C.girar(0, 3, -1), C.girar(1, 3, -1), C.girar(0, 0, 1)], [1, 0, 2, 0, 0]);
});

// ── o ponteiro ─────────────────────────────────────────────────────────

t("toque: as alças nas pontas, o × no canto, o corpo do clipe e o vazio", () => {
    const ed = tres(), v = C.vista(ed, 0, 600, false); // clipes em 0–200, 200–400, 400–600 px
    igual(C.toque(ed, v, 100, 40), { tipo: "clipe", clipe: 0 });
    igual(C.toque(ed, v, 3, 40), { tipo: "inicio", clipe: 0 });
    igual(C.toque(ed, v, 195, 40), { tipo: "fim", clipe: 0 });
    // Na emenda: à esquerda dela é o fim de um, à direita o início do outro.
    igual(C.toque(ed, v, 199, 40), { tipo: "fim", clipe: 0 });
    igual(C.toque(ed, v, 200, 40), { tipo: "inicio", clipe: 1 });
    igual(C.toque(ed, v, 208, 40), { tipo: "inicio", clipe: 1 });
    igual(C.toque(ed, v, 209, 40), { tipo: "clipe", clipe: 1 });
    igual(C.toque(ed, v, 600, 40), { tipo: "fim", clipe: 2 }, "a última ponta também tem alça");
    // O × fica no canto de cima, à esquerda da alça do fim.
    igual(C.toque(ed, v, 380, 10), { tipo: "remover", clipe: 1 });
    igual(C.toque(ed, v, 380, 40), { tipo: "clipe", clipe: 1 });
    igual(C.toque(ed, v, 395, 10), { tipo: "fim", clipe: 1 });
    igual(C.toque(ed, v, -5, 40), { tipo: "nada", clipe: -1 });
    igual(C.toque(ed, v, 601, 40), { tipo: "nada", clipe: -1 });
    igual(C.toque(C.vazia(), v, 100, 40), { tipo: "nada", clipe: -1 });
    eq(C.ALCA, 8);
});

t("toque: num clipe estreito as alças encolhem, sobra corpo e não há ×", () => {
    // Um clipe de 1 s numa edição de 41 s em 600 px: uns 14,6 px.
    const ed = C.remover(C.dividir(C.dividir(tres(), 21), 20), 3);
    igual(clipes(ed), [[0, 0, 20], [0, 20, 21], [0, 21, 40]]);
    const v = C.vista(ed, 0, 600, false), x0 = C.xDoTempo(v, 20), x1 = C.xDoTempo(v, 21);
    igual(C.toque(ed, v, x0 + 1, 40), { tipo: "inicio", clipe: 1 });
    igual(C.toque(ed, v, (x0 + x1) / 2, 40), { tipo: "clipe", clipe: 1 });
    igual(C.toque(ed, v, x1 - 1, 40), { tipo: "fim", clipe: 1 });
    igual(C.toque(ed, v, (x0 + x1) / 2, 10), { tipo: "clipe", clipe: 1 }, "sem × num clipe estreito");
    // Com zoom, as coordenadas seguem a vista deslocada.
    const z = C.vista(ed, 20.5, 600, true);
    igual(C.toque(ed, z, 300, 40), { tipo: "clipe", clipe: 1 });
    igual(C.toque(ed, z, C.xDoTempo(z, 20) + 2, 40), { tipo: "inicio", clipe: 1 });
});

t("marcaDoArrasto: a emenda em que o clipe arrastado vai cair", () => {
    const ed = tres(), v = C.vista(ed, 0, 600, false);
    eq(C.marcaDoArrasto(ed, v, 0, 5), 0, "fica onde está");
    eq(C.marcaDoArrasto(ed, v, 0, 35), 400, "o primeiro vai para depois do segundo");
    eq(C.marcaDoArrasto(ed, v, 0, 55), 600);
    eq(C.marcaDoArrasto(ed, v, 2, 5), 0);
    eq(C.marcaDoArrasto(ed, v, 2, 25), 200);
    eq(C.marcaDoArrasto(ed, v, 1, 31), 200);
});
