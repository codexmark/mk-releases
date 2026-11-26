// apps/mkfiles/qml/Selecao.js: a seleção de um painel (clique, Ctrl, Shift,
// teclado, inverter, por padrão, retângulo do mouse).
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const S = load(REPO + "/apps/mkfiles/qml/Selecao.js");
const itens = ["a.jpg", "b.txt", "c.jpg", "d.txt", "e.JPG"].map((nome) => ({ nome, caminho: "/p/" + nome }));
const sel = (e) => Object.keys(e.sel).map((c) => c.slice(3)).sort().join(" ");

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["nova", "feita", "copia", "tem", "so", "comIntervalo", "clicar", "alternar", "andar", "todos", "inverter", "porPadrao",
        "comIndices", "porCaminhos", "lista", "caminhos", "reancorar", "passo", "noRetangulo", "destino", "mover"];
    const funcoes = Object.keys(S).filter((k) => typeof S[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("nova e so: vazia, e um item só", () => {
    igual(S.nova(), { sel: {}, n: 0, ancora: -1, cursor: -1, onde: "" });
    igual(S.so(itens, 2), { sel: { "/p/c.jpg": true }, n: 1, ancora: 2, cursor: 2, onde: "/p/c.jpg" });
    eq(S.so(itens, 99).n, 0);
    eq(S.tem(S.so(itens, 2), itens[2]), true);
    eq(S.tem(S.so(itens, 2), itens[1]), false);
    eq(S.tem(S.nova(), null), false);
    igual(S.feita({}, 0, -1, 9, itens).onde, "");
    igual(S.copia({ a: true }), { a: true });
});

t("clicar: simples troca, Ctrl liga e desliga, Shift estende da âncora", () => {
    let e = S.clicar(S.nova(), itens, 1, {});
    eq(sel(e), "b.txt");
    e = S.clicar(e, itens, 3, { ctrl: true });
    eq(sel(e), "b.txt d.txt");
    eq(e.n, 2);
    e = S.clicar(e, itens, 1, { ctrl: true });
    eq(sel(e), "d.txt");
    eq(e.ancora, 1);
    e = S.clicar(S.so(itens, 1), itens, 3, { shift: true });
    eq(sel(e), "b.txt c.jpg d.txt");
    eq(e.ancora, 1);
    eq(e.cursor, 3);
    // Shift de novo, para o outro lado: o intervalo sai da mesma âncora.
    e = S.clicar(e, itens, 0, { shift: true });
    eq(sel(e), "a.jpg b.txt");
    // Ctrl+Shift soma o intervalo ao que havia.
    e = S.clicar(S.alternar(S.so(itens, 0), itens, 3), itens, 4, { ctrl: true, shift: true });
    eq(sel(e), "a.jpg d.txt e.JPG");
    eq(e.n, 3);
    // No vazio: limpa, a não ser com um modificador.
    eq(S.clicar(S.so(itens, 1), itens, -1, {}).n, 0);
    eq(S.clicar(S.so(itens, 1), itens, -1, { ctrl: true }).n, 1);
    eq(S.comIntervalo({}, 0, itens, 3, 1), 3);
});

t("andar: o teclado seleciona, estende com Shift e só anda com Ctrl", () => {
    let e = S.andar(S.so(itens, 0), itens, 1, {});
    eq(sel(e), "b.txt");
    e = S.andar(e, itens, 3, { shift: true });
    eq(sel(e), "b.txt c.jpg d.txt");
    e = S.andar(e, itens, 4, { ctrl: true });
    eq(sel(e), "b.txt c.jpg d.txt");
    eq(e.cursor, 4);
    eq(e.onde, "/p/e.JPG");
    eq(S.andar(e, itens, 99, {}), e);
    eq(S.alternar(e, itens, -1), e);
});

t("todos, inverter e porPadrao", () => {
    eq(S.todos(itens).n, 5);
    eq(S.todos([]).cursor, -1);
    eq(sel(S.inverter(S.clicar(S.so(itens, 0), itens, 1, { shift: true }), itens)), "c.jpg d.txt e.JPG");
    eq(S.inverter(S.todos(itens), itens).n, 0);
    const e = S.porPadrao(S.so(itens, 1), itens, "*.jpg");
    eq(sel(e), "a.jpg b.txt c.jpg e.JPG");
    eq(e.n, 4);
    eq(e.cursor, 0);
    eq(S.porPadrao(S.so(itens, 1), itens, "*.png").n, 1);
});

t("comIndices e porCaminhos: o retângulo soma à base; o recém-criado fica selecionado", () => {
    const e = S.comIndices(S.so(itens, 0), itens, [2, 3, 99, 0]);
    eq(sel(e), "a.jpg c.jpg d.txt");
    eq(e.n, 3);
    eq(sel(S.comIndices(S.nova(), itens, [])), "");
    const c = S.porCaminhos(itens, ["/p/d.txt", "/p/nao-ha", "/p/b.txt"]);
    eq(sel(c), "b.txt d.txt");
    eq(c.cursor, 1);
});

t("lista e caminhos: o selecionado, na ordem da tela", () => {
    const e = S.alternar(S.so(itens, 3), itens, 0);
    igual(S.lista(e, itens).map((i) => i.nome), ["a.jpg", "d.txt"]);
    igual(S.caminhos(e, itens), ["/p/a.jpg", "/p/d.txt"]);
    igual(S.lista(S.nova(), itens), []);
});

t("reancorar: depois de reordenar, fica o que ainda existe e o cursor acha o item", () => {
    const e = S.clicar(S.so(itens, 1), itens, 3, { shift: true });
    const novos = [itens[4], itens[3], itens[1]];
    const r = S.reancorar(e, novos);
    eq(sel(r), "b.txt d.txt");
    eq(r.n, 2);
    eq(r.cursor, 1);
    eq(r.onde, "/p/d.txt");
    // O item do cursor sumiu: o cursor fica na mesma altura, dentro da lista.
    eq(S.reancorar(S.so(itens, 4), itens.slice(0, 2)).cursor, 1);
    eq(S.reancorar(S.nova(), itens).cursor, -1);
});

t("passo: setas, Home/End e páginas, na grade e na lista", () => {
    eq(S.passo("baixo", 0, 10, 1, 5), 1);
    eq(S.passo("cima", 0, 10, 1, 5), 0);
    eq(S.passo("fim", 3, 10, 1, 5), 9);
    eq(S.passo("inicio", 3, 10, 1, 5), 0);
    eq(S.passo("pagina-baixo", 2, 10, 1, 5), 7);
    eq(S.passo("pagina-baixo", 8, 10, 1, 5), 9);
    eq(S.passo("pagina-cima", 2, 10, 1, 5), 0);
    // Grade de 4 colunas com 10 itens (a última linha tem 2).
    eq(S.passo("direita", 3, 10, 4, 2), 4);
    eq(S.passo("esquerda", 0, 10, 4, 2), 0);
    eq(S.passo("baixo", 1, 10, 4, 2), 5);
    eq(S.passo("baixo", 6, 10, 4, 2), 9, "sem item embaixo, vai ao último da última linha");
    eq(S.passo("baixo", 9, 10, 4, 2), 9);
    eq(S.passo("cima", 5, 10, 4, 2), 1);
    eq(S.passo("cima", 2, 10, 4, 2), 2);
    eq(S.passo("pagina-baixo", 0, 100, 4, 2), 8);
    // Sem cursor: a primeira tecla põe no começo (ou no fim).
    eq(S.passo("baixo", -1, 10, 1, 5), 0);
    eq(S.passo("fim", -1, 10, 1, 5), 9);
    eq(S.passo("baixo", 0, 0, 1, 5), -1);
});

t("noRetangulo: os índices que o retângulo do mouse toca", () => {
    const grade = { n: 10, colunas: 4, larguraDaCelula: 100, alturaDaCelula: 80 };
    igual(S.noRetangulo(grade, { x: 10, y: 10, l: 150, a: 100 }), [0, 1, 4, 5]);
    // Arrastado para cima e para a esquerda dá no mesmo.
    igual(S.noRetangulo(grade, { x: 160, y: 110, l: -150, a: -100 }), [0, 1, 4, 5]);
    igual(S.noRetangulo(grade, { x: 250, y: 170, l: 500, a: 500 }), [], "a última linha só tem 2");
    igual(S.noRetangulo(grade, { x: 0, y: 170, l: 500, a: 500 }), [8, 9]);
    const lista = { n: 5, colunas: 1, larguraDaCelula: 600, alturaDaCelula: 20 };
    igual(S.noRetangulo(lista, { x: 50, y: 25, l: 10, a: 30 }), [1, 2]);
    igual(S.noRetangulo(lista, { x: 50, y: 500, l: 10, a: 30 }), []);
    igual(S.noRetangulo({ n: 0, colunas: 1, larguraDaCelula: 1, alturaDaCelula: 1 }, { x: 0, y: 0, l: 5, a: 5 }), []);
});

// --- o cursor: para onde cada tecla leva (os casos vistos em uso) ---------------
const sete = Array.from({ length: 7 }, (_, i) => ({ nome: "i" + i, caminho: "/p/i" + i }));
// Aperta as teclas em sequência a partir de um estado, e devolve o estado.
const aperta = (teclas, g, de, mods) => teclas.reduce((e, k) => S.mover(e, sete, k, mods || {}, g), de || S.nova());

t("destino: sem cursor a primeira tecla põe no primeiro item, e a segunda já anda", () => {
    const lista = { visao: "lista", porPagina: 5 };
    eq(S.destino("baixo", -1, 7, lista), 0);
    eq(S.destino("direita", -1, 7, { visao: "icones", colunas: 3 }), 0);
    eq(S.destino("fim", -1, 7, lista), 6);
    eq(S.destino("baixo", 0, 7, lista), 1);
    eq(S.destino("baixo", 0, 0, lista), -1, "pasta vazia");
});

t("lista: Down, Down para no item 1; Down três vezes, no 2", () => {
    const lista = { visao: "lista", porPagina: 5 };
    eq(aperta(["baixo", "baixo"], lista).cursor, 1);
    eq(aperta(["baixo", "baixo", "baixo"], lista).cursor, 2);
    eq(aperta(["baixo", "baixo", "cima"], lista).cursor, 0);
    eq(aperta(["baixo", "pagina-baixo"], lista).cursor, 5);
    eq(aperta(["fim", "inicio"], lista).cursor, 0);
    // Esquerda e direita, na lista, andam como cima e baixo: nenhuma seta morre.
    eq(aperta(["baixo", "direita"], lista).cursor, 1);
});

t("ícones: 7 itens em 3 colunas, as setas andam na grade; a última linha incompleta leva ao último", () => {
    const g = { visao: "icones", colunas: 3, porPagina: 2 };
    eq(aperta(["baixo", "baixo", "baixo"], g).cursor, 6);
    eq(aperta(["baixo", "baixo"], g).cursor, 3);
    eq(aperta(["baixo", "direita"], g).cursor, 1);
    eq(aperta(["baixo", "direita", "direita", "baixo"], g).cursor, 5);
    eq(aperta(["baixo", "direita", "direita", "baixo", "baixo"], g).cursor, 6, "sem item embaixo, vai ao último");
    eq(aperta(["fim", "cima", "cima"], g).cursor, 0);
    eq(aperta(["fim", "inicio"], g).cursor, 0);
});

t("ícones numa linha só (janela larga, 7 itens em 8 colunas): Down não morre, anda para o vizinho", () => {
    const g = { visao: "icones", colunas: 8, porPagina: 3 };
    eq(aperta(["baixo", "baixo", "baixo"], g).cursor, 2);
    eq(aperta(["baixo", "direita"], g).cursor, 1);
    eq(aperta(["fim", "cima"], g).cursor, 5);
    eq(S.destino("baixo", 6, 7, g), 6, "no último, fica");
    // Com mais de uma linha, baixo volta a ser baixo.
    eq(S.destino("baixo", 0, 9, g), 8);
});

t("compacta: enche de cima para baixo, então baixo é o vizinho e direita é a coluna seguinte", () => {
    const g = { visao: "compacta", linhas: 3, porPagina: 2 };
    eq(aperta(["baixo", "baixo"], g).cursor, 1);
    eq(aperta(["baixo", "direita"], g).cursor, 3);
    eq(aperta(["baixo", "direita", "direita", "esquerda"], g).cursor, 3);
});

t("mover: sozinha seleciona só o destino; com Shift estende da âncora; com Ctrl só leva o cursor", () => {
    const lista = { visao: "lista", porPagina: 5 };
    const dois = aperta(["baixo", "baixo"], lista);
    igual(S.caminhos(dois, sete), ["/p/i1"]);
    const estendido = aperta(["baixo", "baixo"], lista, dois, { shift: true });
    igual(S.caminhos(estendido, sete), ["/p/i1", "/p/i2", "/p/i3"]);
    eq(estendido.cursor, 3);
    const solto = aperta(["baixo"], lista, dois, { ctrl: true });
    igual(S.caminhos(solto, sete), ["/p/i1"]);
    eq(solto.cursor, 2);
    // Pasta vazia: o estado é o mesmo objeto (nada a redesenhar).
    const vazio = S.nova();
    eq(S.mover(vazio, [], "baixo", {}, lista), vazio);
});

t("o cursor sobrevive à lista refeita: a tecla seguinte anda a partir do mesmo item", () => {
    const lista = { visao: "lista", porPagina: 5 };
    const antes = aperta(["baixo", "baixo", "baixo"], lista);
    // Chegou um lote novo e a ordem mudou: o item do cursor foi para outra posição.
    const outros = [{ nome: "a", caminho: "/p/a" }].concat(sete);
    const depois = S.reancorar(antes, outros);
    eq(outros[depois.cursor].caminho, "/p/i2");
    eq(outros[S.mover(depois, outros, "baixo", {}, lista).cursor].caminho, "/p/i3");
});
