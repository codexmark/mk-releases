import QtQuick
import "Menu.js" as Menu

// O menu de contexto: uma lista de entradas (Menu.js) que abre onde o mouse
// clicou (ou junto do item, pelo teclado). Setas e Enter escolhem; Esc ou um
// clique fora fecham. Quem trata a escolha é o App.
Item {
    id: menu

    property bool aberto: false
    property var entradas: []
    property int atual: -1
    property real px: 0
    property real py: 0
    signal escolhido(string id)

    function abrir(x, y, lista) {
        entradas = lista;
        px = x;
        py = y;
        atual = -1;
        aberto = lista.length > 0;
    }
    function fechar() { aberto = false; }
    function andar(delta) { atual = Menu.proxima(entradas, atual, delta); }
    function confirmar() {
        if (atual >= 0 && atual < entradas.length && !entradas[atual].sep && entradas[atual].ligado)
            escolher(entradas[atual].id);
    }
    function escolher(id) {
        aberto = false;
        escolhido(id);
    }

    visible: aberto

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.AllButtons
        hoverEnabled: true
        onPressed: menu.fechar()
        onWheel: wheel => wheel.accepted = true
    }

    Rectangle {
        id: caixa
        x: Math.max(4, Math.min(menu.px, menu.width - width - 4))
        y: Math.max(4, Math.min(menu.py, menu.height - height - 4))
        width: 280
        height: Math.min(menu.height - 8, linhas.implicitHeight + 10)
        color: Theme.raised
        border.width: 1
        border.color: Theme.line
        clip: true

        MouseArea { anchors.fill: parent; acceptedButtons: Qt.AllButtons }

        Flickable {
            anchors.fill: parent
            anchors.margins: 5
            contentHeight: linhas.implicitHeight
            boundsBehavior: Flickable.StopAtBounds

            Column {
                id: linhas
                width: parent.width

                Repeater {
                    model: menu.entradas

                    Item {
                        id: entrada
                        required property var modelData
                        required property int index
                        readonly property bool sep: modelData.sep === true
                        width: linhas.width
                        height: sep ? 9 : 26

                        Rectangle {
                            visible: entrada.sep
                            anchors.verticalCenter: parent.verticalCenter
                            width: parent.width
                            height: 1
                            color: Theme.hairline
                        }
                        Rectangle {
                            anchors.fill: parent
                            visible: !entrada.sep && entrada.index === menu.atual
                            color: Theme.wash
                            border.width: 1
                            border.color: Theme.copper
                        }
                        Text {
                            visible: !entrada.sep
                            anchors.left: parent.left
                            anchors.leftMargin: 10
                            anchors.right: tecla.left
                            anchors.rightMargin: 8
                            anchors.verticalCenter: parent.verticalCenter
                            elide: Text.ElideRight
                            text: entrada.sep ? "" : entrada.modelData.texto
                            color: entrada.modelData.ligado ? Theme.text : Theme.textFaint
                            font.family: Theme.sans
                            font.pixelSize: 12
                        }
                        Text {
                            id: tecla
                            visible: !entrada.sep
                            anchors.right: parent.right
                            anchors.rightMargin: 10
                            anchors.verticalCenter: parent.verticalCenter
                            text: entrada.sep ? "" : entrada.modelData.atalho
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                        }
                        MouseArea {
                            anchors.fill: parent
                            enabled: !entrada.sep && entrada.modelData.ligado === true
                            hoverEnabled: true
                            onEntered: menu.atual = entrada.index
                            onClicked: menu.escolher(entrada.modelData.id)
                        }
                    }
                }
            }
        }
    }
}
