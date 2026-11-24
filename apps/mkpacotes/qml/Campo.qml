import QtQuick

// Campo de busca: uma linha fina, cursor de cobre. Enter avisa (aceito).
Item {
    id: campo

    property alias text: entrada.text
    property string dica: ""
    signal aceito()

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
        font.family: Theme.sans
        font.pixelSize: 13
        clip: true
        cursorDelegate: Rectangle {
            width: 2
            color: Theme.copper
            visible: entrada.activeFocus
        }
        Keys.onEscapePressed: entrada.text = ""
        Keys.onReturnPressed: campo.aceito()
        Keys.onEnterPressed: campo.aceito()

        Text {
            anchors.verticalCenter: parent.verticalCenter
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
