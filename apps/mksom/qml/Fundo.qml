import QtQuick

// O fundo de uma folha (formulário, confirmação, atalhos): apaga a janela e
// segura os cliques e a roda. Um clique fora do cartão fecha.
Item {
    id: fundo

    signal fechado()

    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: 0.88
    }
    MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        acceptedButtons: Qt.AllButtons
        onClicked: fundo.fechado()
        onWheel: wheel => wheel.accepted = true
    }
}
