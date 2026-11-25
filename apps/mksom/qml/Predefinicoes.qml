import QtQuick
import "Som.js" as Som

// O seletor de predefinições: as de fábrica e as da pessoa. Clicar (ou ↑/↓ e
// Enter) aplica; as da pessoa têm renomear e apagar.
Item {
    id: seletor

    property var lista: []
    property string atual: ""
    property int marcada: 0
    signal escolhida(string nome)
    signal renomearPedido(string nome)
    signal apagarPedido(string nome)
    signal fechado()

    onListaChanged: marcada = Math.max(0, Math.min(marcada, lista.length - 1))

    // Ao abrir, a marca começa na que está aplicada.
    function abrir() {
        marcada = 0;
        for (let i = 0; i < lista.length; i++)
            if (lista[i].nome === atual)
                marcada = i;
    }

    function andar(passo) {
        if (lista.length > 0)
            marcada = Math.max(0, Math.min(lista.length - 1, marcada + passo));
    }

    function confirmar() {
        if (marcada >= 0 && marcada < lista.length)
            seletor.escolhida(lista[marcada].nome);
    }

    Fundo {
        anchors.fill: parent
        onFechado: seletor.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(parent.width - 32, 380)
        height: Math.min(parent.height - 20, folha.implicitHeight + 40)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        MouseArea { anchors.fill: parent }

        Column {
            id: folha
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 20
            spacing: 8

            Text {
                text: "Predefinições"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }

            Flickable {
                width: parent.width
                height: Math.min(itens.implicitHeight, seletor.height - 130)
                contentHeight: itens.implicitHeight
                clip: true
                boundsBehavior: Flickable.StopAtBounds

                Column {
                    id: itens
                    width: parent.width

                    Repeater {
                        model: seletor.lista

                        Rectangle {
                            id: item
                            required property int index
                            required property var modelData
                            readonly property bool marcado: index === seletor.marcada
                            width: itens.width
                            height: 30
                            color: marcado ? Theme.wash : Qt.rgba(0, 0, 0, 0)
                            border.width: marcado ? 1 : 0
                            border.color: Theme.line

                            MouseArea {
                                anchors.fill: parent
                                cursorShape: Qt.PointingHandCursor
                                onClicked: seletor.escolhida(item.modelData.nome)
                            }
                            Text {
                                anchors.left: parent.left
                                anchors.leftMargin: 10
                                anchors.right: acoes.left
                                anchors.rightMargin: 8
                                anchors.verticalCenter: parent.verticalCenter
                                elide: Text.ElideRight
                                text: item.modelData.nome
                                color: item.modelData.nome === seletor.atual ? Theme.copper : Theme.textSoft
                                font.family: Theme.sans
                                font.pixelSize: 13
                            }
                            Row {
                                id: acoes
                                anchors.right: parent.right
                                anchors.rightMargin: 6
                                anchors.verticalCenter: parent.verticalCenter
                                spacing: 6

                                Rotulo {
                                    anchors.verticalCenter: parent.verticalCenter
                                    visible: item.modelData.fabrica === true
                                    text: "de fábrica"
                                }
                                Botao {
                                    visible: item.modelData.fabrica !== true
                                    implicitHeight: 22
                                    texto: "renomear"
                                    onAcionado: seletor.renomearPedido(item.modelData.nome)
                                }
                                Botao {
                                    visible: item.modelData.fabrica !== true
                                    implicitHeight: 22
                                    texto: "apagar"
                                    onAcionado: seletor.apagarPedido(item.modelData.nome)
                                }
                            }
                        }
                    }
                }
            }
            Rotulo { text: "↑ ↓ escolhem  ·  enter aplica  ·  esc fecha" }
        }
    }
}
