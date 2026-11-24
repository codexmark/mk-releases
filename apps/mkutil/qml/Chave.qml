import QtQuick

// Liga e desliga, como o do Maré: a bolinha vai para o cobre quando ligado.
Rectangle {
    id: chave

    property bool ligado: false
    property bool mudavel: true
    signal trocado()

    implicitWidth: 34
    implicitHeight: 18
    radius: 9
    color: ligado ? Theme.copperA(0.25) : Theme.boneA(0.06)
    border.width: 1
    border.color: ligado ? Theme.copper : Theme.line
    opacity: mudavel ? 1 : 0.4

    Rectangle {
        x: chave.ligado ? parent.width - width - 3 : 3
        anchors.verticalCenter: parent.verticalCenter
        width: 12
        height: 12
        radius: 6
        color: chave.ligado ? Theme.copper : Theme.textDim
        Behavior on x { NumberAnimation { duration: 120 } }
    }

    MouseArea {
        anchors.fill: parent
        anchors.margins: -6
        enabled: chave.mudavel
        cursorShape: Qt.PointingHandCursor
        onClicked: chave.trocado()
    }
}
