import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Artista: foto, descrição, músicas mais tocadas e prateleiras (álbuns, singles, parecidos).
ListView {
    id: page
    property string browseId
    property bool canGoBack: true
    signal back()

    property string ref: "a" + Math.random().toString(36).slice(2)
    property var data: ({ status: "loading", tracks: [], shelves: [] })
    Connections {
        target: daemon
        function onMessage(m) { if (m.type === "page" && m.ref === page.ref) { page.data = m; Qt.callLater(() => page.contentY = -page.headerItem.height - page.topMargin); } }
    }
    Component.onCompleted: daemon.send({ cmd: "open", kind: "artist", id: browseId, ref: ref })

    clip: true
    spacing: 36
    cacheBuffer: 1200
    boundsBehavior: Flickable.StopAtBounds
    leftMargin: 32
    rightMargin: 32
    topMargin: 28
    bottomMargin: 32
    ScrollBar.vertical: ScrollBar {}
    model: data.shelves || []
    delegate: ShelfRow {
        required property var modelData
        width: ListView.view.width - page.leftMargin - page.rightMargin
        title: modelData.title
        items: modelData.items
        onActivated: (item, items) => app.activate(item, items)
        onMenu: (item, anchorItem) => app.trackMenu.openFor(item, anchorItem)
    }

    header: ColumnLayout {
        width: page.width - page.leftMargin - page.rightMargin
        spacing: 20
        PageHeader { label: "Artista"; canGoBack: true; onBack: page.back() }
        BusyIndicator { visible: page.data.status === "loading"; running: visible }
        Hint { visible: page.data.status === "unavailable"; text: "Não deu para abrir agora. Confira a internet." }

        RowLayout {
            visible: page.data.status === "ready"
            Layout.fillWidth: true
            spacing: 24
            Cover { Layout.preferredWidth: 180; Layout.preferredHeight: 180; round: true; source: page.data.artwork || "" }
            ColumnLayout {
                Layout.fillWidth: true
                spacing: 8
                Text { Layout.fillWidth: true; text: page.data.title || ""; color: Theme.text; font.family: Theme.serif; font.pixelSize: 40; wrapMode: Text.WordWrap; maximumLineCount: 2; elide: Text.ElideRight }
                Text {
                    Layout.fillWidth: true
                    visible: text !== ""
                    text: page.data.audience || ""
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 14
                }
                // Descrição: três linhas; "Mais" abre inteira.
                Text {
                    id: about
                    property bool open: false
                    Layout.fillWidth: true
                    text: page.data.subtitle || ""
                    visible: text !== ""
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 13
                    lineHeight: 1.2
                    wrapMode: Text.WordWrap
                    maximumLineCount: open ? 1000 : 3
                    elide: Text.ElideRight
                }
                Text {
                    visible: about.visible && (about.truncated || about.open)
                    text: about.open ? "Menos" : "Mais"
                    color: Theme.copper
                    font.family: Theme.sans
                    font.pixelSize: 12
                    font.weight: Font.Medium
                    HoverHandler { cursorShape: Qt.PointingHandCursor }
                    TapHandler { onTapped: about.open = !about.open }
                }
                RowLayout {
                    Layout.topMargin: 8
                    spacing: 8
                    PlayButtons { tracks: page.data.tracks || [] }
                    MkButton {
                        visible: (page.data.radioPlaylistId || "") !== ""
                        text: "Rádio do artista"
                        glyph: Theme.icon.radio
                        onClicked: daemon.send({ cmd: "playCollection", playlistId: page.data.radioPlaylistId })
                    }
                }
            }
        }
        Caption { visible: (page.data.tracks || []).length > 0; text: "Mais tocadas" }
        TrackList { Layout.fillWidth: true; tracks: (page.data.tracks || []).slice(0, 5); numbered: true }
    }
}
