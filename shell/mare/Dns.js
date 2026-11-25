.pragma library

// O DNS dos Ajustes (SettingsDnsPage.qml), como dados e funções puras: a
// tabela de opções, a validação do que a pessoa escreve em "outro", que opção
// uma lista de servidores é e os textos da página. Quem lê e grava o sistema
// é o scripts/mare-dns; aqui nada toca em nada. Testado em
// tests/js/Dns.test.mjs.
//
// A tabela de endereços existe também no scripts/mare-dns (o `aplicar <id>`
// da linha de comando); o teste confere que as duas dizem o mesmo.

// No máximo quatro endereços. (A glibc só consulta os três primeiros do
// /etc/resolv.conf; o quarto fica de reserva para quem lê o arquivo inteiro.)
var max = 4;

var providers = [
    { id: "rede", name: "da rede", servers: [],
      desc: "O que o roteador (ou o DHCP da rede) entrega. É o de fábrica: o Mk não deixa arquivo nenhum." },
    { id: "cloudflare", name: "Cloudflare", desc: "Rápido e sem registro de quem pergunta; não filtra nada.",
      servers: ["1.1.1.1", "1.0.0.1", "2606:4700:4700::1111", "2606:4700:4700::1001"] },
    { id: "cloudflare-familia", name: "Cloudflare família", desc: "O mesmo, mas bloqueia malware e conteúdo adulto.",
      servers: ["1.1.1.3", "1.0.0.3", "2606:4700:4700::1113", "2606:4700:4700::1003"] },
    { id: "quad9", name: "Quad9", desc: "Bloqueia domínios de malware e de golpe; de uma fundação sem fins lucrativos.",
      servers: ["9.9.9.9", "149.112.112.112", "2620:fe::fe", "2620:fe::9"] },
    { id: "google", name: "Google", desc: "Rápido e em todo lugar; não filtra nada.",
      servers: ["8.8.8.8", "8.8.4.4", "2001:4860:4860::8888", "2001:4860:4860::8844"] },
    { id: "adguard", name: "AdGuard", desc: "Bloqueia anúncios e rastreadores para todos os programas.",
      servers: ["94.140.14.14", "94.140.15.15", "2a10:50c0::ad1:ff", "2a10:50c0::ad2:ff"] },
    { id: "outro", name: "outro", servers: [],
      desc: "Os endereços que você escrever: de um a quatro, IPv4 ou IPv6." }
];

// O que a página diz sem rodeio, no rodapé.
var limits = "Aqui o DNS vai sem cifra: DNS cifrado (DoT/DoH) fica de fora, porque o NetworkManager sozinho não faz "
    + "(precisaria do systemd-resolved, que o Mk não usa). Uma VPN ligada pode impor o DNS dela enquanto durar. "
    + "Com um DNS escolhido aqui, nomes que só o roteador conhece (a intranet do trabalho, o portal do Wi-Fi de um "
    + "hotel) podem deixar de abrir: volte para o da rede.";

function provider(id) {
    for (var i = 0; i < providers.length; i++)
        if (providers[i].id === id)
            return providers[i];
    return null;
}

// Quatro números de 0 a 255, sem zero à esquerda, e que sirva de servidor:
// nada de 0.x, multicast (224 em diante) ou broadcast.
function isIPv4(s) {
    if (typeof s !== "string" || !/^(0|[1-9][0-9]{0,2})(\.(0|[1-9][0-9]{0,2})){3}$/.test(s))
        return false;
    var parts = s.split(".");
    for (var i = 0; i < 4; i++)
        if (Number(parts[i]) > 255)
            return false;
    var first = Number(parts[0]);
    return first >= 1 && first <= 223;
}

// Oito grupos de um a quatro dígitos hexadecimais, com um "::" no máximo no
// lugar de um ou mais grupos de zeros. Ficam de fora, de propósito: a forma
// com IPv4 no fim (::ffff:1.2.3.4), a zona (fe80::1%wlan0), o "::" sozinho e
// o multicast (ff00::/8).
function isIPv6(s) {
    if (typeof s !== "string" || s.length < 3 || s.length > 39 || !/^[0-9A-Fa-f:]+$/.test(s))
        return false;
    var halves = s.split("::");
    if (halves.length > 2)
        return false;
    var groups = [];
    for (var h = 0; h < halves.length; h++) {
        if (halves[h] === "")
            continue;
        var gs = halves[h].split(":");
        for (var i = 0; i < gs.length; i++) {
            if (!/^[0-9A-Fa-f]{1,4}$/.test(gs[i]))
                return false;
            groups.push(gs[i]);
        }
    }
    if (halves.length === 1 ? groups.length !== 8 : groups.length > 7 || groups.length < 1)
        return false;
    return !/^[fF][fF]/.test(s) || s.charAt(2) === ":" || s.charAt(3) === ":";
}

function isAddress(s) {
    return isIPv4(s) || isIPv6(s);
}

// O texto do campo "outro" vira a lista de endereços: separados por espaço,
// vírgula, ponto e vírgula ou quebra de linha; IPv6 em minúsculas; repetido
// entra uma vez. Um só que não seja endereço reprova tudo.
function parse(text) {
    var words = String(text === null || text === undefined ? "" : text).split(/[\s,;]+/);
    var servers = [];
    for (var i = 0; i < words.length; i++) {
        var w = words[i];
        if (w === "")
            continue;
        if (!isAddress(w))
            return { servers: [], error: "não é um endereço IP: " + (w.length > 40 ? w.slice(0, 40) + "…" : w) };
        w = w.toLowerCase();
        if (servers.indexOf(w) < 0)
            servers.push(w);
    }
    if (servers.length === 0)
        return { servers: [], error: "escreva ao menos um endereço" };
    if (servers.length > max)
        return { servers: [], error: "no máximo " + max + " endereços" };
    return { servers: servers, error: "" };
}

// Que opção é esta lista de servidores (a do arquivo do Mk): vazia é "da
// rede"; igual à de um provedor, na ordem, é ele; o resto é "outro".
function optionOf(servers) {
    if (!servers || servers.length === 0)
        return "rede";
    var line = servers.join(" ");
    for (var i = 0; i < providers.length; i++)
        if (providers[i].servers.length > 0 && providers[i].servers.join(" ") === line)
            return providers[i].id;
    return "outro";
}

function addresses(list) {
    var out = [];
    if (list && typeof list === "object" && list.length > 0)
        for (var i = 0; i < list.length; i++)
            if (isAddress(list[i]))
                out.push(list[i]);
    return out;
}

// O estado que o `mare-dns estado` contou, só com o que a página usa e do
// tipo certo (o resultado da última troca, `last`, vai à parte).
function stateOf(data) {
    var d = data && typeof data === "object" ? data : {};
    var known = typeof d.opcao === "string" && provider(d.opcao) !== null;
    return {
        opcao: d.opcao === undefined ? "rede" : known ? d.opcao : "outro",
        arquivo: addresses(d.arquivo),
        servidores: addresses(d.servidores),
        valendo: d.valendo === undefined ? true : d.valendo === true,
        ajudante: d.ajudante === true,
        conexao: typeof d.conexao === "string" ? d.conexao : "",
        pending: typeof d.pending === "string" ? d.pending : ""
    };
}

// A opção marcada: durante a troca, já a que foi pedida.
function selected(state, asking) {
    return asking !== "" ? asking : state.opcao;
}

// A frase do alto da página.
function summary(state) {
    if (state.opcao === "rede") {
        if (state.conexao !== "")
            return "da rede, mas a conexão " + state.conexao + " tem um DNS fixo próprio";
        return state.servidores.length > 0 ? "da rede: o que o roteador entrega"
            : "da rede: nenhum servidor agora (sem conexão?)";
    }
    var name = provider(state.opcao).name;
    return state.valendo ? name + ", em todas as redes"
        : name + " está gravado, mas o sistema usa outros servidores agora";
}

function serversLine(servers) {
    return servers && servers.length > 0 ? servers.join(" · ") : "—";
}

// Os argumentos do scripts/mare-dns para a opção escolhida (`text` é o campo
// de "outro"). Sem comando quando a opção não existe ou o texto não passa.
function argsFor(id, text) {
    if (id === "rede")
        return { args: ["limpar"], error: "" };
    if (id === "outro") {
        var p = parse(text);
        return p.error !== "" ? { args: [], error: p.error } : { args: ["aplicar", "outro"].concat(p.servers), error: "" };
    }
    var known = provider(id);
    if (known === null || known.servers.length === 0)
        return { args: [], error: "opção desconhecida" };
    return { args: ["aplicar", id], error: "" };
}

// O recado da última troca ({ acao, ok, message, resolve }, do mare-dns):
// o texto, se é má notícia, e se cabe oferecer "voltar para o da rede".
function noteOf(last) {
    if (!last || typeof last !== "object")
        return { text: "", error: false, back: false };
    var text = typeof last.message === "string" && last.message !== "" ? last.message : "";
    if (last.ok !== true)
        return { text: text !== "" ? text : "não deu certo", error: true, back: false };
    if (last.resolve === true)
        return { text: text + ". resolve nomes.", error: false, back: false };
    if (last.resolve === false)
        return last.acao === "aplicar"
            ? { text: text + ". mas não resolve nomes: sem internet, ou esses servidores não respondem daqui.", error: true, back: true }
            : { text: text + ". mas não resolve nomes: sem internet?", error: true, back: false };
    return { text: text, error: false, back: false };
}

// O recado do botão "testar" ({ resolve, nome }, do `mare-dns testar`), com a
// opção em vigor: se não resolve com um DNS escolhido, oferece voltar.
function testNote(result, opcao) {
    var r = result && typeof result === "object" ? result : {};
    var name = typeof r.nome === "string" && r.nome !== "" ? " (" + r.nome + ")" : "";
    if (r.resolve === true)
        return { text: "resolve" + name, error: false, back: false };
    return { text: "não resolve" + name, error: true, back: opcao !== "rede" };
}
