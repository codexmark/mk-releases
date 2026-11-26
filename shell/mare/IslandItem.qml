import QtQuick

// Um item da ilha: ícone e, se houver, um valor curto. `plate` desenha a
// placa de osso do item ativo (como o workspace atual). Clicar abre `panel`
// logo abaixo deste item; sem painel, emite `clicked`.
Item {
    id: item

    property string icon: ""
    property string text: ""
    property string panel: ""
    property color iconColor: Theme.textDim
    property color textColor: Theme.textSoft
    property bool plate: false
    property string textFamily: Theme.mono
    property int textSize: 11
    property bool dot: false
    // O texto mais largo que este item mostra (ex.: "100%"). Com ele, o item
    // guarda esse lugar e não muda de largura quando o valor ganha ou perde
    // um dígito; vazio, a largura é a do próprio texto.
    property string reserve: ""

    signal clicked
    signal scrolled(int delta)

    implicitWidth: row.implicitWidth + (plate ? 16 : 10)
    implicitHeight: 24

    Rectangle {
        anchors.fill: parent
        radius: 9
        color: item.plate ? Theme.text : hover.hovered ? Theme.wash : "transparent"

        Behavior on color {
            ColorAnimation { duration: Motion.fast }
        }
    }

    TextMetrics {
        id: reserved
        text: item.reserve
        font.family: item.textFamily
        font.pixelSize: item.textSize
        font.features: ({ "tnum": 1 })
    }

    Row {
        id: row
        anchors.centerIn: parent
        spacing: 5

        Text {
            visible: item.icon !== ""
            anchors.verticalCenter: parent.verticalCenter
            text: item.icon
            color: item.plate ? Theme.ground : item.iconColor
            font.family: Theme.icons
            font.pixelSize: 15
        }

        Text {
            visible: item.text !== ""
            anchors.verticalCenter: parent.verticalCenter
            text: item.text
            color: item.plate ? Theme.ground : item.textColor
            font.family: item.textFamily
            font.pixelSize: item.textSize
            font.features: ({ "tnum": 1 })
            width: Math.max(implicitWidth, Math.ceil(reserved.advanceWidth))
        }
    }

    // Ponto de cobre: algo pedindo atenção (notificação, música tocando).
    Rectangle {
        visible: item.dot
        x: parent.width - 7
        y: 3
        width: 5
        height: 5
        radius: 3
        color: Theme.copper
    }

    HoverHandler {
        id: hover
        cursorShape: Qt.PointingHandCursor
    }

    TapHandler {
        onTapped: {
            if (item.panel !== "")
                Mare.openAt(item.panel, item.mapToItem(null, item.width / 2, 0).x);
            else
                item.clicked();
        }
    }

    WheelHandler {
        onWheel: event => item.scrolled(event.angleDelta.y)
    }
}
