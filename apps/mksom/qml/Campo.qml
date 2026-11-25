import QtQuick

// Campo de uma linha: uma linha fina, cursor de cobre. Enter avisa (aceito).
// Usado para o nome de uma predefinição.
Item {
    id: campo

    property alias text: entrada.text
    property string dica: ""
    property bool mono: false
    readonly property bool comFoco: entrada.activeFocus
    signal aceito()
    signal apagarVazio()

    function focar() { entrada.forceActiveFocus(); }

    implicitWidth: 260
    implicitHeight: 30

    TextInput {
        id: entrada
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        color: Theme.text
        selectionColor: Theme.copper
        font.family: campo.mono ? Theme.mono : Theme.sans
        font.pixelSize: campo.mono ? 12 : 13
        clip: true
        selectByMouse: true
        activeFocusOnTab: true
        cursorDelegate: Rectangle {
            width: 2
            color: Theme.copper
            visible: entrada.activeFocus
        }
        Keys.onReturnPressed: campo.aceito()
        // Um campo de texto fica com o Delete para si; vazio, não há o que
        // apagar, e a tecla vale para quem usa o campo (nada, aqui).
        Keys.onDeletePressed: event => {
            if (text === "")
                campo.apagarVazio();
            else
                event.accepted = false;
        }
        Keys.onEnterPressed: campo.aceito()

        Text {
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width
            elide: Text.ElideRight
            visible: entrada.text === ""
            text: campo.dica
            color: Theme.textFaint
            font.family: Theme.serif
            font.pixelSize: 13
            font.italic: true
        }
    }

    Rectangle {
        anchors.bottom: parent.bottom
        width: parent.width
        height: 1
        color: entrada.activeFocus ? Theme.line : Theme.hairline
    }
}
