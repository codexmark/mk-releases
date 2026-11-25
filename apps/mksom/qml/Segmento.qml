import QtQuick

// Escolha de uma entre poucas: [{ v, texto }]. A escolhida fica em cobre.
Row {
    id: segmento

    property var opcoes: []
    property string valor: ""
    signal escolhido(string v)

    spacing: -1

    Repeater {
        model: segmento.opcoes

        Rectangle {
            id: op
            required property var modelData
            readonly property bool atual: modelData.v === segmento.valor
            width: t.implicitWidth + 22
            height: 26
            z: atual ? 1 : 0
            color: atual ? Theme.wash : Qt.rgba(0, 0, 0, 0)
            border.width: 1
            border.color: atual ? Theme.copper : Theme.line

            Text {
                id: t
                anchors.centerIn: parent
                text: op.modelData.texto
                color: op.atual ? Theme.copper : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 10
                font.capitalization: Font.AllUppercase
                font.letterSpacing: 1.2
            }

            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                onClicked: segmento.escolhido(op.modelData.v)
            }
        }
    }
}
