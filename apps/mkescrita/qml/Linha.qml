import QtQuick

// Campo de uma linha para as barras (buscar, substituir, filtrar fontes): uma
// linha fina, cursor de cobre. Enter confirma (com ou sem Shift), Esc cancela.
Item {
    id: linha

    property alias text: entrada.text
    property alias temFoco: entrada.activeFocus
    property string dica: ""
    signal confirmado(bool comShift)
    signal cancelado()
    signal desce()
    signal sobe()

    function foca() {
        entrada.forceActiveFocus();
        entrada.selectAll();
    }

    implicitWidth: 220
    implicitHeight: 28

    TextInput {
        id: entrada
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        color: Theme.text
        selectionColor: Theme.copper
        selectedTextColor: Theme.ground
        font.family: Theme.mono
        font.pixelSize: 12
        clip: true
        cursorDelegate: Rectangle {
            width: 2
            color: Theme.copper
            visible: entrada.activeFocus
        }
        Keys.onPressed: event => {
            if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                linha.confirmado((event.modifiers & Qt.ShiftModifier) !== 0);
                event.accepted = true;
            } else if (event.key === Qt.Key_Escape) {
                linha.cancelado();
                event.accepted = true;
            } else if (event.key === Qt.Key_Down) {
                linha.desce();
                event.accepted = true;
            } else if (event.key === Qt.Key_Up) {
                linha.sobe();
                event.accepted = true;
            }
        }

        Text {
            anchors.verticalCenter: parent.verticalCenter
            visible: entrada.text === ""
            text: linha.dica
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 12
        }
    }

    Rectangle {
        anchors.bottom: parent.bottom
        width: parent.width
        height: 1
        color: entrada.activeFocus ? Theme.line : Theme.hairline
    }
}
