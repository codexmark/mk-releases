import QtQuick

// O alto de cada janela: de quem é, o título em serifa e uma frase.
Column {
    id: cabecalho

    property string titulo: ""
    property string frase: ""

    spacing: 4

    Rotulo { text: "mk  /  " + cabecalho.titulo }
    Text {
        text: cabecalho.titulo
        color: Theme.text
        font.family: Theme.serif
        font.pixelSize: 28
        font.weight: Font.Light
    }
    Text {
        text: cabecalho.frase
        color: Theme.textDim
        font.family: Theme.sans
        font.pixelSize: 12
    }
}
