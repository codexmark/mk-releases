import QtQuick

// Botão arredondado dos painéis. `primary` usa a cor do projeto.
Rectangle {
    id: btn

    property string text: ""
    property bool primary: false
    property bool selected: false

    signal clicked

    implicitWidth: label.implicitWidth + 28
    implicitHeight: 34
    radius: 2
    color: hover.hovered ? Theme.raised : "transparent"
    border.width: 1
    border.color: primary || selected ? Projects.accent : Theme.line

    Behavior on color {
        ColorAnimation { duration: Motion.fast }
    }

    Text {
        id: label
        anchors.centerIn: parent
        text: btn.text
        color: btn.primary || btn.selected ? Projects.accent : Theme.text
        font.family: Theme.mono
        font.pixelSize: 12
    }

    HoverHandler {
        id: hover
        cursorShape: Qt.PointingHandCursor
    }

    TapHandler {
        onTapped: btn.clicked()
    }
}
