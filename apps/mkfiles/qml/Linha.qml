import QtQuick

// Um item nas visões de lista e compacta: o ícone pequeno, o nome e, na
// lista, as colunas (tamanho, tipo, modificado, permissões). Leve como o
// Cartao: sem área de mouse, lendo o item pelo índice.
Item {
    id: linha

    required property int index
    property var dono: null
    property bool compacta: false
    readonly property var it: dono ? dono.itens[index] || null : null
    readonly property bool marcado: it !== null && dono.selecao.sel[it.caminho] === true
    readonly property bool noCursor: dono !== null && dono.ativo && dono.selecao.cursor === index
    readonly property bool alvo: dono !== null && dono.alvoDeSoltar === index

    Rectangle {
        anchors.fill: parent
        anchors.rightMargin: linha.compacta ? 6 : 0
        visible: linha.marcado || linha.noCursor || linha.alvo
        color: linha.marcado ? Theme.copperA(0.16) : Qt.rgba(0, 0, 0, 0)
        border.width: linha.noCursor || linha.alvo ? 1 : 0
        border.color: linha.alvo ? Theme.copper : Theme.line
    }
    Icone {
        id: figura
        x: 8
        anchors.verticalCenter: parent.verticalCenter
        lado: 16
        nome: linha.it ? linha.it.icone : ""
        reserva: linha.it ? linha.it.icone_reserva : "text-x-generic"
        opacity: linha.it && linha.it.oculto ? 0.55 : 1
    }
    Text {
        x: 32
        width: (linha.compacta ? linha.width - 8 : (linha.dono ? linha.dono.larguraDoNome : 200)) - 38
        anchors.verticalCenter: parent.verticalCenter
        elide: Text.ElideMiddle
        text: linha.it ? linha.it.nome + (linha.it.tipo === "link" ? (linha.it.quebrado ? "  ⤫" : "  ↗") : "") : ""
        color: linha.marcado ? Theme.text : linha.it && linha.it.oculto ? Theme.textDim : Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Row {
        visible: !linha.compacta
        x: linha.dono ? linha.dono.larguraDoNome : 200
        height: parent.height

        Repeater {
            model: linha.compacta || !linha.dono ? [] : linha.dono.cols

            Text {
                required property var modelData
                width: modelData.largura
                height: linha.height
                leftPadding: 8
                rightPadding: 8
                verticalAlignment: Text.AlignVCenter
                horizontalAlignment: modelData.id === "tamanho" ? Text.AlignRight : Text.AlignLeft
                elide: modelData.id === "tipo" ? Text.ElideMiddle : Text.ElideRight
                text: linha.it ? linha.dono.celula(linha.it, modelData.id) : ""
                color: Theme.textDim
                font.family: modelData.id === "tipo" ? Theme.sans : Theme.mono
                font.pixelSize: 11
            }
        }
    }
}
