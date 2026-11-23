import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Lista de faixas da biblioteca: curtidas, listas inteligentes ou uma playlist sua.
// `source` = "liked" | "smart:<título>" | "playlist:<id>": segue a biblioteca ao vivo.
ListView {
    id: page
    property string source
    property bool canGoBack: true
    signal back()

    readonly property string playlistId: source.startsWith("playlist:") ? source.slice(9) : ""
    readonly property var playlist: playlistId === "" ? null : (daemon.library.playlists.find(p => p.id === playlistId) || null)
    readonly property string title: source === "liked" ? "Curtidas"
        : source === "local" ? "No computador"
        : source.startsWith("smart:") ? source.slice(6)
        : (playlist ? playlist.title : "Playlist")
    readonly property var tracks: source === "liked" ? daemon.library.liked
        : source === "local" ? daemon.local.tracks
        : source.startsWith("smart:") ? ((daemon.library.smart.find(s => s.title === source.slice(6)) || { tracks: [] }).tracks)
        : (playlist ? playlist.tracks : [])

    // Playlist apagada: volta sozinho.
    onPlaylistChanged: if (playlistId !== "" && playlist === null && daemon.library.playlists !== undefined) page.back()

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
    model: tracks
    delegate: TrackRow {
        required property var modelData
        required property int index
        width: ListView.view.width - page.leftMargin - page.rightMargin
        track: modelData
        current: daemon.now.track.id === modelData.id
        onPlay: daemon.send({ cmd: "play", tracks: page.tracks, index: index })
        onMenu: anchorItem => app.trackMenu.openFor(modelData, anchorItem, page.playlistId, index)
    }

    header: ColumnLayout {
        width: page.width - page.leftMargin - page.rightMargin
        spacing: 16
        PageHeader { label: page.playlistId !== "" ? "Playlist" : "Biblioteca"; title: page.title; canGoBack: true; onBack: page.back() }
        Hint { text: Theme.count(page.tracks.length, "música", "músicas") }
        RowLayout {
            spacing: 8
            PlayButtons { tracks: page.tracks }
            Glyph { glyph: Theme.icon.download; label: "Baixar tudo"; enabled: page.tracks.length > 0; onClicked: daemon.send({ cmd: "downloadAll", tracks: page.tracks }) }
            Glyph {
                id: optionsButton
                visible: page.playlistId !== ""
                glyph: Theme.icon.more
                label: "Opções da playlist"
                onClicked: playlistMenu.popup(optionsButton, 0, optionsButton.height)
            }
        }
        Hint {
            visible: page.tracks.length === 0
            text: page.source === "liked" ? "Curta músicas pelo ♥ no player ou pelo menu ⋯." : "Nada aqui ainda. Adicione pelo menu ⋯ de uma faixa."
        }
        Item { height: 4 }
    }

    MkMenu {
        id: playlistMenu
        MenuEntry { text: "Renomear…"; glyph: Theme.icon.playlistAdd; onTriggered: app.nameDialog.askRename(page.playlistId, page.title) }
        MenuEntry { text: "Apagar playlist"; glyph: Theme.icon.remove; onTriggered: app.confirm("Apagar a playlist \"" + page.title + "\"?", "As músicas continuam no catálogo.", "Apagar", () => daemon.send({ cmd: "deletePlaylist", id: page.playlistId })) }
    }
}
