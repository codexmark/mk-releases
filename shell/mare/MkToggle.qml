import QtQuick

// Chave liga/desliga dos Ajustes. `checked` é o estado; clicar emite toggled(novo).
Rectangle {
    id: toggle
    property bool checked: false
    signal toggled(bool value)

    implicitWidth: 36
    implicitHeight: 20
    radius: 10
    color: checked ? Theme.copper : Theme.line

    Behavior on color {
        ColorAnimation { duration: Motion.fast }
    }

    Rectangle {
        x: toggle.checked ? parent.width - width - 2 : 2
        y: 2
        width: 16
        height: 16
        radius: 8
        color: Theme.ground

        Behavior on x {
            NumberAnimation {
                duration: Motion.fast
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.curve
            }
        }
    }

    HoverHandler { cursorShape: Qt.PointingHandCursor }
    TapHandler { onTapped: toggle.toggled(!toggle.checked) }
}
