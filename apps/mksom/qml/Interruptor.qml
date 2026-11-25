import QtQuick

// A chave grande de ligar e desligar o equalizador. Recebe o foco do Tab;
// Enter troca (o Espaço é atalho da janela inteira).
Rectangle {
    id: interruptor

    property bool ligado: false
    property bool mudavel: true
    signal trocado()

    implicitWidth: 56
    implicitHeight: 28
    radius: 14
    color: ligado ? Theme.copperA(0.25) : Theme.boneA(0.06)
    border.width: activeFocus ? 2 : 1
    border.color: ligado || activeFocus ? Theme.copper : Theme.line
    opacity: mudavel ? 1 : 0.4
    activeFocusOnTab: true

    Keys.onReturnPressed: if (mudavel) trocado()
    Keys.onEnterPressed: if (mudavel) trocado()

    Rectangle {
        x: interruptor.ligado ? parent.width - width - 4 : 4
        anchors.verticalCenter: parent.verticalCenter
        width: 20
        height: 20
        radius: 10
        color: interruptor.ligado ? Theme.copper : Theme.textDim
        Behavior on x { NumberAnimation { duration: 120 } }
    }

    MouseArea {
        anchors.fill: parent
        anchors.margins: -4
        enabled: interruptor.mudavel
        cursorShape: Qt.PointingHandCursor
        onClicked: {
            interruptor.forceActiveFocus();
            interruptor.trocado();
        }
    }
}
