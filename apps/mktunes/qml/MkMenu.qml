import QtQuick
import QtQuick.Controls

Menu {
    id: menu
    padding: 6
    implicitWidth: 260
    background: Rectangle {
        implicitWidth: 260
        radius: 10
        color: Theme.raised
        border.color: Theme.line
    }
    delegate: MenuEntry {}
}
