import QtQuick

// Cabeçalho dos painéis, como as seções das folhas do perfil:
// "01  título ——————— INFORMAÇÃO". Número em cobre, título em serifa,
// linha guia até a informação à direita, em mono maiúsculo.
Item {
    id: header

    property string index: ""
    property string title: ""
    property string meta: ""
    property color metaColor: Theme.textDim

    implicitWidth: row.implicitWidth
    implicitHeight: 30

    Row {
        id: row
        anchors.verticalCenter: parent.verticalCenter
        width: parent.width
        spacing: 12

        Text {
            id: num
            visible: header.index !== ""
            anchors.verticalCenter: parent.verticalCenter
            text: header.index
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
            font.letterSpacing: 1.2
        }

        Text {
            id: ttl
            anchors.verticalCenter: parent.verticalCenter
            text: header.title
            color: Theme.text
            font.family: Theme.serif
            font.pixelSize: 19
            font.weight: Font.Light
        }

        Rectangle {
            anchors.verticalCenter: parent.verticalCenter
            width: Math.max(16, header.width - (num.visible ? num.implicitWidth + 12 : 0)
                - ttl.implicitWidth - (meta.visible ? meta.implicitWidth + 12 : 0) - 12)
            height: 1
            color: Theme.hairline
        }

        Text {
            id: meta
            visible: header.meta !== ""
            anchors.verticalCenter: parent.verticalCenter
            text: header.meta
            color: header.metaColor
            font.family: Theme.mono
            font.pixelSize: 10
            font.capitalization: Font.AllUppercase
            font.letterSpacing: 1.6
        }
    }
}
