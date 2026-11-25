import QtQuick
import QtQuick.Layouts
import "Porto.js" as Porto

// A área "Chaves": as chaves públicas de ~/.ssh, com o tipo e a impressão
// digital, e o "copiar a pública". Só as públicas: a privada nunca é lida.
// "nova chave" abre o ssh-keygen num terminal; como ninguém avisa quando ele
// acaba, o "atualizar" relê a pasta.
Item {
    id: pagina

    property var chaves: []
    signal copiar(var chave)
    signal novaChave()
    signal atualizar()

    ColumnLayout {
        anchors.fill: parent
        spacing: 12

        Cabecalho {
            Layout.fillWidth: true
            titulo: "Chaves"
            frase: "As chaves públicas de ~/.ssh. A parte privada nunca é lida pelo MkPorto."
        }

        Row {
            spacing: 8
            Botao {
                texto: "nova chave"
                forte: true
                onAcionado: pagina.novaChave()
            }
            Botao {
                texto: "atualizar"
                onAcionado: pagina.atualizar()
            }
        }

        ListView {
            id: vista
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            spacing: 8
            boundsBehavior: Flickable.StopAtBounds
            model: pagina.chaves

            delegate: Rectangle {
                id: item
                required property var modelData
                width: vista.width
                height: 76
                color: Theme.surface
                border.width: 1
                border.color: Theme.hairline

                Column {
                    anchors.left: parent.left
                    anchors.right: botao.left
                    anchors.margins: 14
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: 4

                    Text {
                        width: parent.width
                        elide: Text.ElideMiddle
                        text: item.modelData.nome
                        color: Theme.text
                        font.family: Theme.sans
                        font.pixelSize: 13
                    }
                    Text {
                        width: parent.width
                        elide: Text.ElideRight
                        text: Porto.tipoEBits(item.modelData) + (item.modelData.comentario ? "  ·  " + item.modelData.comentario : "")
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                    Text {
                        width: parent.width
                        elide: Text.ElideMiddle
                        text: item.modelData.impressao || "sem impressão digital"
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }
                }
                Botao {
                    id: botao
                    anchors.right: parent.right
                    anchors.rightMargin: 14
                    anchors.verticalCenter: parent.verticalCenter
                    texto: "copiar a pública"
                    onAcionado: pagina.copiar(item.modelData)
                }
            }

            Text {
                anchors.centerIn: parent
                width: parent.width - 40
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.WordWrap
                visible: pagina.chaves.length === 0
                text: "Nenhuma chave pública em ~/.ssh.\nCrie a primeira em \"nova chave\"."
                color: Theme.textFaint
                font.family: Theme.serif
                font.pixelSize: 15
                font.italic: true
            }
        }
    }
}
