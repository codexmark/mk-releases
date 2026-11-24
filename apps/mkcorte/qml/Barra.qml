import QtQuick

// Barra de progresso: um fio que enche de cobre, de 0 a 100.
Rectangle {
    id: barra

    property real pct: 0

    implicitHeight: 3
    color: Theme.hairline

    Rectangle {
        height: parent.height
        width: parent.width * Math.max(0, Math.min(100, barra.pct)) / 100
        color: Theme.copper
        Behavior on width { NumberAnimation { duration: 200 } }
    }
}
