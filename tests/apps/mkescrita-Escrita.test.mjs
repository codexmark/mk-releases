// apps/mkescrita/qml/Escrita.js: a lógica da janela do MkEscrita que não
// depende do QML: as edições de Markdown (alternar marcação, continuar lista,
// recuo, link), buscar e substituir, a máquina de estados do salvamento, o
// que fazer ao fechar e os textos do rodapé.
import { t, eq, igual, assert } from "../js/harness.mjs";
import { load, source, REPO } from "../js/load.mjs";

const arquivo = REPO + "/apps/mkescrita/qml/Escrita.js";
const E = load(arquivo);

// Um texto com a seleção marcada: "|" é o cursor; "[" e "]" as pontas da seleção.
function marcado(s) {
    const cursor = s.indexOf("|");
    if (cursor >= 0)
        return { texto: s.replace("|", ""), ini: cursor, fim: cursor };
    const ini = s.indexOf("[");
    const fim = s.indexOf("]") - 1;
    return { texto: s.replace("[", "").replace("]", ""), ini, fim };
}

// Aplica uma edição e devolve o resultado na mesma notação.
function depois(m, e) {
    if (e === null)
        return null;
    const texto = E.aplicar(m.texto, e);
    if (e.selInicio === e.selFim)
        return texto.slice(0, e.selInicio) + "|" + texto.slice(e.selInicio);
    return texto.slice(0, e.selInicio) + "[" + texto.slice(e.selInicio, e.selFim) + "]" + texto.slice(e.selFim);
}

const marca = (s, m) => { const x = marcado(s); return depois(x, E.alternarMarca(x.texto, x.ini, x.fim, m)); };
const enter = (s) => { const x = marcado(s); return depois(x, E.aoEnter(x.texto, x.ini, x.fim)); };
const tab = (s, menos) => { const x = marcado(s); return depois(x, E.recuar(x.texto, x.ini, x.fim, !!menos)); };
const doc = (extra) => Object.assign({ fase: "limpo", pendente: false, caminho: "", recuperado: false, soLeitura: false, vazio: false }, extra || {});

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["edicao", "aplicar", "linhaEm", "linhaDe", "posicaoDaLinha", "paragrafoEm", "letra", "liga", "palavraEm",
        "corridaAntes", "corridaDepois", "temMarca", "alternarMarca", "envolver", "inserirLink", "itemDe", "aoEnter", "recuar",
        "expressao", "ocorrencias", "proxima", "atual", "substituirUma", "substituirTodas", "textoDaBusca", "estadoInicial",
        "transicao", "temMudancas", "deveSalvarSozinho", "deveGuardarRascunho", "aoFechar", "respostaDaTecla", "aoMudarPorFora",
        "ondeAbrir", "nome", "pasta", "textoDoEstado", "tituloDaJanela", "milhar", "textoDasPalavras", "tempoDeLeitura", "haQuanto",
        "fraseDoConflito", "zoom", "bytesUtf8", "argsSalvar", "argsTexto", "argsRascunho", "argsExportar", "nomeDoExportado", "idDeRascunho",
        "rascunhoParaOferecer", "caminhoDeUrl", "urlDaPasta", "linkAbrivel", "arquivoDeExemplo", "tituloAtual", "filtrarFontes",
        "estiloDaPrevia", "atalhos"];
    const funcoes = Object.keys(E).filter((k) => typeof E[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("o arquivo só usa JavaScript que o motor do Qt entende (sem \\p{…}, sem olhar para trás, sem trimStart)", () => {
    const fonte = source(arquivo).split("\n").filter((l) => !/^\s*\/\//.test(l)).join("\n");
    for (const proibido of ["\\p{", "(?<", ".trimStart(", ".trimEnd(", ".replaceAll(", ".at(", "?.", "??", ".matchAll(", ".flat("])
        assert.ok(!fonte.includes(proibido), "usa " + proibido);
    assert.match(source(arquivo), /^\.pragma library\n/);
});

// --- pedaços do texto -----------------------------------------------------------

t("linhaEm, linhaDe e posicaoDaLinha: a linha do cursor, o número dela e a volta", () => {
    const s = "um\ndois\n\nquatro";
    igual(E.linhaEm(s, 0), { inicio: 0, fim: 2, texto: "um" });
    igual(E.linhaEm(s, 2), { inicio: 0, fim: 2, texto: "um" });
    igual(E.linhaEm(s, 3), { inicio: 3, fim: 7, texto: "dois" });
    igual(E.linhaEm(s, 8), { inicio: 8, fim: 8, texto: "" });
    igual(E.linhaEm(s, s.length), { inicio: 9, fim: 15, texto: "quatro" });
    igual(E.linhaEm("", 0), { inicio: 0, fim: 0, texto: "" });
    igual([0, 2, 3, 7, 8, 9, 15].map((p) => E.linhaDe(s, p)), [1, 1, 2, 2, 3, 4, 4]);
    igual([1, 2, 3, 4, 5, 99].map((n) => E.posicaoDaLinha(s, n)), [0, 3, 8, 9, 15, 15]);
    eq(E.posicaoDaLinha(s, 0), 0);
    for (let n = 1; n <= 4; n++)
        eq(E.linhaDe(s, E.posicaoDaLinha(s, n)), n, "ida e volta da linha " + n);
});

t("paragrafoEm: as linhas seguidas em volta do cursor; numa linha em branco, só ela", () => {
    const s = "um\ndois\n\ntrês\n\n\nfim";
    igual(E.paragrafoEm(s, 0), { inicio: 0, fim: 7 });
    igual(E.paragrafoEm(s, 5), { inicio: 0, fim: 7 });
    igual(E.paragrafoEm(s, 8), { inicio: 8, fim: 8 });
    igual(E.paragrafoEm(s, 10), { inicio: 9, fim: 13 });
    igual(E.paragrafoEm(s, s.length), { inicio: 16, fim: 19 });
    igual(E.paragrafoEm("", 0), { inicio: 0, fim: 0 });
});

t("palavraEm: letras com acento, algarismos e o hífen do meio; marcação e espaço não são palavra", () => {
    const s = "a ação-reação d'água **x** - 42";
    igual(E.palavraEm(s, 3), { inicio: 2, fim: 13 });
    igual(E.palavraEm(s, 2), { inicio: 2, fim: 13 });
    igual(E.palavraEm(s, 13), { inicio: 2, fim: 13 });
    igual(E.palavraEm(s, 16), { inicio: 14, fim: 20 });
    igual(E.palavraEm(s, 23), { inicio: 23, fim: 24 });
    eq(E.palavraEm(s, 22), null, "entre os asteriscos");
    eq(E.palavraEm(s, 28), null, "no traço solto");
    igual(E.palavraEm(s, s.length), { inicio: 29, fim: 31 });
    eq(E.palavraEm("", 0), null);
    eq(E.letra("é"), true);
    eq(E.letra("7"), true);
    eq(E.letra("*"), false);
    eq(E.letra(undefined), false);
    eq(E.liga("a-b", 1), true);
    eq(E.liga("a- b", 1), false);
    eq(E.liga("- b", 0), false);
});

t("corridas e temMarca: o itálico está em * e em ***, não em **", () => {
    eq(E.corridaAntes("a***", 4, "*"), 3);
    eq(E.corridaAntes("a***", 1, "*"), 0);
    eq(E.corridaDepois("***a", 0, "*"), 3);
    eq(E.corridaDepois("***a", 3, "*"), 0);
    igual([0, 1, 2, 3, 4].map((n) => E.temMarca(n, "*")), [false, true, false, true, true]);
    igual([0, 1, 2, 3].map((n) => E.temMarca(n, "**")), [false, false, true, true]);
    igual([0, 1, 2].map((n) => E.temMarca(n, "~~")), [false, false, true]);
});

// --- alternar marcação ----------------------------------------------------------

t("alternarMarca: põe negrito, itálico e tachado na seleção, que continua selecionada", () => {
    eq(marca("um [texto] aqui", "**"), "um **[texto]** aqui");
    eq(marca("um [texto] aqui", "*"), "um *[texto]* aqui");
    eq(marca("um [texto] aqui", "~~"), "um ~~[texto]~~ aqui");
    eq(marca("[tudo]", "**"), "**[tudo]**");
    eq(marca("[duas palavras]", "*"), "*[duas palavras]*");
});

t("alternarMarca: os espaços das pontas da seleção ficam fora das marcas", () => {
    eq(marca("um[ texto ]aqui", "**"), "um **[texto]** aqui");
    eq(marca("[texto ]aqui", "*"), "*[texto]* aqui");
});

t("alternarMarca: desfaz a marcação que já existe, em volta da seleção ou dentro dela", () => {
    eq(marca("um **[texto]** aqui", "**"), "um [texto] aqui");
    eq(marca("um [**texto**] aqui", "**"), "um [texto] aqui");
    eq(marca("um *[texto]* aqui", "*"), "um [texto] aqui");
    eq(marca("um [*texto*] aqui", "*"), "um [texto] aqui");
    eq(marca("um ~~[texto]~~ aqui", "~~"), "um [texto] aqui");
    eq(marca("um [~~texto~~] aqui", "~~"), "um [texto] aqui");
});

t("alternarMarca: ida e volta devolve o texto e a seleção de antes", () => {
    for (const m of ["**", "*", "~~"]) {
        for (const s of ["um [texto] aqui", "[a]", "x [duas palavras] y", "começo [é] fim"]) {
            const ida = marca(s, m);
            eq(marca(ida, m), s, m + " em " + s);
        }
    }
});

t("alternarMarca: negrito e itálico não se confundem", () => {
    // Itálico num trecho em negrito soma (***), não tira o negrito.
    eq(marca("**[texto]**", "*"), "***[texto]***");
    // E tirar o itálico de *** deixa o negrito.
    eq(marca("***[texto]***", "*"), "**[texto]**");
    // Tirar o negrito de *** deixa o itálico.
    eq(marca("***[texto]***", "**"), "*[texto]*");
    // Negrito num trecho em itálico soma.
    eq(marca("*[texto]*", "**"), "***[texto]***");
});

t("alternarMarca: sem seleção, vale para a palavra sob o cursor, e o cursor fica na mesma letra", () => {
    eq(marca("uma pala|vra aqui", "**"), "uma **pala|vra** aqui");
    eq(marca("uma **pala|vra** aqui", "**"), "uma pala|vra aqui");
    eq(marca("uma |palavra aqui", "*"), "uma *|palavra* aqui");
    eq(marca("uma palavra| aqui", "~~"), "uma ~~palavra|~~ aqui");
    eq(marca("guarda-chu|va", "**"), "**guarda-chu|va**");
    eq(marca("açã|o", "*"), "*açã|o*");
    // Cursor encostado nas marcas, do lado de fora da palavra.
    eq(marca("uma **palavra|** aqui", "**"), "uma palavra| aqui");
    eq(marca("uma **|palavra** aqui", "**"), "uma |palavra aqui");
});

t("alternarMarca: sem seleção e sem palavra, abre o par com o cursor no meio; de novo, desfaz", () => {
    eq(marca("um | dois", "**"), "um **|** dois");
    eq(marca("um **|** dois", "**"), "um | dois");
    eq(marca("|", "*"), "*|*");
    eq(marca("*|*", "*"), "|");
    eq(marca("|", "~~"), "~~|~~");
    eq(marca("~~|~~", "~~"), "|");
    // O traço de uma lista não é palavra: não vira "**-**".
    eq(marca("-| item", "**"), "-**|** item");
});

t("alternarMarca: a seleção ao contrário (do fim para o começo) dá no mesmo", () => {
    const e1 = E.alternarMarca("um texto aqui", 3, 8, "**");
    const e2 = E.alternarMarca("um texto aqui", 8, 3, "**");
    igual(e1, e2);
});

t("envolver: digitar *, _ ou ` com seleção envolve o trecho; sem seleção, a tecla segue normal", () => {
    const x = marcado("um [texto] aqui");
    eq(depois(x, E.envolver(x.texto, x.ini, x.fim, "*")), "um *[texto]* aqui");
    eq(depois(x, E.envolver(x.texto, x.ini, x.fim, "_")), "um _[texto]_ aqui");
    eq(depois(x, E.envolver(x.texto, x.ini, x.fim, "`")), "um `[texto]` aqui");
    eq(depois(x, E.envolver(x.texto, x.fim, x.ini, "*")), "um *[texto]* aqui", "seleção ao contrário");
    eq(E.envolver(x.texto, x.ini, x.fim, "a"), null);
    eq(E.envolver(x.texto, x.ini, x.fim, "("), null);
    eq(E.envolver(x.texto, 3, 3, "*"), null);
    // Duas vezes dá negrito.
    const y = marcado("um *[texto]* aqui");
    eq(depois(y, E.envolver(y.texto, y.ini, y.fim, "*")), "um **[texto]** aqui");
});

t("inserirLink: a seleção vira o texto do link e o cursor vai para o endereço", () => {
    const link = (s) => { const x = marcado(s); return depois(x, E.inserirLink(x.texto, x.ini, x.fim)); };
    eq(link("veja [aqui] já"), "veja [aqui](|) já");
    eq(link("veja | já"), "veja [|]() já");
    eq(link("veja [https://exemplo.org/a] já"), "veja [|](https://exemplo.org/a) já");
    eq(link("[www.exemplo.org]"), "[|](www.exemplo.org)");
    eq(link("[duas palavras]"), "[duas palavras](|)");
    eq(link("[http://a com espaço]"), "[http://a com espaço](|)");
});

// --- listas ----------------------------------------------------------------------

t("itemDe: reconhece marcador, número, tarefa e citação", () => {
    eq(E.itemDe("- a").marca, "-");
    eq(E.itemDe("  * a").recuo, "  ");
    eq(E.itemDe("12. a").numero, 12);
    eq(E.itemDe("3) a").fecho, ")");
    eq(E.itemDe("- [ ] a").tarefa, true);
    eq(E.itemDe("- [x] a").prefixo, "- [x] ");
    eq(E.itemDe("- a").tarefa, false);
    eq(E.itemDe("> a").citacao, true);
    eq(E.itemDe("> > a").prefixo, "> > ");
    eq(E.itemDe("-").prefixo, "-");
    for (const naoE of ["texto", "-colado", "1.colado", "**negrito**", "", "#. x", "1986 foi", "[ ] solto"])
        eq(E.itemDe(naoE), null, naoE);
});

t("aoEnter: continua a lista com a mesma marca", () => {
    eq(enter("- um|"), "- um\n- |");
    eq(enter("* um|"), "* um\n* |");
    eq(enter("+ um|"), "+ um\n+ |");
    eq(enter("  - aninhado|"), "  - aninhado\n  - |");
    eq(enter("antes\n- um|\ndepois"), "antes\n- um\n- |\ndepois");
});

t("aoEnter: a lista numerada segue com o número seguinte e o mesmo fecho", () => {
    eq(enter("1. um|"), "1. um\n2. |");
    eq(enter("9. nove|"), "9. nove\n10. |");
    eq(enter("3) três|"), "3) três\n4) |");
    eq(enter("   1. aninhado|"), "   1. aninhado\n   2. |");
});

t("aoEnter: tarefa nova nasce desmarcada", () => {
    eq(enter("- [ ] fazer|"), "- [ ] fazer\n- [ ] |");
    eq(enter("- [x] feito|"), "- [x] feito\n- [ ] |");
    eq(enter("1. [X] feito|"), "1. [X] feito\n2. [ ] |");
});

t("aoEnter: no meio do item, o resto da linha desce para o item novo, sem espaço sobrando", () => {
    eq(enter("- um |dois"), "- um\n- |dois");
    eq(enter("- um|dois"), "- um\n- |dois");
    eq(enter("- um|  dois"), "- um\n- |dois");
});

t("aoEnter: num item vazio encerra a lista; aninhado, volta um nível", () => {
    eq(enter("- um\n- |"), "- um\n|");
    eq(enter("- um\n-|"), "- um\n|");
    eq(enter("1. um\n2. |"), "1. um\n|");
    eq(enter("- [ ] um\n- [ ] |"), "- [ ] um\n|");
    eq(enter("- um\n  - |"), "- um\n- |");
    eq(enter("1. um\n   1. |"), "1. um\n1. |");
    eq(enter("- um\n- |\ndepois"), "- um\n|\ndepois");
});

t("aoEnter: citação continua e encerra do mesmo jeito", () => {
    eq(enter("> dito|"), "> dito\n> |");
    eq(enter("> dito\n> |"), "> dito\n|");
    eq(enter(">colado|"), ">colado\n> |");
});

t("aoEnter: fora de lista, com seleção ou antes da marca, o Enter segue normal", () => {
    eq(enter("texto comum|"), null);
    eq(enter("|"), null);
    eq(enter("# Título|"), null);
    eq(enter("|- um"), null);
    eq(enter("-| um"), null);
    eq(enter("- [um]"), null);
    eq(enter("**negrito**|"), null);
});

t("recuar: Tab aninha o item (2 espaços; 3 na numerada) e o cursor acompanha", () => {
    eq(tab("- um|"), "  - um|");
    eq(tab("- |um"), "  - |um");
    eq(tab("|- um"), "  |- um");
    eq(tab("1. um|"), "   1. um|");
    eq(tab("- a\n- b|\n- c"), "- a\n  - b|\n- c");
    eq(tab("- [ ] tarefa|"), "  - [ ] tarefa|");
});

t("recuar: Shift+Tab volta um nível e não passa da margem", () => {
    eq(tab("  - um|", true), "- um|");
    eq(tab("    - um|", true), "  - um|");
    eq(tab(" - um|", true), "- um|");
    eq(tab("   1. um|", true), "1. um|");
    eq(tab("- um|", true), null);
    eq(tab(" |- um", true), "|- um");
});

t("recuar: com várias linhas selecionadas, mexe em todas as que são item e mantém a seleção", () => {
    // A seleção que começa no começo da linha passa a incluir o recuo.
    eq(tab("- a\n[- b\n- c]\n- d"), "- a\n[  - b\n  - c]\n- d");
    eq(tab("- a\n  [- b\n  - c]\n- d", true), "- a\n[- b\n- c]\n- d");
    eq(tab("[- a\ntexto\n- c]"), "[  - a\ntexto\n  - c]");
    // A seleção que acaba no começo da linha seguinte não a inclui.
    eq(tab("[- a\n]- b"), "[  - a\n]- b");
});

t("recuar: fora de lista não faz nada (a tecla segue normal)", () => {
    eq(tab("texto comum|"), null);
    eq(tab("|"), null);
    eq(tab("> citação|"), null);
    eq(tab("[um\ndois]"), null);
    eq(tab("  texto recuado|", true), null);
});

t("recuar e aoEnter: ida e volta devolve o texto", () => {
    for (const s of ["- um|", "1. um|", "- a\n[- b\n- c]", "x\n- |y"])
        eq(tab(tab(s), true), s, s);
});

// --- buscar e substituir ----------------------------------------------------------

t("ocorrencias: as posições de cada uma, sem diferenciar maiúsculas", () => {
    igual(E.ocorrencias("Casa casa CASA casaco", "casa"), [{ inicio: 0, fim: 4 }, { inicio: 5, fim: 9 }, { inicio: 10, fim: 14 }, { inicio: 15, fim: 19 }]);
    igual(E.ocorrencias("AÇÃO ação Ação", "ação").map((o) => o.inicio), [0, 5, 10]);
    igual(E.ocorrencias("nada aqui", "xyz"), []);
    igual(E.ocorrencias("texto", ""), []);
    igual(E.ocorrencias("", "a"), []);
});

t("ocorrencias: com a opção de diferenciar, só a grafia exata", () => {
    igual(E.ocorrencias("Casa casa CASA", "casa", true), [{ inicio: 5, fim: 9 }]);
    igual(E.ocorrencias("Casa casa CASA", "CASA", true), [{ inicio: 10, fim: 14 }]);
    igual(E.ocorrencias("Ação ação", "ação", true), [{ inicio: 5, fim: 9 }]);
});

t("ocorrencias: o termo é texto, não expressão regular; e não se sobrepõe", () => {
    igual(E.ocorrencias("a.b a+b a*b (x) [y] a\\b $1 ^", ".").length, 1);
    igual(E.ocorrencias("1+1=2", "1+1").length, 1);
    igual(E.ocorrencias("f(x) e f(y)", "f(").map((o) => o.inicio), [0, 7]);
    igual(E.ocorrencias("[ ] e [x]", "[ ]").length, 1);
    igual(E.ocorrencias("a\\b", "\\").length, 1);
    igual(E.ocorrencias("custa $5", "$5").length, 1);
    igual(E.ocorrencias("a-b a/b", "a-b").length, 1);
    igual(E.ocorrencias("a-b a/b", "a/b").length, 1);
    igual(E.ocorrencias("aaaa", "aa").map((o) => o.inicio), [0, 2]);
    igual(E.ocorrencias("um\ndois\num", "um").map((o) => o.inicio), [0, 8]);
    assert.ok(E.expressao("a.b", false) instanceof RegExp);
    eq(E.expressao("a.b", true).flags.includes("i"), false);
});

t("ocorrencias: para de contar num limite (uma letra num texto enorme)", () => {
    const n = E.ocorrencias("a".repeat(E.MAXIMO_DE_OCORRENCIAS + 50), "a").length;
    eq(n, E.MAXIMO_DE_OCORRENCIAS);
});

t("proxima: a que vem a partir do cursor, dando a volta; para trás, a anterior", () => {
    const o = E.ocorrencias("um dois um três um", "um"); // 0, 8, 16
    eq(E.proxima(o, 0, false), 0);
    eq(E.proxima(o, 1, false), 1);
    eq(E.proxima(o, 8, false), 1);
    eq(E.proxima(o, 17, false), 0, "depois da última, volta para a primeira");
    eq(E.proxima(o, 18, true), 2);
    eq(E.proxima(o, 16, true), 1);
    eq(E.proxima(o, 10, true), 1);
    eq(E.proxima(o, 2, true), 0);
    eq(E.proxima(o, 0, true), 2, "antes da primeira, volta para a última");
    eq(E.proxima([], 0, false), -1);
    eq(E.proxima([], 0, true), -1);
});

t("proxima: Enter repetido percorre todas e volta ao começo", () => {
    const o = E.ocorrencias("a b a b a", "a");
    let pos = 0;
    const visitadas = [];
    for (let i = 0; i < 4; i++) {
        const k = E.proxima(o, pos, false);
        visitadas.push(k);
        pos = o[k].fim;
    }
    igual(visitadas, [0, 1, 2, 0]);
    pos = o[0].inicio;
    const voltando = [];
    for (let i = 0; i < 3; i++) {
        const k = E.proxima(o, pos, true);
        voltando.push(k);
        pos = o[k].inicio;
    }
    igual(voltando, [2, 1, 0]);
});

t("atual: qual ocorrência é a seleção", () => {
    const o = E.ocorrencias("um dois um", "um");
    eq(E.atual(o, 8, 10), 1);
    eq(E.atual(o, 10, 8), 1);
    eq(E.atual(o, 8, 9), -1);
    eq(E.atual(o, 3, 3), -1);
});

t("substituirUma: troca só a ocorrência selecionada, ao pé da letra", () => {
    const s = "um dois um";
    let e = E.substituirUma(s, "um", "três", false, 8, 10);
    eq(E.aplicar(s, e), "um dois três");
    eq(e.selInicio, 12);
    // O "$&" e o "$1" do texto de troca são texto.
    e = E.substituirUma(s, "um", "$& $1", false, 0, 2);
    eq(E.aplicar(s, e), "$& $1 dois um");
    // A seleção não é uma ocorrência: nada troca.
    eq(E.substituirUma(s, "um", "x", false, 3, 7), null);
    eq(E.substituirUma(s, "um", "x", false, 0, 0), null);
    eq(E.substituirUma("Um", "um", "x", true, 0, 2), null);
    // Trocar por nada apaga.
    eq(E.aplicar(s, E.substituirUma(s, "um ", "", false, 0, 3)), "dois um");
});

t("substituirTodas: todas de uma vez, numa edição só, e diz quantas", () => {
    let r = E.substituirTodas("um dois UM três um", "um", "1", false);
    eq(r.quantas, 3);
    eq(E.aplicar("um dois UM três um", r.edicao), "1 dois 1 três 1");
    r = E.substituirTodas("um dois UM três um", "um", "1", true);
    eq(r.quantas, 2);
    eq(E.aplicar("um dois UM três um", r.edicao), "1 dois UM três 1");
    // A edição só cobre da primeira à última ocorrência.
    r = E.substituirTodas("antes x meio x depois", "x", "yy", false);
    eq(r.edicao.inicio, 6);
    eq(r.edicao.fim, 14);
    eq(r.edicao.texto, "yy meio yy");
    // O texto de troca que contém o termo não entra em laço.
    r = E.substituirTodas("a a", "a", "aa", false);
    eq(E.aplicar("a a", r.edicao), "aa aa");
    r = E.substituirTodas("a.b", ".", "$&$&", false);
    eq(E.aplicar("a.b", r.edicao), "a$&$&b");
    igual(E.substituirTodas("nada", "x", "y", false), { edicao: null, quantas: 0 });
    igual(E.substituirTodas("nada", "", "y", false), { edicao: null, quantas: 0 });
});

t("textoDaBusca: o que a barra diz", () => {
    eq(E.textoDaBusca("", 0, -1), "");
    eq(E.textoDaBusca("x", 0, -1), "nada encontrado");
    eq(E.textoDaBusca("x", 12, 2), "3 de 12");
    eq(E.textoDaBusca("x", 1, -1), "1 ocorrência");
    eq(E.textoDaBusca("x", 1234, -1), "1.234 ocorrências");
    eq(E.textoDaBusca("x", E.MAXIMO_DE_OCORRENCIAS, 0), "1 de muitas");
});

// --- o salvamento -----------------------------------------------------------------

const anda = (...eventos) => eventos.reduce((e, ev) => E.transicao(e, ev), E.estadoInicial());

t("salvamento: limpo → sujo → salvando → salvo", () => {
    igual(E.estadoInicial(), { fase: "limpo", pendente: false });
    eq(anda("editou").fase, "sujo");
    eq(anda("editou", "salvar").fase, "salvando");
    eq(anda("editou", "salvar", "salvou").fase, "salvo");
    eq(anda("editou", "salvar", "salvou", "editou").fase, "sujo");
});

t("salvamento: digitar durante o salvamento não se perde: volta a sujo quando ele acaba", () => {
    igual(anda("editou", "salvar", "editou"), { fase: "salvando", pendente: true });
    igual(anda("editou", "salvar", "editou", "salvou"), { fase: "sujo", pendente: false });
    igual(anda("editou", "salvar", "editou", "editou", "salvou", "salvar", "salvou"), { fase: "salvo", pendente: false });
});

t("salvamento: conflito trava tudo até a pessoa decidir", () => {
    eq(anda("editou", "salvar", "conflito").fase, "conflito");
    eq(anda("editou", "salvar", "conflito", "editou").fase, "conflito", "digitar não tira do conflito");
    eq(anda("conflito").fase, "conflito", "mesmo limpo, se o arquivo sumiu");
    eq(anda("editou", "conflito", "manter").fase, "sujo", "manter a minha: volta a valer o meu texto");
    eq(anda("editou", "conflito", "abriu").fase, "limpo", "recarregar: vale o do disco");
    eq(anda("editou", "conflito", "salvar").fase, "salvando", "salvar à força ou salvar como");
    eq(anda("editou", "conflito", "salvar", "salvou").fase, "salvo");
    eq(anda("editou", "manter").fase, "sujo", "manter fora de conflito não faz nada");
});

t("salvamento: falha ao gravar fica à vista e a próxima mudança tenta de novo", () => {
    eq(anda("editou", "salvar", "falhou").fase, "erro");
    eq(anda("editou", "salvar", "falhou", "editou").fase, "sujo");
    eq(anda("editou", "salvar", "falhou", "salvar").fase, "salvando");
    eq(anda("editou", "falhou").fase, "sujo", "falhou sem estar salvando não muda nada");
});

t("salvamento: eventos fora de hora não estragam o estado", () => {
    eq(anda("salvar").fase, "limpo", "salvar o que está limpo não faz nada");
    eq(anda("salvou").fase, "limpo");
    eq(anda("editou", "salvou").fase, "sujo", "um 'salvou' atrasado não limpa o que está sujo");
    eq(anda("editou", "salvar", "salvar").fase, "salvando");
    eq(anda("editou", "salvar", "salvou", "salvar").fase, "salvo");
    eq(anda("editou", "inventado").fase, "sujo");
    // A transição não mexe no estado que recebeu.
    const e = E.estadoInicial();
    E.transicao(e, "editou");
    igual(e, { fase: "limpo", pendente: false });
});

t("temMudancas: só limpo e salvo estão no disco", () => {
    igual(["limpo", "salvo", "sujo", "salvando", "conflito", "erro"].map((fase) => E.temMudancas({ fase })), [false, false, true, true, true, true]);
});

t("salvamento automático: só documento com nome, sujo e sem conflito", () => {
    eq(E.deveSalvarSozinho(doc({ fase: "sujo", caminho: "/t/a.md" })), true);
    // Documento sem nome nunca é salvo sozinho.
    eq(E.deveSalvarSozinho(doc({ fase: "sujo", caminho: "" })), false);
    // Em conflito, para até a pessoa decidir.
    eq(E.deveSalvarSozinho(doc({ fase: "conflito", caminho: "/t/a.md" })), false);
    // Um rascunho recuperado nunca é gravado sozinho por cima do arquivo.
    eq(E.deveSalvarSozinho(doc({ fase: "sujo", caminho: "/t/a.md", recuperado: true })), false);
    eq(E.deveSalvarSozinho(doc({ fase: "sujo", caminho: "/t/a.md", soLeitura: true })), false);
    for (const fase of ["limpo", "salvo", "salvando", "erro"])
        eq(E.deveSalvarSozinho(doc({ fase, caminho: "/t/a.md" })), false, fase);
});

t("rascunho: guarda tudo que não está no disco e não é vazio", () => {
    eq(E.deveGuardarRascunho(doc({ fase: "sujo" })), true);
    eq(E.deveGuardarRascunho(doc({ fase: "sujo", caminho: "/t/a.md" })), true);
    eq(E.deveGuardarRascunho(doc({ fase: "conflito", caminho: "/t/a.md" })), true);
    eq(E.deveGuardarRascunho(doc({ fase: "erro", caminho: "/t/a.md" })), true);
    eq(E.deveGuardarRascunho(doc({ fase: "sujo", vazio: true })), false);
    eq(E.deveGuardarRascunho(doc({ fase: "limpo" })), false);
    eq(E.deveGuardarRascunho(doc({ fase: "salvo", caminho: "/t/a.md" })), false);
});

t("ao fechar: sem mudança fecha; com nome grava e fecha; o resto pergunta", () => {
    eq(E.aoFechar(doc({ fase: "limpo" })), "fechar");
    eq(E.aoFechar(doc({ fase: "salvo", caminho: "/t/a.md" })), "fechar");
    eq(E.aoFechar(doc({ fase: "sujo", caminho: "/t/a.md" })), "salvar");
    eq(E.aoFechar(doc({ fase: "salvando", caminho: "/t/a.md" })), "salvar");
    eq(E.aoFechar(doc({ fase: "salvando", caminho: "/t/a.md", pendente: true })), "salvar");
    // Sem nome e com texto: o app nunca escolhe um nome sozinho.
    eq(E.aoFechar(doc({ fase: "sujo", caminho: "" })), "perguntar");
    // Sem nome e sem texto nenhum: não há o que perder.
    eq(E.aoFechar(doc({ fase: "sujo", caminho: "", vazio: true })), "fechar");
    // Com nome e vazio: apagar tudo é uma mudança como outra.
    eq(E.aoFechar(doc({ fase: "sujo", caminho: "/t/a.md", vazio: true })), "salvar");
    eq(E.aoFechar(doc({ fase: "conflito", caminho: "/t/a.md" })), "perguntar");
    eq(E.aoFechar(doc({ fase: "erro", caminho: "/t/a.md" })), "perguntar");
    eq(E.aoFechar(doc({ fase: "sujo", caminho: "/t/a.md", recuperado: true })), "perguntar");
    eq(E.aoFechar(doc({ fase: "salvando", caminho: "/t/a.md", recuperado: true })), "perguntar");
    eq(E.aoFechar(doc({ fase: "sujo", caminho: "/t/a.md", soLeitura: true })), "perguntar");
});

t("ao fechar: a pergunta se responde por S, D, C, Enter e Esc", () => {
    igual(["s", "S", "Enter", "Return"].map(E.respostaDaTecla), ["salvar", "salvar", "salvar", "salvar"]);
    igual(["d", "D"].map(E.respostaDaTecla), ["descartar", "descartar"]);
    igual(["c", "C", "Escape", "Esc"].map(E.respostaDaTecla), ["cancelar", "cancelar", "cancelar", "cancelar"]);
    igual(["x", "", " ", "ç", undefined].map(E.respostaDaTecla), ["", "", "", "", ""]);
});

t("mudou por fora: o que eu mesmo gravei é ignorado; sem mudança minha, recarrega; com, é conflito", () => {
    const mudou = (sha) => ({ evento: "mudou", impressao: { sha256: sha } });
    eq(E.aoMudarPorFora(doc({ fase: "salvo", caminho: "/a" }), mudou("abc"), "abc"), "ignorar");
    eq(E.aoMudarPorFora(doc({ fase: "sujo", caminho: "/a" }), mudou("abc"), "abc"), "ignorar");
    eq(E.aoMudarPorFora(doc({ fase: "limpo", caminho: "/a" }), mudou("novo"), "abc"), "recarregar");
    eq(E.aoMudarPorFora(doc({ fase: "salvo", caminho: "/a" }), mudou("novo"), "abc"), "recarregar");
    eq(E.aoMudarPorFora(doc({ fase: "sujo", caminho: "/a" }), mudou("novo"), "abc"), "conflito");
    eq(E.aoMudarPorFora(doc({ fase: "salvando", caminho: "/a" }), mudou("novo"), "abc"), "conflito");
    eq(E.aoMudarPorFora(doc({ fase: "erro", caminho: "/a" }), mudou("novo"), "abc"), "conflito");
    eq(E.aoMudarPorFora(doc({ fase: "conflito", caminho: "/a" }), mudou("novo"), "abc"), "conflito");
    // Rascunho recuperado ainda limpo na conta do estado: não troca o texto da pessoa.
    eq(E.aoMudarPorFora(doc({ fase: "limpo", caminho: "/a", recuperado: true }), mudou("novo"), "abc"), "conflito");
    // O arquivo não existia e apareceu.
    eq(E.aoMudarPorFora(doc({ fase: "sujo", caminho: "/a" }), mudou("novo"), "ausente"), "conflito");
    // Evento sem impressão não é "o meu": trata como mudança.
    eq(E.aoMudarPorFora(doc({ fase: "sujo", caminho: "/a" }), { evento: "mudou", impressao: null }, ""), "conflito");
});

t("sumiu por fora: é conflito mesmo sem mudança (o texto só existe na janela), menos se nunca existiu", () => {
    const sumiu = { evento: "sumiu", impressao: null };
    eq(E.aoMudarPorFora(doc({ fase: "limpo", caminho: "/a" }), sumiu, "abc"), "conflito");
    eq(E.aoMudarPorFora(doc({ fase: "sujo", caminho: "/a" }), sumiu, "abc"), "conflito");
    eq(E.aoMudarPorFora(doc({ fase: "sujo", caminho: "/a" }), sumiu, "ausente"), "ignorar");
});

t("ondeAbrir: nunca troca um texto não salvo por outro", () => {
    eq(E.ondeAbrir(doc({ fase: "limpo" })), "aqui");
    eq(E.ondeAbrir(doc({ fase: "salvo", caminho: "/a" })), "aqui");
    eq(E.ondeAbrir(doc({ fase: "sujo", vazio: true })), "aqui");
    eq(E.ondeAbrir(doc({ fase: "sujo" })), "nova");
    eq(E.ondeAbrir(doc({ fase: "sujo", caminho: "/a" })), "nova");
    eq(E.ondeAbrir(doc({ fase: "conflito", caminho: "/a" })), "nova");
    eq(E.ondeAbrir(doc({ fase: "salvando", caminho: "/a" })), "nova");
});

// --- textos ------------------------------------------------------------------------

t("nome e pasta de um caminho", () => {
    eq(E.nome("/casa/textos/carta.md"), "carta.md");
    eq(E.nome("solto.md"), "solto.md");
    eq(E.nome(""), "");
    eq(E.nome(undefined), "");
    eq(E.pasta("/casa/textos/carta.md"), "/casa/textos");
    eq(E.pasta("/carta.md"), "/");
    eq(E.pasta("carta.md"), "");
    eq(E.pasta(""), "");
});

t("textoDoEstado: salvo, salvando, não salvo, conflito", () => {
    eq(E.textoDoEstado(doc({ fase: "limpo", caminho: "/a.md" })), "salvo");
    eq(E.textoDoEstado(doc({ fase: "salvo", caminho: "/a.md" })), "salvo");
    eq(E.textoDoEstado(doc({ fase: "salvando", caminho: "/a.md" })), "salvando…");
    eq(E.textoDoEstado(doc({ fase: "sujo", caminho: "/a.md" })), "não salvo");
    eq(E.textoDoEstado(doc({ fase: "sujo" })), "não salvo");
    eq(E.textoDoEstado(doc({ fase: "conflito", caminho: "/a.md" })), "conflito");
    eq(E.textoDoEstado(doc({ fase: "erro", caminho: "/a.md" })), "não salvou");
    eq(E.textoDoEstado(doc({ fase: "limpo" })), "novo");
    eq(E.textoDoEstado(doc({ fase: "limpo", caminho: "/a.md", soLeitura: true })), "só leitura");
    eq(E.textoDoEstado(doc({ fase: "sujo", caminho: "/a.md", recuperado: true })), "rascunho recuperado");
});

t("tituloDaJanela: o nome do arquivo, um ponto quando há mudança, e o nome do app", () => {
    eq(E.tituloDaJanela(doc({ fase: "limpo" })), "sem título — MkEscrita");
    eq(E.tituloDaJanela(doc({ fase: "sujo" })), "● sem título — MkEscrita");
    eq(E.tituloDaJanela(doc({ fase: "limpo", caminho: "/t/carta.md" })), "carta.md — MkEscrita");
    eq(E.tituloDaJanela(doc({ fase: "salvo", caminho: "/t/carta.md" })), "carta.md — MkEscrita");
    eq(E.tituloDaJanela(doc({ fase: "sujo", caminho: "/t/carta.md" })), "● carta.md — MkEscrita");
    eq(E.tituloDaJanela(doc({ fase: "salvando", caminho: "/t/carta.md" })), "● carta.md — MkEscrita");
    eq(E.tituloDaJanela(doc({ fase: "conflito", caminho: "/t/carta.md" })), "● carta.md — MkEscrita");
});

t("palavras e tempo de leitura, como o rodapé mostra", () => {
    eq(E.milhar(0), "0");
    eq(E.milhar(999), "999");
    eq(E.milhar(1000), "1.000");
    eq(E.milhar(1234567), "1.234.567");
    eq(E.textoDasPalavras(0), "0 palavras");
    eq(E.textoDasPalavras(1), "1 palavra");
    eq(E.textoDasPalavras(12345), "12.345 palavras");
    eq(E.textoDasPalavras(null), "");
    eq(E.textoDasPalavras(undefined), "");
    eq(E.tempoDeLeitura(0), "");
    eq(E.tempoDeLeitura(null), "");
    eq(E.tempoDeLeitura(20), "menos de 1 min");
    eq(E.tempoDeLeitura(60), "1 min");
    eq(E.tempoDeLeitura(150), "3 min");
    eq(E.tempoDeLeitura(3600), "1 h");
    eq(E.tempoDeLeitura(3900), "1 h 5 min");
});

t("haQuanto: a idade do rascunho", () => {
    eq(E.haQuanto(1000, 990), "agora há pouco");
    eq(E.haQuanto(1000, 2000), "agora há pouco", "relógio que andou para trás");
    eq(E.haQuanto(10000, 10000 - 300), "há 5 min");
    eq(E.haQuanto(100000, 100000 - 7200), "há 2 h");
    eq(E.haQuanto(1000000, 1000000 - 86400), "ontem");
    eq(E.haQuanto(1000000, 1000000 - 3 * 86400), "há 3 dias");
});

t("fraseDoConflito: diz o que houve com o arquivo", () => {
    eq(E.fraseDoConflito("mudou", "carta.md"), "carta.md mudou por fora desde que foi aberto.");
    assert.match(E.fraseDoConflito("sumiu", "carta.md"), /sumiu do disco/);
    assert.match(E.fraseDoConflito("apareceu", "carta.md"), /Já existe/);
    assert.match(E.fraseDoConflito("mudou", ""), /^o arquivo mudou/);
});

t("zoom: passos de 10, entre 50% e 300%", () => {
    eq(E.zoom(100, 1), 110);
    eq(E.zoom(100, -1), 90);
    eq(E.zoom(100, 0), 100);
    eq(E.zoom(300, 1), 300);
    eq(E.zoom(295, 1), 300);
    eq(E.zoom(50, -1), 50);
    eq(E.zoom(55, -3), 50);
    eq(E.zoom(123, 1), 130);
    eq(E.zoom(undefined, 1), 110);
    eq(E.zoom(NaN, 0), 100);
    eq(E.zoom(9999, 0), 300);
    eq(E.ZOOM_MINIMO, 50);
    eq(E.ZOOM_MAXIMO, 300);
});

// --- a conversa com o motor ---------------------------------------------------------

t("bytesUtf8: bate com a conta de verdade, inclusive acento, emoji e meio par solto", () => {
    for (const s of ["", "abc", "ação", "€", "😀", "a😀b", "日本語", "\n\t", "x\ud83dy", "\udc00", "a\ud83d"])
        eq(E.bytesUtf8(s), Buffer.byteLength(s, "utf8"), JSON.stringify(s));
});

t("argsSalvar: sempre diz quantos bytes vão e o que espera achar no disco", () => {
    igual(E.argsSalvar("/t/a.md", "ação", "abc", { quebraFinal: true }),
        ["salvar", "--bytes", "6", "--esperado", "abc", "--quebra-final", "--", "/t/a.md"]);
    igual(E.argsSalvar("/t/a.md", "x", "abc", { fimDeLinha: "crlf" }),
        ["salvar", "--bytes", "1", "--esperado", "abc", "--fim-de-linha", "crlf", "--", "/t/a.md"]);
    // Arquivo que não existia: espera que continue não existindo.
    igual(E.argsSalvar("/t/novo.md", "", "", {}), ["salvar", "--bytes", "0", "--esperado", "ausente", "--", "/t/novo.md"]);
    igual(E.argsSalvar("/t/novo.md", "", "ausente"), ["salvar", "--bytes", "0", "--esperado", "ausente", "--", "/t/novo.md"]);
    // À força (manter a minha, salvar como): sem --esperado.
    igual(E.argsSalvar("/t/a.md", "x", "abc", { forcar: true }), ["salvar", "--bytes", "1", "--forcar", "--", "/t/a.md"]);
    // O caminho vai depois de "--": nome começando com traço chega inteiro.
    const args = E.argsSalvar("/t/-rf.md", "x", "abc", { fimDeLinha: "lf" });
    eq(args[args.length - 2], "--");
    eq(args.includes("--fim-de-linha"), false, "lf é o padrão: não precisa dizer");
    eq(E.argsSalvar("/t/a.md", "x", "abc", { fimDeLinha: "misto" }).includes("--fim-de-linha"), false);
});

t("argsRascunho e argsExportar", () => {
    igual(E.argsTexto("contar", "é"), ["contar", "--bytes", "2"]);
    igual(E.argsTexto("html", "x", ["--pagina"]), ["html", "--bytes", "1", "--pagina"]);
    igual(E.argsRascunho("r-1", "é", "/t/a.md", "abc"), ["rascunho", "guardar", "r-1", "--bytes", "2", "--origem", "/t/a.md", "--base", "abc"]);
    igual(E.argsRascunho("r-1", "x", "", ""), ["rascunho", "guardar", "r-1", "--bytes", "1"]);
    igual(E.argsExportar("/t/a.html", "x", false), ["exportar", "html", "--bytes", "1", "--", "/t/a.html"]);
    igual(E.argsExportar("/t/a.html", "x", true), ["exportar", "html", "--bytes", "1", "--forcar", "--", "/t/a.html"]);
});

t("nomeDoExportado: o nome do documento com .html", () => {
    eq(E.nomeDoExportado("/t/carta.md"), "carta.html");
    eq(E.nomeDoExportado("/t/carta.markdown"), "carta.html");
    eq(E.nomeDoExportado("/t/notas.TXT"), "notas.html");
    eq(E.nomeDoExportado("/t/sem-extensao"), "sem-extensao.html");
    eq(E.nomeDoExportado("/t/v1.2.md"), "v1.2.html");
    eq(E.nomeDoExportado("/t/.md"), ".md.html");
    eq(E.nomeDoExportado(""), "documento.html");
});

t("idDeRascunho: só o que o motor aceita, e diferente a cada sorteio", () => {
    const a = E.idDeRascunho(1795539592104, 0.123);
    const b = E.idDeRascunho(1795539592104, 0.987);
    assert.match(a, /^[a-z0-9][a-z0-9-]{0,63}$/);
    assert.match(b, /^[a-z0-9][a-z0-9-]{0,63}$/);
    assert.notEqual(a, b);
    assert.match(E.idDeRascunho(0, 0), /^[a-z0-9][a-z0-9-]{0,63}$/);
    assert.match(E.idDeRascunho(1e13, 0.999999), /^[a-z0-9][a-z0-9-]{0,63}$/);
    assert.match(E.idDeRascunho(1e13, undefined), /^[a-z0-9][a-z0-9-]{0,63}$/);
});

t("rascunhoParaOferecer: nunca o de uma janela viva; com arquivo, só o dele", () => {
    const itens = [
        { id: "a", origem: "", vivo: true }, { id: "b", origem: "/t/x.md", vivo: false },
        { id: "c", origem: "", vivo: false }, { id: "d", origem: "/t/y.md", vivo: false }];
    eq(E.rascunhoParaOferecer(itens, "").id, "b", "sem arquivo: o mais recente que não é de janela viva");
    eq(E.rascunhoParaOferecer(itens, "/t/y.md").id, "d");
    eq(E.rascunhoParaOferecer(itens, "/t/z.md"), null);
    eq(E.rascunhoParaOferecer([{ id: "a", origem: "/t/x.md", vivo: true }], "/t/x.md"), null);
    eq(E.rascunhoParaOferecer([], ""), null);
    eq(E.rascunhoParaOferecer(undefined, ""), null);
});

t("caminhoDeUrl e urlDaPasta: do seletor de arquivos para o motor e de volta", () => {
    eq(E.caminhoDeUrl("file:///casa/textos/a%20%C3%A7%C3%A3o.md"), "/casa/textos/a ção.md");
    eq(E.caminhoDeUrl("file://localhost/a.md"), "/a.md");
    eq(E.caminhoDeUrl("https://exemplo.org/a.md"), "");
    eq(E.caminhoDeUrl("file://servidor/a.md"), "");
    eq(E.caminhoDeUrl("file:///a%ZZ.md"), "");
    eq(E.caminhoDeUrl(""), "");
    eq(E.caminhoDeUrl(undefined), "");
    eq(E.urlDaPasta("/casa/meus textos/a.md"), "file:///casa/meus%20textos/");
    eq(E.urlDaPasta("/a.md"), "file:///");
    eq(E.urlDaPasta(""), "");
    eq(E.caminhoDeUrl(E.urlDaPasta("/casa/ação #1/a.md") + "a.md"), "/casa/ação #1/a.md");
});

t("linkAbrivel: da pré-visualização só sai http e https", () => {
    eq(E.linkAbrivel("https://exemplo.org/a"), true);
    eq(E.linkAbrivel("HTTP://exemplo.org"), true);
    for (const nao of ["javascript:alert(1)", "file:///etc/passwd", "mailto:a@b", "arquivo.md", "", undefined, "https://a b"])
        eq(E.linkAbrivel(nao), false, String(nao));
});

t("arquivoDeExemplo: cada pedido tem a sua resposta gravada", () => {
    eq(E.arquivoDeExemplo(["abrir", "--", "/a.md"]), "abrir.json");
    eq(E.arquivoDeExemplo(["salvar", "--bytes", "1", "--", "/a.md"]), "salvar.json");
    eq(E.arquivoDeExemplo(["rascunho", "listar"]), "rascunho-listar.json");
    eq(E.arquivoDeExemplo(["rascunho", "guardar", "r-1"]), "rascunho-guardar.json");
    eq(E.arquivoDeExemplo(["preferencias", "ler"]), "preferencias-ler.json");
    eq(E.arquivoDeExemplo(["html"]), "html.json");
    eq(E.arquivoDeExemplo(["html", "--pagina"]), "html-pagina.json");
    eq(E.arquivoDeExemplo(["contar"]), "contar.json");
    eq(E.arquivoDeExemplo(["estrutura"]), "estrutura.json");
    eq(E.arquivoDeExemplo(["exportar", "html", "--", "/a.html"]), "exportar.json");
    eq(E.arquivoDeExemplo(["vigiar", "/a.md"]), "vigiar.json");
});

t("tituloAtual: em que seção o cursor está", () => {
    const ts = [{ linha: 1 }, { linha: 10 }, { linha: 20 }];
    igual([1, 5, 10, 19, 20, 99].map((l) => E.tituloAtual(ts, l)), [0, 0, 1, 1, 2, 2]);
    eq(E.tituloAtual([{ linha: 5 }], 2), -1);
    eq(E.tituloAtual([], 2), -1);
    eq(E.tituloAtual(undefined, 2), -1);
});

t("filtrarFontes: em ordem, sem repetir, casando em qualquer parte do nome", () => {
    const f = ["Space Mono", "Inter", "JetBrainsMono Nerd Font", "Inter", "", "noto Sans"];
    igual(E.filtrarFontes(f, ""), ["Inter", "JetBrainsMono Nerd Font", "noto Sans", "Space Mono"]);
    igual(E.filtrarFontes(f, "MONO"), ["JetBrainsMono Nerd Font", "Space Mono"]);
    igual(E.filtrarFontes(f, "  inter "), ["Inter"]);
    igual(E.filtrarFontes(f, "xyz"), []);
    igual(E.filtrarFontes(undefined, ""), []);
});

t("estiloDaPrevia: só as cores dadas, e o nome da fonte não escapa da folha de estilo", () => {
    const cores = { destaque: "#ce5e32", texto: "#cdc4ba", suave: "#b0a9a0", apagado: "#958f87" };
    const s = E.estiloDaPrevia(cores, "Space Mono");
    assert.match(s, /^<style>.*<\/style>$/);
    assert.ok(s.includes("a { color: #ce5e32; }"));
    assert.ok(s.includes("font-family: 'Space Mono'"));
    const mau = E.estiloDaPrevia(cores, "x'; } </style><script>");
    eq(mau.split("</style>").length, 2);
    assert.ok(!mau.includes("<script>"));
});

t("atalhos: a folha do F1 tem o que a janela promete", () => {
    const todos = E.atalhos().flatMap((g) => g.itens.map((i) => i[0])).join(" | ");
    for (const tecla of ["Ctrl+S", "Ctrl+Shift+S", "Ctrl+O", "Ctrl+N", "Ctrl+B", "Ctrl+I", "Ctrl+Shift+X", "Ctrl+K", "Ctrl+F", "Ctrl+G",
        "Ctrl+H", "Ctrl+Shift+P", "Ctrl+Shift+O", "Ctrl+Shift+F", "F11", "Ctrl+0", "Ctrl+Shift+C", "Ctrl+Z", "Ctrl+Y", "F1", "Ctrl+/"])
        assert.ok(todos.includes(tecla), "falta " + tecla);
    for (const g of E.atalhos()) {
        assert.ok(g.grupo.length > 0);
        for (const i of g.itens)
            eq(i.length, 2);
    }
});

t("edicao e aplicar: a troca de um trecho", () => {
    igual(E.edicao(1, 2, "x", 3), { inicio: 1, fim: 2, texto: "x", selInicio: 3, selFim: 3 });
    igual(E.edicao(1, 2, "x", 3, 5), { inicio: 1, fim: 2, texto: "x", selInicio: 3, selFim: 5 });
    eq(E.aplicar("abcdef", E.edicao(1, 3, "XY!", 0)), "aXY!def");
    eq(E.aplicar("abc", E.edicao(3, 3, "d", 0)), "abcd");
    eq(E.aplicar("abc", E.edicao(0, 3, "", 0)), "");
});
