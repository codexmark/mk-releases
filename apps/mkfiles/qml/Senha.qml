import QtQuick

// O campo de senha de um compactado: o que se digita não aparece. A senha
// mora só aqui, na memória, enquanto a folha está aberta: a janela a passa ao
// motor no pedido e não a guarda em lugar nenhum.
Item {
    id: senha

    property alias text: entrada.text
    property string dica: "senha"
    readonly property bool comFoco: entrada.activeFocus
    signal aceito()

    function focar() { entrada.forceActiveFocus(); }
    function limpar() { entrada.text = ""; }

    implicitWidth: 260
    implicitHeight: 30

    TextInput {
        id: entrada
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        echoMode: TextInput.Password
        passwordCharacter: "•"
        color: Theme.text
        selectionColor: Theme.copper
        font.family: Theme.mono
        font.pixelSize: 13
        clip: true
        activeFocusOnTab: true
        cursorDelegate: Rectangle {
            width: 2
            color: Theme.copper
            visible: entrada.activeFocus
        }
        Keys.onReturnPressed: senha.aceito()
        Keys.onEnterPressed: senha.aceito()

        Text {
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width
            elide: Text.ElideRight
            visible: entrada.text === ""
            text: senha.dica
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
