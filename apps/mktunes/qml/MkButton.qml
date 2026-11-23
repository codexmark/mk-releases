import QtQuick

// Botão de texto: contornado, ou cheio de cobre quando é a ação principal.
Rectangle {
    id: button
    property string text
    property string glyph: ""
    property bool primary: false
    signal clicked()
    implicitWidth: row.implicitWidth + 28
    implicitHeight: 36
    radius: 18
    color: primary ? Theme.copper : (hover.hovered ? Theme.wash : "transparent")
    border.color: primary ? Theme.copper : Theme.line
    opacity: enabled ? 1 : 0.4
    Accessible.role: Accessible.Button
    Accessible.name: text
    Accessible.onPressAction: button.clicked()
    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { onTapped: button.clicked() }
    Row {
        id: row
        anchors.centerIn: parent
        spacing: 8
        Text {
            visible: button.glyph !== ""
            text: button.glyph
            color: button.primary ? Theme.ground : Theme.text
            font.family: Theme.symbols
            font.pixelSize: 15
            anchors.verticalCenter: parent.verticalCenter
        }
        Text {
            text: button.text
            color: button.primary ? Theme.ground : Theme.text
            font.family: Theme.sans
            font.pixelSize: 13
            font.weight: Font.Medium
            anchors.verticalCenter: parent.verticalCenter
        }
    }
}
