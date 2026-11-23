import QtQuick
import QtQuick.Layouts

// Uma prateleira: título e cartões rolando de lado (roda do mouse vira rolagem lateral).
ColumnLayout {
    id: shelf
    property string title
    property var items: []
    signal activated(var item, var items)
    signal menu(var item, var anchorItem)
    spacing: 12

    RowLayout {
        Layout.fillWidth: true
        spacing: 12
        Text { text: "//"; color: Theme.copper; font.family: Theme.mono; font.pixelSize: 14 }
        Text {
            text: shelf.title
            color: Theme.text
            font.family: Theme.sans
            font.pixelSize: 18
            font.weight: Font.DemiBold
        }
        Rectangle { Layout.fillWidth: true; height: 1; color: Theme.hairline }
    }

    ListView {
        id: list
        Layout.fillWidth: true
        Layout.preferredHeight: 220
        orientation: ListView.Horizontal
        spacing: 16
        clip: true
        cacheBuffer: 400
        boundsBehavior: Flickable.StopAtBounds
        model: shelf.items
        reuseItems: true
        delegate: Card {
            required property var modelData
            item: modelData
            onActivated: shelf.activated(modelData, shelf.items)
            onMenu: anchorItem => shelf.menu(modelData, anchorItem)
        }
        // Shift + roda rola de lado; a roda sozinha continua rolando a página.
        WheelHandler {
            acceptedDevices: PointerDevice.Mouse
            acceptedModifiers: Qt.ShiftModifier
            onWheel: event => list.flick((event.angleDelta.y || event.angleDelta.x) * 10, 0)
        }
    }
}
