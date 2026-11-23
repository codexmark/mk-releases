import QtQuick
import QtQuick.Controls

TextField {
    id: field
    placeholderTextColor: Theme.textFaint
    color: Theme.text
    selectionColor: Theme.copper
    selectedTextColor: Theme.ground
    font.family: Theme.sans
    font.pixelSize: 14
    leftPadding: 14
    rightPadding: 14
    topPadding: 10
    bottomPadding: 10
    background: Rectangle {
        radius: 10
        color: Theme.surface
        border.color: field.activeFocus ? Theme.copper : Theme.line
    }
}
