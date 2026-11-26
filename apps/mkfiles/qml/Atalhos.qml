import QtQuick
import "Menu.js" as Menu

// A folha de atalhos que o F1 mostra. Esc, F1 ou um clique fora fecham.
Folha {
    id: atalhos

    titulo: "Atalhos"
    largura: 640

    Flickable {
        id: rolo
        width: parent.width
        height: Math.min(grade.implicitHeight, atalhos.height - 150)
        contentHeight: grade.implicitHeight
        boundsBehavior: Flickable.StopAtBounds
        clip: true

        // Por cima da folha (não rola com ela).
        Rolagem {
            parent: rolo
            z: 2
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            alvo: rolo
        }

        Grid {
            id: grade
            width: rolo.width - 12
            columns: width > 520 ? 2 : 1
            columnSpacing: 18
            rowSpacing: 5

            Repeater {
                model: Menu.ATALHOS

                Row {
                    id: atalho
                    required property var modelData
                    width: (grade.width - (grade.columns - 1) * grade.columnSpacing) / grade.columns
                    spacing: 10

                    Text {
                        width: 130
                        elide: Text.ElideRight
                        text: atalho.modelData[0]
                        color: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                    Text {
                        width: atalho.width - 140
                        text: atalho.modelData[1]
                        color: Theme.textSoft
                        font.family: Theme.sans
                        font.pixelSize: 12
                        elide: Text.ElideRight
                    }
                }
            }
        }
    }
    Rotulo { text: "esc fecha" }
}
