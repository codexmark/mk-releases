import QtQuick

// O alto de cada página: o número da área, o título em serifa e uma frase.
Column {
    id: cabecalho

    property string numero: "01"
    property string titulo: ""
    property string frase: ""

    spacing: 4

    Rotulo { text: "sistema  /  " + cabecalho.titulo + "     " + cabecalho.numero }
    Text {
        text: cabecalho.titulo
        color: Theme.text
        font.family: Theme.serif
        font.pixelSize: 30
        font.weight: Font.Light
    }
    Text {
        text: cabecalho.frase
        color: Theme.textDim
        font.family: Theme.sans
        font.pixelSize: 12
    }
}
