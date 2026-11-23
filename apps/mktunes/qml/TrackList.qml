import QtQuick

// Lista de faixas que toca a partir da clicada. Rola sozinha; usada dentro de páginas.
ListView {
    id: list
    property var tracks: []
    property bool numbered: false
    property bool covers: true
    property string playlistId: ""
    interactive: false
    implicitHeight: contentHeight
    height: contentHeight
    spacing: 2
    model: tracks
    reuseItems: true
    delegate: TrackRow {
        required property var modelData
        required property int index
        width: ListView.view.width
        track: modelData
        number: list.numbered ? index : -1
        showCover: list.covers
        current: daemon.now.track.id === modelData.id
        onPlay: daemon.send({ cmd: "play", tracks: list.tracks, index: index })
        onMenu: anchorItem => app.trackMenu.openFor(modelData, anchorItem, list.playlistId, index)
    }
}
