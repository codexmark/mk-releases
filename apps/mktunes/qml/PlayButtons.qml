import QtQuick
import QtQuick.Layouts

// Tocar e Aleatório para uma lista inteira.
RowLayout {
    id: row
    property var tracks: []
    spacing: 8
    MkButton {
        text: "Tocar"
        glyph: Theme.icon.play
        primary: true
        enabled: row.tracks.length > 0
        onClicked: daemon.send({ cmd: "play", tracks: row.tracks, index: 0 })
    }
    MkButton {
        text: "Aleatório"
        glyph: Theme.icon.shuffle
        enabled: row.tracks.length > 0
        onClicked: daemon.send({ cmd: "play", tracks: row.tracks, index: 0, shuffle: true })
    }
}
