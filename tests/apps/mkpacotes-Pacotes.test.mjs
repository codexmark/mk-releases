// apps/mkpacotes/qml/Pacotes.js: a lógica da janela do MkPacotes que não
// depende do QML: escrever tamanhos e datas, filtrar, procurar e ordenar os
// instalados, marcar vários, montar os argumentos de uma ação, resumir o
// plano e entender o que o `agir` transmite.
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const P = load(REPO + "/apps/mkpacotes/qml/Pacotes.js");

const pacote = (nome, extra) => Object.assign({
    nome, versao: "1.0-1", descricao: "", tamanho_bytes: 1000, instalado_em_s: 1000, motivo: "pedido",
    origem: "extra", orfao: false, atualizacao: null, protegido: false
}, extra || {});

const lista = [
    pacote("bash", { descricao: "The GNU Bourne Again shell", tamanho_bytes: 10000000, instalado_em_s: 300, origem: "core", atualizacao: "5.3.21-1" }),
    pacote("do-aur", { descricao: "Veio do AUR", tamanho_bytes: 3000, instalado_em_s: 600, origem: "aur" }),
    pacote("glibc", { descricao: "GNU C Library", tamanho_bytes: 50000000, instalado_em_s: 100, motivo: "dependencia", origem: "core", protegido: true }),
    pacote("libvelha", { descricao: "Sobrou", tamanho_bytes: null, instalado_em_s: null, motivo: "dependencia", origem: null, orfao: true })
];
const nomes = (l) => l.map((p) => p.nome);

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["valido", "num", "tamanho", "data", "quando", "fold", "casa", "filtrar", "contagens", "ordenar", "origem",
        "motivo", "nomeValido", "alterna", "marcados", "argsDeAcao", "tituloDoPlano", "resumoDoPlano", "podeConfirmar",
        "interpreta", "empurra", "noFiltro", "versoes", "linhaDeComando", "plural"];
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
    eq(P.num(12.5), "13");
    eq(P.num(9.96), "10");
});

t("tamanho: de bytes para a unidade que cabe", () => {
    eq(P.tamanho(0), "0 B");
    eq(P.tamanho(512), "512 B");
    eq(P.tamanho(1024), "1,0 KB");
    eq(P.tamanho(184319), "180 KB");
    eq(P.tamanho(10055840), "9,6 MB");
    eq(P.tamanho(19884238411), "19 GB");
    eq(P.tamanho(3 * 1024 * 1024 * 1024 * 1024), "3,0 TB");
});

t("o que o coletor não soube (null) vira um traço, nunca zero", () => {
    for (const f of [P.tamanho, P.data])
        for (const v of [null, undefined, NaN, "x", -1])
            eq(f(v), "—", f.name + " " + String(v));
    eq(P.quando(null, 1000), "—");
    eq(P.quando(1000, null), "—");
});

t("data: dia, mês e ano, do jeito daqui", () => {
    const meioDia = new Date(2026, 10, 20, 12, 0, 0).getTime() / 1000;
    eq(P.data(meioDia), "20/11/2026");
    eq(P.data(new Date(2026, 0, 5, 12).getTime() / 1000), "05/01/2026");
});

t("quando: hoje, ontem, há N dias e depois a data", () => {
    const agora = new Date(2026, 10, 20, 15, 0, 0).getTime() / 1000;
    const em = (dia, hora) => new Date(2026, 10, dia, hora, 0, 0).getTime() / 1000;
    eq(P.quando(em(20, 9), agora), "hoje");
    eq(P.quando(em(19, 23), agora), "ontem", "a virada é à meia-noite, não 24 h atrás");
    eq(P.quando(em(17, 10), agora), "há 3 dias");
    eq(P.quando(em(1, 10), agora), "há 19 dias");
    eq(P.quando(new Date(2026, 8, 2, 10).getTime() / 1000, agora), "02/09/2026");
    eq(P.quando(em(21, 10), agora), "hoje", "relógio para trás não vira número negativo");
});

t("fold: minúsculas e sem acento, para a busca", () => {
    eq(P.fold("Área de Transferência"), "area de transferencia");
    eq(P.fold(null), "");
    eq(P.fold(42), "42");
});

t("casa: todas as palavras no nome ou na descrição", () => {
    eq(P.casa(lista[0], "bash"), true);
    eq(P.casa(lista[0], "BOURNE shell"), true);
    eq(P.casa(lista[0], "bash zsh"), false);
    eq(P.casa(lista[0], ""), true, "busca vazia mostra tudo");
    eq(P.casa(lista[0], "   "), true);
    eq(P.casa({ nome: "x", descricao: null }, "x"), true);
});

t("filtrar: todos, pedidos por mim, do AUR, órfãos, com atualização, e a busca por cima", () => {
    igual(nomes(P.filtrar(lista, "todos", "")), ["bash", "do-aur", "glibc", "libvelha"]);
    igual(nomes(P.filtrar(lista, "pedidos", "")), ["bash", "do-aur"]);
    igual(nomes(P.filtrar(lista, "aur", "")), ["do-aur"]);
    igual(nomes(P.filtrar(lista, "orfaos", "")), ["libvelha"]);
    igual(nomes(P.filtrar(lista, "atualizacoes", "")), ["bash"]);
    igual(nomes(P.filtrar(lista, "todos", "gnu")), ["bash", "glibc"]);
    igual(nomes(P.filtrar(lista, "pedidos", "gnu")), ["bash"]);
    igual(nomes(P.filtrar(lista, "filtro-que-nao-existe", "")), ["bash", "do-aur", "glibc", "libvelha"]);
    igual(P.filtrar(null, "todos", ""), []);
    eq(lista.length, 4, "a lista de entrada não muda");
});

t("contagens: quantos há em cada filtro", () => {
    igual(P.contagens(lista), { todos: 4, pedidos: 2, aur: 1, orfaos: 1, atualizacoes: 1 });
    igual(P.contagens(null), { todos: 0, pedidos: 0, aur: 0, orfaos: 0, atualizacoes: 0 });
});

t("ordenar: por nome, tamanho ou data; null vai para o fim nos dois sentidos; empate pelo nome", () => {
    igual(nomes(P.ordenar(lista, "nome", false)), ["bash", "do-aur", "glibc", "libvelha"]);
    igual(nomes(P.ordenar(lista, "nome", true)), ["libvelha", "glibc", "do-aur", "bash"]);
    igual(nomes(P.ordenar(lista, "tamanho_bytes", true)), ["glibc", "bash", "do-aur", "libvelha"]);
    igual(nomes(P.ordenar(lista, "tamanho_bytes", false)), ["do-aur", "bash", "glibc", "libvelha"]);
    igual(nomes(P.ordenar(lista, "instalado_em_s", true)), ["do-aur", "bash", "glibc", "libvelha"]);
    const empate = [pacote("b", { tamanho_bytes: 5 }), pacote("a", { tamanho_bytes: 5 })];
    igual(nomes(P.ordenar(empate, "tamanho_bytes", true)), ["a", "b"]);
    eq(lista[0].nome, "bash", "a lista de entrada não muda");
    igual(P.ordenar(null, "nome", false), []);
});

t("origem e motivo: como a pessoa lê", () => {
    eq(P.origem("aur"), "AUR");
    eq(P.origem("extra"), "extra");
    eq(P.origem(null), "—");
    eq(P.origem(undefined), "—");
    eq(P.motivo("pedido"), "pedido por você");
    eq(P.motivo("dependencia"), "dependência");
    eq(P.motivo(null), "—");
});

t("nomeValido: a mesma regra do coletor", () => {
    for (const n of ["bash", "gtk4", "libc++", "python-pip", "a@b", "9base", "kde.x"])
        eq(P.nomeValido(n), true, n);
    for (const n of ["", "-Syu", "--noconfirm", ".x", "a b", "a;b", "Bash", "ação", "a/b", null, undefined, 3, "a\nb"])
        eq(P.nomeValido(n), false, String(n));
});

t("alterna e marcados: marcar vários sem mudar o mapa de entrada", () => {
    const a = P.alterna({}, "htop");
    igual(a, { htop: true });
    const b = P.alterna(a, "fd");
    igual(P.marcados(b), ["fd", "htop"], "em ordem de nome");
    igual(a, { htop: true });
    igual(P.marcados(P.alterna(b, "htop")), ["fd"]);
    igual(P.marcados(null), []);
    igual(P.alterna(null, "x"), { x: true });
});

t("argsDeAcao: os nomes sempre depois de --, e só os que são nome de pacote", () => {
    igual(P.argsDeAcao("remover", ["htop", "fd"], false), ["remover", "--", "htop", "fd"]);
    igual(P.argsDeAcao("remover", ["htop"], true), ["remover", "--mesmo-assim", "--", "htop"]);
    igual(P.argsDeAcao("instalar", ["htop", "-Syu", "a b", "fd"], false), ["instalar", "--", "htop", "fd"]);
    igual(P.argsDeAcao("orfaos", [], false), ["orfaos"]);
    igual(P.argsDeAcao("cache", null, false), ["cache"]);
    igual(P.argsDeAcao("atualizar", ["htop"], false), ["atualizar"], "ação sem pacote não leva pacote");
    igual(P.argsDeAcao("orfaos", [], true), ["orfaos", "--mesmo-assim"]);
});

t("plural: 1 pacote, 2 pacotes", () => {
    eq(P.plural(1, "pacote", "pacotes"), "1 pacote");
    eq(P.plural(2, "pacote", "pacotes"), "2 pacotes");
    eq(P.plural(0, "pacote", "pacotes"), "0 pacotes");
});

const plano = (extra) => Object.assign({
    acao: "remover", pedidos: ["htop"], entra: [], sai: [], baixar_bytes: null, liberar_bytes: null, comandos: [],
    avisos: [], travado: false, nada_a_fazer: false, cache: null
}, extra || {});

t("tituloDoPlano: a pergunta da folha", () => {
    eq(P.tituloDoPlano(plano()), "Remover htop?");
    eq(P.tituloDoPlano(plano({ pedidos: ["a", "b", "c"] })), "Remover 3 pacotes?");
    eq(P.tituloDoPlano(plano({ acao: "instalar", pedidos: ["fd"] })), "Instalar fd?");
    eq(P.tituloDoPlano(plano({ acao: "instalar", pedidos: ["fd", "rg"] })), "Instalar 2 pacotes?");
    eq(P.tituloDoPlano(plano({ acao: "orfaos", pedidos: ["x", "y"] })), "Remover os 2 órfãos?");
    eq(P.tituloDoPlano(plano({ acao: "orfaos", pedidos: ["x"] })), "Remover o órfão x?");
    eq(P.tituloDoPlano(plano({ acao: "cache", pedidos: [] })), "Limpar o cache de pacotes?");
    eq(P.tituloDoPlano(plano({ acao: "atualizar", pedidos: [] })), "Atualizar tudo?");
    eq(P.tituloDoPlano(null), "");
});

t("resumoDoPlano: o que entra, o que sai e quanto pesa", () => {
    eq(P.resumoDoPlano(plano({ sai: [{ nome: "htop", pedido: true }, { nome: "libnl", pedido: false }], liberar_bytes: 1900000 })),
        "Saem 2 pacotes: o que você pediu e 1 dependência que ficaria sem uso. Libera 1,8 MB.");
    eq(P.resumoDoPlano(plano({ sai: [{ nome: "htop", pedido: true }], liberar_bytes: 400000 })), "Sai 1 pacote. Libera 391 KB.");
    eq(P.resumoDoPlano(plano({ acao: "instalar", entra: [{ nome: "fd", pedido: true }, { nome: "x", pedido: false }, { nome: "y", pedido: false }], baixar_bytes: 3000000 })),
        "Entram 3 pacotes: o que você pediu e 2 dependências. Baixa 2,9 MB.");
    eq(P.resumoDoPlano(plano({ acao: "instalar", entra: [{ nome: "fd", pedido: true }], baixar_bytes: null })), "Entra 1 pacote.");
    eq(P.resumoDoPlano(plano({ acao: "cache", liberar_bytes: 13463798667, cache: { arquivos: 4174, bytes: 19529508058, sobram: 2172 } })),
        "Saem do cache 2002 arquivos de pacotes que não estão mais instalados. Libera 13 GB; ficam 2172.");
    eq(P.resumoDoPlano(plano({ acao: "atualizar" })), "Abre o mk-update num terminal: ele atualiza o Mk, o Arch e o AUR.");
    eq(P.resumoDoPlano(plano({ nada_a_fazer: true })), "Nada a fazer.");
    eq(P.resumoDoPlano(null), "");
});

t("podeConfirmar: só com plano, com algo a fazer e sem o pacman travado", () => {
    eq(P.podeConfirmar(plano()), true);
    eq(P.podeConfirmar(plano({ nada_a_fazer: true })), false);
    eq(P.podeConfirmar(plano({ travado: true })), false);
    eq(P.podeConfirmar(plano({ acao: "atualizar", travado: true })), true, "o mk-update lida com a trava");
    eq(P.podeConfirmar(null), false);
});

t("interpreta: cada linha que o agir transmite", () => {
    igual(P.interpreta('{"linha":"removendo htop..."}'), { tipo: "linha", texto: "removendo htop..." });
    igual(P.interpreta('{"ok":true,"acao":"remover","pacotes":["htop"]}'), { tipo: "ok", acao: "remover", pacotes: ["htop"] });
    igual(P.interpreta('{"erro":"a senha não foi dada: nada foi mudado"}'), { tipo: "erro", texto: "a senha não foi dada: nada foi mudado" });
    igual(P.interpreta("isto não é json"), { tipo: "nada" });
    igual(P.interpreta(""), { tipo: "nada" });
    igual(P.interpreta("null"), { tipo: "nada" });
    igual(P.interpreta('{"outra":1}'), { tipo: "nada" });
});

t("empurra: o registro anda, com no máximo N linhas, sem mudar o de entrada", () => {
    const a = ["1", "2", "3"];
    igual(P.empurra(a, "4", 3), ["2", "3", "4"]);
    igual(a, ["1", "2", "3"]);
    igual(P.empurra(null, "1", 3), ["1"]);
});

t("noFiltro: o pacote está no filtro?", () => {
    eq(P.noFiltro(lista[0], "pedidos"), true);
    eq(P.noFiltro(lista[2], "pedidos"), false);
    eq(P.noFiltro(lista[1], "aur"), true);
    eq(P.noFiltro(lista[3], "aur"), false, "origem desconhecida (null) não é AUR");
    eq(P.noFiltro(lista[3], "orfaos"), true);
    eq(P.noFiltro(lista[0], "atualizacoes"), true);
    eq(P.noFiltro(lista[1], "atualizacoes"), false);
    eq(P.noFiltro(lista[1], "todos"), true);
});

t("versoes: as versões de uma linha do histórico", () => {
    eq(P.versoes({ acao: "atualizado", de: "3.5.2-1", para: "3.5.3-1" }), "3.5.2-1 → 3.5.3-1");
    eq(P.versoes({ acao: "rebaixado", de: "2-1", para: "1-1" }), "2-1 → 1-1");
    eq(P.versoes({ acao: "instalado", de: null, para: "10.2.0-1" }), "10.2.0-1");
    eq(P.versoes({ acao: "removido", de: "2.9.9-6", para: null }), "2.9.9-6");
    eq(P.versoes({ acao: "reinstalado", de: "1-1", para: "1-1" }), "1-1");
    eq(P.versoes(null), "");
});

t("linhaDeComando: o comando como a pessoa digitaria", () => {
    eq(P.linhaDeComando(["pkexec", "pacman", "-Rns", "--noconfirm", "--", "htop"]), "pkexec pacman -Rns --noconfirm -- htop");
    eq(P.linhaDeComando(["kitty", "--title=Mk · atualizações", "/usr/bin/mk-update"]), "kitty '--title=Mk · atualizações' /usr/bin/mk-update");
    eq(P.linhaDeComando(null), "");
});
