import QtQuick
import "Som.js" as Som

// A folha de atalhos que o F1 mostra. Esc, F1 ou um clique fecham.
Item {
    id: atalhos

    signal fechado()

    Fundo {
        anchors.fill: parent
        onFechado: atalhos.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(parent.width - 32, 460)
        height: Math.min(parent.height - 20, folha.implicitHeight + 44)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        MouseArea {
            anchors.fill: parent
            onClicked: atalhos.fechado()
        }

        Column {
            id: folha
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 22
            spacing: 10

            Text {
                text: "Atalhos"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }
            Column {
                width: parent.width
                spacing: 5

                Repeater {
                    model: Som.ATALHOS

                    Row {
                        id: atalho
                        required property var modelData
                        spacing: 10

                        Text {
                            width: 120
                            text: atalho.modelData[0]
                            color: Theme.copper
                            font.family: Theme.mono
                            font.pixelSize: 11
                        }
                        Text {
                            width: folha.width - 130
                            text: atalho.modelData[1]
                            color: Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 12
                            elide: Text.ElideRight
                        }
                    }
                }
            }
            Rotulo { text: "esc fecha" }
        }
    }
}
