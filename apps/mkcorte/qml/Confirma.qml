import QtQuick
import "Corte.js" as Corte

// Uma pergunta por cima da janela, que se responde pelo teclado: Tab e as
// setas andam pelas respostas, Enter escolhe a marcada, Esc é "cancelar". O
// Editor entrega as teclas (tecla()); o ponteiro também serve.
Item {
    id: confirma

    property string titulo: ""
    property string texto: ""
    // [{ v, texto, forte }]. A resposta "cancelar" é a do Esc.
    property var opcoes: []
    property int marcada: 0
    signal escolhido(string v)

    // acao: o que o Corte.acaoDoDialogo devolveu.
    function tecla(acao) {
        if (acao === "proximo" || acao === "baixo")
            confirma.marcada = Corte.girar(confirma.marcada, confirma.opcoes.length, 1);
        else if (acao === "anterior" || acao === "cima")
            confirma.marcada = Corte.girar(confirma.marcada, confirma.opcoes.length, -1);
        else if (acao === "confirmar" && confirma.opcoes.length > 0)
            confirma.escolhido(confirma.opcoes[confirma.marcada].v);
        else if (acao === "cancelar")
            confirma.escolhido("cancelar");
    }

    onVisibleChanged: marcada = 0

    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: 0.86
    }
    // O que está embaixo não recebe clique nem roda.
    MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        onWheel: wheel => wheel.accepted = true
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(parent.width - 40, 460)
        height: conteudo.implicitHeight + 44
        color: Theme.surface
        border.width: 1
        border.color: Theme.line

        Column {
            id: conteudo
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 22
            spacing: 12

            Text {
                width: parent.width
                text: confirma.titulo
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
                wrapMode: Text.WordWrap
            }
            Text {
                width: parent.width
                visible: text !== ""
                text: confirma.texto
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 13
                wrapMode: Text.WordWrap
            }
            Row {
                spacing: 8
                topPadding: 6

                Repeater {
                    model: confirma.opcoes

                    Column {
                        id: resposta
                        required property var modelData
                        required property int index
                        spacing: 3

                        Botao {
                            texto: resposta.modelData.texto
                            forte: resposta.modelData.forte === true
                            onAcionado: confirma.escolhido(resposta.modelData.v)
                        }
                        // A resposta que o Enter escolhe.
                        Rectangle {
                            width: parent.width
                            height: 2
                            color: confirma.marcada === resposta.index ? Theme.copper : Qt.rgba(0, 0, 0, 0)
                        }
                    }
                }
            }
            Rotulo { text: "tab ou setas  ·  enter escolhe  ·  esc cancela" }
        }
    }
}
