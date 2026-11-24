import QtQuick

// A marca de uma linha da tabela: um quadradinho que enche de cobre. Serve
// para escolher vários pacotes para uma ação só.
Rectangle {
    id: marca

    property bool marcado: false
    property bool ligado: true
    signal trocado()

    implicitWidth: 14
    implicitHeight: 14
    color: marcado ? Theme.copperA(0.25) : Qt.rgba(0, 0, 0, 0)
    border.width: 1
    border.color: marcado ? Theme.copper : toque.containsMouse ? Theme.textDim : Theme.line
    opacity: ligado ? 1 : 0.3

    Rectangle {
        anchors.centerIn: parent
        visible: marca.marcado
        width: 6
        height: 6
        color: Theme.copper
    }

    MouseArea {
        id: toque
        anchors.fill: parent
        anchors.margins: -8
        hoverEnabled: true
        enabled: marca.ligado
        cursorShape: Qt.PointingHandCursor
        onClicked: marca.trocado()
    }
}
