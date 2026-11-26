import QtQuick
import "Arquivos.js" as Arquivos

// Um item na visão de ícones: o ícone (ou a miniatura) e o nome embaixo. É
// leve de propósito: não tem área de mouse (o painel cuida do mouse inteiro)
// e lê o item do array do painel pelo índice.
Item {
    id: cartao

    required property int index
    property var dono: null
    readonly property var it: dono ? dono.itens[index] || null : null
    readonly property bool marcado: it !== null && dono.selecao.sel[it.caminho] === true
    readonly property bool noCursor: dono !== null && dono.ativo && dono.selecao.cursor === index
    readonly property bool alvo: dono !== null && dono.alvoDeSoltar === index
    readonly property string mini: it !== null && dono.miniVersao >= 0 ? dono.minis[it.caminho] || "" : ""
    readonly property int lado: dono ? dono.iconePx : 64

    onItChanged: {
        if (it !== null)
            dono.querMiniatura(it);
    }

    Rectangle {
        anchors.fill: parent
        anchors.margins: 3
        visible: cartao.marcado || cartao.noCursor || cartao.alvo
        color: cartao.marcado ? Theme.copperA(0.16) : Qt.rgba(0, 0, 0, 0)
        border.width: cartao.noCursor || cartao.alvo ? 1 : 0
        border.color: cartao.alvo ? Theme.copper : Theme.line
    }
    Icone {
        id: figura
        anchors.horizontalCenter: parent.horizontalCenter
        y: 8
        lado: cartao.lado
        nome: cartao.it ? cartao.it.icone : ""
        reserva: cartao.it ? cartao.it.icone_reserva : "text-x-generic"
        arquivo: cartao.mini
        opacity: cartao.it && cartao.it.oculto ? 0.55 : 1
    }
    // A marca de link, no canto do ícone.
    Text {
        visible: cartao.it !== null && cartao.it.tipo === "link"
        anchors.right: figura.right
        anchors.bottom: figura.bottom
        text: cartao.it && cartao.it.quebrado ? "⤫" : "↗"
        color: Theme.copper
        font.pixelSize: 13
    }
    Text {
        id: rotulo
        anchors.top: figura.bottom
        anchors.topMargin: 4
        x: 6
        width: parent.width - 12
        horizontalAlignment: Text.AlignHCenter
        wrapMode: Text.WrapAnywhere
        maximumLineCount: 2
        elide: Text.ElideRight
        text: cartao.it ? cartao.it.nome : ""
        color: cartao.marcado ? Theme.text : cartao.it && cartao.it.oculto ? Theme.textDim : Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Text {
        visible: cartao.dono !== null && cartao.dono.extra && cartao.it !== null
        anchors.top: rotulo.bottom
        x: 6
        width: parent.width - 12
        horizontalAlignment: Text.AlignHCenter
        elide: Text.ElideMiddle
        text: visible ? cartao.dono.celula(cartao.it, "tipo") : ""
        color: Theme.textFaint
        font.family: Theme.sans
        font.pixelSize: 10
    }
}
