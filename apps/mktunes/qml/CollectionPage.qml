import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Álbum ou playlist do catálogo: pede a página ao daemon e mostra quando ela chega.
ListView {
    id: page
    property string kind: "album"
    property string browseId
    property bool canGoBack: true
    signal back()

    property string ref: "c" + Math.random().toString(36).slice(2)
    property var data: ({ status: "loading", tracks: [] })
    property bool saved: false

    Connections {
        target: daemon
        function onMessage(m) {
            if (m.type === "page" && m.ref === page.ref) { page.data = m; page.saved = m.saved === true; Qt.callLater(() => page.contentY = -page.headerItem.height - page.topMargin); }
            else if (m.type === "saved" && m.id === page.browseId) page.saved = m.saved;
        }
    }
    Component.onCompleted: daemon.send({ cmd: "open", kind: kind, id: browseId, ref: ref })

    clip: true
    spacing: 2
    cacheBuffer: 1200
    boundsBehavior: Flickable.StopAtBounds
    leftMargin: 32
    rightMargin: 32
    topMargin: 28
    bottomMargin: 32
    ScrollBar.vertical: ScrollBar {}
    reuseItems: true
    model: data.tracks || []
    delegate: TrackRow {
        required property var modelData
        required property int index
        width: ListView.view.width - page.leftMargin - page.rightMargin
        track: modelData
        number: index
        showCover: page.kind !== "album"
        current: daemon.now.track.id === modelData.id
        onPlay: daemon.send({ cmd: "play", tracks: page.data.tracks, index: index })
        onMenu: anchorItem => app.trackMenu.openFor(modelData, anchorItem)
    }

    header: ColumnLayout {
        width: page.width - page.leftMargin - page.rightMargin
        spacing: 20
        PageHeader { label: page.kind === "album" ? "Álbum" : "Playlist"; canGoBack: true; onBack: page.back() }

        BusyIndicator { visible: page.data.status === "loading"; running: visible }
        ColumnLayout {
            visible: page.data.status === "unavailable"
            spacing: 12
            Hint { text: "Não deu para abrir agora. Confira a internet." }
            MkButton {
                text: "Tentar de novo"
                glyph: Theme.icon.retry
                onClicked: { page.data = { status: "loading", tracks: [] }; daemon.send({ cmd: "open", kind: page.kind, id: page.browseId, ref: page.ref }); }
            }
        }

        RowLayout {
            visible: page.data.status === "ready"
            Layout.fillWidth: true
            spacing: 24
            Cover { Layout.preferredWidth: 200; Layout.preferredHeight: 200; source: page.data.artwork || "" }
            // Largura preferida mínima: o texto quebra ou corta, em vez de empurrar a coluna.
            ColumnLayout {
                Layout.fillWidth: true
                Layout.preferredWidth: 1
                spacing: 6
                Text { Layout.fillWidth: true; text: page.data.title || ""; color: Theme.text; font.family: Theme.serif; font.pixelSize: 36; wrapMode: Text.WordWrap; maximumLineCount: 2; elide: Text.ElideRight }
                Text { Layout.fillWidth: true; elide: Text.ElideRight; text: page.data.owner || ""; visible: text !== ""; color: Theme.textSoft; font.family: Theme.sans; font.pixelSize: 15 }
                Text { Layout.fillWidth: true; wrapMode: Text.WordWrap; maximumLineCount: 2; elide: Text.ElideRight; text: [page.data.subtitle, page.data.detail].filter(s => s).join(" · "); color: Theme.textDim; font.family: Theme.sans; font.pixelSize: 13 }
                RowLayout {
                    Layout.topMargin: 12
                    spacing: 8
                    PlayButtons { tracks: page.data.tracks || [] }
                    Glyph {
                        glyph: page.saved ? Theme.icon.check : Theme.icon.plus
                        label: page.saved ? "Tirar da biblioteca" : "Salvar na biblioteca"
                        active: page.saved
                        onClicked: daemon.send({ cmd: "toggleSaved", id: page.browseId })
                    }
                    Glyph {
                        glyph: Theme.icon.download
                        label: "Baixar tudo"
                        onClicked: daemon.send({ cmd: "downloadAll", tracks: page.data.tracks })
                    }
                }
            }
        }
        Item { height: 4 }
    }
}
