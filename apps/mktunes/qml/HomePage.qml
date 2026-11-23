import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Início: saudação, moods, tocadas recentemente e as prateleiras do catálogo, com mais ao rolar.
ListView {
    id: page
    property bool canGoBack: false
    signal back()

    readonly property var home: daemon.home
    clip: true
    spacing: 36
    cacheBuffer: 1200
    boundsBehavior: Flickable.StopAtBounds
    leftMargin: 32
    rightMargin: 32
    topMargin: 28
    bottomMargin: 32
    ScrollBar.vertical: ScrollBar {}

    // Só troca o modelo quando as prateleiras mudam de verdade: o resto do estado (tocadas
    // recentemente, "carregando mais") não pode jogar a rolagem para o topo.
    property string shelvesKey: ""
    property var shelves: []
    Connections {
        target: daemon
        function onHomeChanged() { page.syncShelves(); }
    }
    function syncShelves() {
        const key = JSON.stringify(daemon.home.shelves);
        if (key !== page.shelvesKey) {
            page.shelvesKey = key;
            page.shelves = daemon.home.shelves;
        }
    }
    Component.onCompleted: { syncShelves(); daemon.send({ cmd: "home" }); }
    onVisibleChanged: if (visible) daemon.send({ cmd: "home" })

    model: page.shelves
    // O cabeçalho cresce depois de aberto (retomar, recentes): sem o ouvinte ter rolado, a
    // página continua no topo em vez de aparecer rolada para baixo.
    property bool scrolled: false
    onMovementStarted: scrolled = true
    onAtYEndChanged: if (atYEnd && home.hasMore && !home.loadingMore && !home.moreFailed) daemon.send({ cmd: "homeMore" })

    function greeting() {
        const h = new Date().getHours();
        return h < 5 ? "Boa noite" : h < 12 ? "Bom dia" : h < 18 ? "Boa tarde" : "Boa noite";
    }

    header: ColumnLayout {
        width: page.width - page.leftMargin - page.rightMargin
        spacing: 20
        onHeightChanged: if (!page.scrolled) Qt.callLater(() => page.contentY = -height - page.topMargin)
        PageHeader { label: "01 · Início · Ouvir"; title: page.greeting() }

        // Moods numa linha só, rolando de lado (roda do mouse em cima dela rola de lado).
        ListView {
            id: chips
            Layout.fillWidth: true
            Layout.preferredHeight: 34
            visible: page.home.chips.length > 0
            orientation: ListView.Horizontal
            spacing: 8
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            model: page.home.chips
            delegate: Chip {
                required property var modelData
                text: modelData.title
                selected: page.home.chip === modelData.token
                onClicked: daemon.send({ cmd: "homeChip", token: modelData.token })
            }
            WheelHandler {
                acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
                onWheel: event => chips.contentX = Math.max(0, Math.min(chips.contentWidth - chips.width, chips.contentX - (event.angleDelta.y || event.angleDelta.x)))
            }
            // Esmaece a borda direita quando há mais moods para o lado.
            Rectangle {
                anchors.right: parent.right
                width: 48
                height: parent.height
                visible: !chips.atXEnd
                gradient: Gradient {
                    orientation: Gradient.Horizontal
                    GradientStop { position: 0; color: "transparent" }
                    GradientStop { position: 1; color: Theme.ground }
                }
            }
        }

        // A música que ficou pausada, para retomar de onde parou.
        ColumnLayout {
            Layout.fillWidth: true
            visible: daemon.now.status === "paused" && !daemon.now.live
            spacing: 12
            Caption { text: "Continuar ouvindo" }
            ContinueCard { Layout.fillWidth: true }
        }

        ColumnLayout {
            visible: page.home.status === "unavailable"
            spacing: 12
            Hint { text: "Não foi possível carregar o início. Confira a internet." }
            MkButton { text: "Tentar de novo"; glyph: Theme.icon.retry; onClicked: daemon.send({ cmd: "homeRetry" }) }
        }
        BusyIndicator { visible: page.home.status === "loading" && page.shelves.length === 0; running: visible }

        // Tocadas recentemente: ladrilhos compactos, duas ou três colunas conforme a largura.
        ColumnLayout {
            Layout.fillWidth: true
            Layout.bottomMargin: 16
            visible: page.home.recent.length > 0
            spacing: 12
            Caption { text: "Tocadas recentemente" }
            GridLayout {
                id: recentGrid
                Layout.fillWidth: true
                columns: width >= 900 ? 3 : 2
                columnSpacing: 10
                rowSpacing: 10
                Repeater {
                    model: page.home.recent.slice(0, recentGrid.columns * 2)
                    RecentTile {
                        required property var modelData
                        required property int index
                        Layout.fillWidth: true
                        Layout.preferredWidth: 1
                        track: modelData
                        current: daemon.now.track.id === modelData.id
                        onPlay: daemon.send({ cmd: "play", tracks: page.home.recent, index: index })
                        onMenu: anchorItem => app.trackMenu.openFor(modelData, anchorItem)
                    }
                }
            }
        }
    }

    delegate: ShelfRow {
        required property var modelData
        width: ListView.view.width - page.leftMargin - page.rightMargin
        title: modelData.title
        items: modelData.items
        onActivated: (item, items) => app.activate(item, items)
        onMenu: (item, anchorItem) => app.trackMenu.openFor(item, anchorItem)
    }

    footer: Item {
        width: page.width - page.leftMargin - page.rightMargin
        height: page.home.loadingMore || page.home.moreFailed ? 72 : 0
        BusyIndicator { anchors.centerIn: parent; visible: page.home.loadingMore; running: visible }
        MkButton {
            anchors.centerIn: parent
            visible: page.home.moreFailed
            text: "Tentar carregar mais"
            glyph: Theme.icon.retry
            onClicked: daemon.send({ cmd: "homeMore" })
        }
    }
}
