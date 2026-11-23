import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Rádio: as estações mais votadas da região, ou as que têm o nome buscado.
ListView {
    id: page
    property bool canGoBack: false
    signal back()
    readonly property var radio: daemon.radio
    Component.onCompleted: daemon.send({ cmd: "radio" })

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
    model: radio.stations
    delegate: TrackRow {
        required property var modelData
        required property int index
        width: ListView.view.width - page.leftMargin - page.rightMargin
        track: modelData
        detail: "AO VIVO"
        current: daemon.now.track.id === modelData.id
        onPlay: daemon.send({ cmd: "play", tracks: [modelData], index: 0 })
        onMenu: anchorItem => app.trackMenu.openFor(modelData, anchorItem)
    }

    header: ColumnLayout {
        width: page.width - page.leftMargin - page.rightMargin
        spacing: 16
        PageHeader { label: "03 · Rádio · Ao vivo"; title: "Rádio" }
        MkField {
            Layout.fillWidth: true
            placeholderText: "Buscar estação pelo nome"
            onAccepted: daemon.send({ cmd: "radio", query: text.trim() })
        }
        BusyIndicator { visible: page.radio.status === "loading"; running: visible }
        Hint {
            visible: text !== ""
            text: page.radio.status === "nothing" ? "Nenhuma estação com esse nome."
                : page.radio.status === "unavailable" ? "O diretório de rádios não respondeu. Confira a internet." : ""
        }
        Item { height: 4 }
    }
}
