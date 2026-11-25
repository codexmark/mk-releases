import QtQuick
import "Porto.js" as Porto

// "Enviar chave": escolher qual das chaves públicas vai para o host. O envio
// é o ssh-copy-id, num terminal: a senha de lá, se for pedida, é digitada lá.
Item {
    id: escolha

    property string apelido: ""
    property var chaves: []
    signal escolhida(string nome)
    signal fechado()

    // A chave marcada (pelas setas ou pelo ponteiro); Enter envia essa.
    property int marcada: 0
    onChavesChanged: marcada = 0

    function andar(passo) {
        if (chaves.length > 0)
            marcada = Math.max(0, Math.min(chaves.length - 1, marcada + passo));
    }

    function confirmar() {
        if (marcada >= 0 && marcada < chaves.length)
            escolha.escolhida(chaves[marcada].nome);
    }

    Fundo {
        anchors.fill: parent
        onFechado: escolha.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(520, parent.width - 32)
        height: Math.min(parent.height - 24, corpo.implicitHeight + 44)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        MouseArea { anchors.fill: parent }

        Flickable {
            id: rolo
            anchors.fill: parent
            anchors.margins: 22
            contentHeight: corpo.implicitHeight
            boundsBehavior: Flickable.StopAtBounds
            clip: true

            Column {
                id: corpo
                width: rolo.width
                spacing: 10

                Rotulo { text: "enviar chave  /  " + escolha.apelido }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    text: "Qual chave pública?"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 22
                    font.weight: Font.Light
                }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    text: escolha.chaves.length > 0
                        ? "Abre um terminal com o ssh-copy-id. Se a máquina pedir senha, é lá que se digita. Enter envia a marcada."
                        : "Não há chave pública (*.pub) em ~/.ssh. Crie uma num terminal com ssh-keygen -t ed25519 e volte aqui."
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 12
                }

                Repeater {
                    model: escolha.chaves

                    Rectangle {
                        id: item
                        required property var modelData
                        required property int index
                        readonly property bool marcado: escolha.marcada === index
                        width: corpo.width
                        height: 46
                        color: marcado ? Theme.wash : Qt.rgba(0, 0, 0, 0)
                        border.width: 1
                        border.color: marcado ? Theme.copper : Theme.hairline

                        Column {
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.margins: 12
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 3
                            Text {
                                width: parent.width
                                elide: Text.ElideMiddle
                                text: item.modelData.nome
                                color: Theme.text
                                font.family: Theme.sans
                                font.pixelSize: 13
                            }
                            Text {
                                width: parent.width
                                elide: Text.ElideRight
                                text: Porto.tipoEBits(item.modelData) + (item.modelData.comentario ? "  ·  " + item.modelData.comentario : "")
                                color: Theme.textDim
                                font.family: Theme.mono
                                font.pixelSize: 10
                            }
                        }
                        MouseArea {
                            id: toque
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onEntered: escolha.marcada = item.index
                            onClicked: escolha.escolhida(item.modelData.nome)
                        }
                    }
                }

                Botao {
                    texto: "cancelar"
                    onAcionado: escolha.fechado()
                }
            }
        }
    }
}
