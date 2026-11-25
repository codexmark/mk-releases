import QtQuick
import "Som.js" as Som

// O medidor de nível: a barra é o nível de agora; o risco, o pico que cai
// devagar. De −60 dBFS (vazio) a 0 (cheio).
Rectangle {
    id: medidor

    property real nivel: Som.PISO
    property real pico: Som.PISO
    property bool ligado: true

    implicitHeight: 12
    color: Theme.wash
    border.width: 1
    border.color: Theme.hairline

    Rectangle {
        x: 1
        y: 1
        height: parent.height - 2
        width: (parent.width - 2) * Som.fracao(medidor.nivel)
        color: medidor.ligado ? Theme.copperA(0.7) : Theme.boneA(0.2)
    }
    // Onde começa a saturar.
    Rectangle {
        x: 1 + (parent.width - 2) * Som.fracao(Som.SATURA_EM)
        width: 1
        height: parent.height
        color: Theme.line
    }
    Rectangle {
        visible: Som.fracao(medidor.pico) > 0
        x: Math.min(parent.width - 3, 1 + (parent.width - 2) * Som.fracao(medidor.pico))
        y: 1
        width: 2
        height: parent.height - 2
        color: Theme.text
    }
}
