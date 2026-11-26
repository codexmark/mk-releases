import QtQuick

// Uma opção de ligar e desligar com o nome ao lado.
Row {
    id: opcao

    property string texto: ""
    property bool ligado: false
    signal trocado()

    spacing: 7

    Chave {
        anchors.verticalCenter: parent.verticalCenter
        ligado: opcao.ligado
        onTrocado: opcao.trocado()
    }
    Text {
        anchors.verticalCenter: parent.verticalCenter
        text: opcao.texto
        color: opcao.ligado ? Theme.text : Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 12
        MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: opcao.trocado()
        }
    }
}
