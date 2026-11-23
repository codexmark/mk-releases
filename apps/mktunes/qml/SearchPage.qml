import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Buscar: campo com sugestões, filtros, buscas recentes e os resultados.
ListView {
    id: page
    property bool canGoBack: false
    signal back()
    // O campo mora no cabeçalho da lista: as funções da página o alcançam por aqui.
    readonly property var field: headerItem ? headerItem.field : null
    function focusField() { if (field) { field.forceActiveFocus(); field.selectAll(); } }

    readonly property var search: daemon.search
    readonly property bool songs: search.filter === "songs"
    readonly property bool everything: search.filter === "all"
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

    model: songs && search.status === "results" ? search.tracks : []
    delegate: TrackRow {
        required property var modelData
        required property int index
        width: ListView.view.width - page.leftMargin - page.rightMargin
        track: modelData
        current: daemon.now.track.id === modelData.id
        onPlay: daemon.send({ cmd: "play", tracks: page.search.tracks, index: index })
        onMenu: anchorItem => app.trackMenu.openFor(modelData, anchorItem)
    }

    readonly property var filters: [
        { key: "all", title: "Tudo" }, { key: "songs", title: "Músicas" }, { key: "albums", title: "Álbuns" },
        { key: "artists", title: "Artistas" }, { key: "playlists", title: "Playlists" }
    ]

    header: ColumnLayout {
        property alias field: field
        width: page.width - page.leftMargin - page.rightMargin
        spacing: 16
        PageHeader { label: "02 · Buscar"; title: "" }

        MkField {
            id: field
            Layout.fillWidth: true
            font.pixelSize: 16
            placeholderText: "Buscar faixas, artistas, álbuns"
            onTextEdited: daemon.send({ cmd: "suggest", text: text })
            onAccepted: if (text.trim() !== "") daemon.send({ cmd: "search", query: text.trim() })
            Keys.onEscapePressed: { text = ""; daemon.send({ cmd: "clearSearch" }); }
            Keys.onDownPressed: if (suggestions.count > 0) { suggestions.forceActiveFocus(); suggestions.currentIndex = 0; }
            Glyph {
                anchors.right: parent.right
                anchors.rightMargin: 6
                anchors.verticalCenter: parent.verticalCenter
                visible: field.text !== ""
                glyph: Theme.icon.close
                label: "Limpar"
                size: 30
                onClicked: { field.text = ""; daemon.send({ cmd: "clearSearch" }); field.forceActiveFocus(); }
            }
        }

        ListView {
            id: suggestions
            Layout.fillWidth: true
            Layout.preferredHeight: contentHeight
            visible: count > 0 && field.text.trim().length >= 2
            interactive: false
            model: page.search.suggestions
            delegate: Rectangle {
                required property string modelData
                required property int index
                width: ListView.view.width
                height: 36
                radius: 6
                color: hov.hovered || (suggestions.activeFocus && suggestions.currentIndex === index) ? Theme.wash : "transparent"
                HoverHandler { id: hov; cursorShape: Qt.PointingHandCursor }
                TapHandler { onTapped: page.choose(modelData) }
                Row {
                    anchors.verticalCenter: parent.verticalCenter
                    x: 12
                    spacing: 12
                    Text { text: Theme.icon.search; color: Theme.textFaint; font.family: Theme.symbols; font.pixelSize: 14 }
                    Text { text: modelData; color: Theme.text; font.family: Theme.sans; font.pixelSize: 14 }
                }
            }
            Keys.onReturnPressed: page.choose(model[currentIndex])
            Keys.onEscapePressed: field.forceActiveFocus()
        }

        Flow {
            Layout.fillWidth: true
            spacing: 8
            Repeater {
                model: page.filters
                Chip {
                    required property var modelData
                    text: modelData.title
                    selected: page.search.filter === modelData.key
                    onClicked: daemon.send({ cmd: "search", filter: modelData.key, query: field.text.trim() })
                }
            }
        }

        Hint {
            Layout.fillWidth: true
            visible: text !== ""
            text: {
                switch (page.search.status) {
                case "loading": return "Buscando…";
                case "nothing": return "Nada encontrado para \"" + page.search.query + "\".";
                case "unavailable": return "Não foi possível buscar agora. Confira a internet.";
                default: return "";
                }
            }
        }

        // Recentes, enquanto nada foi buscado.
        ColumnLayout {
            Layout.fillWidth: true
            visible: page.search.status === "idle" && page.search.recent.length > 0
            spacing: 4
            RowLayout {
                Layout.fillWidth: true
                Caption { text: "Buscas recentes" }
                Item { Layout.fillWidth: true }
                MkButton { text: "Limpar"; onClicked: daemon.send({ cmd: "clearRecent" }) }
            }
            Repeater {
                model: page.search.recent
                Rectangle {
                    required property string modelData
                    Layout.fillWidth: true
                    height: 40
                    radius: 6
                    color: rh.hovered ? Theme.wash : "transparent"
                    HoverHandler { id: rh; cursorShape: Qt.PointingHandCursor }
                    TapHandler { onTapped: page.choose(modelData) }
                    Row {
                        anchors.verticalCenter: parent.verticalCenter
                        x: 12
                        spacing: 12
                        Text { text: Theme.icon.retry; color: Theme.textFaint; font.family: Theme.symbols; font.pixelSize: 14 }
                        Text { text: modelData; color: Theme.text; font.family: Theme.sans; font.pixelSize: 14 }
                    }
                }
            }
        }

        // Tudo: o melhor resultado ao lado das primeiras músicas, depois uma prateleira por tipo.
        RowLayout {
            Layout.fillWidth: true
            visible: page.everything && page.search.status === "results"
            spacing: 24
            ColumnLayout {
                Layout.preferredWidth: 360
                Layout.maximumWidth: 360
                Layout.alignment: Qt.AlignTop
                visible: page.search.top !== null && page.search.top !== undefined
                spacing: 12
                Caption { text: "Melhor resultado" }
                TopResultCard {
                    Layout.fillWidth: true
                    item: page.search.top || ({})
                    onActivated: {
                        const top = page.search.top;
                        if (top.kind === "song") {
                            const at = page.search.tracks.findIndex(t => t.id === top.id);
                            daemon.send({ cmd: "play", tracks: page.search.tracks, index: Math.max(0, at) });
                        } else app.activate(top, []);
                    }
                }
            }
            ColumnLayout {
                Layout.fillWidth: true
                Layout.alignment: Qt.AlignTop
                visible: page.search.tracks.length > 0
                spacing: 12
                RowLayout {
                    Layout.fillWidth: true
                    Caption { text: "Músicas" }
                    Item { Layout.fillWidth: true }
                    MkButton {
                        visible: page.search.tracks.length > 4
                        text: "Ver todas"
                        onClicked: daemon.send({ cmd: "search", filter: "songs", query: page.search.query })
                    }
                }
                TrackList { Layout.fillWidth: true; tracks: page.everything ? page.search.tracks.slice(0, 4) : [] }
            }
        }
        Repeater {
            model: page.everything && page.search.status === "results" ? page.search.sections : []
            ShelfRow {
                required property var modelData
                Layout.fillWidth: true
                Layout.topMargin: 12
                title: modelData.title
                items: modelData.items
                onActivated: (item, items) => app.activate(item, items)
            }
        }

        // Páginas (álbuns, artistas, playlists) em grade.
        Flow {
            Layout.fillWidth: true
            visible: !page.songs && !page.everything && page.search.status === "results"
            spacing: 20
            Repeater {
                model: page.songs || page.everything ? [] : page.search.items
                Card {
                    required property var modelData
                    item: modelData
                    size: 160
                    onActivated: app.activate(modelData, page.search.items)
                }
            }
        }
    }

    function choose(text) {
        if (field) field.text = text;
        if (text.trim() === "") daemon.send({ cmd: "clearSearch" });
        else daemon.send({ cmd: "search", query: text });
        focusField();
    }
}
