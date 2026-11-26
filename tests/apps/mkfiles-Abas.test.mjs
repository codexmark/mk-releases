// apps/mkfiles/qml/Abas.js: as abas, o histórico de navegação de cada uma, os
// dois painéis e o estado que se grava ao fechar.
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const B = load(REPO + "/apps/mkfiles/qml/Abas.js");
const caminhos = (p) => p.abas.map((a) => a.caminho);

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["eLixeira", "nova", "ir", "podeVoltar", "podeAvancar", "podeSubir", "voltar", "avancar", "subir", "titulo", "painel",
        "abaAtual", "abrir", "duplicar", "fechar", "escolher", "trocar", "mover", "comAba", "paraEstado", "caminhoValido", "doEstado",
        "inicial", "pedidosDe", "eCompactado"];
    const funcoes = Object.keys(B).filter((k) => typeof B[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("uma aba: ir, voltar, avançar e subir, com o histórico", () => {
    let a = B.nova("/home/ana");
    eq(B.podeVoltar(a), false);
    a = B.ir(a, "/home/ana/Doc");
    a = B.ir(a, "/home/ana/Doc/x");
    eq(B.ir(a, "/home/ana/Doc/x"), a, "ir para onde já está não muda nada");
    igual(a.tras, ["/home/ana", "/home/ana/Doc"]);
    a = B.voltar(a);
    eq(a.caminho, "/home/ana/Doc");
    igual(a.frente, ["/home/ana/Doc/x"]);
    eq(B.podeAvancar(a), true);
    a = B.avancar(a);
    eq(a.caminho, "/home/ana/Doc/x");
    eq(B.podeAvancar(a), false);
    eq(B.avancar(a), a);
    a = B.voltar(B.voltar(a));
    eq(a.caminho, "/home/ana");
    eq(B.voltar(a), a);
    // Ir para outro lugar no meio do histórico apaga o "avançar".
    a = B.ir(a, "/tmp");
    igual(a.frente, []);
    a = B.subir(a);
    eq(a.caminho, "/");
    eq(B.podeSubir(a), false);
    eq(B.subir(a), a);
    igual(a.tras, ["/home/ana", "/tmp"]);
});

t("a lixeira é um lugar: não sobe, e tem nome", () => {
    const l = B.nova(B.LIXEIRA);
    eq(B.eLixeira(l.caminho), true);
    eq(B.eLixeira("/home"), false);
    eq(B.podeSubir(l), false);
    eq(B.titulo(l, "/home/ana"), "Lixeira");
    eq(B.titulo(B.nova("/home/ana"), "/home/ana"), "Pasta pessoal");
    eq(B.titulo(B.nova("/home/ana/Meus docs"), "/home/ana"), "Meus docs");
    eq(B.titulo(B.nova("/"), "/home/ana"), "/");
});

t("o painel: abrir, duplicar, fechar, escolher, trocar e mover abas", () => {
    let p = B.painel(["/a", "/b", "/c"], 1);
    eq(B.abaAtual(p).caminho, "/b");
    p = B.abrir(p, "/n");
    igual(caminhos(p), ["/a", "/b", "/n", "/c"]);
    eq(p.atual, 2);
    p = B.duplicar(p, 0);
    igual(caminhos(p), ["/a", "/a", "/b", "/n", "/c"]);
    eq(p.atual, 1);
    eq(new Set(p.abas.map((a) => a.id)).size, 5, "cada aba tem o seu id");
    p = B.fechar(p, 1);
    igual(caminhos(p), ["/a", "/b", "/n", "/c"]);
    eq(p.atual, 1);
    p = B.fechar(p, 0);
    eq(B.abaAtual(p).caminho, "/b");
    p = B.fechar(B.escolher(p, 2), 2);
    eq(B.abaAtual(p).caminho, "/n", "fechar a última vai para a anterior");
    eq(B.escolher(p, 9), p);
    p = B.trocar(B.painel(["/a", "/b", "/c"], 2), 1);
    eq(p.atual, 0);
    eq(B.trocar(p, -1).atual, 2);
    p = B.mover(B.painel(["/a", "/b", "/c"], 0), 0, 2);
    igual(caminhos(p), ["/b", "/c", "/a"]);
    eq(p.atual, 2, "a aba à vista continua à vista");
    eq(B.mover(p, 1, 1), p);
    eq(B.mover(p, 1, 9), p);
    // A última aba não fecha, e há um teto.
    const um = B.painel(["/a"], 0);
    eq(B.fechar(um, 0), um);
    let cheio = B.painel(Array(40).fill("/x"), 0);
    eq(cheio.abas.length, 30);
    eq(B.abrir(cheio, "/y"), cheio);
    eq(B.duplicar(cheio, 0), cheio);
    const q = B.comAba(B.painel(["/a", "/b"], 1), B.nova("/z"));
    igual(caminhos(q), ["/a", "/z"]);
});

t("paraEstado e doEstado: o que se grava ao fechar volta igual", () => {
    const paineis = [B.painel(["/a", B.LIXEIRA], 1), B.painel(["/c"], 0)];
    const e = B.paraEstado(paineis, 1, true, 251.6);
    igual(e, { paineis: [{ abas: ["/a", "lixeira:"], atual: 1 }, { abas: ["/c"], atual: 0 }], ativo: 1, duplo: true, lateral_px: 252 });
    eq(B.paraEstado(paineis, 0, false, 9999).lateral_px, 480);
    eq(B.paraEstado(paineis, 5, false, 0).lateral_px, 200);
    const de = B.doEstado(e, "/home/ana");
    igual(de.map(caminhos), [["/a", "lixeira:"], ["/c"]]);
    eq(de[0].atual, 1);
    // Estado vazio ou torto: a pasta pessoal nos dois.
    igual(B.doEstado({}, "/home/ana").map(caminhos), [["/home/ana"], ["/home/ana"]]);
    igual(B.doEstado({ paineis: [{ abas: ["relativo", 3, "/ok"], atual: 7 }, "x"] }, "/home/ana").map(caminhos), [["/ok"], ["/home/ana"]]);
    igual(B.doEstado(null, "/home/ana").length, 2);
    eq(B.caminhoValido("/a"), true);
    eq(B.caminhoValido("a"), false);
    eq(B.caminhoValido(null), false);
});

t("inicial: `mkfiles pasta arquivo` abre as pastas em abas e seleciona o arquivo", () => {
    const estado = { paineis: [{ abas: ["/velho"], atual: 0 }, { abas: ["/outro"], atual: 0 }] };
    let r = B.inicial([], estado, "/home/ana");
    igual(r.paineis.map(caminhos), [["/velho"], ["/outro"]]);
    igual(r.selecionar, {});
    r = B.inicial([{ caminho: "/a/doc", pasta: true }, { caminho: "/a/doc/x.txt", pasta: false }, { caminho: "/b/y.png", pasta: false },
        { caminho: "/b/z.png", pasta: false }], estado, "/home/ana");
    igual(r.paineis.map(caminhos), [["/a/doc", "/b"], ["/outro"]]);
    igual(r.selecionar, { "/a/doc": ["/a/doc/x.txt"], "/b": ["/b/y.png", "/b/z.png"] });
});

t("pedidosDe: as linhas de MKFILES_ABRIR, só as absolutas", () => {
    igual(B.pedidosDe("/a b\nrelativo\n\n/c\n"), ["/a b", "/c"]);
    igual(B.pedidosDe(""), []);
    igual(B.pedidosDe(undefined), []);
});

t("uma aba dentro de um compactado: sobe até sair dele, e volta na pasta do arquivo", () => {
    eq(B.eCompactado("/x/a.zip//pasta"), true);
    eq(B.eCompactado("/x/a.zip"), false);
    eq(B.eCompactado(B.LIXEIRA), false);
    let a = B.ir(B.nova("/x"), "/x/a.zip//");
    a = B.ir(a, "/x/a.zip//pasta");
    eq(B.titulo(a, "/home/ana"), "pasta");
    eq(B.podeSubir(a), true);
    a = B.subir(a);
    eq(a.caminho, "/x/a.zip//");
    eq(B.titulo(a, "/home/ana"), "a.zip");
    eq(B.subir(a).caminho, "/x");
    eq(B.voltar(a).caminho, "/x/a.zip//pasta");
    // O estado gravado nunca leva o lugar de dentro: vai a pasta do arquivo.
    const p = B.painel(["/x/a.zip//pasta", "/y"], 0);
    igual(B.paraEstado([p, B.painel(["/z"], 0)], 0, false, 200).paineis[0].abas, ["/x", "/y"]);
    eq(B.caminhoValido("/x/a.zip//pasta"), false);
    igual(B.doEstado({ paineis: [{ abas: ["/x/a.zip//", "/ok"], atual: 0 }] }, "/home/ana")[0].abas.map((x) => x.caminho), ["/ok"]);
});
