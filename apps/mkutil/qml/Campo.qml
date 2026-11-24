import QtQuick

// Campo de uma linha: uma linha fina, cursor de cobre. `so` restringe o que
// se digita ("numero": só algarismos e a vírgula).
Item {
    id: campo

    property alias text: entrada.text
    property string dica: ""
    property string so: ""
    signal confirmado()

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
        validator: campo.so === "numero" ? numeros : null
        onAccepted: campo.confirmado()
        cursorDelegate: Rectangle {
            width: 2
            color: Theme.copper
            visible: entrada.activeFocus
        }
        Keys.onEscapePressed: entrada.text = ""

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

    RegularExpressionValidator {
        id: numeros
        regularExpression: /[0-9]{0,6}([.,][0-9]{0,2})?/
    }

    Rectangle {
        anchors.bottom: parent.bottom
        width: parent.width
        height: 1
        color: entrada.activeFocus ? Theme.line : Theme.hairline
    }
}
