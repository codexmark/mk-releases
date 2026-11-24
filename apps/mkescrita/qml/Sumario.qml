import QtQuick

// O sumário (Ctrl+Shift+O): os títulos do texto, recuados pelo nível. Clicar
// (ou Enter) salta para a linha; setas andam; Esc fecha.
Rectangle {
    id: sumario

    // [{ nivel, texto, linha }], do `estrutura` do motor.
    property var titulos: []
    // O índice do título em que o cursor do texto está.
    property int atual: -1
    signal escolhido(int linha)
    signal fechar()

    function foca() {
        lista.currentIndex = Math.max(0, atual);
        lista.forceActiveFocus();
    }

    color: Theme.surface

    Rectangle {
        anchors.right: parent.right
        width: 1
        height: parent.height
        color: Theme.hairline
    }

    Rotulo {
        id: cabeca
        x: 20
        y: 20
        text: "Sumário"
    }

    Text {
        anchors.top: cabeca.bottom
        anchors.topMargin: 16
        x: 20
        width: parent.width - 40
        visible: sumario.titulos.length === 0
        text: "Nenhum título ainda. Uma linha que começa com # vira um."
        color: Theme.textFaint
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
        wrapMode: Text.Wrap
    }

    ListView {
        id: lista
        anchors.top: cabeca.bottom
        anchors.topMargin: 12
        anchors.bottom: parent.bottom
        anchors.bottomMargin: 12
        width: parent.width - 1
        clip: true
        model: sumario.titulos
        boundsBehavior: Flickable.StopAtBounds
        keyNavigationEnabled: true

        Keys.onPressed: event => {
            if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                if (lista.currentIndex >= 0 && lista.currentIndex < sumario.titulos.length)
                    sumario.escolhido(sumario.titulos[lista.currentIndex].linha);
                event.accepted = true;
            } else if (event.key === Qt.Key_Escape) {
                sumario.fechar();
                event.accepted = true;
            }
        }

        delegate: Rectangle {
            id: item
            required property var modelData
            required property int index
            width: lista.width
            height: 28
            color: area.containsMouse || (lista.activeFocus && lista.currentIndex === index) ? Theme.wash : Qt.rgba(0, 0, 0, 0)

            Rectangle {
                visible: sumario.atual === item.index
                width: 2
                height: parent.height
                color: Theme.copper
            }

            Text {
                anchors.verticalCenter: parent.verticalCenter
                x: 20 + (Math.max(1, item.modelData.nivel) - 1) * 14
                width: parent.width - x - 14
                text: item.modelData.texto || "(sem texto)"
                color: sumario.atual === item.index ? Theme.text : item.modelData.nivel <= 1 ? Theme.textSoft : Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 13
                elide: Text.ElideRight
            }

            MouseArea {
                id: area
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: sumario.escolhido(item.modelData.linha)
            }
        }
    }
}
