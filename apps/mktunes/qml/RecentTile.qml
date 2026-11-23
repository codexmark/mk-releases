import QtQuick
import QtQuick.Layouts

// Ladrilho compacto de "Tocadas recentemente": capa pequena, título e artista; toca ao clicar.
Rectangle {
    id: tile
    property var track: ({})
    property bool current: false
    signal play()
    signal menu(var anchorItem)
    implicitHeight: 64
    radius: 10
    color: hover.hovered ? Theme.raised : Theme.surface
    Behavior on color { ColorAnimation { duration: 120 } }
    Accessible.role: Accessible.Button
    Accessible.name: (track.title || "") + ", " + (track.artist || "")

    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { acceptedButtons: Qt.LeftButton; onTapped: tile.play() }
    TapHandler { acceptedButtons: Qt.RightButton; onTapped: tile.menu(tile) }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 8
        spacing: 12
        Cover { Layout.preferredWidth: 48; Layout.preferredHeight: 48; source: tile.track.artwork || "" }
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2
            Text { Layout.fillWidth: true; text: tile.track.title || ""; color: tile.current ? Theme.copper : Theme.text; font.family: Theme.sans; font.pixelSize: 13; font.weight: Font.Medium; elide: Text.ElideRight }
            Text { Layout.fillWidth: true; text: tile.track.artist || ""; color: Theme.textDim; font.family: Theme.sans; font.pixelSize: 12; elide: Text.ElideRight }
        }
        Text {
            visible: hover.hovered
            text: Theme.icon.play
            color: Theme.copper
            font.family: Theme.symbols
            font.pixelSize: 20
        }
    }
}
