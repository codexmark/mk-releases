import QtQuick

// A moldura de toda folha (pergunta, propriedades, atalhos): apaga a janela
// atrás, põe um cartão no meio com o rótulo e o título, e o conteúdo dentro.
// Um clique fora fecha.
Item {
    id: folha

    property string rotulo: ""
    property string titulo: ""
    property int largura: 440
    default property alias conteudo: corpo.data
    signal fechado()

    Fundo {
        anchors.fill: parent
        onFechado: folha.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(folha.largura, parent.width - 32)
        height: Math.min(parent.height - 20, corpo.implicitHeight + 44)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        MouseArea {
            anchors.fill: parent
            acceptedButtons: Qt.AllButtons
            onWheel: wheel => wheel.accepted = true
        }

        Column {
            id: corpo
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 22
            spacing: 10

            Rotulo {
                visible: folha.rotulo !== ""
                text: folha.rotulo
            }
            Text {
                visible: folha.titulo !== ""
                width: parent.width
                wrapMode: Text.WrapAnywhere
                maximumLineCount: 3
                elide: Text.ElideRight
                text: folha.titulo
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 22
                font.weight: Font.Light
            }
        }
    }
}
