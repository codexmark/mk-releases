import QtQuick

// Uma pergunta antes de algo que não tem volta (forçar o desligamento,
// restaurar ou apagar um snapshot): nada acontece sem um "sim" explícito.
Item {
    id: pergunta

    property string rotulo: "confirmação"
    property string titulo: ""
    property string texto: ""
    property string acao: "confirmar"
    property string erro: ""
    property bool ocupado: false
    signal confirmado()
    signal fechado()

    Fundo {
        anchors.fill: parent
        onFechado: pergunta.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(460, parent.width - 32)
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

            Rotulo { text: pergunta.rotulo }
            Text {
                width: parent.width
                wrapMode: Text.WrapAnywhere
                text: pergunta.titulo
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 22
                font.weight: Font.Light
            }
            Text {
                width: parent.width
                wrapMode: Text.WordWrap
                text: pergunta.texto
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Text {
                width: parent.width
                wrapMode: Text.WordWrap
                visible: pergunta.erro !== ""
                text: pergunta.erro
                color: Theme.copper
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Row {
                spacing: 8
                Botao {
                    texto: pergunta.ocupado ? "um instante…" : pergunta.acao
                    forte: true
                    ligado: !pergunta.ocupado
                    onAcionado: pergunta.confirmado()
                }
                Botao {
                    texto: "cancelar"
                    onAcionado: pergunta.fechado()
                }
            }
            Rotulo { text: "esc cancela" }
        }
    }
}
