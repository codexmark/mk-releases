import QtQuick
import "Corte.js" as Corte

// A folha de atalhos que o "?" mostra. Qualquer tecla (ou um clique) fecha.
Item {
    id: atalhos

    signal fechado()

    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: 0.9
    }
    MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        onClicked: atalhos.fechado()
        onWheel: wheel => wheel.accepted = true
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(parent.width - 40, 700)
        height: Math.min(parent.height - 20, folha.implicitHeight + 44)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        Column {
            id: folha
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 22
            spacing: 12

            Text {
                text: "Atalhos"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }
            Grid {
                width: parent.width
                columns: width >= 560 ? 2 : 1
                columnSpacing: 24
                rowSpacing: 5

                Repeater {
                    model: Corte.ATALHOS

                    Row {
                        id: atalho
                        required property var modelData
                        spacing: 10

                        Text {
                            width: 150
                            text: atalho.modelData[0]
                            color: Theme.copper
                            font.family: Theme.mono
                            font.pixelSize: 11
                        }
                        Text {
                            width: (folha.width - 24) / (folha.width >= 560 ? 2 : 1) - 160
                            text: atalho.modelData[1]
                            color: Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 12
                            elide: Text.ElideRight
                        }
                    }
                }
            }
            Rotulo { text: "qualquer tecla fecha" }
        }
    }
}
