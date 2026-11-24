// shell/mare/SettingsVpnPage.qml, o trecho entre "puro: início" e "puro: fim":
// a chave e o estado de cada conexão (contando a ação em curso), o mapa de
// erros, o caminho de um arquivo solto na página e o recado da importação.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("SettingsVpnPage.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["importedNote", "kindName", "onFor", "pathFromUrl", "patch", "stateFor"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("SettingsVpnPage.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

const A = "6f1c2c1e-0b1d-4c55-9a43-2d1f0a7b9e10";
const B = "a2b7d3f4-88aa-4b0e-8f0e-5c1d2e3f4a5b";
// Uma conexão como o `mare-vpn list` devolve.
const con = (mais) => Object.assign({ uuid: A, name: "trabalho", kind: "wireguard", state: "off", iface: "", ip: "",
    secrets: false, username: "", autoconnect: false }, mais || {});

t("patch: põe a chave numa cópia; undefined tira", () => {
    const antes = { [A]: "erro velho" };
    igual(M.patch(antes, B, "sem rota"), { [A]: "erro velho", [B]: "sem rota" });
    igual(M.patch(antes, A, "outro"), { [A]: "outro" });
    igual(M.patch(antes, A, undefined), {});
    igual(M.patch({}, A, undefined), {});
    // O mapa de quem chamou não muda (o QML só percebe um mapa novo).
    igual(antes, { [A]: "erro velho" });
    assert.ok(M.patch(antes, B, true) !== antes);
});

t("patch: valor falso, vazio ou nulo é valor (só undefined tira)", () => {
    igual(M.patch({}, "k", false), { k: false });
    igual(M.patch({}, "k", ""), { k: "" });
    igual(M.patch({}, "k", null), { k: null });
    igual(M.patch({ k: 1 }, "outra", undefined), { k: 1 });
});

t("kindName: o tipo como veio do script", () => {
    eq(M.kindName(con()), "wireguard");
    eq(M.kindName(con({ kind: "openvpn" })), "openvpn");
    eq(M.kindName(con({ kind: "openconnect" })), "openconnect");
    eq(M.kindName(con({ kind: "vpn" })), "vpn");
});

t("onFor: sem ação em curso, a chave segue o estado", () => {
    eq(M.onFor(con({ state: "on" }), "", ""), true);
    eq(M.onFor(con({ state: "connecting" }), "", ""), true);
    eq(M.onFor(con({ state: "off" }), "", ""), false);
    eq(M.onFor(con({ state: "disconnecting" }), "", ""), false);
});

t("onFor: durante a ação nesta conexão, a chave já mostra para onde vai", () => {
    eq(M.onFor(con({ state: "off" }), A, "up"), true);
    eq(M.onFor(con({ state: "on" }), A, "down"), false);
    eq(M.onFor(con({ state: "on" }), A, "delete"), false);
    // "conectar ao iniciar" não mexe na chave de ligar.
    eq(M.onFor(con({ state: "on" }), A, "autoconnect"), true);
    eq(M.onFor(con({ state: "off" }), A, "autoconnect"), false);
});

t("onFor: a ação em outra conexão não mexe nesta", () => {
    eq(M.onFor(con({ state: "off" }), B, "up"), false);
    eq(M.onFor(con({ state: "on" }), B, "down"), true);
});

t("stateFor: os estados do NetworkManager, por extenso", () => {
    eq(M.stateFor(con({ state: "on" }), "", "", {}, {}), "conectada");
    eq(M.stateFor(con({ state: "off" }), "", "", {}, {}), "desligada");
    eq(M.stateFor(con({ state: "connecting" }), "", "", {}, {}), "conectando…");
    eq(M.stateFor(con({ state: "disconnecting" }), "", "", {}, {}), "desconectando…");
});

t("stateFor: a ação em curso nesta conexão fala primeiro", () => {
    eq(M.stateFor(con({ state: "off" }), A, "up", {}, {}), "conectando…");
    eq(M.stateFor(con({ state: "on" }), A, "down", {}, {}), "desconectando…");
    eq(M.stateFor(con({ state: "on" }), A, "delete", {}, {}), "esquecendo…");
    eq(M.stateFor(con({ state: "off" }), A, "delete", { [A]: "erro antigo" }, {}), "esquecendo…");
    // A ação em outra conexão não muda o texto desta.
    eq(M.stateFor(con({ state: "on" }), B, "down", {}, {}), "conectada");
    eq(M.stateFor(con({ state: "off" }), B, "up", {}, {}), "desligada");
});

t("stateFor: desligada com erro mostra o erro; conectada não mostra erro velho", () => {
    eq(M.stateFor(con(), "", "", { [A]: "o servidor não respondeu" }, {}), "falhou: o servidor não respondeu");
    eq(M.stateFor(con({ state: "on" }), "", "", { [A]: "o servidor não respondeu" }, {}), "conectada");
    // O erro é de outra conexão.
    eq(M.stateFor(con(), "", "", { [B]: "x" }, {}), "desligada");
});

t("stateFor: o erro vai como veio, com aspas, dois-pontos, barra e acento", () => {
    const erro = "Error: \"wg0\": não achou C:\\chave <b>";
    eq(M.stateFor(con(), "", "", { [A]: erro }, {}), "falhou: " + erro);
});

t("stateFor: a que pede senha avisa do terminal, pelo perfil ou por já ter falhado assim", () => {
    eq(M.stateFor(con({ kind: "openvpn", secrets: true }), "", "", {}, {}), "pede senha: conecta pelo terminal");
    eq(M.stateFor(con({ kind: "openvpn" }), "", "", {}, { [A]: true }), "pede senha: conecta pelo terminal");
    // O erro, quando há, vem antes do aviso.
    eq(M.stateFor(con({ secrets: true }), "", "", { [A]: "senha recusada" }, { [A]: true }), "falhou: senha recusada");
    // Conectada, não precisa avisar.
    eq(M.stateFor(con({ state: "on", secrets: true }), "", "", {}, {}), "conectada");
});

t("pathFromUrl: um arquivo solto (file://) vira caminho, com os %XX desfeitos", () => {
    eq(M.pathFromUrl("file:///home/u/Downloads/trabalho.conf"), "/home/u/Downloads/trabalho.conf");
    eq(M.pathFromUrl("file:///home/u/Meus%20Documentos/vpn%20casa.ovpn"), "/home/u/Meus Documentos/vpn casa.ovpn");
    eq(M.pathFromUrl("file:///home/u/a%C3%A7%C3%A3o/t%C3%BAnel.conf"), "/home/u/ação/túnel.conf");
    eq(M.pathFromUrl("file:///tmp/a%3Ab%22c%27d%5Ce.conf"), "/tmp/a:b\"c'd\\e.conf");
    eq(M.pathFromUrl("file:///tmp/100%25.conf"), "/tmp/100%.conf");
});

t("pathFromUrl: o que não é file:// passa como veio", () => {
    eq(M.pathFromUrl("/home/u/trabalho.conf"), "/home/u/trabalho.conf");
    eq(M.pathFromUrl("~/vpn casa.ovpn"), "~/vpn casa.ovpn");
    eq(M.pathFromUrl(""), "");
    eq(M.pathFromUrl("https://exemplo.com/a%20b.conf"), "https://exemplo.com/a%20b.conf");
    // Um caminho escrito à mão com % não é desfeito.
    eq(M.pathFromUrl("/tmp/100%25.conf"), "/tmp/100%25.conf");
});

t("pathFromUrl: % malformado não lança; fica o caminho cru", () => {
    eq(M.pathFromUrl("file:///tmp/100%.conf"), "/tmp/100%.conf");
    eq(M.pathFromUrl("file:///tmp/%E0%A4%A.conf"), "/tmp/%E0%A4%A.conf");
});

t("pathFromUrl: aceita o objeto de URL do QML (qualquer coisa com toString)", () => {
    eq(M.pathFromUrl({ toString: () => "file:///tmp/a%20b.conf" }), "/tmp/a b.conf");
    eq(M.pathFromUrl(new URL("file:///tmp/a b.conf")), "/tmp/a b.conf");
});

t("importedNote: o WireGuard avisa que foi desligado depois de importar", () => {
    eq(M.importedNote("trabalho", "wireguard"),
        "importada: trabalho (wireguard). o networkmanager a liga ao importar; já foi desligada. ligue na lista acima.");
    eq(M.importedNote("casa: \"ação\"", "openvpn"), "importada: casa: \"ação\" (openvpn). está desligada; ligue na lista acima.");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
