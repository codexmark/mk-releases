// shell/mare/NetworkPanel.qml, o trecho entre "puro: início" e "puro: fim":
// a ordem das redes ao redor, a leitura do ping, a janela das últimas medidas,
// a perda, as redes guardadas que o mare-wifi lista e a linha da VPN.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("NetworkPanel.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["last", "lossOf", "ms", "nearest", "pingMs", "push", "savedMap", "vpnLine"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("NetworkPanel.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

const rede = (name, signalStrength, connected) => ({ name: name, signalStrength: signalStrength, connected: connected === true });
const nomes = (lista) => lista.map((r) => r.name);

t("nearest: a conectada primeiro, depois do sinal mais forte para o mais fraco", () => {
    const lista = [rede("Vizinho", 0.9), rede("Casa", 0.55, true), rede("Padaria: wi-fi", 0.2), rede("Café \"Ação\"", 0.7)];
    igual(nomes(M.nearest(lista, 6)), ["Casa", "Vizinho", "Café \"Ação\"", "Padaria: wi-fi"]);
});

t("nearest: corta no máximo e não mexe na lista que recebeu", () => {
    const lista = [0.1, 0.8, 0.3, 0.9, 0.5, 0.7, 0.2, 0.6].map((s, i) => rede("r" + i, s));
    const antes = nomes(lista).join(",");
    igual(nomes(M.nearest(lista, 6)), ["r3", "r1", "r5", "r7", "r4", "r2"]);
    eq(nomes(lista).join(","), antes);
    igual(M.nearest([], 6), []);
    eq(M.nearest(lista, 20).length, 8);
});

t("nearest: a conectada fica em cima mesmo com o pior sinal e fora do corte", () => {
    const lista = [0.9, 0.8, 0.7, 0.6, 0.5, 0.4].map((s, i) => rede("r" + i, s)).concat([rede("minha", 0.05, true)]);
    const perto = M.nearest(lista, 6);
    eq(perto[0].name, "minha");
    eq(perto.length, 6);
    assert.ok(!nomes(perto).includes("r5"));
});

t("last: a última medida; sem medida, -1", () => {
    eq(M.last([12.5, 13.1, 9.8]), 9.8);
    eq(M.last([-1]), -1);
    eq(M.last([]), -1);
});

t("push: junta a medida nova e guarda só as últimas 24", () => {
    igual(M.push([], 5), [5]);
    igual(M.push([1, 2], 3), [1, 2, 3]);
    let lista = [];
    for (let i = 1; i <= 30; i++)
        lista = M.push(lista, i);
    eq(lista.length, 24);
    eq(lista[0], 7);
    eq(lista[23], 30);
});

t("push não mexe na lista que recebeu (o QML só percebe uma lista nova)", () => {
    const antes = [1, 2, 3];
    const depois = M.push(antes, 4);
    igual(antes, [1, 2, 3]);
    assert.ok(depois !== antes);
});

t("ms: uma casa abaixo de 10, inteiro dali em diante, travessão sem resposta", () => {
    eq(M.ms(3.24), "3,2 ms");
    eq(M.ms(0.412), "0,4 ms");
    eq(M.ms(0), "0,0 ms");
    eq(M.ms(10), "10 ms");
    eq(M.ms(12.6), "13 ms");
    eq(M.ms(248.3), "248 ms");
    eq(M.ms(-1), "—");
});

t("ms: perto de 10 não escreve '10,0 ms' (9,96 arredonda para 10 ms)", () => {
    eq(M.ms(9.94), "9,9 ms");
    eq(M.ms(9.96), "10 ms");
    eq(M.ms(9.999), "10 ms");
    const ruins = [];
    for (let v = 0; v < 12; v += 0.001)
        if (/^10,0 ms$/.test(M.ms(v)))
            ruins.push(v.toFixed(3));
    vazio(ruins, "valores escritos como '10,0 ms'");
});

t("lossOf: a porcentagem das medidas sem resposta", () => {
    eq(M.lossOf([]), 0);
    eq(M.lossOf([12, 13, 11]), 0);
    eq(M.lossOf([-1, -1]), 100);
    eq(M.lossOf([10, -1, 10, 10]), 25);
    eq(M.lossOf([10, 10, -1]), 33);
    eq(M.lossOf([-1, -1, 10]), 67);
    // 0 ms é resposta, não perda.
    eq(M.lossOf([0, 0.4]), 0);
});

// O que `LC_ALL=C ping -n -c 1 -W 1 <ip>` imprime.
const PING = `PING 192.168.0.1 (192.168.0.1) 56(84) bytes of data.
64 bytes from 192.168.0.1: icmp_seq=1 ttl=64 time=2.47 ms

--- 192.168.0.1 ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 0ms
rtt min/avg/max/mdev = 2.470/2.470/2.470/0.000 ms
`;
const SEM_RESPOSTA = `PING 1.1.1.1 (1.1.1.1) 56(84) bytes of data.

--- 1.1.1.1 ping statistics ---
1 packets transmitted, 0 received, 100% packet loss, time 0ms

`;

t("pingMs: o tempo da resposta, em ms", () => {
    eq(M.pingMs(PING), 2.47);
    eq(M.pingMs("64 bytes from 1.1.1.1: icmp_seq=1 ttl=57 time=118 ms\n"), 118);
    eq(M.pingMs("64 bytes from 1.1.1.1: icmp_seq=1 ttl=57 time=0.041 ms\n"), 0.041);
});

t("pingMs: sem resposta, saída vazia ou erro contam como perda (-1)", () => {
    eq(M.pingMs(SEM_RESPOSTA), -1);
    eq(M.pingMs(""), -1);
    eq(M.pingMs("ping: connect: Network is unreachable\n"), -1);
    eq(M.pingMs("From 192.168.0.10 icmp_seq=1 Destination Host Unreachable\n"), -1);
});

t("pingMs: o 'time 0ms' do resumo não é confundido com a resposta", () => {
    // Na saída sem resposta só existe o "time 0ms" (sem o sinal de igual).
    assert.ok(SEM_RESPOSTA.includes("time 0ms"));
    eq(M.pingMs(SEM_RESPOSTA), -1);
});

t("savedMap: nome → reconecta sozinho?", () => {
    igual(M.savedMap([{ ssid: "Casa", autoconnect: true }, { ssid: "Hotel", autoconnect: false }]), { "Casa": true, "Hotel": false });
    igual(M.savedMap([]), {});
});

t("savedMap: dois perfis com o mesmo nome, vale o que reconecta (em qualquer ordem)", () => {
    igual(M.savedMap([{ ssid: "Casa", autoconnect: false }, { ssid: "Casa", autoconnect: true }]), { "Casa": true });
    igual(M.savedMap([{ ssid: "Casa", autoconnect: true }, { ssid: "Casa", autoconnect: false }]), { "Casa": true });
    igual(M.savedMap([{ ssid: "Casa", autoconnect: false }, { ssid: "Casa", autoconnect: false }]), { "Casa": false });
});

t("savedMap: nome com dois-pontos, barra, aspas, acento e espaço é a chave inteira", () => {
    const tortos = ["Café: \"Ação\"", "a\\b", "it's", " espaço ", "日本", "a:b:c"];
    const mapa = M.savedMap(tortos.map((s) => ({ ssid: s, autoconnect: true })));
    igual(Object.keys(mapa).sort(), tortos.slice().sort());
});

t("vpnLine: o nome, quantas mais, a interface e o IP", () => {
    eq(M.vpnLine({ up: true, count: 1, name: "trabalho", kind: "wireguard", iface: "wg0", ip: "10.8.0.2" }), "trabalho  ·  wg0  ·  10.8.0.2");
    eq(M.vpnLine({ up: true, count: 3, name: "trabalho", kind: "wireguard", iface: "wg0", ip: "10.8.0.2" }), "trabalho +2  ·  wg0  ·  10.8.0.2");
    eq(M.vpnLine({ up: true, count: 2, name: "casa: \"vpn\"", kind: "openvpn", iface: "", ip: "" }), "casa: \"vpn\" +1");
    eq(M.vpnLine({ up: true, count: 1, name: "ação", kind: "openvpn", iface: "tun0", ip: "" }), "ação  ·  tun0");
    eq(M.vpnLine({ up: true, count: 1, name: "x", kind: "openvpn", iface: "", ip: "10.0.0.5" }), "x  ·  10.0.0.5");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
