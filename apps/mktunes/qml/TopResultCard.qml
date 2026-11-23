import QtQuick
import QtQuick.Layouts

// O melhor resultado da busca, em destaque: capa grande, tipo e o que fazer com ele.
Rectangle {
    id: card
    property var item: ({})
    property var songs: []
    signal activated()
    readonly property string kindLabel: ({ song: "Música", album: "Álbum", playlist: "Playlist", artist: "Artista" })[item.kind] || ""
    implicitHeight: 220
    radius: 14
    color: hover.hovered ? Theme.raised : Theme.surface
    border.color: Theme.hairline
    Behavior on color { ColorAnimation { duration: 120 } }
    Accessible.role: Accessible.Button
    Accessible.name: "Melhor resultado: " + (item.title || "")

    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { onTapped: card.activated() }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 10
        Cover {
            Layout.preferredWidth: 104
            Layout.preferredHeight: 104
            source: card.item.artwork || ""
            round: card.item.kind === "artist"
        }
        Text {
            Layout.fillWidth: true
            Layout.topMargin: 4
            text: card.item.title || ""
            color: Theme.text
            font.family: Theme.serif
            font.pixelSize: 26
            elide: Text.ElideRight
        }
        RowLayout {
            Layout.fillWidth: true
            spacing: 8
            Rectangle {
                implicitWidth: kind.implicitWidth + 16
                implicitHeight: 22
                radius: 11
                color: Theme.wash
                Text { id: kind; anchors.centerIn: parent; text: card.kindLabel; color: Theme.textSoft; font.family: Theme.sans; font.pixelSize: 11; font.weight: Font.Medium }
            }
            Text {
                Layout.fillWidth: true
                text: card.item.kind === "artist" ? "" : (card.item.subtitle || "")
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 13
                elide: Text.ElideRight
            }
        }
        Item { Layout.fillHeight: true }
    }

    // Música: tocar direto daqui.
    Rectangle {
        visible: card.item.kind === "song"
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.margins: 20
        width: 48; height: 48; radius: 24
        color: Theme.copper
        opacity: hover.hovered ? 1 : 0.85
        Text { anchors.centerIn: parent; text: Theme.icon.play; color: Theme.ground; font.family: Theme.symbols; font.pixelSize: 24 }
    }
}
