pragma Singleton
import QtQuick
import Quickshell

// Os tokens do Maré (o shell do Mk): preto quente, texto em osso e um só
// destaque, o cobre. Nenhuma outra cor entra na janela.
Singleton {
    readonly property color ground: "#0b0b0b"
    readonly property color surface: "#141211"
    readonly property color raised: "#1b1917"
    readonly property color hairline: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.13)
    readonly property color line: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.26)
    readonly property color wash: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.05)
    readonly property color text: "#cdc4ba"
    readonly property color textSoft: "#b0a9a0"
    readonly property color textDim: "#958f87"
    readonly property color textFaint: "#716b69"
    readonly property color copper: "#ce5e32"

    function copperA(a) { return Qt.rgba(206 / 255, 94 / 255, 50 / 255, a); }
    function boneA(a) { return Qt.rgba(205 / 255, 196 / 255, 186 / 255, a); }

    readonly property var families: Qt.fontFamilies()
    readonly property string serif: families.indexOf("Fraunces") >= 0 ? "Fraunces" : "Noto Serif"
    readonly property string sans: families.indexOf("Space Grotesk") >= 0 ? "Space Grotesk" : "Inter"
    readonly property string mono: families.indexOf("Space Mono") >= 0 ? "Space Mono" : "JetBrainsMono Nerd Font"
    readonly property real labelSpacing: 1.6
}
