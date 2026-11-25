import QtQuick

// A pergunta antes de remover um host: nada sai sem um "sim" explícito.
Item {
    id: confirma

    property string apelido: ""
    property string erro: ""
    property bool ocupado: false
    signal confirmado()
    signal fechado()

    Fundo {
        anchors.fill: parent
        onFechado: confirma.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(440, parent.width - 32)
        height: corpo.implicitHeight + 44
        color: Theme.surface
        border.width: 1
        border.color: Theme.line

        MouseArea { anchors.fill: parent }

        Column {
            id: corpo
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 22
            spacing: 10

            Rotulo { text: "confirmação" }
            Text {
                width: parent.width
                wrapMode: Text.WrapAnywhere
                text: "Remover " + confirma.apelido + "?"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 22
                font.weight: Font.Light
            }
            Text {
                width: parent.width
                wrapMode: Text.WordWrap
                text: "O host sai de ~/.ssh/config.d/mkporto.conf. Nada muda na máquina de lá, nem no resto do seu ~/.ssh/config."
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Text {
                width: parent.width
                wrapMode: Text.WordWrap
                visible: confirma.erro !== ""
                text: confirma.erro
                color: Theme.copper
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Row {
                spacing: 8
                Botao {
                    texto: confirma.ocupado ? "removendo…" : "remover"
                    forte: true
                    ligado: !confirma.ocupado
                    onAcionado: confirma.confirmado()
                }
                Botao {
                    texto: "cancelar"
                    onAcionado: confirma.fechado()
                }
            }
            Rotulo { text: "enter remove  ·  esc cancela" }
        }
    }
}
