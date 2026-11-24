import QtQuick

// O rodapé: à esquerda o nome do arquivo e o estado (salvo, salvando, não
// salvo, conflito); à direita as palavras, o tempo de leitura, a fonte e o
// tamanho do texto. Discreto: some no modo foco.
Item {
    id: rodape

    property string nome: ""
    property string estado: ""
    // O estado pede atenção (conflito, não salvou): fica em cobre e clicável.
    property bool alerta: false
    property string aviso: ""
    property string palavras: ""
    property string leitura: ""
    property string fonte: ""
    property int zoom: 100
    signal estadoClicado()
    signal fonteClicada()
    signal zoomClicado()

    implicitHeight: 30

    Rectangle {
        anchors.top: parent.top
        width: parent.width
        height: 1
        color: Theme.hairline
    }

    Row {
        id: esquerda
        anchors.left: parent.left
        anchors.leftMargin: 16
        anchors.verticalCenter: parent.verticalCenter
        spacing: 10

        Text {
            text: rodape.nome
            color: Theme.textSoft
            font.family: Theme.mono
            font.pixelSize: 11
            elide: Text.ElideMiddle
            width: Math.min(implicitWidth, rodape.width * 0.3)
        }

        Text {
            text: rodape.estado
            color: rodape.alerta ? Theme.copper : Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 11

            MouseArea {
                anchors.fill: parent
                anchors.margins: -6
                enabled: rodape.alerta
                cursorShape: rodape.alerta ? Qt.PointingHandCursor : Qt.ArrowCursor
                onClicked: rodape.estadoClicado()
            }
        }

        Text {
            visible: text !== ""
            text: rodape.aviso
            color: Theme.textDim
            font.family: Theme.serif
            font.pixelSize: 12
            font.italic: true
            elide: Text.ElideRight
            width: Math.min(implicitWidth, Math.max(0, rodape.width - esquerda.x - direita.width - 260))
        }
    }

    Row {
        id: direita
        anchors.right: parent.right
        anchors.rightMargin: 16
        anchors.verticalCenter: parent.verticalCenter
        spacing: 14

        Text {
            visible: text !== ""
            text: rodape.palavras
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 11
        }

        Text {
            visible: text !== ""
            text: rodape.leitura
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 11
        }

        Text {
            text: rodape.fonte
            color: areaDaFonte.containsMouse ? Theme.copper : Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 11
            elide: Text.ElideRight
            width: Math.min(implicitWidth, 180)

            MouseArea {
                id: areaDaFonte
                anchors.fill: parent
                anchors.margins: -6
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: rodape.fonteClicada()
            }
        }

        Text {
            text: rodape.zoom + "%"
            color: areaDoZoom.containsMouse ? Theme.copper : Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 11

            MouseArea {
                id: areaDoZoom
                anchors.fill: parent
                anchors.margins: -6
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: rodape.zoomClicado()
            }
        }
    }
}
