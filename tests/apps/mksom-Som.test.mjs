// apps/mksom/qml/Som.js: a lógica da janela do MkSom que não depende do QML:
// os passos dos controles, a curva resultante, o aviso de distorção, o ritmo
// das aplicações ao arrastar, as predefinições, o medidor e o diagnóstico do
// microfone, os textos e os atalhos.
import { readFileSync } from "node:fs";
import { t, eq, igual, perto, assert } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const S = load(REPO + "/apps/mksom/qml/Som.js");
const exemplo = (nome) => JSON.parse(readFileSync(REPO + "/tests/apps/mksom-exemplo/" + nome + ".json", "utf8"));
const zeros = () => [0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
const com = (i, v) => { const g = zeros(); g[i] = v; return g; };
const quase = (obtido, esperado, folga, msg) =>
    assert.ok(Math.abs(obtido - esperado) <= folga, (msg || "") + " esperado ~" + esperado + ", obtido " + obtido);

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["prender", "noPasso", "ganho", "pre", "mover", "daPosicao", "paraPosicao", "noTrilho", "andar", "trocar", "plano", "mesmos", "rotulo",
        "textoDoGanho", "textoDb", "tipoDaBanda", "resposta", "somaEm", "curva", "pico", "sobra", "podeDistorcer", "preSugerido",
        "avisoDeDistorcao", "pontos", "xDaFrequencia", "pedido", "devoAplicar", "esperaPara", "predefinicaoDe", "acha", "daPessoa",
        "aparado", "erroDoNome", "fracao", "picoQueCai", "junta", "maiorPico", "entradaDe", "entradaInicial", "comEntrada", "comPadrao", "volume",
        "textoDoVolume", "diagnostico", "textoDaGravacao", "estreita", "baixa", "area", "textoDaSaida", "arquivoDeExemplo"];
    const funcoes = Object.keys(S).filter((k) => typeof S[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("as bandas e os limites são os do contrato (e os do motor)", () => {
    igual(S.BANDAS, [31, 62, 125, 250, 500, 1000, 2000, 4000, 8000, 16000]);
    igual([S.GANHO_MIN, S.GANHO_MAX, S.PRE_MIN, S.PRE_MAX, S.PASSO, S.PASSO_GRANDE, S.Q], [-12, 12, -12, 0, 0.5, 3, 1]);
    igual(exemplo("estado").eq.bandas, S.BANDAS);
    igual(S.plano(), zeros());
});

t("prender, noPasso, ganho e pre: sempre um valor que o motor aceita", () => {
    eq(S.prender(5, 0, 3), 3);
    eq(S.prender(-5, 0, 3), 0);
    eq(S.prender("x", -12, 12), 0);
    eq(S.prender(NaN, 2, 9), 2);
    eq(S.noPasso(3.24), 3);
    eq(S.noPasso(3.26), 3.5);
    eq(S.noPasso(-0.2), 0);
    assert.ok(Object.is(S.noPasso(-0.2), 0), "sem −0");
    eq(S.ganho(99), 12);
    eq(S.ganho(-99), -12);
    eq(S.ganho(2.74), 2.5);
    eq(S.ganho(undefined), 0);
    eq(S.pre(3), 0);
    eq(S.pre(-99), -12);
    eq(S.pre(-2.8), -3);
});

t("mover: ↑/↓ meio dB, Page Up/Down 3 dB, 0 zera, sem passar dos limites", () => {
    eq(S.mover(0, "cima"), 0.5);
    eq(S.mover(0, "baixo"), -0.5);
    eq(S.mover(1, "pagina_cima"), 4);
    eq(S.mover(1, "pagina_baixo"), -2);
    eq(S.mover(7.5, "zero"), 0);
    eq(S.mover(12, "cima"), 12);
    eq(S.mover(11, "pagina_cima"), 12);
    eq(S.mover(-12, "baixo"), -12);
    eq(S.mover(3, "outra"), 3);
    // O pré-ganho não passa de 0.
    eq(S.mover(0, "cima", true), 0);
    eq(S.mover(-1, "pagina_cima", true), 0);
    eq(S.mover(-11, "pagina_baixo", true), -12);
    eq(S.mover(-4, "zero", true), 0);
});

t("daPosicao e paraPosicao: o alto do trilho é o máximo, o pé o mínimo", () => {
    eq(S.daPosicao(0, 200, -12, 12), 12);
    eq(S.daPosicao(200, 200, -12, 12), -12);
    eq(S.daPosicao(100, 200, -12, 12), 0);
    eq(S.daPosicao(-50, 200, -12, 12), 12);
    eq(S.daPosicao(999, 200, -12, 12), -12);
    eq(S.daPosicao(71, 200, -12, 12), 3.5);
    eq(S.daPosicao(10, 0, -12, 12), 0);
    eq(S.daPosicao(0, 100, -12, 0), 0);
    eq(S.daPosicao(100, 100, -12, 0), -12);
    eq(S.paraPosicao(12, 200, -12, 12), 0);
    eq(S.paraPosicao(-12, 200, -12, 12), 200);
    eq(S.paraPosicao(0, 200, -12, 12), 100);
    eq(S.paraPosicao(99, 200, -12, 12), 0);
    // Ida e volta em todos os passos.
    for (let g = -12; g <= 12; g += 0.5)
        eq(S.daPosicao(S.paraPosicao(g, 173, -12, 12), 173, -12, 12), g);
});

t("noTrilho e andar: os controles deitados (pré-ganho e volume), no passo e nos limites", () => {
    eq(S.noTrilho(0, 200, -12, 0, 0.5), -12);
    eq(S.noTrilho(200, 200, -12, 0, 0.5), 0);
    eq(S.noTrilho(100, 200, -12, 0, 0.5), -6);
    eq(S.noTrilho(51, 200, -12, 0, 0.5), -9);
    eq(S.noTrilho(-10, 200, 0, 150, 1), 0);
    eq(S.noTrilho(999, 200, 0, 150, 1), 150);
    eq(S.noTrilho(99, 200, 0, 150, 1), 74);
    eq(S.noTrilho(5, 0, 0, 150, 1), 0);
    eq(S.andar(74, 1, 0, 150, 1), 75);
    eq(S.andar(74, -10, 0, 150, 1), 64);
    eq(S.andar(149, 10, 0, 150, 1), 150);
    eq(S.andar(3, -10, 0, 150, 1), 0);
    eq(S.andar(-0.5, 0.5, -12, 0, 0.5), 0);
    assert.ok(Object.is(S.andar(-0.5, 0.5, -12, 0, 0.5), 0), "sem −0");
    eq(S.andar(-11, -3, -12, 0, 0.5), -12);
    eq(S.andar(-3, 3, -12, 0, 0.5), 0);
});

t("trocar devolve uma lista nova, com o valor no passo", () => {
    const g = zeros();
    const novo = S.trocar(g, 3, 4.26);
    eq(novo[3], 4.5);
    eq(g[3], 0);
    assert.notStrictEqual(novo, g);
    igual(S.trocar(g, 99, 4), g);
    igual(S.trocar(g, -1, 4), g);
});

t("mesmos", () => {
    eq(S.mesmos([1, 2], [1, 2]), true);
    eq(S.mesmos([1, 2], [1, 2.5]), false);
    eq(S.mesmos([1, 2], [1]), false);
    eq(S.mesmos(null, [1]), false);
    eq(S.mesmos([0], [-0]), true);
});

t("rotulo: inteiro quando cabe, curto quando a janela estreita", () => {
    igual(S.BANDAS.map((f) => S.rotulo(f)), ["31 Hz", "62 Hz", "125 Hz", "250 Hz", "500 Hz", "1 kHz", "2 kHz", "4 kHz", "8 kHz", "16 kHz"]);
    igual(S.BANDAS.map((f) => S.rotulo(f, true)), ["31", "62", "125", "250", "500", "1k", "2k", "4k", "8k", "16k"]);
    eq(S.rotulo(1500), "1,5 kHz");
    eq(S.rotulo(1500, true), "1,5k");
});

t("textoDoGanho e textoDb: sinal, vírgula e o menos de verdade", () => {
    eq(S.textoDoGanho(3.5), "+3,5");
    eq(S.textoDoGanho(-2), "−2");
    eq(S.textoDoGanho(0), "0");
    eq(S.textoDoGanho(-0), "0");
    eq(S.textoDoGanho(12), "+12");
    eq(S.textoDoGanho("x"), "0");
    eq(S.textoDb(-4.5), "−4,5 dB");
    eq(S.textoDb(0), "0 dB");
});

t("tipoDaBanda: prateleira nas pontas, sino no meio", () => {
    igual(S.BANDAS.map((_, i) => S.tipoDaBanda(i)),
        ["lowshelf", "peaking", "peaking", "peaking", "peaking", "peaking", "peaking", "peaking", "peaking", "highshelf"]);
});

t("resposta: o sino dá o ganho inteiro no centro e some longe dele", () => {
    quase(S.resposta("peaking", 1000, 6, 1000), 6, 1e-6);
    quase(S.resposta("peaking", 1000, -9, 1000), -9, 1e-6);
    eq(S.resposta("peaking", 1000, 0, 1000), 0);
    // Uma oitava ao lado, bem menos; três oitavas, quase nada.
    const lado = S.resposta("peaking", 1000, 6, 2000);
    assert.ok(lado > 1 && lado < 3, "uma oitava: " + lado);
    assert.ok(Math.abs(S.resposta("peaking", 1000, 6, 125)) < 0.3);
    // Simétrico em dB: cortar é o espelho de levantar.
    perto(S.resposta("peaking", 500, -6, 700), -S.resposta("peaking", 500, 6, 700));
});

t("resposta: as prateleiras dão metade do ganho no canto e o ganho inteiro além dele", () => {
    quase(S.resposta("lowshelf", 250, 6, 250), 3, 0.01);
    quase(S.resposta("lowshelf", 250, 6, 20), 6, 0.1);
    quase(S.resposta("lowshelf", 250, 6, 8000), 0, 0.05);
    quase(S.resposta("highshelf", 4000, 6, 4000), 3, 0.01);
    quase(S.resposta("highshelf", 4000, 6, 20000), 6, 0.1);
    quase(S.resposta("highshelf", 4000, 6, 100), 0, 0.05);
    quase(S.resposta("lowshelf", 31, -8, 31), -4, 0.01);
});

t("somaEm e curva: plano dá zero; o pré-ganho desce tudo junto", () => {
    eq(S.somaEm(zeros(), 1000), 0);
    eq(S.somaEm(null, 1000), 0);
    quase(S.somaEm(com(5, 6), 1000), 6, 1e-6);
    // Vizinhas somam: duas bandas em +6 dão mais de 6 entre elas.
    const g = zeros(); g[4] = 6; g[5] = 6;
    assert.ok(S.somaEm(g, 707) > 6.5, "soma entre as bandas: " + S.somaEm(g, 707));
    const c = S.curva(zeros(), 0, 48);
    eq(c.length, 48);
    perto(c[0].f, 20);
    quase(c[47].f, 20000, 1e-6);
    assert.ok(c.every((p) => p.db === 0));
    assert.ok(c.every((p, i) => i === 0 || p.f > c[i - 1].f), "frequências crescem");
    assert.ok(S.curva(zeros(), -4, 8).every((p) => p.db === -4));
    eq(S.curva(zeros(), 0, 1).length, 2);
    eq(S.curva(zeros(), 0).length, 64);
});

t("pico, sobra e podeDistorcer: os ganhos positivos contra o pré-ganho", () => {
    eq(S.pico(zeros()), 0);
    // Só cortes: nada a compensar.
    eq(S.pico([-6, -4, -2, 0, 0, 0, 0, -1, -3, -5]), 0);
    quase(S.pico(com(5, 6)), 6, 0.01);
    eq(S.podeDistorcer(com(5, 6), 0), true);
    eq(S.podeDistorcer(com(5, 6), -6), false);
    eq(S.podeDistorcer(com(5, 6), -5.5), true);
    quase(S.sobra(com(5, 6), -2), 4, 0.01);
    eq(S.podeDistorcer(zeros(), 0), false);
    eq(S.podeDistorcer(com(5, -6), 0), false);
    // Tudo no máximo: o pico passa (bem) de 12.
    assert.ok(S.pico(S.BANDAS.map(() => 12)) > 12);
});

t("preSugerido: o pré-ganho que cobre o pico, no passo e dentro do limite", () => {
    eq(S.preSugerido(zeros()), 0);
    eq(S.preSugerido(com(5, 6)), -6);
    eq(S.preSugerido(com(5, 0.5)), -0.5);
    eq(S.preSugerido(com(5, -6)), 0);
    eq(S.preSugerido(S.BANDAS.map(() => 12)), -12);
    for (const g of [com(0, 4), com(9, 7.5), [3, 0, 2, 0, 5, 0, 1, 0, 4, 0], [1, 1, 1, 1, 1, 1, 1, 1, 1, 1]]) {
        const p = S.preSugerido(g);
        eq(S.podeDistorcer(g, p), false, "com o sugerido não distorce: " + g);
        eq(S.podeDistorcer(g, p + 0.5), true, "meio dB acima já distorce: " + g);
    }
});

t("avisoDeDistorcao: diz para onde baixar; vazio quando cabe", () => {
    eq(S.avisoDeDistorcao(zeros(), 0), "");
    eq(S.avisoDeDistorcao(com(5, 6), -6), "");
    eq(S.avisoDeDistorcao(com(5, 6), 0), "pode distorcer: baixe o pré-ganho para −6 dB");
    eq(S.avisoDeDistorcao(com(5, 6), -3), "pode distorcer: baixe o pré-ganho para −6 dB");
    // Nem o mínimo resolve: não manda baixar para onde já está.
    eq(S.avisoDeDistorcao(S.BANDAS.map(() => 12), -12), "pode distorcer: os ganhos passam do limite do pré-ganho");
});

t("as predefinições de fábrica vêm com o pré-ganho justo: nenhuma avisa, nenhuma sobra", () => {
    const fabrica = exemplo("estado").eq.predefinicoes.filter((p) => p.fabrica);
    igual(fabrica.map((p) => p.nome), ["plano", "graves", "agudos", "voz", "noite", "fone pequeno"]);
    for (const p of fabrica) {
        eq(S.podeDistorcer(p.ganhos, p.pre), false, p.nome + " avisa");
        eq(p.pre, S.preSugerido(p.ganhos), p.nome + ": o pré-ganho não é o sugerido");
    }
});

t("pontos e xDaFrequencia: a curva dentro do desenho, 0 dB no meio", () => {
    const p = S.pontos(zeros(), 0, 300, 80, 31);
    eq(p.length, 31);
    eq(p[0].x, 0);
    eq(p[30].x, 300);
    assert.ok(p.every((q) => q.y === 40));
    // Levantar sobe (y menor); o que passa da escala fica na borda.
    const alto = S.pontos(S.BANDAS.map(() => 12), 0, 300, 80, 31);
    assert.ok(alto.every((q) => q.y >= 0 && q.y <= 40));
    assert.ok(alto.some((q) => q.y === 0));
    const baixo = S.pontos(S.BANDAS.map(() => -12), -12, 300, 80, 31);
    assert.ok(baixo.every((q) => q.y <= 80 && q.y >= 40));
    eq(S.xDaFrequencia(20, 300), 0);
    perto(S.xDaFrequencia(20000, 300), 300);
    quase(S.xDaFrequencia(632.5, 300), 150, 0.1);
    eq(S.xDaFrequencia(1, 300), 0);
});

t("pedido: o que vai para o eq ajustar é sempre válido", () => {
    igual(S.pedido([3, 2.5, 0, -1, 0, 0.5, 0, -12, 12, 4], -4.5), { ganhos: [3, 2.5, 0, -1, 0, 0.5, 0, -12, 12, 4], pre: -4.5 });
    igual(S.pedido([99, 0.26], 5), { ganhos: [12, 0.5, 0, 0, 0, 0, 0, 0, 0, 0], pre: 0 });
    igual(S.pedido(null, null), { ganhos: zeros(), pre: 0 });
});

t("devoAplicar e esperaPara: no arraste, no máximo 10 aplicações por segundo e uma de cada vez", () => {
    eq(S.INTERVALO_MS, 100);
    eq(S.devoAplicar(1000, 0, false), true);
    eq(S.devoAplicar(1000, 950, false), false);
    eq(S.devoAplicar(1000, 900, false), true);
    eq(S.devoAplicar(1000, 0, true), false);
    eq(S.esperaPara(1000, 950), 50);
    eq(S.esperaPara(1000, 900), 0);
    eq(S.esperaPara(1000, 0), 0);
    eq(S.esperaPara(1000, 2000), 100);
    // Um arraste de 1 s com um evento a cada 5 ms: 10 aplicações, não 200.
    let ultima = -1000, aplicadas = 0;
    for (let agora = 0; agora < 1000; agora += 5)
        if (S.devoAplicar(agora, ultima, false)) { aplicadas++; ultima = agora; }
    eq(aplicadas, 10);
});

const lista = [
    { nome: "plano", fabrica: true, ganhos: zeros(), pre: 0 },
    { nome: "graves", fabrica: true, ganhos: [4, 3.5, 2.5, 1, 0, 0, 0, 0, 0, 0], pre: -5 },
    { nome: "Minha", fabrica: false, ganhos: com(0, 2), pre: -2 }
];

t("predefinicaoDe, acha e daPessoa", () => {
    eq(S.predefinicaoDe(lista, zeros(), 0), "plano");
    eq(S.predefinicaoDe(lista, com(0, 2), -2), "Minha");
    eq(S.predefinicaoDe(lista, com(0, 2), -1), "");
    eq(S.predefinicaoDe(null, zeros(), 0), "");
    eq(S.acha(lista, "MINHA").pre, -2);
    eq(S.acha(lista, "outra"), null);
    eq(S.acha(null, "x"), null);
    igual(S.daPessoa(lista).map((p) => p.nome), ["Minha"]);
    igual(S.daPessoa(null), []);
});

t("erroDoNome: as regras do motor, e as de fábrica não se trocam", () => {
    for (const bom of ["minha", "Fone do trabalho", "rock & jazz (2)", "água", "é".repeat(40), "Minha", "  com espaços  "])
        eq(S.erroDoNome(bom, lista), "", bom);
    eq(S.erroDoNome("", lista), "dê um nome à predefinição");
    eq(S.erroDoNome("   ", lista), "dê um nome à predefinição");
    eq(S.erroDoNome(null, lista), "dê um nome à predefinição");
    eq(S.erroDoNome("x".repeat(41), lista), "o nome pode ter até 40 caracteres");
    eq(S.erroDoNome("a/b", lista), "o nome não pode ter barra");
    eq(S.erroDoNome("a\\b", lista), "o nome não pode ter barra");
    eq(S.erroDoNome("-rf", lista), "o nome não pode começar com - nem ser só pontos");
    eq(S.erroDoNome("..", lista), "o nome não pode começar com - nem ser só pontos");
    eq(S.erroDoNome("a\nb", lista), "o nome não pode ter caractere de controle");
    eq(S.erroDoNome("a\u007fb", lista), "o nome não pode ter caractere de controle");
    eq(S.erroDoNome("a\n", lista), "o nome não pode ter caractere de controle");
    eq(S.erroDoNome("\t", lista), "o nome não pode ter caractere de controle");
    eq(S.aparado("  a b  "), "a b");
    eq(S.aparado("a\u00a0"), "a\u00a0");
    eq(S.erroDoNome("Graves", lista), "“Graves” é uma predefinição de fábrica: escolha outro nome");
});

t("fracao: de −60 dBFS (vazio) a 0 (cheio)", () => {
    eq(S.fracao(0), 1);
    eq(S.fracao(-60), 0);
    eq(S.fracao(-90), 0);
    eq(S.fracao(-30), 0.5);
    eq(S.fracao(3), 1);
    eq(S.fracao("x"), 0);
});

t("picoQueCai: sobe na hora, cai devagar, nunca abaixo do piso", () => {
    eq(S.picoQueCai(-30, -10, 66), -10);
    perto(S.picoQueCai(-10, -50, 1000), -28);
    perto(S.picoQueCai(-10, -50, 100), -11.8);
    eq(S.picoQueCai(-10, -11, 1000), -11);
    eq(S.picoQueCai(-89, -90, 5000), -90);
    eq(S.picoQueCai(undefined, -20, 0), -20);
    eq(S.picoQueCai(-10, undefined, 0), -10);
});

t("junta e maiorPico: os últimos 2 s de níveis", () => {
    let n = [];
    for (let i = 0; i < 40; i++)
        n = S.junta(n, { pico: -40 + i, rms: -50 });
    eq(n.length, S.JANELA);
    eq(n[0].pico, -30);
    eq(S.maiorPico(n), -1);
    eq(S.maiorPico([]), -90);
    eq(S.maiorPico(null), -90);
    // O que não é um nível (o {"fim": true}, um erro) não entra.
    igual(S.junta(n, { fim: true }), n);
    igual(S.junta(null, null), []);
    const antes = n;
    assert.notStrictEqual(S.junta(n, { pico: 0, rms: 0 }), antes);
});

const entradas = [
    { id: 52, nome: "a", descricao: "Microfone interno", padrao: false, volume: 74, mudo: false },
    { id: 53, nome: "b", descricao: "Microfone USB", padrao: true, volume: 100, mudo: true }
];

t("entradaDe, entradaInicial, comEntrada, comPadrao", () => {
    eq(S.entradaDe(entradas, 53).descricao, "Microfone USB");
    eq(S.entradaDe(entradas, 9), null);
    eq(S.entradaDe(null, 9), null);
    eq(S.entradaInicial(entradas), 53);
    eq(S.entradaInicial([entradas[0]]), 52);
    eq(S.entradaInicial([]), -1);
    eq(S.entradaInicial(null), -1);
    const novas = S.comEntrada(entradas, 52, { volume: 30, mudo: true });
    igual([novas[0].volume, novas[0].mudo, novas[1].volume], [30, true, 100]);
    eq(entradas[0].volume, 74);
    igual(S.comPadrao(entradas, 52).map((e) => e.padrao), [true, false]);
    igual(entradas.map((e) => e.padrao), [false, true]);
    igual(S.comPadrao(null, 1), []);
});

t("volume e textoDoVolume", () => {
    eq(S.volume(74.4), 74);
    eq(S.volume(999), 150);
    eq(S.volume(-3), 0);
    eq(S.volume("x"), 0);
    eq(S.textoDoVolume(74), "74%");
    eq(S.textoDoVolume(null), "—");
});

const niveis = (n, pico) => Array.from({ length: n }, () => ({ pico, rms: pico - 6 }));

t("diagnostico: as frases do contrato, a partir dos níveis", () => {
    igual(S.diagnostico([], null, [], false), { texto: "nenhuma entrada", problema: true });
    igual(S.diagnostico(null, null, [], false), { texto: "nenhuma entrada", problema: true });
    igual(S.diagnostico(entradas, null, [], true), { texto: "escolha uma entrada", problema: false });
    // Muda: nem olha os níveis.
    igual(S.diagnostico(entradas, entradas[1], niveis(30, -10), true), { texto: "a entrada está muda", problema: true });
    igual(S.diagnostico(entradas, entradas[0], niveis(30, -70), true),
        { texto: "o nível está muito baixo (fale perto / suba o volume)", problema: true });
    igual(S.diagnostico(entradas, entradas[0], niveis(30, -41), true).problema, true);
    igual(S.diagnostico(entradas, entradas[0], niveis(30, -18), true), { texto: "o nível está bom", problema: false });
    igual(S.diagnostico(entradas, entradas[0], niveis(30, -0.5), true), { texto: "está saturando (baixe o volume)", problema: true });
    // Saturar aparece na hora; "baixo" espera juntar níveis.
    igual(S.diagnostico(entradas, entradas[0], niveis(1, 0), true).texto, "está saturando (baixe o volume)");
    igual(S.diagnostico(entradas, entradas[0], niveis(3, -70), true), { texto: "ouvindo…", problema: false });
    igual(S.diagnostico(entradas, entradas[0], [], false), { texto: "o medidor está parado", problema: false });
    // Um pico bom no meio de silêncio (a pessoa falou): está bom.
    igual(S.diagnostico(entradas, entradas[0], niveis(20, -70).concat(niveis(1, -15)), true).texto, "o nível está bom");
});

t("o exemplo do medidor passa pelas três situações", () => {
    const fluxo = exemplo("mic-nivel");
    assert.ok(fluxo.length >= 20);
    eq(fluxo[fluxo.length - 1].fim, true);
    let n = [];
    const frases = new Set();
    for (const linha of fluxo) {
        n = S.junta(n, linha);
        frases.add(S.diagnostico(entradas, entradas[0], n, true).texto);
    }
    assert.ok(frases.has("ouvindo…") && frases.has("o nível está bom"), [...frases].join(" | "));
});

t("textoDaGravacao", () => {
    eq(S.textoDaGravacao("", 0), "gravar e ouvir");
    eq(S.textoDaGravacao("gravando", 4), "gravando… 4 s");
    eq(S.textoDaGravacao("gravando", undefined), "gravando… 0 s");
    eq(S.textoDaGravacao("ouvindo", 0), "tocando o que gravou…");
});

t("estreita e baixa: a janela cabe em 420×420", () => {
    eq(S.estreita(420), true);
    eq(S.estreita(479), true);
    eq(S.estreita(480), false);
    eq(S.estreita(560), false);
    eq(S.baixa(420), true);
    eq(S.baixa(520), false);
});

t("area: só as duas", () => {
    eq(S.area("microfone"), "microfone");
    eq(S.area("equalizador"), "equalizador");
    eq(S.area(""), "equalizador");
    eq(S.area("outra"), "equalizador");
});

t("textoDaSaida: por onde o som sai", () => {
    const e = exemplo("estado");
    eq(S.textoDaSaida(e), "equalizando Alto-falantes");
    eq(S.textoDaSaida(null), "o PipeWire não respondeu");
    eq(S.textoDaSaida({ som: { responde: false, erro: "o PipeWire não respondeu (x)" } }), "o PipeWire não respondeu (x)");
    const som = { responde: true, erro: null };
    const padrao = { id: 51, nome: "a", descricao: "Alto-falantes" };
    eq(S.textoDaSaida({ som, eq: { ligado: false }, saida_padrao: padrao }), "saída: Alto-falantes");
    eq(S.textoDaSaida({ som, eq: { ligado: false }, saida_padrao: null }), "nenhuma saída de som");
    eq(S.textoDaSaida({ som, eq: { ligado: true, na_saida: false }, saida_padrao: padrao }), "ligado, mas a saída padrão é outra: Alto-falantes");
    eq(S.textoDaSaida({ som, eq: { ligado: true, na_saida: true, saida_anterior: null } }), "equalizando a saída");
    eq(S.textoDaSaida({ som, eq: { ligado: true, na_saida: true, saida_anterior: { nome: "x", descricao: null, existe: false } } }), "equalizando a saída");
});

t("arquivoDeExemplo: um arquivo por subcomando, sem o que a pessoa escolheu", () => {
    eq(S.arquivoDeExemplo(["estado"]), "estado.json");
    eq(S.arquivoDeExemplo(["eq", "ligar"]), "eq-ligar.json");
    eq(S.arquivoDeExemplo(["eq", "desligar"]), "eq-desligar.json");
    eq(S.arquivoDeExemplo(["eq", "ajustar"]), "eq-ajustar.json");
    eq(S.arquivoDeExemplo(["eq", "predefinicao", "aplicar", "--", "graves"]), "eq-predefinicao-aplicar.json");
    eq(S.arquivoDeExemplo(["eq", "predefinicao", "renomear", "--", "a", "b"]), "eq-predefinicao-renomear.json");
    eq(S.arquivoDeExemplo(["mic", "listar"]), "mic-listar.json");
    eq(S.arquivoDeExemplo(["mic", "volume", "52", "74"]), "mic-volume.json");
    eq(S.arquivoDeExemplo(["mic", "mudo", "52", "sim"]), "mic-mudo.json");
    eq(S.arquivoDeExemplo(["mic", "nivel", "52"]), "mic-nivel.json");
    eq(S.arquivoDeExemplo(["mic", "ouvir"]), "mic-ouvir.json");
    // Um nome de predefinição com cara de caminho não escolhe arquivo.
    eq(S.arquivoDeExemplo(["eq", "predefinicao", "aplicar", "--", "../../etc/passwd"]), "eq-predefinicao-aplicar.json");
    eq(S.arquivoDeExemplo(["mic", "padrao", "../x"]), "mic-padrao.json");
});

t("os atalhos da folha são pares [tecla, o que faz]", () => {
    assert.ok(S.ATALHOS.length >= 10);
    for (const a of S.ATALHOS) {
        eq(a.length, 2);
        assert.ok(a[0] !== "" && a[1] !== "");
    }
    for (const tecla of ["Tab", "↑ ↓", "Page Up / Down", "0", "Espaço", "Ctrl+1", "Ctrl+2", "F1", "Esc"])
        assert.ok(S.ATALHOS.some((a) => a[0] === tecla), "falta " + tecla);
});
