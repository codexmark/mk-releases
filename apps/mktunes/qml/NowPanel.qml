import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Tocando agora: capa, título e, em abas, a fila e a letra.
Rectangle {
    id: panel
    readonly property var now: daemon.now
    readonly property bool hasTrack: now.status !== "idle"
    property int tab: 0
    radius: 14
    color: Theme.surface
    border.color: Theme.hairline

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 12

        Caption { text: "Tocando agora"; accent: true }
        Cover {
            Layout.fillWidth: true
            Layout.preferredHeight: width
            Layout.maximumHeight: 300
            source: panel.now.artwork
            Text {
                anchors.centerIn: parent
                visible: !panel.hasTrack
                text: "Nada tocando"
                color: Theme.textFaint
                font.family: Theme.serif
                font.pixelSize: 18
            }
        }
        Text { Layout.fillWidth: true; text: panel.now.title; color: Theme.text; font.family: Theme.serif; font.pixelSize: 22; elide: Text.ElideRight }
        Text {
            Layout.fillWidth: true
            Layout.topMargin: -8
            text: panel.now.status === "loading" ? panel.now.artist + " · carregando…"
                : panel.now.status === "failed" ? "Não deu para tocar esta faixa." : panel.now.artist
            color: panel.now.status === "failed" ? Theme.copper : Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 13
            elide: Text.ElideRight
        }

        RowLayout {
            Layout.topMargin: 4
            spacing: 6
            Chip { text: "A seguir"; selected: panel.tab === 0; onClicked: panel.tab = 0 }
            Chip { text: "Letra"; selected: panel.tab === 1; onClicked: panel.tab = 1 }
            Item { Layout.fillWidth: true }
            Glyph { visible: panel.tab === 0 && panel.now.upNext.length > 1; glyph: Theme.icon.shuffle; label: "Embaralhar a fila"; size: 30; onClicked: daemon.send({ cmd: "shuffleUpNext" }) }
            Glyph { visible: panel.tab === 0 && panel.now.upNext.length > 0; glyph: Theme.icon.remove; label: "Limpar a fila"; size: 30; onClicked: daemon.send({ cmd: "clearUpNext" }) }
        }

        StackLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            currentIndex: panel.tab

            ListView {
                id: queue
                clip: true
                spacing: 2
                cacheBuffer: 600
                reuseItems: true
                boundsBehavior: Flickable.StopAtBounds
                model: panel.now.upNext
                ScrollBar.vertical: ScrollBar {}
                delegate: TrackRow {
                    id: queued
                    required property var modelData
                    required property int index
                    width: ListView.view.width
                    track: modelData
                    detail: ""
                    onPlay: daemon.send({ cmd: "playUpNext", index: index })
                    onMenu: anchorItem => queueMenu.openAt(index, modelData, anchorItem)
                }
                Hint {
                    anchors.fill: parent
                    visible: queue.count === 0
                    text: panel.hasTrack ? "A fila acabou. Com \"músicas parecidas\" ligado, o MkTunes continua sozinho." : ""
                }
            }

            Lyrics { offsetMs: app.lyricsOffset }
        }
    }

    MkMenu {
        id: queueMenu
        property int index: -1
        property var track: ({})
        function openAt(i, t, anchorItem) { index = i; track = t; popup(anchorItem, 0, anchorItem.height); }
        MenuEntry { text: "Tocar agora"; glyph: Theme.icon.play; onTriggered: daemon.send({ cmd: "playUpNext", index: queueMenu.index }) }
        MenuEntry { text: "Subir"; glyph: "↑"; enabled: queueMenu.index > 0; onTriggered: daemon.send({ cmd: "moveUpNext", from: queueMenu.index, to: queueMenu.index - 1 }) }
        MenuEntry { text: "Descer"; glyph: "↓"; enabled: queueMenu.index < panel.now.upNext.length - 1; onTriggered: daemon.send({ cmd: "moveUpNext", from: queueMenu.index, to: queueMenu.index + 1 }) }
        MenuEntry { text: "Tirar da fila"; glyph: Theme.icon.remove; onTriggered: daemon.send({ cmd: "removeUpNext", index: queueMenu.index }) }
        MenuEntry { text: daemon.isLiked(queueMenu.track.id) ? "Descurtir" : "Curtir"; glyph: Theme.icon.heartOutline; onTriggered: daemon.send({ cmd: "like", track: queueMenu.track }) }
        MenuEntry { text: "Baixar"; glyph: Theme.icon.download; onTriggered: daemon.send({ cmd: "download", track: queueMenu.track }) }
    }
}
