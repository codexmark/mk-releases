import QtQuick
import QtQuick.Layouts

// Uma faixa: número ou capa, título, artista, curtida, duração e o menu ⋯.
Rectangle {
    id: row
    property var track: ({})
    property int number: -1
    property bool current: false
    property bool showCover: true
    property string detail: track.length || ""
    signal play()
    signal menu(var anchorItem)

    height: 56
    radius: 8
    color: hover.hovered ? Theme.wash : "transparent"
    Accessible.role: Accessible.ListItem
    Accessible.name: (track.title || "") + ", " + (track.artist || "")

    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { acceptedButtons: Qt.LeftButton; onTapped: row.play() }
    TapHandler { acceptedButtons: Qt.RightButton; onTapped: row.menu(moreButton) }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: 8
        anchors.rightMargin: 4
        spacing: 12

        Text {
            visible: row.number >= 0
            Layout.preferredWidth: 22
            horizontalAlignment: Text.AlignRight
            text: row.number + 1
            color: row.current ? Theme.copper : Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 12
        }
        Cover {
            visible: row.showCover
            Layout.preferredWidth: 40
            Layout.preferredHeight: 40
            source: row.track.artwork || ""
        }
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2
            Text {
                Layout.fillWidth: true
                text: row.track.title || ""
                color: row.current ? Theme.copper : Theme.text
                font.family: Theme.sans
                font.pixelSize: 14
                font.weight: Font.Medium
                elide: Text.ElideRight
            }
            Text {
                Layout.fillWidth: true
                text: row.track.artist || ""
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 12
                elide: Text.ElideRight
            }
        }
        Text {
            visible: daemon.isLiked(row.track.id)
            text: Theme.icon.heart
            color: Theme.copper
            font.family: Theme.symbols
            font.pixelSize: 14
        }
        Text {
            text: row.detail
            visible: text !== ""
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 11
        }
        Glyph {
            id: moreButton
            glyph: Theme.icon.more
            label: "Mais opções"
            size: 32
            opacity: hover.hovered || row.activeFocus ? 1 : 0.35
            onClicked: row.menu(moreButton)
        }
    }
}
