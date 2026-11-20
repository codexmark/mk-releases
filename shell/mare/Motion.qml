pragma Singleton
import QtQuick
import Quickshell

// Tokens de movimento. Os painéis seguem o comportamento da ilha do Ryoku
// (orpheus): abrem rápido e assentam, fecham acelerando, e deslizam até o item
// clicado com um leve passar do ponto. Os detalhes usam a curva do perfil.
Singleton {
    // Velocidade escolhida nos Ajustes: 0.7 rápido, 1 normal, 1.4 calmo, 0 sem.
    readonly property real speed: Settings.get("motion.speed")

    readonly property int fast: Math.round(120 * speed)
    readonly property int base: Math.round(260 * speed)
    readonly property int slow: Math.round(420 * speed)
    readonly property int contentDelay: Math.round(90 * speed)
    readonly property int content: Math.round(220 * speed)

    // Curva do perfil codexmark: cubic-bezier(.215, .61, .355, 1).
    readonly property var curve: [0.215, 0.61, 0.355, 1, 1, 1]

    // Painel abrindo / fechando.
    readonly property int enter: Math.round(400 * speed)
    readonly property var enterCurve: [0.05, 0.7, 0.1, 1, 1, 1]
    readonly property int exit: Math.round(300 * speed)
    readonly property var exitCurve: [0.3, 0, 0.8, 0.15, 1, 1]

    // Painel indo de um item da ilha a outro (passa um pouco do ponto e volta).
    readonly property int spatial: Math.round(500 * speed)
    readonly property var spatialCurve: [0.38, 1.21, 0.22, 1, 1, 1]
}
