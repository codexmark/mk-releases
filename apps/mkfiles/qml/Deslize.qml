import QtQuick

// Um controle de arrastar entre 0 e 1 (a posição de uma música, o volume).
Item {
    id: deslize

    property real valor: 0
    signal movido(real v)

    implicitWidth: 120
    implicitHeight: 18

    Rectangle {
        anchors.verticalCenter: parent.verticalCenter
        width: parent.width
        height: 2
        color: Theme.line
    }
    Rectangle {
        anchors.verticalCenter: parent.verticalCenter
        width: Math.max(0, Math.min(1, deslize.valor)) * parent.width
        height: 2
        color: Theme.copper
    }
    Rectangle {
        x: Math.max(0, Math.min(1, deslize.valor)) * (parent.width - width)
        anchors.verticalCenter: parent.verticalCenter
        width: 8
        height: 12
        color: toque.pressed || toque.containsMouse ? Theme.copper : Theme.textSoft
    }
    MouseArea {
        id: toque
        anchors.fill: parent
        hoverEnabled: true
        preventStealing: true
        cursorShape: Qt.PointingHandCursor
        function diz(x) { deslize.movido(Math.max(0, Math.min(1, x / Math.max(1, width)))); }
        onPressed: mouse => diz(mouse.x)
        onPositionChanged: mouse => {
            if (pressed)
                diz(mouse.x);
        }
    }
}
