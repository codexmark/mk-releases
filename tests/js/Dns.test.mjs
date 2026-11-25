// shell/mare/Dns.js: a tabela de provedores de DNS, a validação dos endereços
// que a pessoa escreve (IPv4 e IPv6 de verdade, no máximo quatro), que opção
// uma lista de servidores é, o que a página diz do estado e do teste, e os
// argumentos que ela passa ao scripts/mare-dns.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { load, source } from "./load.mjs";

const D = load("Dns.js");
const ids = () => D.providers.map((p) => p.id);

t("a tabela: da rede primeiro, os cinco provedores e 'outro' por último", () => {
    igual(ids(), ["rede", "cloudflare", "cloudflare-familia", "quad9", "google", "adguard", "outro"]);
});

t("cada opção tem nome e uma linha dizendo o que faz", () => {
    const ruins = [];
    for (const p of D.providers) {
        if (typeof p.name !== "string" || p.name === "")
            ruins.push(p.id + ": sem nome");
        if (typeof p.desc !== "string" || p.desc.length < 12)
            ruins.push(p.id + ": sem a linha do que faz");
        if (!/^[a-z0-9]+(-[a-z0-9]+)*$/.test(p.id))
            ruins.push(p.id + ": id com caractere que o script não aceita");
    }
    vazio(ruins);
});

t("a linha de cada provedor diz o que ele filtra (ou que não filtra)", () => {
    assert.match(D.provider("cloudflare").desc, /não filtra/);
    assert.match(D.provider("cloudflare-familia").desc, /malware/);
    assert.match(D.provider("cloudflare-familia").desc, /adulto/);
    assert.match(D.provider("quad9").desc, /malware/);
    assert.match(D.provider("google").desc, /não filtra/);
    assert.match(D.provider("adguard").desc, /anúncio/);
    assert.match(D.provider("rede").desc, /roteador/);
});

t("os endereços de cada provedor: dois IPv4 e dois IPv6, nessa ordem", () => {
    igual(D.provider("cloudflare").servers, ["1.1.1.1", "1.0.0.1", "2606:4700:4700::1111", "2606:4700:4700::1001"]);
    igual(D.provider("cloudflare-familia").servers, ["1.1.1.3", "1.0.0.3", "2606:4700:4700::1113", "2606:4700:4700::1003"]);
    igual(D.provider("quad9").servers, ["9.9.9.9", "149.112.112.112", "2620:fe::fe", "2620:fe::9"]);
    igual(D.provider("google").servers, ["8.8.8.8", "8.8.4.4", "2001:4860:4860::8888", "2001:4860:4860::8844"]);
    igual(D.provider("adguard").servers, ["94.140.14.14", "94.140.15.15", "2a10:50c0::ad1:ff", "2a10:50c0::ad2:ff"]);
    igual(D.provider("rede").servers, []);
    igual(D.provider("outro").servers, []);
});

t("todo endereço da tabela passa na própria validação, sem repetição, e cabe no máximo", () => {
    const ruins = [], vistos = new Set();
    for (const p of D.providers) {
        if (p.servers.length > D.max)
            ruins.push(p.id + ": mais de " + D.max);
        p.servers.forEach((s, i) => {
            if (!(i < 2 ? D.isIPv4(s) : D.isIPv6(s)))
                ruins.push(p.id + ": " + s);
            if (vistos.has(s))
                ruins.push(p.id + ": " + s + " repetido");
            vistos.add(s);
        });
    }
    vazio(ruins);
    eq(D.max, 4);
});

t("provider: o id desconhecido não devolve nada", () => {
    eq(D.provider("opendns"), null);
    eq(D.provider(""), null);
    eq(D.provider("__proto__"), null);
});

// A tabela existe também no script (o `aplicar <id>` da linha de comando) e
// no ajudante que escreve em /etc não: as duas têm de dizer o mesmo.
t("a tabela do scripts/mare-dns é a mesma daqui", () => {
    const sh = source("scripts/mare-dns");
    const doScript = {};
    for (const m of sh.matchAll(/^\s+([a-z0-9-]+)\) echo "([0-9a-f:. ]+)" ;;$/gm))
        doScript[m[1]] = m[2].split(" ");
    const daqui = {};
    for (const p of D.providers)
        if (p.servers.length > 0)
            daqui[p.id] = p.servers;
    igual(doScript, daqui);
});

t("isIPv4: quatro números de 0 a 255, sem zero à esquerda", () => {
    for (const s of ["1.1.1.1", "9.9.9.9", "192.168.1.1", "10.0.0.53", "127.0.0.1", "149.112.112.112", "223.255.255.254"])
        eq(D.isIPv4(s), true, s);
    for (const s of ["", "1.1.1", "1.1.1.1.1", "256.1.1.1", "1.1.1.256", "01.1.1.1", "1.1.1.01", "1.1.1.", ".1.1.1",
        "1..1.1", "1.1.1.1 ", " 1.1.1.1", "1.1.1.1\n", "a.b.c.d", "1.1.1.1/24", "1.1.1.1:53", "0x1.1.1.1", "１.1.1.1",
        "-1.1.1.1", "1.1.1.1e0", "1,1,1,1"])
        eq(D.isIPv4(s), false, JSON.stringify(s));
});

t("isIPv4: endereço que não serve de servidor (0.0.0.0, multicast, broadcast) não passa", () => {
    for (const s of ["0.0.0.0", "0.1.2.3", "224.0.0.251", "239.255.255.250", "240.0.0.1", "255.255.255.255"])
        eq(D.isIPv4(s), false, s);
});

t("isIPv6: oito grupos de até quatro dígitos hexadecimais, com um '::' no máximo", () => {
    for (const s of ["2606:4700:4700::1111", "2620:fe::fe", "2620:fe::9", "2a10:50c0::ad1:ff", "::1", "fd00::53",
        "2001:db8:0:0:0:0:0:1", "2001:0db8:0000:0000:0000:0000:0000:0001", "2001:DB8::A", "2001:db8::", "1:2:3:4:5:6:7::",
        "::2:3:4:5:6:7:8"])
        eq(D.isIPv6(s), true, s);
    for (const s of ["", ":", "::", ":::", "2001:db8", "1:2:3:4:5:6:7", "1:2:3:4:5:6:7:8:9", "1::2::3", "2001:db8::g",
        "2001:db8::12345", ":1:2:3:4:5:6:7", "1:2:3:4:5:6:7:", "2001:db8:::1", "1:2:3:4:5:6:7:8::", "::1:2:3:4:5:6:7:8",
        "2001:db8::1 ", "2001:db8::1\n", "[2001:db8::1]", "2001:db8::1/64", "fe80::1%wlan0", "::ffff:1.2.3.4", "1.1.1.1"])
        eq(D.isIPv6(s), false, JSON.stringify(s));
});

t("isIPv6: multicast (ff00::/8) não serve de servidor", () => {
    eq(D.isIPv6("ff02::fb"), false);
    eq(D.isIPv6("FF02::1"), false);
});

t("isAddress: um ou outro; o que não é texto não passa", () => {
    eq(D.isAddress("9.9.9.9"), true);
    eq(D.isAddress("2620:fe::fe"), true);
    eq(D.isAddress("dns.quad9.net"), false);
    eq(D.isAddress(null), false);
    eq(D.isAddress(undefined), false);
    eq(D.isAddress(9), false);
    eq(D.isAddress(["9.9.9.9"]), false);
});

t("o que quebraria o arquivo do NetworkManager nunca é endereço", () => {
    for (const s of ["1.1.1.1\n[main]", "1.1.1.1,8.8.8.8", "1.1.1.1;8.8.8.8", "servers=1.1.1.1", "[global-dns]", "1.1.1.1 #x",
        "dns+tls://1.1.1.1", "1.1.1.1\\", "$(reboot)", "`reboot`", "1.1.1.1\r", "1.1.1.1\t", "::1\u0000"])
        eq(D.isAddress(s), false, JSON.stringify(s));
});

t("parse: separa por espaço, vírgula, ponto e vírgula e quebra de linha", () => {
    igual(D.parse("9.9.9.9 149.112.112.112"), { servers: ["9.9.9.9", "149.112.112.112"], error: "" });
    igual(D.parse(" 9.9.9.9,  2620:fe::fe ;10.0.0.1\n10.0.0.2 "),
        { servers: ["9.9.9.9", "2620:fe::fe", "10.0.0.1", "10.0.0.2"], error: "" });
    igual(D.parse("192.168.1.1"), { servers: ["192.168.1.1"], error: "" });
});

t("parse: o IPv6 sai em minúsculas e o repetido entra uma vez só", () => {
    igual(D.parse("2620:FE::FE 2620:fe::fe 9.9.9.9 9.9.9.9"), { servers: ["2620:fe::fe", "9.9.9.9"], error: "" });
});

t("parse: vazio pede um endereço", () => {
    igual(D.parse(""), { servers: [], error: "escreva ao menos um endereço" });
    igual(D.parse("  , ; \n"), { servers: [], error: "escreva ao menos um endereço" });
    igual(D.parse(null), { servers: [], error: "escreva ao menos um endereço" });
    igual(D.parse(undefined), { servers: [], error: "escreva ao menos um endereço" });
});

t("parse: um que não é endereço reprova tudo e diz qual", () => {
    igual(D.parse("9.9.9.9 dns.quad9.net"), { servers: [], error: "não é um endereço IP: dns.quad9.net" });
    igual(D.parse("1.1.1.256"), { servers: [], error: "não é um endereço IP: 1.1.1.256" });
    igual(D.parse("9.9.9.9 [main]"), { servers: [], error: "não é um endereço IP: [main]" });
    igual(D.parse("servers=1.1.1.1"), { servers: [], error: "não é um endereço IP: servers=1.1.1.1" });
});

t("parse: o texto recusado aparece cortado (não vai uma página inteira para o recado)", () => {
    const longo = "x".repeat(200);
    const r = D.parse(longo);
    igual(r.servers, []);
    assert.ok(r.error.length < 80, r.error);
    assert.ok(r.error.endsWith("…"));
});

t("parse: mais de quatro não passa", () => {
    igual(D.parse("10.0.0.1 10.0.0.2 10.0.0.3 10.0.0.4"),
        { servers: ["10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"], error: "" });
    igual(D.parse("10.0.0.1 10.0.0.2 10.0.0.3 10.0.0.4 10.0.0.5"), { servers: [], error: "no máximo 4 endereços" });
    // Repetidos não contam duas vezes.
    igual(D.parse("10.0.0.1 10.0.0.1 10.0.0.2 10.0.0.3 10.0.0.4").error, "");
});

t("optionOf: lista vazia é 'da rede'; a de um provedor é ele; o resto é 'outro'", () => {
    eq(D.optionOf([]), "rede");
    eq(D.optionOf(null), "rede");
    eq(D.optionOf(undefined), "rede");
    for (const p of D.providers)
        if (p.servers.length > 0)
            eq(D.optionOf(p.servers.slice()), p.id);
    eq(D.optionOf(["192.168.1.1"]), "outro");
    // Só parte de um provedor, ou a ordem trocada, já não é o que o Mk gravou.
    eq(D.optionOf(["1.1.1.1", "1.0.0.1"]), "outro");
    eq(D.optionOf(["1.0.0.1", "1.1.1.1", "2606:4700:4700::1111", "2606:4700:4700::1001"]), "outro");
    eq(D.optionOf(["1.1.1.1", "1.0.0.1", "2606:4700:4700::1111", "2606:4700:4700::1001", "9.9.9.9"]), "outro");
});

// O que o `mare-dns estado` responde.
const estado = (mais) => Object.assign({
    opcao: "rede", arquivo: [], servidores: ["192.168.1.1"], valendo: true, ajudante: true, conexao: "", pending: "", last: null
}, mais || {});

t("stateOf: fica com os campos do estado, do tipo certo; resposta vazia vira 'da rede'", () => {
    igual(D.stateOf(estado()), { opcao: "rede", arquivo: [], servidores: ["192.168.1.1"], valendo: true, ajudante: true,
        conexao: "", pending: "" });
    igual(D.stateOf({}), { opcao: "rede", arquivo: [], servidores: [], valendo: true, ajudante: false, conexao: "", pending: "" });
    igual(D.stateOf(null), { opcao: "rede", arquivo: [], servidores: [], valendo: true, ajudante: false, conexao: "", pending: "" });
    igual(D.stateOf({ opcao: "inventada", arquivo: "x", servidores: null, valendo: "sim", ajudante: 1, conexao: 5, pending: null }),
        { opcao: "outro", arquivo: [], servidores: [], valendo: false, ajudante: false, conexao: "", pending: "" });
});

t("stateOf: só endereço de verdade fica nas listas (o resolv.conf é um arquivo qualquer)", () => {
    const s = D.stateOf(estado({ opcao: "outro", arquivo: ["10.0.0.1", "<b>x</b>", 7], servidores: ["10.0.0.1", "lixo", null] }));
    igual(s.arquivo, ["10.0.0.1"]);
    igual(s.servidores, ["10.0.0.1"]);
});

t("stateOf: o resultado da última troca não faz parte do estado", () => {
    assert.ok(!("last" in D.stateOf(estado({ last: { acao: "aplicar", ok: true } }))));
});

t("selected: a opção marcada é a gravada; durante a troca, já a pedida", () => {
    eq(D.selected(D.stateOf(estado()), ""), "rede");
    eq(D.selected(D.stateOf(estado({ opcao: "quad9", arquivo: D.provider("quad9").servers })), ""), "quad9");
    eq(D.selected(D.stateOf(estado()), "adguard"), "adguard");
});

t("summary: 'da rede' diz de onde vêm os servidores", () => {
    eq(D.summary(D.stateOf(estado())), "da rede: o que o roteador entrega");
    eq(D.summary(D.stateOf(estado({ servidores: [] }))), "da rede: nenhum servidor agora (sem conexão?)");
});

t("summary: 'da rede' com DNS fixo na conexão avisa que não é o do roteador", () => {
    eq(D.summary(D.stateOf(estado({ conexao: "Casa 5G", servidores: ["1.1.1.1"] }))),
        "da rede, mas a conexão Casa 5G tem um DNS fixo próprio");
});

t("summary: um provedor gravado e em uso", () => {
    const q = D.provider("quad9").servers;
    eq(D.summary(D.stateOf(estado({ opcao: "quad9", arquivo: q, servidores: q.slice(0, 3) }))), "Quad9, em todas as redes");
    eq(D.summary(D.stateOf(estado({ opcao: "outro", arquivo: ["10.0.0.1"], servidores: ["10.0.0.1"] }))),
        "outro, em todas as redes");
});

t("summary: gravado mas o sistema usa outros servidores (uma VPN, ou o NetworkManager não recarregou)", () => {
    const q = D.provider("quad9").servers;
    eq(D.summary(D.stateOf(estado({ opcao: "quad9", arquivo: q, servidores: ["10.8.0.1"], valendo: false }))),
        "Quad9 está gravado, mas o sistema usa outros servidores agora");
});

t("serversLine: os servidores em uso, numa linha; sem nenhum, um traço", () => {
    eq(D.serversLine(["1.1.1.1", "1.0.0.1"]), "1.1.1.1 · 1.0.0.1");
    eq(D.serversLine([]), "—");
    eq(D.serversLine(null), "—");
});

t("argsFor: 'da rede' limpa, um provedor vai pelo id, 'outro' leva os endereços validados", () => {
    igual(D.argsFor("rede", ""), { args: ["limpar"], error: "" });
    igual(D.argsFor("quad9", "lixo que não importa"), { args: ["aplicar", "quad9"], error: "" });
    igual(D.argsFor("outro", "10.0.0.1, FD00::53"), { args: ["aplicar", "outro", "10.0.0.1", "fd00::53"], error: "" });
});

t("argsFor: 'outro' com texto ruim não gera comando nenhum", () => {
    igual(D.argsFor("outro", ""), { args: [], error: "escreva ao menos um endereço" });
    igual(D.argsFor("outro", "9.9.9.9; rm -rf /"), { args: [], error: "não é um endereço IP: rm" });
    igual(D.argsFor("outro", "--help"), { args: [], error: "não é um endereço IP: --help" });
});

t("argsFor: opção que não existe não gera comando", () => {
    igual(D.argsFor("opendns", ""), { args: [], error: "opção desconhecida" });
    igual(D.argsFor("", ""), { args: [], error: "opção desconhecida" });
    igual(D.argsFor("limpar", ""), { args: [], error: "opção desconhecida" });
});

t("noteOf: sem resultado, sem recado", () => {
    igual(D.noteOf(null), { text: "", error: false, back: false });
    igual(D.noteOf(undefined), { text: "", error: false, back: false });
    igual(D.noteOf("x"), { text: "", error: false, back: false });
});

t("noteOf: a troca que deu certo e resolve nomes", () => {
    igual(D.noteOf({ acao: "aplicar", ok: true, message: "DNS trocado: Quad9", resolve: true }),
        { text: "DNS trocado: Quad9. resolve nomes.", error: false, back: false });
});

t("noteOf: aplicou e não resolve: avisa e oferece voltar para o da rede", () => {
    igual(D.noteOf({ acao: "aplicar", ok: true, message: "DNS trocado: outro", resolve: false }),
        { text: "DNS trocado: outro. mas não resolve nomes: sem internet, ou esses servidores não respondem daqui.",
          error: true, back: true });
});

t("noteOf: voltar para o da rede que não resolve não oferece voltar de novo", () => {
    igual(D.noteOf({ acao: "limpar", ok: true, message: "voltou para o DNS da rede", resolve: false }),
        { text: "voltou para o DNS da rede. mas não resolve nomes: sem internet?", error: true, back: false });
});

t("noteOf: sem o teste (resolve nulo), só o recado", () => {
    igual(D.noteOf({ acao: "aplicar", ok: true, message: "DNS trocado: Google", resolve: null }),
        { text: "DNS trocado: Google", error: false, back: false });
});

t("noteOf: a troca que falhou mostra o erro e não oferece voltar (nada mudou)", () => {
    igual(D.noteOf({ acao: "aplicar", ok: false, error: "a senha não foi dada: nada mudou", message: "a senha não foi dada: nada mudou", resolve: null }),
        { text: "a senha não foi dada: nada mudou", error: true, back: false });
    igual(D.noteOf({ acao: "aplicar", ok: false }), { text: "não deu certo", error: true, back: false });
});

t("testNote: o botão testar diz 'resolve' ou 'não resolve', com o nome testado", () => {
    igual(D.testNote({ resolve: true, nome: "archlinux.org" }, "quad9"), { text: "resolve (archlinux.org)", error: false, back: false });
    igual(D.testNote({ resolve: false, nome: "archlinux.org" }, "rede"), { text: "não resolve (archlinux.org)", error: true, back: false });
});

t("testNote: não resolve com um DNS escolhido: oferece voltar para o da rede", () => {
    igual(D.testNote({ resolve: false, nome: "archlinux.org" }, "quad9"), { text: "não resolve (archlinux.org)", error: true, back: true });
    igual(D.testNote({ resolve: false, nome: "archlinux.org" }, "outro"), { text: "não resolve (archlinux.org)", error: true, back: true });
});

t("testNote: resposta estragada conta como não resolve", () => {
    igual(D.testNote(null, "rede"), { text: "não resolve", error: true, back: false });
    igual(D.testNote({ resolve: "true" }, "quad9"), { text: "não resolve", error: true, back: true });
});

t("os avisos da página: DNS cifrado fica de fora, e a VPN pode impor o dela", () => {
    assert.match(D.limits, /DoT/);
    assert.match(D.limits, /DoH/);
    assert.match(D.limits, /systemd-resolved/);
    assert.match(D.limits, /VPN/);
});

t("o arquivo é uma biblioteca do QML, sem eval e sem rede", () => {
    const src = source("Dns.js");
    assert.ok(src.startsWith(".pragma library\n"));
    assert.ok(!/\beval\s*\(|new Function|XMLHttpRequest|fetch\(/.test(src));
});
