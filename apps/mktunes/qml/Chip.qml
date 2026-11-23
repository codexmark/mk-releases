import QtQuick

Rectangle {
    id: chip
    property string text
    property bool selected: false
    signal clicked()
    implicitWidth: label.implicitWidth + 28
    implicitHeight: 34
    radius: 17
    color: selected ? Theme.copper : (hover.hovered ? Theme.wash : "transparent")
    border.color: selected ? Theme.copper : Theme.line
    Accessible.role: Accessible.Button
    Accessible.name: text
    Accessible.checkable: true
    Accessible.checked: selected
    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { onTapped: chip.clicked() }
    Text {
        id: label
        anchors.centerIn: parent
        text: chip.text
        color: chip.selected ? Theme.ground : Theme.text
        font.family: Theme.sans
        font.pixelSize: 13
        font.weight: Font.Medium
    }
}
