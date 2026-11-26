import QtQuick

// A pergunta antes do que não tem volta (apagar de vez, esvaziar a lixeira) e
// as confirmações: nada sai sem um "sim" explícito. Enter confirma, Esc cancela.
Folha {
    id: pergunta

    property string frase: ""
    property var lista: []
    property string botao: "confirmar"
    signal confirmado()

    rotulo: "confirmação"

    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: pergunta.frase !== ""
        text: pergunta.frase
        color: Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Column {
        width: parent.width
        spacing: 2
        Repeater {
            model: pergunta.lista.slice(0, 6)
            Text {
                required property var modelData
                width: parent.width
                elide: Text.ElideMiddle
                text: modelData
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 11
            }
        }
        Text {
            visible: pergunta.lista.length > 6
            text: "… e mais " + (pergunta.lista.length - 6)
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 11
        }
    }
    Row {
        spacing: 8
        Botao {
            texto: pergunta.botao
            forte: true
            onAcionado: pergunta.confirmado()
        }
        Botao {
            texto: "cancelar"
            onAcionado: pergunta.fechado()
        }
    }
    Rotulo { text: "enter confirma  ·  esc cancela" }
}
