// shell/mare/ClipboardPanel.qml, o trecho entre "puro: início" e "puro: fim":
// a leitura da saída do `cliphist list` e o filtro da busca.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("ClipboardPanel.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["matching", "parsed"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("ClipboardPanel.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

// Como o `cliphist list` responde: "<id>\t<prévia>", o mais novo em cima.
const SAIDA = [
    "1042\tgit commit -m \"ajusta o calendário\"",
    "1041\t[[ binary data 1.2 MiB png 1920x1080 ]]",
    "1040\thttps://exemplo.com.br/caminho?a=1&b=2",
    "1039\t  linha com   espaços\te um tab  ",
    "1038\tC:\\Users\\mk: \"aspas\" 'simples' ção",
    "1037\t[[ binary data 48 KiB jpeg 640x480 ]]",
    ""
].join("\n");

t("parsed: uma entrada por linha, com o id, a prévia e a linha inteira", () => {
    const e = M.parsed(SAIDA);
    eq(e.length, 6);
    igual(e[0], { line: "1042\tgit commit -m \"ajusta o calendário\"", id: "1042",
        text: "git commit -m \"ajusta o calendário\"", image: false });
    igual(e.map((x) => x.id), ["1042", "1041", "1040", "1039", "1038", "1037"]);
});

t("parsed: imagem vira 'imagem <tipo> <largura>×<altura>'", () => {
    const e = M.parsed(SAIDA);
    igual(e[1], { line: "1041\t[[ binary data 1.2 MiB png 1920x1080 ]]", id: "1041", text: "imagem png 1920×1080", image: true });
    eq(e[5].text, "imagem jpeg 640×480");
    eq(M.parsed("7\t[[ binary data 3 KiB JPG 10x10 ]]")[0].text, "imagem JPG 10×10");
    eq(M.parsed("7\t[[ binary data 3 KiB webp ]]")[0].text, "imagem webp");
    for (const tipo of ["gif", "bmp", "jpg"])
        eq(M.parsed("7\t[[ binary data 1 KiB " + tipo + " 2x2 ]]")[0].image, true, tipo);
});

t("parsed: binário que não é imagem e texto que só fala em binário ficam como texto", () => {
    const pdf = M.parsed("9\t[[ binary data 2 MiB application/pdf ]]")[0];
    eq(pdf.image, false);
    eq(pdf.text, "[[ binary data 2 MiB application/pdf ]]");
    const frase = M.parsed("9\tveja [[ binary data 1 KiB png 1x1 ]]")[0];
    eq(frase.image, false);
});

t("parsed: espaços e tabs da prévia viram um espaço só; a linha fica intacta", () => {
    const e = M.parsed(SAIDA)[3];
    eq(e.text, "linha com espaços e um tab");
    eq(e.line, "1039\t  linha com   espaços\te um tab  ");
    eq(e.id, "1039");
});

t("parsed: barra, dois-pontos, aspas e acento passam como vieram", () => {
    eq(M.parsed(SAIDA)[4].text, "C:\\Users\\mk: \"aspas\" 'simples' ção");
});

t("parsed: vazio, linha sem tab e linha que começa com tab não viram entrada", () => {
    igual(M.parsed(""), []);
    igual(M.parsed("\n\n"), []);
    igual(M.parsed("sem tab nenhum\n\tsem id\n12\tcom id"), [{ line: "12\tcom id", id: "12", text: "com id", image: false }]);
});

t("parsed: prévia vazia continua sendo uma entrada (dá para apagar)", () => {
    igual(M.parsed("5\t"), [{ line: "5\t", id: "5", text: "", image: false }]);
});

t("matching: sem busca, os primeiros da lista", () => {
    const e = M.parsed(SAIDA);
    igual(M.matching(e, "", 8), e);
    igual(M.matching(e, "   ", 8), e);
    igual(M.matching(e, "", 2).map((x) => x.id), ["1042", "1041"]);
    igual(M.matching([], "", 8), []);
});

t("matching: procura na prévia, sem diferenciar maiúsculas e sem os espaços das pontas", () => {
    const e = M.parsed(SAIDA);
    igual(M.matching(e, "GIT", 8).map((x) => x.id), ["1042"]);
    igual(M.matching(e, "  imagem ", 8).map((x) => x.id), ["1041", "1037"]);
    igual(M.matching(e, "imagem", 1).map((x) => x.id), ["1041"]);
    igual(M.matching(e, "Calendário", 8).map((x) => x.id), ["1042"]);
    igual(M.matching(e, "não tem", 8), []);
});

t("matching: a busca é texto, não expressão regular", () => {
    const e = M.parsed(SAIDA);
    igual(M.matching(e, "a=1&b", 8).map((x) => x.id), ["1040"]);
    igual(M.matching(e, ".*", 8), []);
    igual(M.matching(e, "\\users\\", 8).map((x) => x.id), ["1038"]);
    igual(M.matching(e, "[[", 8), []);
});

t("matching não mexe na lista que recebeu", () => {
    const e = M.parsed(SAIDA);
    const antes = JSON.stringify(e);
    M.matching(e, "git", 1);
    eq(JSON.stringify(e), antes);
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
