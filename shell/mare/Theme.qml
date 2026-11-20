pragma Singleton
import QtQuick
import Quickshell

// Direção Folha: os tokens do perfil codexmark. Preto que nunca é preto,
// texto em tons de osso, linhas finas e um único destaque, o cobre queimado.
// Projetos se distinguem por número e nome, não por cor.
Singleton {
    // Superfícies: do fundo ao mais elevado, todas quentes, nenhuma preta pura.
    readonly property color ground: "#0b0b0b"
    readonly property color surfaceLow: "#100f0e"
    readonly property color surface: "#141211"
    readonly property color raised: "#1b1917"

    // Linhas: o osso em transparência (13% e 26% no perfil).
    readonly property color hairline: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.13)
    readonly property color line: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.26)
    readonly property color wash: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.05)

    // Texto: osso e cinzas quentes.
    readonly property color text: "#cdc4ba"
    readonly property color textSoft: "#b0a9a0"
    readonly property color textDim: "#958f87"
    readonly property color textFaint: "#716b69"
    readonly property color neutral: "#807a73"

    // O único destaque.
    readonly property color copper: "#ce5e32"

    // Mantido para os painéis que pedem uma cor por projeto: agora é sempre o cobre.
    function accentFor(name) {
        return copper;
    }

    readonly property var families: Qt.fontFamilies()
    readonly property string serif: families.indexOf("Fraunces") >= 0 ? "Fraunces" : "Noto Serif"
    readonly property string sans: families.indexOf("Space Grotesk") >= 0 ? "Space Grotesk" : "Inter"
    readonly property string mono: families.indexOf("Space Mono") >= 0 ? "Space Mono"
        : families.indexOf("SpaceMono Nerd Font") >= 0 ? "SpaceMono Nerd Font"
        : families.indexOf("GeistMono Nerd Font") >= 0 ? "GeistMono Nerd Font" : "JetBrainsMono Nerd Font"
    // Ícones (Nerd Font) quando o mono escolhido não os tem.
    readonly property string icons: families.indexOf("GeistMono Nerd Font") >= 0 ? "GeistMono Nerd Font" : "JetBrainsMono Nerd Font"

    // Workspaces por letra grega, como na ilha do Ryoku: 1 = α, 2 = β...
    readonly property var greek: ["α", "β", "γ", "δ", "ε", "ζ", "η", "θ", "ι", "κ"]
    function wsName(id) {
        return id >= 1 && id <= greek.length ? greek[id - 1] : String(id);
    }

    // Rótulos: mono em maiúsculas espaçadas, como "FUNÇÃO ——— engenheiro".
    readonly property real labelSpacing: 1.6
}
