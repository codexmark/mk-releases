import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Biblioteca em abas: visão geral (curtidas, listas inteligentes e o que você ouviu), suas
// playlists, os álbuns salvos e as músicas que já estão no computador.
Flickable {
    id: page
    property bool canGoBack: false
    signal back()
    readonly property var lib: daemon.library
    readonly property var local: daemon.local
    property int section: 0
    onSectionChanged: { contentY = 0; if (section === 3) daemon.send({ cmd: "local" }); }

    clip: true
    contentHeight: column.implicitHeight + 64
    boundsBehavior: Flickable.StopAtBounds
    ScrollBar.vertical: ScrollBar {}
    Component.onCompleted: daemon.send({ cmd: "library" })
    onVisibleChanged: if (visible) { daemon.send({ cmd: "library" }); if (section === 3) daemon.send({ cmd: "local" }); }

    ColumnLayout {
        id: column
        x: 32
        y: 28
        width: page.width - 64
        spacing: 28

        PageHeader { label: "04 · Biblioteca"; title: "Sua música" }

        Row {
            spacing: 8
            Repeater {
                model: ["Visão geral", "Playlists", "Salvos", "No computador"]
                Chip {
                    required property string modelData
                    required property int index
                    text: modelData
                    selected: page.section === index
                    onClicked: page.section = index
                }
            }
        }

        Flow {
            Layout.fillWidth: true
            visible: page.section === 0
            spacing: 20
            LibraryTile {
                title: "Curtidas"
                subtitle: Theme.count(page.lib.liked.length, "música", "músicas")
                artwork: page.lib.liked.length > 0 ? page.lib.liked[0].artwork : ""
                glyph: Theme.icon.heart
                onActivated: app.push(trackListPage, { source: "liked" })
            }
            Repeater {
                model: page.lib.smart
                LibraryTile {
                    required property var modelData
                    title: modelData.title
                    subtitle: Theme.count(modelData.tracks.length, "música", "músicas")
                    artwork: modelData.tracks.length > 0 ? modelData.tracks[0].artwork : ""
                    onActivated: app.push(trackListPage, { source: "smart:" + modelData.title })
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            visible: page.section === 1
            Caption { text: "Suas playlists" }
            Item { Layout.fillWidth: true }
            MkButton { text: "Nova playlist"; glyph: Theme.icon.plus; onClicked: app.nameDialog.askCreate(null) }
        }
        Hint { visible: page.section === 1 && page.lib.playlists.length === 0; text: "Crie uma playlist aqui ou pelo menu ⋯ de qualquer faixa." }
        Flow {
            Layout.fillWidth: true
            visible: page.section === 1
            spacing: 20
            Repeater {
                model: page.lib.playlists
                LibraryTile {
                    required property var modelData
                    title: modelData.title
                    subtitle: Theme.count(modelData.tracks.length, "música", "músicas")
                    artwork: modelData.artwork
                    onActivated: app.push(trackListPage, { source: "playlist:" + modelData.id })
                }
            }
        }

        Hint { visible: page.section === 2 && page.lib.saved.length === 0; text: "Salve álbuns e playlists do catálogo pelo botão + da página deles." }
        Flow {
            Layout.fillWidth: true
            visible: page.section === 2 && page.lib.saved.length > 0
            spacing: 20
            Repeater {
                model: page.lib.saved
                Card {
                    required property var modelData
                    item: modelData
                    size: 150
                    onActivated: app.activate(modelData, [])
                }
            }
        }

        // No computador: o começo da lista aqui; a lista inteira numa página própria.
        ColumnLayout {
            Layout.fillWidth: true
            visible: page.section === 3
            spacing: 12
            RowLayout {
                Layout.fillWidth: true
                Caption { text: page.local.status === "ready" ? Theme.count(page.local.tracks.length, "música na pasta de Música", "músicas na pasta de Música") : "Pasta de Música" }
                Item { Layout.fillWidth: true }
                PlayButtons { visible: page.local.tracks.length > 0; tracks: page.local.tracks }
            }
            BusyIndicator { visible: page.local.status === "scanning"; running: visible }
            Hint {
                visible: page.local.status === "empty"
                text: "Nenhuma música na sua pasta de Música. As baixadas pelo MkTunes ficam em Downloads."
            }
            TrackList { Layout.fillWidth: true; tracks: page.section === 3 ? page.local.tracks.slice(0, 50) : [] }
            MkButton {
                visible: page.local.tracks.length > 50
                text: "Ver todas as " + page.local.tracks.length
                onClicked: app.push(trackListPage, { source: "local" })
            }
        }

        RowLayout {
            Layout.fillWidth: true
            visible: page.section === 0
            Caption { text: "O que você ouviu" }
            Item { Layout.fillWidth: true }
            Repeater {
                model: [{ key: "day", title: "Hoje" }, { key: "week", title: "7 dias" }, { key: "month", title: "30 dias" }]
                Chip {
                    required property var modelData
                    text: modelData.title
                    selected: page.lib.period === modelData.key
                    onClicked: daemon.send({ cmd: "statsPeriod", period: modelData.key })
                }
            }
        }
        RowLayout {
            visible: page.section === 0
            spacing: 40
            Stat { value: page.lib.stats.plays || 0; caption: "músicas tocadas" }
            Stat { value: page.lib.stats.minutes || 0; caption: "minutos" }
        }
        RowLayout {
            Layout.fillWidth: true
            spacing: 32
            visible: page.section === 0 && (page.lib.stats.plays || 0) > 0
            ColumnLayout {
                Layout.preferredWidth: 240
                Layout.alignment: Qt.AlignTop
                spacing: 8
                Caption { text: "Artistas" }
                Repeater {
                    model: page.lib.stats.artists || []
                    RowLayout {
                        required property var modelData
                        required property int index
                        spacing: 10
                        Text { text: index + 1; color: Theme.copper; font.family: Theme.mono; font.pixelSize: 12 }
                        Text { text: modelData.name; color: Theme.text; font.family: Theme.sans; font.pixelSize: 13; elide: Text.ElideRight; Layout.maximumWidth: 180 }
                        Text { text: modelData.plays + "×"; color: Theme.textFaint; font.family: Theme.mono; font.pixelSize: 11 }
                    }
                }
            }
            ColumnLayout {
                Layout.fillWidth: true
                Layout.alignment: Qt.AlignTop
                spacing: 4
                Caption { text: "Músicas" }
                TrackList { Layout.fillWidth: true; tracks: page.lib.stats.tracks || [] }
            }
        }
    }

    component Stat: ColumnLayout {
        property int value
        property string caption
        spacing: 0
        Text { text: parent.value; color: Theme.text; font.family: Theme.serif; font.pixelSize: 40 }
        Text { text: parent.caption; color: Theme.textDim; font.family: Theme.sans; font.pixelSize: 12 }
    }
}
