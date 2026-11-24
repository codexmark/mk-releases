// apps/mkpulso/qml/Pulso.js: a lógica da janela do MkPulso que não depende do
// QML: escrever tamanhos, taxas e tempos, escolher o valor da janela de tempo
// (agora, 1 min, 3 min), procurar, ordenar e separar os grupos em seções.
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const P = load(REPO + "/apps/mkpulso/qml/Pulso.js");

const grupo = (id, tipo, nome, extra) => Object.assign({
    id, tipo, nome, icone: "", unidade: "", escopo: "", janelas: [], n: 1, pid: 1,
    cpu_pct: 0, mem_kb: 0, disco_bps: 0, gpu_pct: null, energia: "nenhuma",
    media: { cpu_pct_1m: 0, cpu_pct_3m: 0, mem_kb_1m: 0, mem_kb_3m: 0, disco_bps_1m: 0, disco_bps_3m: 0 },
    processos: []
}, extra || {});

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["tamanho", "taxa", "pct", "duracao", "desde", "fold", "valor", "casa", "filtrar", "ordenar",
        "secoes", "nivelEnergia", "calor", "empurra", "num", "valido", "linhas", "teto", "chave"];
    const funcoes = Object.keys(P).filter((k) => typeof P[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("valido: só número de verdade, finito e não negativo", () => {
    for (const v of [0, 1, 0.5, 1e12])
        eq(P.valido(v), true, String(v));
    for (const v of [null, undefined, NaN, Infinity, -1, "3", {}, []])
        eq(P.valido(v), false, String(v));
});

t("num: uma casa só quando o número é pequeno, com vírgula", () => {
    eq(P.num(5.94), "5,9");
    eq(P.num(0.04), "0,0");
    eq(P.num(12.5), "13");
    eq(P.num(9.96), "10");
    eq(P.num(196.4), "196");
});

t("tamanho: de kB para a unidade que cabe, como a pessoa lê", () => {
    eq(P.tamanho(0), "0 KB");
    eq(P.tamanho(336), "336 KB");
    eq(P.tamanho(1023), "1023 KB");
    eq(P.tamanho(1024), "1,0 MB");
    eq(P.tamanho(46 * 1024), "46 MB");
    eq(P.tamanho(839 * 1024), "839 MB");
    eq(P.tamanho(5.4 * 1024 * 1024), "5,4 GB");
    eq(P.tamanho(2 * 1024 * 1024 * 1024), "2,0 TB");
});

t("taxa: bytes por segundo", () => {
    eq(P.taxa(0), "0 B/s");
    eq(P.taxa(512), "512 B/s");
    eq(P.taxa(8 * 1024), "8,0 KB/s");
    eq(P.taxa(1.4 * 1024 * 1024), "1,4 MB/s");
});

t("pct: porcentagem; acima de 100 (vários núcleos) sem casa", () => {
    eq(P.pct(5.94), "5,9%");
    eq(P.pct(0), "0,0%");
    eq(P.pct(96), "96%");
    eq(P.pct(196.2), "196%");
});

t("o que o coletor não soube ler (null) vira um traço, nunca zero", () => {
    for (const f of [P.tamanho, P.taxa, P.pct, P.duracao])
        for (const v of [null, undefined, NaN, "x", -1])
            eq(f(v), "—", f.name + " " + String(v));
});

t("duracao: segundos em palavras curtas", () => {
    eq(P.duracao(0), "agora");
    eq(P.duracao(59), "59 s");
    eq(P.duracao(60), "1 min");
    eq(P.duracao(3599), "59 min");
    eq(P.duracao(3600), "1 h");
    eq(P.duracao(3 * 3600 + 12 * 60), "3 h 12 min");
    eq(P.duracao(86400), "1 d");
    eq(P.duracao(2 * 86400 + 4 * 3600 + 5), "2 d 4 h");
});

t("desde: há quanto tempo, a partir de um instante e do agora", () => {
    eq(P.desde(1000, 1000 + 125), "2 min");
    eq(P.desde(1000, 900), "agora", "relógio para trás não vira número negativo");
    eq(P.desde(null, 1000), "—");
});

t("fold: minúsculas e sem acento, para a busca", () => {
    eq(P.fold("Área de Transferência"), "area de transferencia");
    eq(P.fold(null), "");
    eq(P.fold(42), "42");
});

t("valor: o da janela de tempo pedida (agora, 1 min, 3 min)", () => {
    const g = grupo("app:x", "app", "X", { cpu_pct: 10, mem_kb: 100, disco_bps: 5,
        media: { cpu_pct_1m: 6, cpu_pct_3m: 3, mem_kb_1m: 90, mem_kb_3m: 80, disco_bps_1m: 4, disco_bps_3m: 2 } });
    eq(P.valor(g, "cpu_pct", "agora"), 10);
    eq(P.valor(g, "cpu_pct", "1m"), 6);
    eq(P.valor(g, "cpu_pct", "3m"), 3);
    eq(P.valor(g, "mem_kb", "3m"), 80);
    eq(P.valor(g, "disco_bps", "1m"), 4);
});

t("valor: coluna sem média (GPU, PID) vale a de agora em qualquer janela; sem média guardada também", () => {
    const g = grupo("app:x", "app", "X", { gpu_pct: 7, pid: 99, cpu_pct: 10, media: null });
    eq(P.valor(g, "gpu_pct", "3m"), 7);
    eq(P.valor(g, "pid", "1m"), 99);
    eq(P.valor(g, "cpu_pct", "1m"), 10);
    eq(P.valor(null, "cpu_pct", "agora"), null);
});

t("casa: procura no nome, no id, na unidade, nas janelas, no pid e nos processos", () => {
    const g = grupo("app:firefox", "app", "Firefox", { pid: 1234, unidade: "app-firefox-1.scope",
        janelas: [{ titulo: "Notícias do dia — Firefox", workspace: "2", endereco: "0x1" }],
        processos: [{ pid: 1240, nome: "Web Content", cmd: "/usr/lib/firefox/firefox -contentproc" }] });
    for (const q of ["fire", "FIREFOX", "noticias", "1234", "1240", "contentproc", "web content", "scope"])
        eq(P.casa(g, q), true, q);
    eq(P.casa(g, "chrome"), false);
    eq(P.casa(g, ""), true, "busca vazia mostra tudo");
    eq(P.casa(g, "fire noticias"), true, "todas as palavras têm de casar");
    eq(P.casa(g, "fire chrome"), false);
});

t("filtrar: só os grupos que casam, sem mexer na lista de entrada", () => {
    const lista = [grupo("app:a", "app", "Alfa"), grupo("app:b", "app", "Beta"), grupo("servico:x.service", "servico", "x")];
    igual(P.filtrar(lista, "beta").map((g) => g.id), ["app:b"]);
    igual(P.filtrar(lista, "").length, 3);
    eq(lista.length, 3);
    igual(P.filtrar(null, "a"), []);
});

t("ordenar: pela coluna e pelo sentido; null vai para o fim nos dois sentidos; empate pelo nome", () => {
    const lista = [
        grupo("app:a", "app", "Alfa", { cpu_pct: 5 }), grupo("app:b", "app", "Beta", { cpu_pct: 50 }),
        grupo("app:c", "app", "Gama", { cpu_pct: 5, gpu_pct: 3 }), grupo("app:d", "app", "Delta", { cpu_pct: 0 })
    ];
    igual(P.ordenar(lista, "cpu_pct", true, "agora").map((g) => g.nome), ["Beta", "Alfa", "Gama", "Delta"]);
    igual(P.ordenar(lista, "cpu_pct", false, "agora").map((g) => g.nome), ["Delta", "Alfa", "Gama", "Beta"]);
    igual(P.ordenar(lista, "gpu_pct", true, "agora").map((g) => g.nome)[0], "Gama");
    igual(P.ordenar(lista, "gpu_pct", false, "agora").map((g) => g.nome)[0], "Gama");
    igual(P.ordenar(lista, "nome", false, "agora").map((g) => g.nome), ["Alfa", "Beta", "Delta", "Gama"]);
    eq(lista[0].nome, "Alfa", "a lista de entrada não muda");
});

t("ordenar: energia segue a escala (nenhuma … muito alta), não o alfabeto", () => {
    const lista = ["baixa", "muito alta", "nenhuma", "moderada", "alta", "muito baixa"]
        .map((e, i) => grupo("app:" + i, "app", "G" + i, { energia: e }));
    igual(P.ordenar(lista, "energia", true, "agora").map((g) => g.energia),
        ["muito alta", "alta", "moderada", "baixa", "muito baixa", "nenhuma"]);
});

t("nivelEnergia: de 0 (nenhuma) a 5 (muito alta); texto desconhecido é 0", () => {
    igual(["nenhuma", "muito baixa", "baixa", "moderada", "alta", "muito alta"].map(P.nivelEnergia), [0, 1, 2, 3, 4, 5]);
    eq(P.nivelEnergia("banana"), 0);
    eq(P.nivelEnergia(null), 0);
});

t("secoes: apps, serviços e sistema, nessa ordem, com a contagem; seção vazia não aparece", () => {
    const lista = [grupo("servico:a.service", "servico", "a"), grupo("app:x", "app", "X"),
        grupo("sistema:kernel", "sistema", "Kernel"), grupo("app:y", "app", "Y")];
    const s = P.secoes(lista);
    igual(s.map((x) => [x.tipo, x.titulo, x.grupos.length]), [["app", "Apps", 2], ["servico", "Serviços", 1], ["sistema", "Sistema", 1]]);
    igual(P.secoes([grupo("app:x", "app", "X")]).map((x) => x.tipo), ["app"]);
    igual(P.secoes([]), []);
    igual(P.secoes(null), []);
});

t("calor: quanto pintar a célula, de 0 a 1, e nada para zero ou null", () => {
    eq(P.calor(0, 100), 0);
    eq(P.calor(null, 100), 0);
    eq(P.calor(50, 100), 0.5);
    eq(P.calor(300, 100), 1);
    eq(P.calor(5, 0), 0);
});

t("empurra: a série do gráfico anda, com no máximo N pontos, sem mudar a de entrada", () => {
    const a = [1, 2, 3];
    igual(P.empurra(a, 4, 3), [2, 3, 4]);
    igual(a, [1, 2, 3]);
    igual(P.empurra([], 1, 3), [1]);
    igual(P.empurra(null, 1, 3), [1]);
    igual(P.empurra([1], null, 3), [1, 0], "um valor que não veio vira 0 no gráfico");
});

t("linhas: a tabela achatada: seção, grupos dela e, nos grupos abertos, os processos", () => {
    const a = grupo("app:a", "app", "Alfa", { processos: [{ pid: 1, nome: "a" }, { pid: 2, nome: "a-filho" }] });
    const b = grupo("app:b", "app", "Beta", { processos: [{ pid: 3, nome: "b" }] });
    const s = grupo("servico:s.service", "servico", "s", { processos: [{ pid: 4, nome: "s" }] });
    const linhas = P.linhas(P.secoes([a, b, s]), { [P.chave(a)]: true }, {});
    igual(linhas.map((l) => l.tipo + ":" + (l.tipo === "secao" ? l.secao.tipo : l.tipo === "grupo" ? l.grupo.id : l.processo.pid)),
        ["secao:app", "grupo:app:a", "processo:1", "processo:2", "grupo:app:b", "secao:servico", "grupo:servico:s.service"]);
    eq(linhas[2].grupo.id, "app:a", "a linha do processo sabe de que grupo é");
});

t("linhas: uma seção recolhida mostra só o título; grupo de um processo só não abre", () => {
    const a = grupo("app:a", "app", "Alfa", { processos: [{ pid: 1, nome: "a" }] });
    const s = grupo("servico:s.service", "servico", "s", { processos: [{ pid: 4, nome: "s" }, { pid: 5, nome: "t" }] });
    const linhas = P.linhas(P.secoes([a, s]), { [P.chave(a)]: true, [P.chave(s)]: true }, { servico: true });
    igual(linhas.map((l) => l.tipo), ["secao", "grupo", "secao"]);
    igual(P.linhas(null, {}, {}), []);
    igual(P.linhas(P.secoes([a]), null, null).map((l) => l.tipo), ["secao", "grupo"]);
});

t("teto: o maior valor de uma coluna entre os grupos, para a escala do calor", () => {
    const lista = [grupo("app:a", "app", "A", { cpu_pct: 5 }), grupo("app:b", "app", "B", { cpu_pct: 50 }), grupo("app:c", "app", "C", { cpu_pct: null })];
    eq(P.teto(lista, "cpu_pct", "agora"), 50);
    eq(P.teto([], "cpu_pct", "agora"), 0);
    eq(P.teto(null, "cpu_pct", "agora"), 0);
});

t("chave: o id com o escopo, porque a mesma unidade pode existir no sistema e na sessão", () => {
    const sis = grupo("servico:dbus-broker.service", "servico", "dbus-broker", { escopo: "sistema", processos: [{ pid: 1 }, { pid: 2 }] });
    const usu = grupo("servico:dbus-broker.service", "servico", "dbus-broker", { escopo: "usuario", processos: [{ pid: 3 }, { pid: 4 }] });
    eq(P.chave(sis) === P.chave(usu), false);
    eq(P.chave(grupo("app:x", "app", "X")), "app:x|");
    // Abrir um não abre o outro.
    const linhas = P.linhas(P.secoes([sis, usu]), { [P.chave(usu)]: true }, {});
    igual(linhas.filter((l) => l.tipo === "processo").map((l) => l.processo.pid), [3, 4]);
});
