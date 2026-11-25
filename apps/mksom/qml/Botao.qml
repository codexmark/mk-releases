import QtQuick

// Botão de uma linha, na linguagem do Maré: contorno fino, cobre quando é o
// que importa (forte) ou quando o ponteiro passa.
Rectangle {
    id: botao

    property string texto: ""
    property bool forte: false
    property bool ligado: true
    signal acionado()

    implicitWidth: rotulo.implicitWidth + 26
    implicitHeight: 28
    color: area.containsMouse && ligado ? Theme.wash : Qt.rgba(0, 0, 0, 0)
    border.width: 1
    border.color: !ligado ? Theme.hairline : forte || area.containsMouse ? Theme.copper : Theme.line
    opacity: ligado ? 1 : 0.45

    Text {
        id: rotulo
        anchors.centerIn: parent
        text: botao.texto
        color: botao.forte && botao.ligado ? Theme.copper : Theme.textSoft
        font.family: Theme.mono
        font.pixelSize: 11
        font.letterSpacing: 0.6
    }

    MouseArea {
        id: area
        anchors.fill: parent
        hoverEnabled: true
        enabled: botao.ligado
        cursorShape: Qt.PointingHandCursor
        onClicked: botao.acionado()
    }
}
