// shell/mare/BluetoothPanel.qml, o trecho entre "puro: início" e "puro: fim":
// o ícone de cada tipo de aparelho e o texto da bateria dele.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("BluetoothPanel.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["batteryText", "icon"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("BluetoothPanel.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

const PADRAO = "󰂯";

t("icon: cada tipo que o BlueZ informa tem o seu ícone", () => {
    // Os nomes de ícone do BlueZ (org.bluez.Device1.Icon).
    const tipos = ["audio-headset", "audio-headphones", "audio-card", "multimedia-player", "input-mouse", "input-keyboard",
        "input-gaming", "input-tablet", "phone", "computer", "printer", "camera-photo", "video-display"];
    const vistos = new Set();
    for (const tipo of tipos) {
        const i = M.icon({ icon: tipo });
        assert.ok(i !== PADRAO && i.length > 0, tipo + " caiu no ícone padrão");
        vistos.add(i);
    }
    // audio-card e multimedia-player são a mesma caixa de som.
    eq(vistos.size, tipos.length - 1);
    eq(M.icon({ icon: "audio-card" }), M.icon({ icon: "multimedia-player" }));
    eq(M.icon({ icon: "input-gaming" }), M.icon({ icon: "joystick" }));
});

t("icon: headset e headphone não caem no ícone genérico de áudio", () => {
    assert.notEqual(M.icon({ icon: "audio-headset" }), M.icon({ icon: "audio-card" }));
    assert.notEqual(M.icon({ icon: "audio-headphones" }), M.icon({ icon: "audio-card" }));
    assert.notEqual(M.icon({ icon: "audio-headset" }), M.icon({ icon: "audio-headphones" }));
});

t("icon: tipo desconhecido, vazio, nulo ou ausente fica com o do Bluetooth", () => {
    eq(M.icon({ icon: "network-wireless" }), PADRAO);
    eq(M.icon({ icon: "" }), PADRAO);
    eq(M.icon({ icon: null }), PADRAO);
    eq(M.icon({}), PADRAO);
});

t("batteryText: a bateria vem de 0 a 1", () => {
    eq(M.batteryText({ connected: true, batteryAvailable: true, battery: 0.8 }), "80%");
    eq(M.batteryText({ connected: true, batteryAvailable: true, battery: 0.555 }), "56%");
    eq(M.batteryText({ connected: true, batteryAvailable: true, battery: 1 }), "100%");
    eq(M.batteryText({ connected: true, batteryAvailable: true, battery: 0 }), "0%");
});

t("batteryText: se vier em porcentagem, não estoura", () => {
    eq(M.batteryText({ connected: true, batteryAvailable: true, battery: 80 }), "80%");
    eq(M.batteryText({ connected: true, batteryAvailable: true, battery: 100 }), "100%");
    eq(M.batteryText({ connected: true, batteryAvailable: true, battery: 37.4 }), "37%");
});

t("batteryText: desconectado ou sem o dado, fica vazio", () => {
    eq(M.batteryText({ connected: false, batteryAvailable: true, battery: 0.8 }), "");
    eq(M.batteryText({ connected: true, batteryAvailable: false, battery: 0 }), "");
    eq(M.batteryText({ connected: false, batteryAvailable: false }), "");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
