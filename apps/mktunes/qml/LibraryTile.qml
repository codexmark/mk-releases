import QtQuick

// Cartão da biblioteca: capa (ou um símbolo), título e quantas músicas.
Item {
    id: tile
    property string title
    property string subtitle
    property string artwork
    property string glyph: Theme.icon.library
    signal activated()
    width: 150
    height: 202
    Accessible.role: Accessible.Button
    Accessible.name: title
    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { onTapped: tile.activated() }
    Rectangle {
        id: art
        width: 150
        height: 150
        radius: 12
        color: Theme.raised
        border.color: Theme.hairline
        clip: true
        scale: hover.hovered ? 1.03 : 1
        Behavior on scale { NumberAnimation { duration: 140; easing.type: Easing.OutCubic } }
        Cover { anchors.fill: parent; source: tile.artwork; visible: tile.artwork !== "" }
        Text {
            anchors.centerIn: parent
            visible: tile.artwork === ""
            text: tile.glyph
            color: Theme.copper
            font.family: Theme.symbols
            font.pixelSize: 44
        }
    }
    Text { anchors.top: art.bottom; anchors.topMargin: 8; width: 150; text: tile.title; color: Theme.text; font.family: Theme.sans; font.pixelSize: 13; font.weight: Font.Medium; elide: Text.ElideRight }
    Text { anchors.top: art.bottom; anchors.topMargin: 28; width: 150; text: tile.subtitle; color: Theme.textDim; font.family: Theme.sans; font.pixelSize: 12; elide: Text.ElideRight }
}
