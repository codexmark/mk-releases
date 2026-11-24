import QtQuick

// Um cartão da página Desempenho: rótulo, o valor de agora em serifa, uma
// linha de detalhe e o gráfico do último minuto.
Rectangle {
    id: cartao

    property string rotulo: ""
    property string valor: "—"
    property string detalhe: ""
    property alias serie: grafico.serie
    property alias serie2: grafico.serie2
    property alias teto: grafico.teto
    default property alias extra: resto.data

    color: Qt.rgba(0, 0, 0, 0)
    border.width: 1
    border.color: Theme.hairline

    Rotulo { x: 16; y: 14; text: cartao.rotulo }
    Text {
        x: 16
        y: 30
        text: cartao.valor
        color: Theme.text
        font.family: Theme.serif
        font.pixelSize: 28
        font.weight: Font.Light
    }
    Text {
        anchors.right: parent.right
        anchors.rightMargin: 16
        y: 40
        text: cartao.detalhe
        color: Theme.textDim
        font.family: Theme.mono
        font.pixelSize: 10
    }
    Grafico {
        id: grafico
        x: 16
        y: 74
        width: parent.width - 32
        height: Math.max(40, parent.height - 74 - resto.height - 16)
        pontos: Coletor.pontos
    }
    Column {
        id: resto
        x: 16
        width: parent.width - 32
        anchors.bottom: parent.bottom
        anchors.bottomMargin: 12
        spacing: 5
    }
}
