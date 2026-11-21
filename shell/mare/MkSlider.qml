import QtQuick

// Controle deslizante dos Ajustes: from/to/step/value, `label` é o texto do
// valor à direita; arrastar emite moved(novo).
Item {
    id: slider
    property real from: 0
    property real to: 1
    property real step: 0.1
    property real value: 0
    property string label: ""
    readonly property real ratio: (value - from) / (to - from)
    signal moved(real value)

    implicitWidth: 260
    implicitHeight: 24

    function pick(x) {
        const r = Math.max(0, Math.min(1, x / track.width));
        const raw = from + r * (to - from);
        const snapped = Math.round(raw / step) * step;
        moved(Math.round(snapped * 1000) / 1000);
    }

    Item {
        id: track
        anchors.left: parent.left
        anchors.right: valueText.left
        anchors.rightMargin: 14
        anchors.verticalCenter: parent.verticalCenter
        height: 14

        Rectangle {
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width
            height: 1
            color: Theme.line
        }

        Rectangle {
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width * slider.ratio
            height: 2
            color: Theme.copper
        }

        Rectangle {
            x: parent.width * slider.ratio - width / 2
            anchors.verticalCenter: parent.verticalCenter
            width: 12
            height: 12
            radius: 6
            color: Theme.text
            border.width: 2
            border.color: Theme.surface
        }

        MouseArea {
            anchors.fill: parent
            anchors.margins: -6
            cursorShape: Qt.PointingHandCursor
            onPressed: mouse => slider.pick(mouse.x - 6)
            onPositionChanged: mouse => {
                if (pressed)
                    slider.pick(mouse.x - 6);
            }
        }
    }

    Text {
        id: valueText
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        width: 56
        horizontalAlignment: Text.AlignRight
        text: slider.label
        color: Theme.text
        font.family: Theme.mono
        font.pixelSize: 11
    }
}
