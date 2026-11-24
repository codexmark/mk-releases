import QtQuick

// Uma tecla da calculadora, na linguagem do Maré: contorno fino, cobre no "="
// e em quem está sob o ponteiro, e um lampejo quando é acionada (também pelo
// teclado físico: `ativa`).
Rectangle {
    id: tecla

    property string texto: ""
    // "numero", "operador" ou "igual".
    property string tipo: "numero"
    property bool ativa: false
    signal acionado()

    implicitWidth: 56
    implicitHeight: 44
    color: area.pressed || ativa ? Theme.copperA(0.16) : area.containsMouse ? Theme.wash : tipo === "numero" ? Theme.surface : Qt.rgba(0, 0, 0, 0)
    border.width: 1
    border.color: tipo === "igual" || area.containsMouse || ativa ? Theme.copper : Theme.hairline

    Text {
        anchors.centerIn: parent
        text: tecla.texto
        color: tecla.tipo === "igual" ? Theme.copper : tecla.tipo === "operador" ? Theme.textSoft : Theme.text
        font.family: Theme.mono
        font.pixelSize: Math.max(13, Math.min(20, tecla.height * 0.42))
    }

    MouseArea {
        id: area
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: tecla.acionado()
    }
}
