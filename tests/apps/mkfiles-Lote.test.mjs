// apps/mkfiles/qml/Lote.js: a conta do renomear em lote (trocar texto, modelo
// com campos, caixa, acentos, espaços, extensão) e os conflitos que a folha
// marca antes de o motor renomear.
process.env.TZ = "UTC";
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const L = load(REPO + "/apps/mkfiles/qml/Lote.js");
const it = (nome, mais) => Object.assign({ nome, caminho: "/p/" + nome, tipo: "arquivo", tamanho: 0, mtime: 1700000000, mime: "text/plain" }, mais || {});
const pasta = (nome) => it(nome, { tipo: "pasta", mime: "inode/directory" });
const com = (mais) => Object.assign(L.novo(), mais);
const novo = (nome, mais, i) => L.nomeNovo(it(nome), i || 0, com(mais));

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["novo", "dois", "formatarData", "numero", "doModelo", "trocar", "tirarAcentos", "comCaixa", "nomeNovo", "erroDoNome", "calcular", "recusa"];
    const funcoes = Object.keys(L).filter((k) => typeof L[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("novo: sem nada escolhido, nenhum nome muda", () => {
    igual(L.novo(), { de: "", para: "", curinga: false, caixaExata: false, modelo: "", inicio: 1, passo: 1, caixa: "", acentos: false, espacos: "", aparar: false, extensao: null });
    for (const n of ["foto.jpg", "a.tar.gz", ".bashrc", "sem", "Ação  dupla .TXT"])
        eq(novo(n, {}), n);
    eq(L.calcular([it("a.txt"), it("b.txt")], L.novo(), null).mudam, 0);
    eq(L.CAIXAS.length, 4);
    eq(L.ESPACOS.map((e) => e.v).join("|"), "|_|-");
});

t("trocar texto: literal (sem diferenciar maiúsculas, a não ser que se peça) e com * e ?", () => {
    eq(novo("IMG_0001.jpg", { de: "img_", para: "foto " }), "foto 0001.jpg");
    eq(novo("IMG_0001.jpg", { de: "img_", para: "foto ", caixaExata: true }), "IMG_0001.jpg");
    eq(novo("a.a.a.txt", { de: "a", para: "b" }), "b.b.b.txt");
    // Sem curinga, * e ? e os sinais de regex são texto.
    eq(novo("o que? (1).txt", { de: "? (1)", para: "" }), "o que.txt");
    eq(novo("a*b.txt", { de: "*", para: "-" }), "a-b.txt");
    eq(novo("a.b+c.txt", { de: ".b+", para: "_" }), "a_c.txt");
    // Com curinga: * é qualquer trecho (o menor), ? é uma letra.
    eq(novo("foto (cópia 2) final.jpg", { de: "(*)", para: "", curinga: true }), "foto  final.jpg");
    eq(novo("IMG_0001.jpg", { de: "IMG_??", para: "", curinga: true }), "01.jpg");
    eq(novo("2024 - viagem.jpg", { de: "* - ", para: "", curinga: true }), "viagem.jpg");
    eq(novo("relatório final.txt", { de: " *", para: "", curinga: true }), "relatório.txt");
    eq(novo("abc.txt", { de: "*", para: "x", curinga: true }), "x.txt");
    // A extensão não entra na troca.
    eq(novo("txt.txt", { de: "txt", para: "doc" }), "doc.txt");
    eq(L.trocar("abc", "", "x", false, false), "abc");
});

t("modelo: {nome}, {ext}, {n} com início, passo e casas, {data} e {pai}", () => {
    eq(novo("praia.jpg", { modelo: "Férias {n}" }, 0), "Férias 1.jpg");
    eq(novo("praia.jpg", { modelo: "Férias {n:3}", inicio: 10, passo: 5 }, 2), "Férias 020.jpg");
    eq(novo("praia.jpg", { modelo: "{n:2} - {nome}" }, 8), "09 - praia.jpg");
    eq(novo("praia.jpg", { modelo: "{data} {nome}" }), "2023-11-14 praia.jpg");
    eq(novo("praia.jpg", { modelo: "{data:DD-MM-AA HHmmss}" }), "14-11-23 221320.jpg");
    eq(novo("praia.jpg", { modelo: "{pai}_{nome}" }), "p_praia.jpg");
    // Com {ext} o modelo dá o nome inteiro; sem, a extensão do arquivo fica.
    eq(novo("praia.JPG", { modelo: "{nome}.{ext}.bak" }), "praia.JPG.bak");
    eq(novo("fonte.tar.gz", { modelo: "{nome}-v2" }), "fonte-v2.tar.gz");
    eq(novo("fonte.tar.gz", { modelo: "{ext}" }), "tar.gz");
    eq(novo("LEIAME", { modelo: "{nome}.{ext}" }), "LEIAME");
    // Campo que não existe fica como está; modelo só de espaços não conta.
    eq(novo("a.txt", { modelo: "{nome} {outro}" }), "a {outro}.txt");
    eq(novo("a.txt", { modelo: "   " }), "a.txt");
    eq(L.nomeNovo(pasta("Fotos.2024"), 0, com({ modelo: "{nome} ({n})" })), "Fotos.2024 (1)");
    eq(L.numero(3, 1, 1, 0), "4");
    eq(L.numero(0, 7, 2, 3), "007");
    eq(L.numero(2, 0, -5, 2), "-10");
    eq(L.numero(1, "x", 1, 0), "1");
    eq(L.formatarData(0, ""), "1970-01-01");
    eq(L.formatarData(1700000000, "AAAA/MM"), "2023/11");
    eq(L.dois(4) + L.dois(12), "0412");
    igual(L.doModelo("{n}", it("a.txt"), 4, com({})), { base: "5", ext: ".txt" });
});

t("caixa, acentos, espaços e aparar, sempre sobre o nome sem a extensão", () => {
    eq(novo("Relatório Final.PDF", { caixa: "maiusculas" }), "RELATÓRIO FINAL.PDF");
    eq(novo("Relatório Final.PDF", { caixa: "minusculas" }), "relatório final.PDF");
    eq(novo("relatório FINAL_de vendas.pdf", { caixa: "primeira" }), "Relatório Final_De Vendas.pdf");
    eq(L.comCaixa("Abc", ""), "Abc");
    eq(novo("Ação e Órgão.txt", { acentos: true }), "Acao e Orgao.txt");
    eq(L.tirarAcentos("ÇÃO ñ ü"), "CAO n u");
    eq(novo("a  b   c.txt", { espacos: "_" }), "a_b_c.txt");
    eq(novo("a b.txt", { espacos: "-" }), "a-b.txt");
    eq(novo("  a   b  .txt", { aparar: true }), "a b.txt");
    // Combinados, na ordem: modelo, trocar, acentos, aparar, espaços, caixa, extensão.
    eq(novo("Foto Ação 1.JPG", { modelo: "{pai} {n:3} {nome}", de: "foto ", para: "", acentos: true, espacos: "_", caixa: "minusculas", extensao: "jpg" }), "p_001_acao_1.jpg");
});

t("mudar só a extensão: com ou sem ponto; vazia tira; pasta não tem", () => {
    eq(novo("a.jpeg", { extensao: "jpg" }), "a.jpg");
    eq(novo("a.jpeg", { extensao: ".JPG" }), "a.JPG");
    eq(novo("a.tar.gz", { extensao: "tgz" }), "a.tgz");
    eq(novo("a.jpeg", { extensao: "" }), "a");
    eq(novo("LEIAME", { extensao: "md" }), "LEIAME.md");
    eq(L.nomeNovo(pasta("Fotos.2024"), 0, com({ extensao: "x" })), "Fotos.2024");
});

t("calcular: os pares do que muda, e nada de par para o que fica igual", () => {
    const r = L.calcular([it("a.txt"), it("B.txt"), it("c.txt")], com({ caixa: "minusculas" }), { pasta: "/p", nomes: ["a.txt", "B.txt", "c.txt", "z.txt"] });
    igual(r.pares, [["/p/B.txt", "b.txt"]]);
    eq(r.mudam, 1);
    eq(r.conflitos, 0);
    igual(r.linhas.map((l) => [l.antes, l.depois, l.muda, l.motivo]), [["a.txt", "a.txt", false, ""], ["B.txt", "b.txt", true, ""], ["c.txt", "c.txt", false, ""]]);
});

t("calcular: nome repetido no lote, nome que já existe na pasta e nome inválido são marcados", () => {
    const pastaP = { pasta: "/p", nomes: ["a1.txt", "a2.txt", "x.txt", "fica.txt"] };
    // Dois viram o mesmo nome.
    let r = L.calcular([it("a1.txt"), it("a2.txt")], com({ de: "?", para: "", curinga: true, modelo: "a" }), pastaP);
    igual(r.linhas.map((l) => l.motivo), ["repetido", "repetido"]);
    eq(r.conflitos, 2);
    igual(r.pares, []);
    eq(r.linhas[0].frase, "repetido no lote");
    // O nome novo é de um arquivo que fica na pasta.
    r = L.calcular([it("a1.txt"), it("a2.txt")], com({ modelo: "{n}", inicio: 0 }), { pasta: "/p", nomes: ["a1.txt", "a2.txt", "1.txt"] });
    igual(r.linhas.map((l) => l.depois + ":" + l.motivo), ["0.txt:", "1.txt:existe"]);
    eq(r.linhas[1].frase, "já existe na pasta");
    igual(r.pares, [["/p/a1.txt", "0.txt"]]);
    // Inválido: vazio, com barra, ponto.
    r = L.calcular([it("a1.txt"), it("a2.txt"), it("x.txt")], com({ modelo: "{ext}", de: "txt", para: "", extensao: "" }), pastaP);
    eq(r.linhas[0].motivo, "invalido");
    eq(r.linhas[0].frase, "o nome não pode ser vazio");
    r = L.calcular([it("a1.txt")], com({ de: "a", para: "b/c" }), pastaP);
    igual([r.linhas[0].motivo, r.linhas[0].frase], ["invalido", "o nome não pode ter «/»"]);
    eq(L.erroDoNome("  "), "o nome não pode ser vazio");
    eq(L.erroDoNome(".."), "«..» não serve de nome");
    eq(L.erroDoNome("a\u0000"), "nome com byte nulo");
    eq(L.erroDoNome("ok.txt"), "");
});

t("calcular: troca em cadeia e ciclo dentro do lote não são conflito (o motor resolve com temporários)", () => {
    const pastaP = { pasta: "/p", nomes: ["1.txt", "2.txt", "3.txt"] };
    // 1→2, 2→3, 3→4: cada nome novo é de um item do próprio lote, que sai de lá.
    let r = L.calcular([it("1.txt"), it("2.txt"), it("3.txt")], com({ modelo: "{n}", inicio: 2 }), pastaP);
    eq(r.conflitos, 0);
    igual(r.pares, [["/p/1.txt", "2.txt"], ["/p/2.txt", "3.txt"], ["/p/3.txt", "4.txt"]]);
    // a↔b.
    r = L.calcular([it("a.txt"), it("b.txt")], com({ modelo: "{n}" }), { pasta: "/p", nomes: ["a.txt", "b.txt", "1.txt"] });
    eq(r.linhas[0].motivo, "existe");
    r = L.calcular([it("1.txt"), it("2.txt")], com({ modelo: "{n}", inicio: 2, passo: -1 }), pastaP);
    igual(r.pares, [["/p/1.txt", "2.txt"], ["/p/2.txt", "1.txt"]]);
    eq(r.conflitos, 0);
    // Mas o item do lote que NÃO muda continua ocupando o nome dele.
    r = L.calcular([it("a.txt"), it("b.txt")], com({ de: "a", para: "b" }), { pasta: "/p", nomes: ["a.txt", "b.txt"] });
    eq(r.conflitos, 2);
});

t("calcular: itens de pastas diferentes (uma busca) só se chocam dentro da mesma pasta", () => {
    const r = L.calcular([it("a.txt", { caminho: "/p/a.txt" }), it("a.txt", { caminho: "/q/a.txt" })], com({ modelo: "novo" }), null);
    eq(r.conflitos, 0);
    igual(r.pares, [["/p/a.txt", "novo.txt"], ["/q/a.txt", "novo.txt"]]);
    // Sem a listagem da pasta ninguém é marcado como "existe": o motor confere.
    eq(L.calcular([it("a.txt")], com({ modelo: "b" }), null).conflitos, 0);
});

t("recusa: o que dizer quando o motor recusa o lote", () => {
    eq(L.recusa({ erro: "há conflitos", conflitos: [{ de: "/p/a.txt", para: "b.txt", motivo: "existe" }] }), "há conflitos: «b.txt» (já existe na pasta)");
    eq(L.recusa({ erro: "há conflitos", conflitos: [{ de: "/p/a", para: "x", motivo: "outro" }] }), "há conflitos: «x» (outro)");
    eq(L.recusa({ erro: "sem permissão" }), "sem permissão");
    eq(L.recusa({ erro: "x", conflitos: [] }), "x");
    eq(L.recusa(null), "");
});
