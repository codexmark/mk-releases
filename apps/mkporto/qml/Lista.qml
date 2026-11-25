import QtQuick
import QtQuick.Layouts
import "Porto.js" as Porto

// A lista de hosts: a busca em cima e, em cada linha, a bolinha do estado, o
// apelido e usuario@host:porta. Um clique escolhe; dois conectam.
Item {
    id: lista

    // Os hosts já filtrados e em ordem (Porto.visiveis), e quantos há ao todo.
    property var hosts: []
    property int total: 0
    property var sondas: ({})
    property string escolhido: ""
    property alias busca: campo.text
    signal escolher(string apelido)
    signal conectar(string apelido)
    signal remover()

    function focarBusca() { campo.focar(); }

    // O escolhido fica à vista quando se anda com as setas.
    onEscolhidoChanged: {
        for (let i = 0; i < hosts.length; i++)
            if (hosts[i].apelido === escolhido)
                vista.positionViewAtIndex(i, ListView.Contain);
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 10

        Campo {
            id: campo
            onApagarVazio: lista.remover()
            Layout.fillWidth: true
            dica: "procurar por apelido, endereço ou usuário"
        }

        ListView {
            id: vista
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            model: lista.hosts

            delegate: Rectangle {
                id: item
                required property var modelData
                readonly property bool atual: modelData.apelido === lista.escolhido
                readonly property string estado: Porto.estado(modelData, lista.sondas)
                width: vista.width
                height: 46
                color: atual ? Theme.wash : toque.containsMouse ? Theme.boneA(0.03) : Qt.rgba(0, 0, 0, 0)

                Rectangle {
                    visible: item.atual
                    width: 2
                    height: parent.height
                    color: Theme.copper
                }
                Bolinha {
                    x: 14
                    anchors.verticalCenter: parent.verticalCenter
                    estado: item.estado
                }
                Column {
                    anchors.left: parent.left
                    anchors.leftMargin: 34
                    anchors.right: demora.left
                    anchors.rightMargin: 8
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: 3

                    Text {
                        width: parent.width
                        elide: Text.ElideRight
                        text: item.modelData.apelido
                        color: item.atual ? Theme.text : Theme.textSoft
                        font.family: Theme.sans
                        font.pixelSize: 13
                    }
                    Text {
                        width: parent.width
                        elide: Text.ElideMiddle
                        text: Porto.linha(item.modelData)
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }
                }
                Text {
                    id: demora
                    anchors.right: parent.right
                    anchors.rightMargin: 10
                    anchors.verticalCenter: parent.verticalCenter
                    text: item.estado === "responde" ? Porto.latencia(lista.sondas[item.modelData.apelido].ms)
                        : item.estado === "salto" ? "salto" : ""
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
                MouseArea {
                    id: toque
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: lista.escolher(item.modelData.apelido)
                    onDoubleClicked: lista.conectar(item.modelData.apelido)
                }
            }

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                y: 24
                width: parent.width - 20
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.WordWrap
                visible: lista.hosts.length === 0 && lista.total > 0
                text: "nenhum host com essa busca"
                color: Theme.textFaint
                font.family: Theme.serif
                font.pixelSize: 14
                font.italic: true
            }
        }
    }
}
