import QtQuick
import "Escrita.js" as Escrita

// A folha de atalhos (F1 ou Ctrl+/). Qualquer tecla ou clique fecha.
Item {
    id: ajuda

    signal fechar()

    function foca() {
        teclas.forceActiveFocus();
    }

    anchors.fill: parent
    onVisibleChanged: if (visible) foca()

    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: 0.72
    }

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.AllButtons
        onClicked: ajuda.fechar()
        onWheel: wheel => wheel.accepted = true
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(parent.width - 48, 720)
        height: Math.min(parent.height - 48, grade.implicitHeight + 72)
        color: Theme.raised
        border.width: 1
        border.color: Theme.line
        clip: true

        Item {
            id: teclas
            Keys.onPressed: event => {
                ajuda.fechar();
                event.accepted = true;
            }
        }

        Rotulo {
            x: 24
            y: 22
            text: "Atalhos"
        }

        Flow {
            id: grade
            x: 24
            y: 48
            width: parent.width - 48
            spacing: 24

            Repeater {
                model: Escrita.atalhos()

                Column {
                    id: grupo
                    required property var modelData
                    width: (grade.width - 24) / 2
                    spacing: 5

                    Text {
                        text: grupo.modelData.grupo
                        color: Theme.copper
                        font.family: Theme.serif
                        font.pixelSize: 14
                        font.italic: true
                    }

                    Repeater {
                        model: grupo.modelData.itens

                        Item {
                            id: atalho
                            required property var modelData
                            width: grupo.width
                            height: 18

                            Text {
                                width: 150
                                text: atalho.modelData[0]
                                color: Theme.textSoft
                                font.family: Theme.mono
                                font.pixelSize: 11
                                elide: Text.ElideRight
                            }

                            Text {
                                x: 158
                                width: parent.width - 158
                                text: atalho.modelData[1]
                                color: Theme.textDim
                                font.family: Theme.sans
                                font.pixelSize: 12
                                elide: Text.ElideRight
                            }
                        }
                    }
                }
            }
        }
    }
}
