// apps/mkfiles/qml/Arquivos.js: a lógica da janela do MkFiles que não depende
// do QML: ordem natural, filtro, textos em pt-BR, nomes livres, caminhos e
// migalhas, os URIs da área de transferência, o evento de pasta e os exemplos.
process.env.TZ = "UTC";
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const A = load(REPO + "/apps/mkfiles/qml/Arquivos.js");
const it = (nome, mais) => Object.assign({ nome, caminho: "/p/" + nome, tipo: "arquivo", tamanho: 0, mtime: 0, oculto: false, mime: "text/plain" }, mais || {});
const pasta = (nome, mais) => it(nome, Object.assign({ tipo: "pasta", mime: "inode/directory" }, mais || {}));
const nomes = (l) => l.map((x) => x.nome);

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["semAcento", "chave", "ePasta", "chaveDe", "comparar", "ordenar", "casaPadrao", "filtrar", "visiveis", "aplicarEvento",
        "pular", "virgula", "tamanho", "dois", "data", "tipo", "partes", "semExtensao", "nomeLivre", "erroDoNome", "pai", "nomeDe", "juntar",
        "dentro", "limpar", "migalhas", "til", "completar", "uri", "deUri", "limparSimples", "listaDeUris", "lerListaDeUris", "copiados",
        "lerCopiados", "pedidoDeCopia", "lerArea", "soma", "plural", "estado", "progresso", "tarefaAcabou", "detalheDaTarefa", "erroDePasta",
        "bit", "temBit", "trocarBit", "octal", "arquivoDeExemplo", "comandoDeExemplo", "caminhosDeTexto", "tamanhoEmBytes", "dataDeTexto",
        "filtrosDaBusca", "temFiltro", "noCompactado", "partesDoCompactado", "pastaReal", "eCompactado", "pastaDeExtracao", "nomeDoCompactado",
        "comFormato", "marcaDoCompactado", "lerMarca"];
    const funcoes = Object.keys(A).filter((k) => typeof A[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("semAcento e chave: a ordem natural entende números e acentos", () => {
    eq(A.semAcento("AÇÃO Órgão"), "acao orgao");
    const ordem = ["foto 10.jpg", "foto 9.jpg", "Foto 1.jpg", "água.txt", "abacaxi", "zebra", "Árvore", "foto 009b.jpg"];
    igual(ordem.slice().sort((a, b) => (A.chave(a) < A.chave(b) ? -1 : 1)),
        ["abacaxi", "água.txt", "Árvore", "Foto 1.jpg", "foto 9.jpg", "foto 009b.jpg", "foto 10.jpg", "zebra"]);
    eq(A.chave("a007") === A.chave("a7"), true);
    eq(A.chave("v2") < A.chave("v10"), true);
    eq(A.chaveDe(it("B2")), A.chave("b2"));
});

t("ePasta: pasta, ou link que leva a uma", () => {
    eq(A.ePasta(pasta("a")), true);
    eq(A.ePasta(it("a", { tipo: "link", alvo_tipo: "pasta" })), true);
    eq(A.ePasta(it("a", { tipo: "link", alvo_tipo: "arquivo" })), false);
    eq(A.ePasta(it("a")), false);
    eq(A.ePasta(null), false);
});

t("ordenar: por nome, tamanho, data e tipo, com as pastas antes e a ordem inversa", () => {
    const l = [it("b.txt", { tamanho: 5, mtime: 30 }), pasta("z"), it("a10.png", { tamanho: 9, mtime: 10, mime: "image/png" }),
        it("a2.png", { tamanho: 1, mtime: 20, mime: "image/png" }), pasta("Á")];
    igual(nomes(A.ordenar(l, { campo: "nome", inversa: false }, true)), ["Á", "z", "a2.png", "a10.png", "b.txt"]);
    igual(nomes(A.ordenar(l, { campo: "nome", inversa: false }, false)), ["Á", "a2.png", "a10.png", "b.txt", "z"]);
    igual(nomes(A.ordenar(l, { campo: "nome", inversa: true }, false)), ["z", "b.txt", "a10.png", "a2.png", "Á"]);
    igual(nomes(A.ordenar(l, { campo: "tamanho", inversa: false }, true)), ["Á", "z", "a2.png", "b.txt", "a10.png"]);
    igual(nomes(A.ordenar(l, { campo: "mtime", inversa: true }, true)), ["z", "Á", "b.txt", "a2.png", "a10.png"]);
    igual(nomes(A.ordenar(l, { campo: "tipo", inversa: false }, true)), ["Á", "z", "a2.png", "a10.png", "b.txt"]);
    eq(l[0].nome, "b.txt", "a lista original não muda");
    eq(A.comparar(it("a"), it("a"), null, true), 0);
    eq(A.comparar(it("x", { apagado_s: 1 }), it("a", { apagado_s: 2 }), { campo: "apagado" }, false) < 0, true);
});

t("casaPadrao: * e ?, sem diferenciar maiúsculas nem acentos; sem curinga é um trecho", () => {
    eq(A.casaPadrao("Foto.JPG", "*.jpg"), true);
    eq(A.casaPadrao("foto.jpeg", "*.jpg"), false);
    eq(A.casaPadrao("relatório final.pdf", "relatorio*"), true);
    eq(A.casaPadrao("a1.txt", "a?.txt"), true);
    eq(A.casaPadrao("a12.txt", "a?.txt"), false);
    eq(A.casaPadrao("Minha Ação.txt", "acao"), true);
    eq(A.casaPadrao("x(1)[2].txt", "x(1)[2].*"), true);
    eq(A.casaPadrao("qualquer", ""), true);
    eq(A.casaPadrao("a.b", "a+b"), false);
});

t("filtrar e visiveis: ocultos só quando pedidos, e o texto do filtro", () => {
    const l = [it("b"), it(".a", { oculto: true }), pasta("c"), it("escondido", { oculto: true })];
    igual(nomes(A.filtrar(l, {})), ["b", "c"]);
    igual(nomes(A.filtrar(l, { ocultos: true })), ["b", ".a", "c", "escondido"]);
    igual(nomes(A.filtrar(l, { ocultos: true, texto: "esc" })), ["escondido"]);
    igual(nomes(A.visiveis(l, { ocultos: true, ordem: { campo: "nome" }, pastas_antes: true })), ["c", ".a", "b", "escondido"]);
    igual(nomes(A.visiveis(l, null)), ["c", "b"]);
});

t("aplicarEvento: o que mudou entra ou troca, o que saiu some", () => {
    const l = [it("a", { tamanho: 1 }), it("b"), it("c")];
    const r = A.aplicarEvento(l, { mudou: [it("a", { tamanho: 9 }), it("novo")], saiu: ["b", "nao-ha"] });
    igual(nomes(r), ["a", "c", "novo"]);
    eq(r[0].tamanho, 9);
    eq(l.length, 3);
    igual(nomes(A.aplicarEvento(l, {})), ["a", "b", "c"]);
    // Renomear "b" para "c": sai um, o outro troca.
    igual(nomes(A.aplicarEvento(l, { mudou: [it("c", { tamanho: 7 })], saiu: ["b"] })), ["a", "c"]);
});

t("pular: digitar leva ao primeiro nome que começa assim, dando a volta", () => {
    const l = [it("Água"), it("bola"), it("Bolo"), it("casa")];
    eq(A.pular(l, "bo", 0), 1);
    eq(A.pular(l, "bo", 2), 2);
    eq(A.pular(l, "ag", 3), 0);
    eq(A.pular(l, "x", 0), -1);
    eq(A.pular(l, "", 0), -1);
    eq(A.pular([], "a", 0), -1);
    eq(A.pular(l, "c", 99), 3);
});

t("tamanho: bytes em pt-BR, com vírgula", () => {
    eq(A.tamanho(0), "0 B");
    eq(A.tamanho(999), "999 B");
    eq(A.tamanho(1000), "1 kB");
    eq(A.tamanho(1536), "1,5 kB");
    eq(A.tamanho(20480), "20,5 kB");
    eq(A.tamanho(123456789), "123 MB");
    eq(A.tamanho(16000000000), "16 GB");
    eq(A.tamanho(999999), "1 MB");
    eq(A.tamanho(null), "—");
    eq(A.virgula(1.25, 1), "1,3");
    eq(A.dois(7), "07");
    eq(A.dois(12), "12");
});

t("data: hoje, ontem, este ano e outros anos", () => {
    const agora = Date.UTC(2026, 2, 10, 15, 0, 0) / 1000;
    eq(A.data(Date.UTC(2026, 2, 10, 9, 5, 0) / 1000, agora), "hoje, 09:05");
    eq(A.data(Date.UTC(2026, 2, 9, 23, 59, 0) / 1000, agora), "ontem, 23:59");
    eq(A.data(Date.UTC(2026, 0, 3, 8, 15, 0) / 1000, agora), "3 de jan., 08:15");
    eq(A.data(Date.UTC(2021, 11, 25, 8, 15, 0) / 1000, agora), "25 de dez. de 2021");
    eq(A.data(0, agora), "—");
});

t("tipo: o MIME como a pessoa lê", () => {
    eq(A.tipo(pasta("a")), "Pasta");
    eq(A.tipo(it("a", { mime: "image/jpeg" })), "Imagem JPEG");
    eq(A.tipo(it("a", { mime: "video/x-matroska" })), "Vídeo MATROSKA");
    eq(A.tipo(it("a", { mime: "application/pdf" })), "Documento PDF");
    eq(A.tipo(it("a", { mime: "text/plain" })), "Texto");
    eq(A.tipo(it("a", { mime: "application/vnd.oasis.opendocument.text" })), "Documento de escritório");
    eq(A.tipo(it("a", { mime: "application/x-foo" })), "Arquivo foo");
    eq(A.tipo(it("a", { tipo: "link", alvo_tipo: "pasta", mime: "inode/directory" })), "Link para pasta");
    eq(A.tipo(it("a", { tipo: "link", quebrado: true, mime: "inode/symlink" })), "Link quebrado");
    eq(A.tipo(null), "");
    eq(A.tipo(it("a", { mime: "" })), "Arquivo");
});

t("partes e semExtensao: o F2 seleciona só o nome", () => {
    igual(A.partes("foto.jpg"), ["foto", ".jpg"]);
    igual(A.partes("a.tar.gz"), ["a", ".tar.gz"]);
    igual(A.partes(".bashrc"), [".bashrc", ""]);
    igual(A.partes("sem"), ["sem", ""]);
    igual(A.partes("fim."), ["fim.", ""]);
    igual(A.partes("pasta.d", true), ["pasta.d", ""]);
    eq(A.semExtensao("relatório final.pdf"), 15);
    eq(A.semExtensao("pasta.v2", true), 8);
});

t("nomeLivre: x (2).ext, x (3).ext, e o número antes do .tar", () => {
    const ha = (lista) => (n) => lista.includes(n);
    eq(A.nomeLivre("a.txt", ha([])), "a.txt");
    eq(A.nomeLivre("a.txt", ha(["a.txt"])), "a (2).txt");
    eq(A.nomeLivre("a.txt", ha(["a.txt", "a (2).txt"])), "a (3).txt");
    eq(A.nomeLivre("a (2).txt", ha(["a (2).txt"])), "a (3).txt");
    eq(A.nomeLivre("b.tar.gz", ha(["b.tar.gz"])), "b (2).tar.gz");
    eq(A.nomeLivre("pasta.v1", ha(["pasta.v1"]), true), "pasta.v1 (2)");
    eq(A.nomeLivre(".oculto", ha([".oculto"])), ".oculto (2)");
});

t("erroDoNome: as mesmas regras do motor", () => {
    eq(A.erroDoNome("a b.txt"), "");
    for (const ruim of ["", ".", "..", "a/b", "x".repeat(256)])
        eq(A.erroDoNome(ruim) !== "", true, ruim);
});

t("pai, nomeDe, juntar e dentro", () => {
    eq(A.pai("/a/b"), "/a");
    eq(A.pai("/a"), "/");
    eq(A.pai("/"), "/");
    eq(A.nomeDe("/a/b c"), "b c");
    eq(A.nomeDe("/"), "/");
    eq(A.juntar("/", "a"), "/a");
    eq(A.juntar("/a", "b"), "/a/b");
    eq(A.dentro("/a", "/a/b"), true);
    eq(A.dentro("/a", "/a"), true);
    eq(A.dentro("/a", "/ab"), false);
    eq(A.dentro("/", "/x"), true);
});

t("limpar: o caminho digitado vira absoluto e limpo, ou null", () => {
    eq(A.limpar("~", "/home/ana"), "/home/ana");
    eq(A.limpar("~/Doc/", "/home/ana"), "/home/ana/Doc");
    eq(A.limpar(" /a//b/./c/../d ", "/home/ana"), "/a/b/d");
    eq(A.limpar("/..", "/home/ana"), "/");
    eq(A.limpar("file:///a/b%20c", "/home/ana"), "/a/b c");
    eq(A.limpar("relativo", "/home/ana"), null);
    eq(A.limpar("", "/home/ana"), null);
    eq(A.limpar("sftp://x/a", "/home/ana"), null);
    eq(A.limparSimples("/a/../b//c/."), "/b/c");
});

t("migalhas e til: a pasta pessoal é uma migalha só", () => {
    igual(A.migalhas("/home/ana/Doc/x y", "/home/ana"), [{ nome: "Pasta pessoal", caminho: "/home/ana" },
        { nome: "Doc", caminho: "/home/ana/Doc" }, { nome: "x y", caminho: "/home/ana/Doc/x y" }]);
    igual(A.migalhas("/usr/share", "/home/ana"), [{ nome: "/", caminho: "/" }, { nome: "usr", caminho: "/usr" }, { nome: "share", caminho: "/usr/share" }]);
    igual(A.migalhas("/", "/home/ana"), [{ nome: "/", caminho: "/" }]);
    igual(A.migalhas("/home/anabela", "/home/ana").length, 3);
    eq(A.til("/home/ana/Doc", "/home/ana"), "~/Doc");
    eq(A.til("/home/ana", "/home/ana"), "~");
    eq(A.til("/etc", "/home/ana"), "/etc");
});

t("completar: o Tab do campo de caminho completa nomes de pasta", () => {
    const l = [pasta("Documentos"), pasta("Downloads"), pasta("Imagens"), it("Doc.txt"), pasta(".config", { oculto: true })];
    igual(A.completar("/home/ana/Im", l), { texto: "/home/ana/Imagens/", opcoes: [] });
    igual(A.completar("/home/ana/d", l), { texto: "/home/ana/Do", opcoes: ["Documentos", "Downloads"] });
    igual(A.completar("/home/ana/Doc", l), { texto: "/home/ana/Documentos/", opcoes: [] });
    igual(A.completar("/home/ana/x", l), { texto: "/home/ana/x", opcoes: [] });
    igual(A.completar("/home/ana/.c", l), { texto: "/home/ana/.config/", opcoes: [] });
    igual(A.completar("/home/ana/", l).opcoes, ["Documentos", "Downloads", "Imagens"]);
    igual(A.completar("solto", l), { texto: "solto", opcoes: [] });
});

t("uri e deUri: a mesma codificação do motor (PathEscape por pedaço)", () => {
    const c = "/home/ana/Meus Documentos/ação #1%.txt";
    eq(A.uri(c), "file:///home/ana/Meus%20Documentos/a%C3%A7%C3%A3o%20%231%25.txt");
    eq(A.deUri(A.uri(c)), c);
    eq(A.uri("/a/b(1)'!*;,?.txt"), "file:///a/b%281%29%27%21%2A%3B%2C%3F.txt");
    eq(A.uri("/a/$&+:=@~-_."), "file:///a/$&+:=@~-_.");
    eq(A.deUri("file://localhost/a/b"), "/a/b");
    for (const ruim of ["sftp://x/a", "file://outra/a", "/a", "file:///a%zz", "file:///a%00b"])
        eq(A.deUri(ruim), null, ruim);
});

t("text/uri-list e gnome-copied-files: montar e ler", () => {
    eq(A.listaDeUris(["/a b", "/c"]), "file:///a%20b\r\nfile:///c\r\n");
    igual(A.lerListaDeUris("# comentário\r\nfile:///a%20b\r\nsftp://x/y\r\nfile:///c\n"), ["/a b", "/c"]);
    eq(A.copiados(["/a b", "/c"], false), "copy\nfile:///a%20b\nfile:///c");
    eq(A.copiados(["/a"], true), "cut\nfile:///a");
    igual(A.lerCopiados("cut\nfile:///a%20b\nfile:///c\n"), { recortar: true, caminhos: ["/a b", "/c"] });
    igual(A.lerCopiados("copy\nfile:///a"), { recortar: false, caminhos: ["/a"] });
    eq(A.lerCopiados("file:///a"), null);
    // Copiar e recortar põem sempre o gnome-copied-files: é o que os gerenciadores leem.
    igual(A.pedidoDeCopia(["/a"], false), { op: "area-por", tipo: "x-special/gnome-copied-files", texto: "copy\nfile:///a" });
    igual(A.pedidoDeCopia(["/a"], true), { op: "area-por", tipo: "x-special/gnome-copied-files", texto: "cut\nfile:///a" });
});

t("lerArea: o que há para colar", () => {
    igual(A.lerArea({ tipo: "x-special/gnome-copied-files", texto: "cut\nfile:///a" }), { tipo: "arquivos", recortar: true, caminhos: ["/a"] });
    igual(A.lerArea({ tipo: "text/uri-list", texto: "file:///a\r\nfile:///b\r\n" }), { tipo: "arquivos", recortar: false, caminhos: ["/a", "/b"] });
    eq(A.lerArea({ tipo: "image/png", texto: null }).tipo, "imagem");
    eq(A.lerArea({ tipo: "text/plain", texto: "olá" }).tipo, "texto");
    igual(A.lerArea({ tipo: "text/plain", texto: "file:///a\n" }).caminhos, ["/a"]);
    // Caminhos e URIs, um por linha, colados como texto: são arquivos (copiar, não recortar).
    igual(A.lerArea({ tipo: "text/plain", texto: "/home/ana/a b.txt\r\nfile:///c%20d\n\n" }), { tipo: "arquivos", recortar: false, caminhos: ["/home/ana/a b.txt", "/c d"] });
    eq(A.lerArea({ tipo: "text/plain", texto: "/home/ana/a.txt\ne mais uma frase" }).tipo, "texto");
    igual(A.caminhosDeTexto("/a/../b\n"), ["/b"]);
    igual(A.caminhosDeTexto("relativo"), []);
    igual(A.caminhosDeTexto(""), []);
    eq(A.lerArea({ tipo: null, texto: null }).tipo, "vazia");
    eq(A.lerArea({ erro: "x" }).tipo, "vazia");
    eq(A.lerArea({ tipo: "text/uri-list", texto: "" }).tipo, "vazia");
    eq(A.lerArea(null).tipo, "vazia");
});

t("soma, plural e estado: a barra de estado", () => {
    igual(A.soma([it("a", { tamanho: 1000 }), pasta("p"), it("b", { tamanho: 500 })]), { n: 3, pastas: 1, tamanho: 1500 });
    eq(A.plural(1, "item", "itens"), "1 item");
    eq(A.plural(0, "item", "itens"), "0 itens");
    eq(A.estado(42, { n: 0, pastas: 0, tamanho: 0 }, null, false), "42 itens");
    eq(A.estado(42, { n: 2, pastas: 0, tamanho: 1500 }, { livre: 5e9, total: 1e10 }, false), "42 itens  ·  2 selecionados (1,5 kB)  ·  5 GB livres de 10 GB");
    eq(A.estado(1, { n: 1, pastas: 1, tamanho: 0 }, null, true), "lendo… 1 item  ·  1 selecionado");
});

t("progresso, tarefaAcabou e detalheDaTarefa: a área de tarefas", () => {
    eq(A.progresso({ bytes: 50, bytes_total: 200, itens: 1, itens_total: 2 }), 0.25);
    eq(A.progresso({ bytes: 0, bytes_total: 0, itens: 1, itens_total: 2 }), 0.5);
    eq(A.progresso({}), 0);
    eq(A.progresso(null), 0);
    eq(A.tarefaAcabou({ estado: "feita" }), true);
    eq(A.tarefaAcabou({ estado: "erro" }), true);
    eq(A.tarefaAcabou({ estado: "conflito" }), false);
    eq(A.detalheDaTarefa({ estado: "rodando", itens: 1, itens_total: 3, bytes: 1e6, bytes_total: 9e6, velocidade: 5e6 }), "1 de 3  ·  1 MB de 9 MB  ·  5 MB/s");
    eq(A.detalheDaTarefa({ estado: "pausada" }), "pausada");
    eq(A.detalheDaTarefa({ estado: "erro", erros: [{ caminho: "/a", erro: "sem permissão" }] }), "1 erro: sem permissão");
});

t("erroDePasta: a janela diz o que houve, nunca fica em branco", () => {
    for (const m of ["nao-existe", "sem-permissao", "nao-e-pasta", "sumiu", "motor", "erro"])
        eq(A.erroDePasta(m, "frase do motor").titulo !== "", true, m);
    eq(A.erroDePasta("erro", "frase do motor").frase, "frase do motor");
    eq(A.erroDePasta("sem-permissao", "x").frase !== "x", true);
});

t("bit, temBit, trocarBit e octal: a folha de permissões", () => {
    eq(A.bit(0, "r"), 0o400);
    eq(A.bit(1, "w"), 0o020);
    eq(A.bit(2, "x"), 0o001);
    eq(A.temBit(0o644, 0, "w"), true);
    eq(A.temBit(0o644, 2, "w"), false);
    eq(A.trocarBit(0o644, 0, "x"), 0o744);
    eq(A.trocarBit(0o755, 2, "r"), 0o751);
    eq(A.octal(0o644), "644");
    eq(A.octal(0o4755), "4755");
    eq(A.octal(0), "000");
});

t("o modo de exemplo: um arquivo por pedido, lido com o jq", () => {
    eq(A.arquivoDeExemplo({ op: "listar", caminho: "/x" }), "listar.json");
    eq(A.arquivoDeExemplo({ op: "../../etc/passwd" }), "etcpasswd.json");
    igual(A.comandoDeExemplo("/ex", { op: "lixeira-listar" }), ["jq", "-c", ".[]", "/ex/lixeira-listar.json"]);
});

t("os filtros da busca: tamanho com unidade e datas viram os quatro campos do pedido", () => {
    eq(A.tamanhoEmBytes("10", "KB"), 10240);
    eq(A.tamanhoEmBytes("1,5", "MB"), 1572864);
    eq(A.tamanhoEmBytes("2", "GB"), 2147483648);
    eq(A.tamanhoEmBytes("7", "outra"), 7);
    for (const ruim of ["", "abc", "-1", "1e3", null])
        eq(A.tamanhoEmBytes(ruim, "MB"), null, String(ruim));
    eq(A.dataDeTexto("2024-02-29"), 1709164800);
    eq(A.dataDeTexto("29/02/2024"), 1709164800);
    for (const ruim of ["2023-02-29", "31/04/2024", "ontem", "", "2024-13-01"])
        eq(A.dataDeTexto(ruim), null, ruim);
    const agora = 1700000000; // 14/11/2023 22:13 UTC
    igual(A.filtrosDaBusca({}, agora), { tamanho_min: null, tamanho_max: null, depois_s: null, antes_s: null });
    igual(A.filtrosDaBusca(null, agora), { tamanho_min: null, tamanho_max: null, depois_s: null, antes_s: null });
    igual(A.filtrosDaBusca({ min: "1", max: "2", unidade: "MB", periodo: "hoje" }, agora), { tamanho_min: 1048576, tamanho_max: 2097152, depois_s: 1699920000, antes_s: null });
    eq(A.filtrosDaBusca({ periodo: "7" }, agora).depois_s, 1699920000 - 6 * 86400);
    eq(A.filtrosDaBusca({ periodo: "30" }, agora).depois_s, 1699920000 - 29 * 86400);
    eq(A.filtrosDaBusca({ periodo: "ano" }, agora).depois_s, 1672531200);
    // Entre datas: o "até" inclui o dia inteiro.
    igual(A.filtrosDaBusca({ periodo: "datas", de: "2023-01-01", ate: "2023-01-31" }, agora), { tamanho_min: null, tamanho_max: null, depois_s: 1672531200, antes_s: 1675209600 });
    igual(A.filtrosDaBusca({ periodo: "datas", de: "", ate: "torto" }, agora), { tamanho_min: null, tamanho_max: null, depois_s: null, antes_s: null });
    eq(A.temFiltro({ min: "", max: "", periodo: "" }), false);
    eq(A.temFiltro(null), false);
    eq(A.temFiltro({ min: "3", unidade: "KB" }), true);
    eq(A.temFiltro({ periodo: "ano" }), true);
    eq(A.PERIODOS.map((p) => p[0]).join(" "), " hoje 7 30 ano datas");
});

t("dentro de um compactado: o lugar é arquivo//interno, com migalhas, subir e nomes", () => {
    eq(A.noCompactado("/x/a.zip", ""), "/x/a.zip//");
    eq(A.noCompactado("/x/a.zip", "/pasta/sub/"), "/x/a.zip//pasta/sub");
    igual(A.partesDoCompactado("/x/a.zip//pasta/sub"), { arquivo: "/x/a.zip", interno: "pasta/sub" });
    igual(A.partesDoCompactado("/x/a.zip//"), { arquivo: "/x/a.zip", interno: "" });
    eq(A.partesDoCompactado("/x/a.zip"), null);
    eq(A.partesDoCompactado("/"), null);
    // Subir: pasta de dentro, raiz do compactado, e daí para a pasta do arquivo.
    eq(A.pai("/x/a.zip//pasta/sub"), "/x/a.zip//pasta");
    eq(A.pai("/x/a.zip//pasta"), "/x/a.zip//");
    eq(A.pai("/x/a.zip//"), "/x");
    eq(A.nomeDe("/x/a.zip//"), "a.zip");
    eq(A.nomeDe("/x/a.zip//pasta/sub"), "sub");
    eq(A.juntar("/x/a.zip//", "f.txt"), "/x/a.zip//f.txt");
    eq(A.juntar("/x/a.zip//pasta", "f.txt"), "/x/a.zip//pasta/f.txt");
    eq(A.pastaReal("/x/a.zip//pasta"), "/x");
    eq(A.pastaReal("/x/y"), "/x/y");
    // As migalhas: «a.zip › pasta», cada uma levando ao lugar certo.
    igual(A.migalhas("/home/ana/a.zip//pasta/sub", "/home/ana"), [{ nome: "Pasta pessoal", caminho: "/home/ana" }, { nome: "a.zip", caminho: "/home/ana/a.zip//" },
        { nome: "pasta", caminho: "/home/ana/a.zip//pasta" }, { nome: "sub", caminho: "/home/ana/a.zip//pasta/sub" }]);
    igual(A.migalhas("/x/a.zip//", "/home/ana").map((m) => m.nome), ["/", "x", "a.zip"]);
});

t("eCompactado, pastaDeExtracao, nomeDoCompactado e comFormato", () => {
    eq(A.eCompactado(it("a.zip", { mime: "application/zip" })), true);
    eq(A.eCompactado(it("a.tar.zst", { mime: "application/x-zstd-compressed-tar" })), true);
    eq(A.eCompactado(it("A.RAR", { mime: "application/octet-stream" })), true);
    eq(A.eCompactado(it("a.txt")), false);
    eq(A.eCompactado(pasta("a.zip")), false);
    eq(A.eCompactado(null), false);
    eq(A.pastaDeExtracao("fotos.zip"), "fotos");
    eq(A.pastaDeExtracao("fonte-1.2.tar.gz"), "fonte-1.2");
    eq(A.pastaDeExtracao("x.tgz"), "x");
    eq(A.pastaDeExtracao("semext"), "semext (extraído)");
    eq(A.nomeDoCompactado([it("relatório.pdf")], "/p", "zip"), "relatório.zip");
    eq(A.nomeDoCompactado([pasta("fotos.2024")], "/p", "tar.gz"), "fotos.2024.tar.gz");
    eq(A.nomeDoCompactado([it("a"), it("b")], "/home/ana/Viagem", "7z"), "Viagem.7z");
    eq(A.nomeDoCompactado([it("a"), it("b")], "/", "zip"), "Arquivos.zip");
    eq(A.comFormato("Viagem.zip", "tar.gz"), "Viagem.tar.gz");
    eq(A.comFormato("Viagem.tar.zst", "7z"), "Viagem.7z");
    eq(A.comFormato("v1.2", "zip"), "v1.2.zip");
    eq(A.comFormato(".zip", "zip"), "Arquivos.zip");
});

t("a marca de quem copiou de dentro de um compactado", () => {
    const m = A.marcaDoCompactado("/x/a.zip", ["pasta/f.txt", "g.txt"]);
    igual(A.lerMarca(m), { arquivo: "/x/a.zip", internos: ["pasta/f.txt", "g.txt"] });
    eq(A.lerMarca("texto qualquer"), null);
    eq(A.lerMarca("mkfiles-compactado:/x/a.zip\n"), null);
    eq(A.lerMarca("mkfiles-compactado:relativo\nf"), null);
    eq(A.lerMarca(null), null);
    // E um compactado que pede senha, ou que falta ferramenta, tem título próprio.
    eq(A.erroDePasta("senha", "x").titulo, "Este compactado pede uma senha");
    eq(A.erroDePasta("falta", "falta o unrar").frase, "falta o unrar");
});
