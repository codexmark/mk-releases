// apps/mkporto/qml/Porto.js: a lógica da janela do MkPorto que não depende do
// QML: procurar e ordenar os hosts, o estado de cada um (a bolinha), a
// validação do formulário ao digitar (as mesmas regras do motor), o pedido do
// `guardar`, os textos e os atalhos.
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const P = load(REPO + "/apps/mkporto/qml/Porto.js");

const host = (apelido, extra) => Object.assign({
    apelido, host: apelido + ".oficina.example", usuario: null, porta: 22, chave: null, salto: null,
    origem: "config", arquivo: "~/.ssh/config"
}, extra || {});

const hosts = [
    host("farol", { host: "192.0.2.10", usuario: "ana" }),
    host("Atlas", { usuario: "marina", porta: 2200, origem: "mkporto", arquivo: "~/.ssh/config.d/mkporto.conf" }),
    host("cofre", { salto: "farol", host: "cofre.interno.test" }),
    host("água", { host: "192.0.2.44" }),
    host("web10"),
    host("web2")
];
const apelidos = (l) => l.map((h) => h.apelido);

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["fold", "plural", "linha", "casa", "filtrar", "ordenar", "visiveis", "escolhido", "vizinho", "comecaSonda",
        "aplicaSonda", "encerraSonda", "estado", "textoDoEstado", "latencia", "resumo", "origem", "editavel", "erroDoApelido", "erroDoHost",
        "erroDoUsuario", "erroDaPorta", "erroDaChave", "erroDoSalto", "validar", "valido", "formularioVazio", "formularioDe",
        "pedido", "estreita", "apertada", "arquivoDeExemplo", "tipoEBits", "abriria", "sugestaoDeChave", "aparado", "compara", "copia", "erroDePalavra",
        "textoDaEntrada", "fonteDosValores", "erroDoNomeDaChave", "erroDoComentario", "validarChave", "chaveValida", "argsDaChave",
        "bolinhaDaMaquina", "haQuanto", "textoDoEstadoDaMaquina", "memoria", "tamanho", "resumoDaMaquina", "cpuPct", "textoDoUso",
        "erroDoNomeDaMaquina", "erroDeInteiro", "erroDaMemoria", "erroDasCpus", "erroDoDisco", "erroDaIso", "erroDoRotulo", "validarMaquina",
        "maquinaValida", "maquinaVazia", "pedidoDeMaquina", "maquinaEscolhida", "maquinaVizinha", "faltas", "podeNaMaquina",
        "porQueNao", "caminhoDeUrl", "resumoDasMaquinas"];
    const funcoes = Object.keys(P).filter((k) => typeof P[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("fold: sem maiúsculas nem acentos, para a busca", () => {
    eq(P.fold("Água Benta"), "agua benta");
    eq(P.fold("ÇÃO"), "cao");
    eq(P.fold(null), "");
    eq(P.fold(undefined), "");
    eq(P.fold(42), "42");
});

t("plural", () => {
    eq(P.plural(0, "host", "hosts"), "0 hosts");
    eq(P.plural(1, "host", "hosts"), "1 host");
    eq(P.plural(6, "host", "hosts"), "6 hosts");
});

t("linha: usuario@host:porta, sem inventar usuário", () => {
    eq(P.linha(hosts[1]), "marina@Atlas.oficina.example:2200");
    eq(P.linha(hosts[0]), "ana@192.0.2.10:22");
    eq(P.linha(hosts[3]), "192.0.2.44:22");
    eq(P.linha(host("v6", { host: "2001:db8::1", usuario: "x" })), "x@[2001:db8::1]:22");
    eq(P.linha(null), "");
});

t("casa e filtrar: todos os termos, no apelido, no endereço, no usuário ou no salto", () => {
    igual(apelidos(P.filtrar(hosts, "")), apelidos(hosts));
    igual(apelidos(P.filtrar(hosts, "   ")), apelidos(hosts));
    igual(apelidos(P.filtrar(hosts, "atl")), ["Atlas"]);
    igual(apelidos(P.filtrar(hosts, "AGUA")), ["água"]);
    igual(apelidos(P.filtrar(hosts, "192.0.2")), ["farol", "água"]);
    igual(apelidos(P.filtrar(hosts, "marina")), ["Atlas"]);
    igual(apelidos(P.filtrar(hosts, "farol")), ["farol", "cofre"]);
    igual(apelidos(P.filtrar(hosts, "web oficina")), ["web10", "web2"]);
    igual(apelidos(P.filtrar(hosts, "web cofre")), []);
    igual(apelidos(P.filtrar(hosts, "2200")), ["Atlas"]);
    eq(P.casa(hosts[0], "FAR"), true);
    eq(P.casa(null, "x"), false);
    igual(P.filtrar(null, "x"), []);
});

t("ordenar: por apelido, sem ligar para maiúscula nem acento, e número como número", () => {
    igual(apelidos(P.ordenar(hosts)), ["água", "Atlas", "cofre", "farol", "web2", "web10"]);
    // Não mexe na lista que recebeu.
    eq(hosts[0].apelido, "farol");
    igual(P.ordenar(null), []);
    igual(apelidos(P.ordenar([host("b"), host("B"), host("a")])), ["a", "B", "b"]);
});

t("visiveis: a busca e a ordem juntas", () => {
    igual(apelidos(P.visiveis(hosts, "web")), ["web2", "web10"]);
    igual(apelidos(P.visiveis(hosts, "")), ["água", "Atlas", "cofre", "farol", "web2", "web10"]);
});

t("escolhido: o host do apelido, ou null", () => {
    eq(P.escolhido(hosts, "cofre").host, "cofre.interno.test");
    eq(P.escolhido(hosts, "fantasma"), null);
    eq(P.escolhido(hosts, ""), null);
    eq(P.escolhido(null, "x"), null);
});

t("vizinho: anda na lista com as setas, sem sair dela", () => {
    const l = P.visiveis(hosts, "");
    eq(P.vizinho(l, "Atlas", 1), "cofre");
    eq(P.vizinho(l, "Atlas", -1), "água");
    eq(P.vizinho(l, "água", -1), "água");
    eq(P.vizinho(l, "web10", 1), "web10");
    // Sem escolhido (ou com um que saiu da busca): o primeiro, ou o último.
    eq(P.vizinho(l, "", 1), "água");
    eq(P.vizinho(l, "fantasma", -1), "web10");
    eq(P.vizinho([], "x", 1), "");
    eq(P.vizinho(null, "x", 1), "");
});

t("comecaSonda e aplicaSonda: o mapa das sondas nunca é mudado no lugar", () => {
    const vazio = {};
    const a = P.comecaSonda(vazio, ["farol", "Atlas"]);
    igual(vazio, {});
    igual(a, { farol: { sondando: true }, Atlas: { sondando: true } });
    const b = P.aplicaSonda(a, { apelido: "farol", responde: true, ms: 12, motivo: null });
    eq(a.farol.sondando, true);
    igual(b.farol, { sondando: false, responde: true, ms: 12, motivo: null, servidor: null, via: null });
    eq(b.Atlas.sondando, true);
    // Linha torta do motor: ignora.
    igual(P.aplicaSonda(a, null), a);
    igual(P.aplicaSonda(a, { erro: "x" }), a);
    igual(P.aplicaSonda(a, { apelido: 3 }), a);
    // Uma sonda nova de um host só não apaga as dos outros.
    const c = P.comecaSonda(b, ["Atlas"]);
    eq(c.farol.responde, true);
    igual(P.comecaSonda(null, null), {});
});

t("estado: a bolinha de cada host", () => {
    const s = {
        farol: { sondando: false, responde: true, ms: 12, motivo: null },
        Atlas: { sondando: false, responde: false, ms: null, motivo: "recusou a conexão" },
        web2: { sondando: true },
        cofre: { sondando: false, responde: null, ms: null, motivo: "por salto" },
        web10: { sondando: false, responde: null, ms: null, motivo: "não conheço esse apelido" }
    };
    eq(P.estado(hosts[0], s), "responde");
    eq(P.estado(hosts[1], s), "nao");
    eq(P.estado(hosts[5], s), "sondando");
    eq(P.estado(hosts[2], s), "salto");
    // Host com salto é "por salto" mesmo antes de qualquer sonda.
    eq(P.estado(hosts[2], {}), "salto");
    eq(P.estado(hosts[3], s), "desconhecido");
    eq(P.estado(hosts[4], s), "desconhecido");
    eq(P.estado(hosts[0], null), "desconhecido");
    eq(P.estado(null, s), "desconhecido");
});

t("textoDoEstado: a frase ao lado da bolinha", () => {
    const s = {
        farol: { sondando: false, responde: true, ms: 12, motivo: null },
        Atlas: { sondando: false, responde: false, ms: null, motivo: "recusou a conexão" },
        web2: { sondando: true },
        web10: { sondando: false, responde: false, ms: null, motivo: null }
    };
    eq(P.textoDoEstado(hosts[0], s), "responde em 12 ms");
    eq(P.textoDoEstado(hosts[1], s), "não responde: recusou a conexão");
    eq(P.textoDoEstado(hosts[4], s), "não responde");
    eq(P.textoDoEstado(hosts[5], s), "sondando…");
    eq(P.textoDoEstado(hosts[2], s), "por salto: só se chega nele por farol");
    eq(P.textoDoEstado(hosts[3], s), "ainda não sondado");
});

t("latencia: o que o motor não mediu vira um traço, nunca zero", () => {
    eq(P.latencia(12), "12 ms");
    eq(P.latencia(0), "<1 ms");
    eq(P.latencia(999), "999 ms");
    eq(P.latencia(1240), "1,2 s");
    for (const v of [null, undefined, NaN, -1, "12", Infinity])
        eq(P.latencia(v), "—", String(v));
});

t("resumo: quantos hosts e quantos respondem", () => {
    eq(P.resumo([], {}), "nenhum host");
    eq(P.resumo(hosts, {}), "6 hosts");
    eq(P.resumo([hosts[0]], {}), "1 host");
    const s = {
        farol: { sondando: false, responde: true, ms: 1 },
        Atlas: { sondando: false, responde: false },
        web2: { sondando: false, responde: true, ms: 3 },
        fantasma: { sondando: false, responde: true, ms: 3 }
    };
    eq(P.resumo(hosts, s), "6 hosts · 2 respondem · 1 não responde");
    eq(P.resumo(hosts, { farol: s.farol }), "6 hosts · 1 responde");
    eq(P.resumo(hosts, { web2: { sondando: true } }), "6 hosts · sondando…");
});

t("origem e editavel: só o que é do MkPorto dá para editar e remover", () => {
    eq(P.origem(hosts[1]), "MkPorto");
    eq(P.origem(hosts[0]), "o seu ~/.ssh/config");
    eq(P.editavel(hosts[1]), true);
    eq(P.editavel(hosts[0]), false);
    eq(P.editavel(null), false);
    eq(P.editavel({ origem: "outra" }), false);
});

t("erroDoApelido: só letras sem acento, algarismos, ponto, hífen e sublinhado; nunca começando com hífen", () => {
    for (const bom of ["a", "atlas", "Web_1.prod-2", "192.0.2.1", "_x", ".x", "a".repeat(64)])
        eq(P.erroDoApelido(bom), "", bom);
    for (const ruim of ["", "-atlas", "-oProxyCommand=x", "a b", "web*", "web?", "!x", "máquina", "a/b", "a\nHost *", "a\rb", "a;b",
        "$(x)", "a".repeat(65), " a", "a "])
        eq(P.erroDoApelido(ruim) !== "", true, JSON.stringify(ruim));
    eq(P.erroDoApelido(""), "falta o apelido");
    eq(P.erroDoApelido("-x"), "o apelido não pode começar com \"-\"");
});

t("erroDoHost: sem espaço, controle, aspas nem #; nunca começando com hífen", () => {
    for (const bom of ["atlas.oficina.example", "192.0.2.1", "2001:db8::1", "a", "servidor_1.test", "a".repeat(253)])
        eq(P.erroDoHost(bom), "", bom);
    for (const ruim of ["", "-oProxyCommand=touch pwn", "-x", "a b", "a\nProxyCommand x", "a\r\nb", "a\tb", "a\u0000b", "a\"b", "#a",
        "a#b", "a".repeat(254), "a b", "a\u007fb"])
        eq(P.erroDoHost(ruim) !== "", true, JSON.stringify(ruim));
    eq(P.erroDoHost(""), "falta o endereço");
    eq(P.erroDoHost("a\nb"), "o endereço não pode ter quebra de linha");
});

t("erroDoUsuario e erroDoSalto: podem ficar vazios; o resto é como o endereço", () => {
    for (const f of [P.erroDoUsuario, P.erroDoSalto]) {
        eq(f(""), "");
        eq(f("ana"), "");
        for (const ruim of ["-l", "a b", "a\nb", "a\"b", "#x"])
            eq(f(ruim) !== "", true, f.name + " " + JSON.stringify(ruim));
    }
    eq(P.erroDoSalto("ana@farol:2222,outro"), "");
    eq(P.erroDoUsuario("a".repeat(65)) !== "", true);
    eq(P.erroDoSalto("-oProxyCommand=x"), "o salto não pode começar com \"-\"");
});

t("erroDaPorta: inteiro de 1 a 65535; vazio é a 22", () => {
    for (const boa of ["", "22", "1", "65535", "2200", " 22 ", 22])
        eq(P.erroDaPorta(boa), "", String(boa));
    for (const ruim of ["0", "65536", "-1", "22.5", "2e3", "vinte", "22 23", "0x16", "99999999999"])
        eq(P.erroDaPorta(ruim), "a porta vai de 1 a 65535", ruim);
});

t("erroDaChave: um caminho, com espaço se for o caso; sem aspas, barra invertida nem controle", () => {
    for (const boa of ["", "~/.ssh/id_ed25519", "~/.ssh/minha chave", "/caminho/inteiro/chave"])
        eq(P.erroDaChave(boa), "", boa);
    for (const ruim of ["-x", "~/.ssh/k\nProxyCommand x", "~/.ssh/k\" x \"", "~/.ssh/k\\", "a\u0007b", "a\rb"])
        eq(P.erroDaChave(ruim) !== "", true, JSON.stringify(ruim));
});

t("validar: os erros de cada campo, e o apelido que já existe", () => {
    const f = { apelido: "novo", host: "novo.test", usuario: "", porta: "", chave: "", salto: "" };
    igual(P.validar(f, hosts, ""), { apelido: "", host: "", usuario: "", porta: "", chave: "", salto: "" });
    eq(P.valido(P.validar(f, hosts, "")), true);

    const ruim = P.validar({ apelido: "-x", host: "a b", usuario: "a\nb", porta: "0", chave: "-k", salto: "a b" }, hosts, "");
    for (const campo of ["apelido", "host", "usuario", "porta", "chave", "salto"])
        eq(ruim[campo] !== "", true, campo);
    eq(P.valido(ruim), false);

    // Já é do config da pessoa: o MkPorto não mexe.
    eq(P.validar(Object.assign({}, f, { apelido: "farol" }), hosts, "").apelido, "farol já está no seu ~/.ssh/config");
    // Já é de outro host do MkPorto.
    eq(P.validar(Object.assign({}, f, { apelido: "Atlas" }), hosts, "").apelido, "já existe um host com esse apelido");
    // Editando o próprio Atlas, o apelido dele pode ficar.
    eq(P.validar(Object.assign({}, f, { apelido: "Atlas" }), hosts, "Atlas").apelido, "");
    // E pode trocar para um novo, mas não para o de outro.
    eq(P.validar(Object.assign({}, f, { apelido: "titan" }), hosts, "Atlas").apelido, "");
    eq(P.validar(Object.assign({}, f, { apelido: "web2" }), hosts, "Atlas").apelido, "web2 já está no seu ~/.ssh/config");
    // Formulário pela metade (campos faltando) não estoura.
    eq(P.valido(P.validar({}, null, "")), false);
    eq(P.valido(null), false);
});

t("formularioVazio e formularioDe: o que o formulário mostra ao abrir", () => {
    igual(P.formularioVazio(), { apelido: "", host: "", usuario: "", porta: "22", chave: "", salto: "" });
    igual(P.formularioDe(hosts[1]), { apelido: "Atlas", host: "Atlas.oficina.example", usuario: "marina", porta: "2200", chave: "", salto: "" });
    igual(P.formularioDe(hosts[2]).salto, "farol");
    igual(P.formularioDe(null), P.formularioVazio());
});

t("pedido: o JSON do guardar, com a porta como número e sem espaço sobrando", () => {
    igual(P.pedido({ apelido: " atlas ", host: " atlas.test ", usuario: "", porta: "", chave: " ~/.ssh/k ", salto: "" }, ""),
        { apelido: "atlas", host: "atlas.test", usuario: "", porta: 22, chave: "~/.ssh/k", salto: "", anterior: "" });
    igual(P.pedido({ apelido: "titan", host: "x.test", usuario: "ana", porta: "2200", chave: "", salto: "farol" }, "atlas"),
        { apelido: "titan", host: "x.test", usuario: "ana", porta: 2200, chave: "", salto: "farol", anterior: "atlas" });
    // O pedido é um objeto: quem vira texto é o JSON.stringify, e uma quebra
    // de linha num campo chega ao motor escapada (e ele recusa).
    const texto = JSON.stringify(P.pedido({ apelido: "a\nb", host: "x" }, ""));
    eq(texto.indexOf("\n"), -1);
    eq(JSON.parse(texto).apelido, "a\nb");
});

t("estreita: abaixo de 720 de largura o detalhe não cabe ao lado da lista", () => {
    eq(P.estreita(900), false);
    eq(P.estreita(720), false);
    eq(P.estreita(719), true);
    eq(P.estreita(520), true);
    eq(P.LARGURA_DE_DUAS_COLUNAS, 720);
});

t("apertada: abaixo de 400 de largura nem o alto cabe inteiro", () => {
    eq(P.apertada(400), false);
    eq(P.apertada(399), true);
    eq(P.apertada(295), true);
});

t("arquivoDeExemplo: a resposta gravada de cada pedido", () => {
    eq(P.arquivoDeExemplo(["listar"]), "listar.json");
    eq(P.arquivoDeExemplo(["sondar"]), "sondar.json");
    eq(P.arquivoDeExemplo(["sondar", "--", "atlas"]), "sondar.json");
    eq(P.arquivoDeExemplo(["chaves"]), "chaves.json");
    eq(P.arquivoDeExemplo(["guardar"]), "guardar.json");
    eq(P.arquivoDeExemplo(["remover", "--", "x"]), "remover.json");
    eq(P.arquivoDeExemplo(["comando", "conectar", "--", "atlas"]), "comando-conectar.json");
    eq(P.arquivoDeExemplo(["comando", "chave", "--", "atlas", "k.pub"]), "comando-chave.json");
    // Nada que venha da pessoa entra no nome do arquivo.
    eq(P.arquivoDeExemplo(["../../etc/passwd"]), "nada.json");
    eq(P.arquivoDeExemplo(["comando", "../x"]), "nada.json");
    eq(P.arquivoDeExemplo(null), "nada.json");
});

t("tipoEBits: o tipo da chave e o tamanho, quando o motor soube", () => {
    eq(P.tipoEBits({ tipo: "ed25519", bits: 256 }), "ed25519 · 256 bits");
    eq(P.tipoEBits({ tipo: "rsa", bits: null }), "rsa");
    eq(P.tipoEBits(null), "");
});

t("abriria: no modo de exemplo, a janela diz o que abriria em vez de abrir", () => {
    eq(P.abriria({ argv: ["ssh", "--", "atlas"] }), "exemplo: abriria um terminal com ssh -- atlas");
    eq(P.abriria(null), "exemplo: não abriria nada");
});

t("sugestaoDeChave: a chave privada que acompanha uma pública, só pelo nome", () => {
    eq(P.sugestaoDeChave({ arquivo: "~/.ssh/id_ed25519.pub" }), "~/.ssh/id_ed25519");
    eq(P.sugestaoDeChave({ arquivo: "~/.ssh/estranho" }), "");
    eq(P.sugestaoDeChave(null), "");
});

t("os atalhos da folha são os que o contrato lista", () => {
    const teclas = P.ATALHOS.map((a) => a[0]);
    for (const tecla of ["Ctrl+F", "Ctrl+N", "Enter", "F5", "Esc", "Ctrl+E", "Ctrl+K", "Delete", "Ctrl+Shift+C", "F1"])
        eq(teclas.includes(tecla), true, tecla);
    for (const a of P.ATALHOS)
        eq(Array.isArray(a) && a.length === 2 && a[1].length > 0, true, String(a));
});

t("os ajudantes: aparado, compara, copia e erroDePalavra", () => {
    eq(P.aparado({ a: "  x  " }, "a"), "x");
    eq(P.aparado({ a: null }, "a"), "");
    eq(P.aparado(null, "a"), "");
    eq(P.aparado({ a: 22 }, "a"), "22");
    eq(P.compara("web2", "web10") < 0, true);
    eq(P.compara("web10", "web2") > 0, true);
    eq(P.compara("a", "a"), 0);
    eq(P.compara("a", "ab") < 0, true);
    eq(P.compara("a1b", "a1c") < 0, true);
    const original = { a: { sondando: true } };
    const outra = P.copia(original);
    outra.b = 1;
    igual(Object.keys(original), ["a"]);
    igual(P.copia(null), {});
    eq(P.erroDePalavra("ana", "o usuário", 64), "");
    eq(P.erroDePalavra("-l", "o usuário", 64), "o usuário não pode começar com \"-\"");
    eq(P.erroDePalavra("a b", "o usuário", 64), "o usuário não pode ter espaço");
    eq(P.erroDePalavra("a#b", "o usuário", 64), "o usuário não pode ter #");
    eq(P.erroDePalavra("abc", "o usuário", 2), "o usuário é comprido demais (até 2)");
});

t("encerraSonda: quando o sondar acaba, ninguém fica sondando para sempre", () => {
    const s = { a: { sondando: true }, b: { sondando: false, responde: true, ms: 3, motivo: null } };
    igual(P.encerraSonda(s), { b: s.b });
    eq(s.a.sondando, true);
    igual(P.encerraSonda(null), {});
    eq(P.estado({ apelido: "a", salto: null }, P.encerraSonda(s)), "desconhecido");
});

// ── a segunda rodada: os valores do ssh, a sonda de verdade, a entrada e as chaves ──

t("aplicaSonda guarda o servidor e por onde a sonda passou", () => {
    const s = P.aplicaSonda({}, { apelido: "cofre", responde: true, ms: 180, motivo: null, servidor: "OpenSSH_9.9", via: "farol" });
    igual(s.cofre, { sondando: false, responde: true, ms: 180, motivo: null, servidor: "OpenSSH_9.9", via: "farol" });
    igual(P.aplicaSonda({}, { apelido: "x", responde: true, servidor: 3, via: {} }).x.servidor, null);
});

t("estado de um host por salto: o que a sonda pelo salto disse, ou \"salto\" se ela não pôde ir", () => {
    const cofre = hosts[2];
    eq(P.estado(cofre, { cofre: { sondando: false, responde: true, ms: 180, via: "farol" } }), "responde");
    eq(P.estado(cofre, { cofre: { sondando: false, responde: false, motivo: "o salto não deixou passar" } }), "nao");
    eq(P.estado(cofre, { cofre: { sondando: false, responde: null, motivo: "por salto" } }), "salto");
    eq(P.estado(cofre, { cofre: { sondando: true } }), "sondando");
    eq(P.estado(cofre, {}), "salto");
    eq(P.estado(cofre, null), "salto");
});

t("textoDoEstado diz por onde respondeu", () => {
    const cofre = hosts[2];
    eq(P.textoDoEstado(cofre, { cofre: { sondando: false, responde: true, ms: 180, via: "farol" } }), "responde por farol em 180 ms");
    eq(P.textoDoEstado(cofre, { cofre: { sondando: false, responde: false, motivo: "o salto não deixou passar: stdio forwarding failed" } }),
        "não responde: o salto não deixou passar: stdio forwarding failed");
    eq(P.textoDoEstado(cofre, {}), "por salto: só se chega nele por farol");
    eq(P.textoDoEstado(hosts[0], { farol: { sondando: false, responde: false, motivo: "a porta abre, mas não é um servidor ssh" } }),
        "não responde: a porta abre, mas não é um servidor ssh");
});

t("resumo conta quem respondeu pelo salto", () => {
    eq(P.resumo(hosts, { cofre: { sondando: false, responde: true, ms: 180, via: "farol" } }), "6 hosts · 1 responde");
});

t("formularioDe parte do que está ESCRITO no arquivo do app, não do que o ssh completou", () => {
    const h = host("atlas", {
        origem: "mkporto", host: "real.oficina.example", usuario: "local", porta: 2200, chave: "~/.ssh/geral", salto: "farol", efetivo: true,
        escrito: { host: "atlas.test", usuario: null, porta: 22, chave: null, salto: null }
    });
    igual(P.formularioDe(h), { apelido: "atlas", host: "atlas.test", usuario: "", porta: "22", chave: "", salto: "" });
    // Sem `escrito` (motor antigo, ou host do config), vale o que há.
    igual(P.formularioDe(host("x", { usuario: "ana", escrito: null })).usuario, "ana");
});

t("fonteDosValores: se é o ssh que diz, ou o leitor do config", () => {
    eq(P.fonteDosValores({ efetivo: true }), "como o ssh os vê");
    eq(P.fonteDosValores({ efetivo: false }), "lidos do config (o ssh não respondeu)");
    eq(P.fonteDosValores({}), "lidos do config (o ssh não respondeu)");
    eq(P.fonteDosValores(null), "");
});

t("textoDaEntrada: a linha do teste de entrada, só para o host a que ele pertence", () => {
    const h = hosts[1];
    eq(P.textoDaEntrada(null, h), "");
    eq(P.textoDaEntrada({ apelido: "outro", entra: true }, h), "");
    eq(P.textoDaEntrada({ apelido: "Atlas", testando: true }, h), "testando a entrada…");
    eq(P.textoDaEntrada({ apelido: "Atlas", entra: true, motivo: null, ms: 340 }, h), "entra: a máquina aceitou uma das suas chaves (340 ms)");
    eq(P.textoDaEntrada({ apelido: "Atlas", entra: false, motivo: "a máquina recusou a conexão", ms: 12 }, h),
        "não entra: a máquina recusou a conexão");
    eq(P.textoDaEntrada({ apelido: "Atlas", entra: false, motivo: null }, h), "não entra");
    eq(P.textoDaEntrada({ apelido: "Atlas", erro: "não achei o ssh nesta máquina" }, h), "não achei o ssh nesta máquina");
    eq(P.textoDaEntrada({ apelido: "Atlas", entra: true }, null), "");
});

t("erroDoNomeDaChave: vira arquivo em ~/.ssh e argumento do ssh-keygen", () => {
    for (const bom of ["trabalho", "id_ed25519_casa", "a", "Chave-2.nova", "_x", "a".repeat(64)])
        eq(P.erroDoNomeDaChave(bom), "", bom);
    for (const ruim of ["", "-x", ".oculta", "a b", "a/b", "../x", "chave.pub", "máquina", "a\nb", "a".repeat(65), "$x", "a;b"])
        eq(P.erroDoNomeDaChave(ruim) !== "", true, JSON.stringify(ruim));
    eq(P.erroDoNomeDaChave(""), "falta o nome da chave");
    eq(P.erroDoNomeDaChave(".x"), "o nome não pode começar com ponto");
});

t("erroDoComentario: pode ficar vazio e ter espaço; sem controle e sem começar com hífen", () => {
    for (const bom of ["", "ana no trabalho", "ana@oficina.example", "ana d'água $USER", "é".repeat(128)])
        eq(P.erroDoComentario(bom), "", bom);
    for (const ruim of ["-x", "a\nb", "a\rb", "a\u0000b", "a\tb", "é".repeat(129)])
        eq(P.erroDoComentario(ruim) !== "", true, JSON.stringify(ruim));
});

t("validarChave, chaveValida e argsDaChave: a folha da chave nova", () => {
    const chaves = [{ nome: "id_ed25519.pub" }, { nome: "trabalho.pub" }];
    igual(P.validarChave({ nome: "nova", comentario: "" }, chaves), { nome: "", comentario: "" });
    eq(P.chaveValida(P.validarChave({ nome: "nova", comentario: "x" }, chaves)), true);
    eq(P.validarChave({ nome: "trabalho", comentario: "" }, chaves).nome, "já existe uma chave com esse nome");
    eq(P.validarChave({ nome: " trabalho ", comentario: "" }, chaves).nome, "já existe uma chave com esse nome");
    eq(P.chaveValida(P.validarChave({ nome: "-x", comentario: "-y" }, chaves)), false);
    eq(P.chaveValida(P.validarChave({}, null)), false);
    eq(P.chaveValida(null), false);
    // O que vai para o motor depois de `comando gerar --`: o nome e, se houver, o comentário.
    igual(P.argsDaChave({ nome: " nova ", comentario: "  " }), ["nova"]);
    igual(P.argsDaChave({ nome: "nova", comentario: " ana no trabalho " }), ["nova", "ana no trabalho"]);
    igual(P.argsDaChave(null), [""]);
});

t("arquivoDeExemplo conhece o entrar e o comando gerar", () => {
    eq(P.arquivoDeExemplo(["entrar", "--", "atlas"]), "entrar.json");
    eq(P.arquivoDeExemplo(["comando", "gerar", "--", "nova"]), "comando-gerar.json");
});

t("abriria também serve para a chave nova, e os atalhos novos estão na folha", () => {
    eq(P.abriria({ argv: ["ssh-keygen", "-t", "ed25519", "-f", "~/.ssh/nova", "-C", "ana"] }),
        "exemplo: abriria um terminal com ssh-keygen -t ed25519 -f ~/.ssh/nova -C ana");
    const teclas = P.ATALHOS.map((a) => a[0]);
    for (const tecla of ["Ctrl+T", "Ctrl+G"])
        eq(teclas.includes(tecla), true, tecla);
});

// ── a terceira rodada: as máquinas ──────────────────────────────────────

const maquina = (nome, extra) => Object.assign({
    nome, memoria_mb: 2048, cpus: 2, disco_gb: 20, iso: null, uefi: false, tres_d: false, porta_ssh: 2222, criada_em: 1795000000,
    estado: "desligada", pid: null, desde: null, disco_bytes: 196616
}, extra || {});
const maquinas = [
    maquina("oficina", { estado: "ligada", pid: 4242, desde: 1795003600, memoria_mb: 4096 }),
    maquina("antiga", { estado: "pausada", pid: 77, desde: 1795000000 }),
    maquina("teste")
];

t("bolinhaDaMaquina: o estado da máquina na mesma bolinha dos hosts", () => {
    eq(P.bolinhaDaMaquina("ligada"), "responde");
    eq(P.bolinhaDaMaquina("pausada"), "salto");
    eq(P.bolinhaDaMaquina("travada"), "nao");
    eq(P.bolinhaDaMaquina("desligada"), "desconhecido");
    eq(P.bolinhaDaMaquina(undefined), "desconhecido");
});

t("haQuanto: há quanto tempo a máquina está ligada", () => {
    eq(P.haQuanto(1000, 1030), "há instantes");
    eq(P.haQuanto(1000, 1060), "há 1 min");
    eq(P.haQuanto(1000, 1000 + 59 * 60), "há 59 min");
    eq(P.haQuanto(1000, 1000 + 3600), "há 1 h");
    eq(P.haQuanto(1000, 1000 + 5 * 3600 + 120), "há 5 h");
    eq(P.haQuanto(1000, 1000 + 49 * 3600), "há 2 d");
    // Relógio torto ou dado que falta: nada, em vez de um número inventado.
    for (const [a, b] of [[null, 1000], [1000, null], [2000, 1000], ["x", 1000], [NaN, 1000]])
        eq(P.haQuanto(a, b), "", a + " " + b);
});

t("textoDoEstadoDaMaquina", () => {
    eq(P.textoDoEstadoDaMaquina(maquinas[0], 1795003600 + 180), "ligada há 3 min");
    eq(P.textoDoEstadoDaMaquina(maquinas[0], null), "ligada");
    eq(P.textoDoEstadoDaMaquina(maquinas[1], 1795003600), "pausada");
    eq(P.textoDoEstadoDaMaquina(maquinas[2], 1795003600), "desligada");
    eq(P.textoDoEstadoDaMaquina(maquina("x", { estado: "travada" }), 1), "travada: o QEMU está de pé, mas não responde");
    eq(P.textoDoEstadoDaMaquina(null, 1), "");
});

t("memoria e tamanho: o que o motor não soube vira um traço", () => {
    eq(P.memoria(512), "512 MB");
    eq(P.memoria(1024), "1 GB");
    eq(P.memoria(4096), "4 GB");
    eq(P.memoria(1536), "1,5 GB");
    eq(P.tamanho(0), "0 B");
    eq(P.tamanho(196616), "192 KB");
    eq(P.tamanho(5368709120), "5,0 GB");
    eq(P.tamanho(21474836480), "20 GB");
    for (const v of [null, undefined, NaN, -1, "3"]) {
        eq(P.memoria(v), "—", String(v));
        eq(P.tamanho(v), "—", String(v));
    }
});

t("resumoDaMaquina e resumoDasMaquinas", () => {
    eq(P.resumoDaMaquina(maquinas[0]), "4 GB · 2 CPUs");
    eq(P.resumoDaMaquina(maquina("x", { cpus: 1 })), "2 GB · 1 CPU");
    eq(P.resumoDaMaquina(null), "");
    eq(P.resumoDasMaquinas(maquinas), "3 máquinas · 1 ligada · 1 pausada");
    eq(P.resumoDasMaquinas([maquinas[2]]), "1 máquina");
    eq(P.resumoDasMaquinas([maquinas[0], maquinas[0]]), "2 máquinas · 2 ligadas");
    eq(P.resumoDasMaquinas([]), "nenhuma máquina");
    eq(P.resumoDasMaquinas(null), "nenhuma máquina");
});

t("cpuPct: a porcentagem de CPU pela diferença entre duas leituras do vm uso", () => {
    const antes = { pid: 4242, ticks: 1000, hz: 100, agora_ms: 10000 };
    // 100 ticks a 100 Hz = 1 s de CPU em 2 s de relógio = 50%.
    eq(P.cpuPct(antes, { pid: 4242, ticks: 1100, hz: 100, agora_ms: 12000 }), 50);
    // Duas CPUs cheias: 200%.
    eq(P.cpuPct(antes, { pid: 4242, ticks: 1400, hz: 100, agora_ms: 12000 }), 200);
    eq(P.cpuPct(antes, { pid: 4242, ticks: 1000, hz: 100, agora_ms: 12000 }), 0);
    // Não dá para saber: primeira leitura, outro processo (a máquina religou),
    // relógio que não andou, contador que voltou, dado torto.
    eq(P.cpuPct(null, antes), null);
    eq(P.cpuPct(antes, null), null);
    eq(P.cpuPct(antes, { pid: 99, ticks: 1100, hz: 100, agora_ms: 12000 }), null);
    eq(P.cpuPct(antes, { pid: 4242, ticks: 1100, hz: 100, agora_ms: 10000 }), null);
    eq(P.cpuPct(antes, { pid: 4242, ticks: 900, hz: 100, agora_ms: 12000 }), null);
    eq(P.cpuPct(antes, { pid: 4242, ticks: 1100, hz: 0, agora_ms: 12000 }), null);
    eq(P.cpuPct(antes, { pid: 4242, ticks: "x", hz: 100, agora_ms: 12000 }), null);
});

t("textoDoUso", () => {
    eq(P.textoDoUso(50, 2097152), "CPU 50% · memória 2,0 GB");
    eq(P.textoDoUso(12.34, 524288), "CPU 12% · memória 512 MB");
    eq(P.textoDoUso(null, 524288), "CPU — · memória 512 MB");
    eq(P.textoDoUso(null, null), "CPU — · memória —");
});

t("erroDoNomeDaMaquina: vira pasta, argumento do QEMU e apelido de host", () => {
    for (const bom of ["a", "oficina", "win-11", "teste_2.novo", "0", "a".repeat(40)])
        eq(P.erroDoNomeDaMaquina(bom), "", bom);
    for (const ruim of ["", "-x", ".x", "_x", "Oficina", "a b", "a/b", "..", "máquina", "a\nb", "a,b", "a".repeat(41)])
        eq(P.erroDoNomeDaMaquina(ruim) !== "", true, JSON.stringify(ruim));
    eq(P.erroDoNomeDaMaquina(""), "falta o nome da máquina");
    eq(P.erroDoNomeDaMaquina("a".repeat(41)), "o nome é comprido demais (até 40)");
});

t("erroDaMemoria, erroDasCpus e erroDoDisco: inteiros dentro dos limites do motor", () => {
    for (const boa of ["512", "65536", "4096", " 2048 ", 2048])
        eq(P.erroDaMemoria(boa), "", String(boa));
    for (const ruim of ["", "511", "65537", "2.5", "2e3", "dois", "-1", "0x800"])
        eq(P.erroDaMemoria(ruim), "a memória vai de 512 a 65536 MB", ruim);
    for (const boa of ["1", "32", "4"])
        eq(P.erroDasCpus(boa), "", boa);
    for (const ruim of ["", "0", "33", "1.5"])
        eq(P.erroDasCpus(ruim), "as CPUs vão de 1 a 32", ruim);
    for (const bom of ["1", "2000", "40"])
        eq(P.erroDoDisco(bom), "", bom);
    for (const ruim of ["", "0", "2001", "20G"])
        eq(P.erroDoDisco(ruim), "o disco vai de 1 a 2000 GB", ruim);
    eq(P.erroDeInteiro("7", 1, 10, "x"), "");
    eq(P.erroDeInteiro("11", 1, 10, "x"), "x");
    eq(P.erroDeInteiro("99999999999", 1, 10, "x"), "x");
});

t("erroDaIso: um caminho inteiro, sem vírgula (o QEMU a lê como separador)", () => {
    for (const boa of ["", "/isos/sistema.iso", "/isos/um sistema (novo).iso"])
        eq(P.erroDaIso(boa), "", boa);
    for (const ruim of ["relativo.iso", "~/x.iso", "/isos/a,b.iso", "/isos/a.iso,readonly=off", "/isos/a\n.iso", "/a\u0007.iso", "-cdrom"])
        eq(P.erroDaIso(ruim) !== "", true, JSON.stringify(ruim));
    eq(P.erroDaIso("/a,b.iso").indexOf("vírgula") >= 0, true);
    eq(P.erroDaIso("x.iso"), "o caminho da ISO tem de ser inteiro, começando por /");
});

t("erroDoRotulo: o rótulo de um snapshot", () => {
    for (const bom of ["antes", "Antes-de-atualizar_2.0", "1", "a".repeat(40)])
        eq(P.erroDoRotulo(bom), "", bom);
    for (const ruim of ["", "-a", "a b", "a/b", "ação", "a\nb", "a".repeat(41)])
        eq(P.erroDoRotulo(ruim) !== "", true, JSON.stringify(ruim));
    eq(P.erroDoRotulo(""), "falta o rótulo");
});

t("validarMaquina, maquinaValida, maquinaVazia e pedidoDeMaquina: a folha da máquina nova", () => {
    const vazia = P.maquinaVazia({ ovmf: { tem: true } });
    igual(vazia, { nome: "", iso: "", memoria_mb: "4096", cpus: "2", disco_gb: "40", uefi: true, tres_d: false });
    eq(P.maquinaVazia({ ovmf: { tem: false } }).uefi, false);
    eq(P.maquinaVazia(null).uefi, false);

    const f = Object.assign({}, vazia, { nome: "nova" });
    igual(P.validarMaquina(f, maquinas, { ovmf: { tem: true } }), { nome: "", memoria_mb: "", cpus: "", disco_gb: "", iso: "", uefi: "" });
    eq(P.maquinaValida(P.validarMaquina(f, maquinas, { ovmf: { tem: true } })), true);
    eq(P.validarMaquina(Object.assign({}, f, { nome: "oficina" }), maquinas, null).nome, "já existe uma máquina com esse nome");
    eq(P.validarMaquina(Object.assign({}, f, { nome: " oficina " }), maquinas, null).nome, "já existe uma máquina com esse nome");
    // UEFI sem o OVMF: a folha já diz, antes de o motor recusar.
    eq(P.validarMaquina(f, maquinas, { ovmf: { tem: false } }).uefi, "UEFI precisa do OVMF: instale o pacote edk2-ovmf");
    eq(P.validarMaquina(Object.assign({}, f, { uefi: false }), maquinas, { ovmf: { tem: false } }).uefi, "");
    const ruim = P.validarMaquina({ nome: "-x", iso: "x.iso", memoria_mb: "1", cpus: "99", disco_gb: "", uefi: false }, maquinas, null);
    for (const campo of ["nome", "memoria_mb", "cpus", "disco_gb", "iso"])
        eq(ruim[campo] !== "", true, campo);
    eq(P.maquinaValida(ruim), false);
    eq(P.maquinaValida(P.validarMaquina({}, null, null)), false);
    eq(P.maquinaValida(null), false);

    igual(P.pedidoDeMaquina({ nome: " nova ", iso: " /isos/a b.iso ", memoria_mb: " 4096 ", cpus: "2", disco_gb: "40", uefi: true, tres_d: false }),
        { nome: "nova", memoria_mb: 4096, cpus: 2, disco_gb: 40, iso: "/isos/a b.iso", uefi: true, tres_d: false });
    igual(P.pedidoDeMaquina({ nome: "x" }), { nome: "x", memoria_mb: 0, cpus: 0, disco_gb: 0, iso: "", uefi: false, tres_d: false });
});

t("maquinaEscolhida e maquinaVizinha: a lista anda pelas setas", () => {
    eq(P.maquinaEscolhida(maquinas, "antiga").pid, 77);
    eq(P.maquinaEscolhida(maquinas, "fantasma"), null);
    eq(P.maquinaEscolhida(null, "x"), null);
    eq(P.maquinaVizinha(maquinas, "oficina", 1), "antiga");
    eq(P.maquinaVizinha(maquinas, "oficina", -1), "oficina");
    eq(P.maquinaVizinha(maquinas, "teste", 1), "teste");
    eq(P.maquinaVizinha(maquinas, "", 1), "oficina");
    eq(P.maquinaVizinha(maquinas, "fantasma", -1), "teste");
    eq(P.maquinaVizinha([], "x", 1), "");
});

t("faltas: o que falta para ter máquinas, do vm requisitos", () => {
    igual(P.faltas({ qemu: { tem: true }, qemu_img: { tem: true }, pronto: true }), []);
    igual(P.faltas({ qemu: { tem: false }, qemu_img: { tem: true }, pronto: false }), ["o QEMU (qemu-system-x86_64)"]);
    igual(P.faltas({ qemu: { tem: false }, qemu_img: { tem: false }, pronto: false }), ["o QEMU (qemu-system-x86_64)", "o qemu-img"]);
    igual(P.faltas(null), []);
});

t("podeNaMaquina e porQueNao: o que cada estado deixa fazer", () => {
    const [ligada, pausada, desligada] = maquinas;
    const travada = maquina("x", { estado: "travada" });
    const tabela = {
        ligar: [false, false, true, false], desligar: [true, true, false, false], forcar: [true, true, false, true],
        pausar: [true, false, false, false], continuar: [false, true, false, false], snapshots: [false, false, true, false],
        remover: [false, false, true, false], host: [true, true, true, true]
    };
    for (const acao in tabela)
        igual([ligada, pausada, desligada, travada].map((m) => P.podeNaMaquina(m, acao)), tabela[acao], acao);
    eq(P.podeNaMaquina(null, "ligar"), false);
    eq(P.podeNaMaquina(desligada, "inventada"), false);
    eq(P.porQueNao(ligada, "snapshots"), "snapshots só com a máquina desligada");
    eq(P.porQueNao(ligada, "remover"), "remover só com a máquina desligada");
    eq(P.porQueNao(desligada, "snapshots"), "");
    eq(P.porQueNao(null, "snapshots"), "");
});

t("caminhoDeUrl: o que o seletor de arquivos devolve vira um caminho", () => {
    eq(P.caminhoDeUrl("file:///isos/um%20sistema%20(novo).iso"), "/isos/um sistema (novo).iso");
    eq(P.caminhoDeUrl("file:///isos/a%C3%A7%C3%A3o.iso"), "/isos/ação.iso");
    eq(P.caminhoDeUrl("/isos/a.iso"), "/isos/a.iso");
    eq(P.caminhoDeUrl("https://isos.example/a.iso"), "");
    eq(P.caminhoDeUrl("file:///isos/%E0%A4%A.iso"), "");
    eq(P.caminhoDeUrl(""), "");
    eq(P.caminhoDeUrl(null), "");
});

t("arquivoDeExemplo conhece as máquinas, e nada da pessoa entra no nome", () => {
    eq(P.arquivoDeExemplo(["vm", "listar"]), "vm-listar.json");
    eq(P.arquivoDeExemplo(["vm", "requisitos"]), "vm-requisitos.json");
    eq(P.arquivoDeExemplo(["vm", "uso", "--", "oficina"]), "vm-uso.json");
    eq(P.arquivoDeExemplo(["vm", "criar"]), "vm-criar.json");
    eq(P.arquivoDeExemplo(["vm", "snapshot", "listar", "--", "oficina"]), "vm-snapshot-listar.json");
    for (const acao of [["ligar", "--", "x"], ["desligar", "--forcar", "--", "x"], ["pausar"], ["continuar"], ["remover"], ["host"],
        ["snapshot", "criar", "--", "x", "y"], ["snapshot", "apagar"]])
        eq(P.arquivoDeExemplo(["vm"].concat(acao)), "vm-acao.json", acao.join(" "));
    eq(P.arquivoDeExemplo(["vm", "../x"]), "nada.json");
    eq(P.arquivoDeExemplo(["vm"]), "nada.json");
    eq(P.ATALHOS.map((a) => a[0]).includes("Ctrl+3"), true);
});
