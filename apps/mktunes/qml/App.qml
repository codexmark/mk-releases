import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell.Io

// A janela inteira: abas à esquerda (cada uma com sua pilha de páginas), o conteúdo,
// "Tocando agora" à direita e a barra do player embaixo.
Rectangle {
    id: app
    color: Theme.ground

    property alias trackMenu: trackMenuItem
    property alias nameDialog: nameDialogItem
    property int tab: 0
    // Janela larga mostra "Tocando agora" sempre (até ser fechado); estreita, só quando pedido.
    readonly property bool wide: app.width >= 1240
    property bool panelClosed: false
    property bool panelOpened: false
    readonly property bool panelShown: wide ? !panelClosed : panelOpened
    property bool fullNow: false
    // Acerto da letra em ms; volta a zero quando muda a música.
    property int lyricsOffset: 0
    readonly property string nowId: daemon.now.track ? (daemon.now.track.id || "") : ""
    onNowIdChanged: lyricsOffset = 0
    function showPanel(tab) {
        if (nowPanel.tab === tab && panelShown) { if (wide) panelClosed = true; else panelOpened = false; return; }
        nowPanel.tab = tab;
        panelClosed = false;
        panelOpened = true;
    }

    Daemon { id: daemon; onNotice: text => toast.show(text) }

    readonly property var tabs: [
        { title: "Início", glyph: Theme.icon.home },
        { title: "Buscar", glyph: Theme.icon.search },
        { title: "Rádio", glyph: Theme.icon.radio },
        { title: "Biblioteca", glyph: Theme.icon.library },
        { title: "Downloads", glyph: Theme.icon.download },
        { title: "Ajustes", glyph: Theme.icon.settings }
    ]

    function stack() { return stacks.children[app.tab]; }

    // Cada aba nasce na primeira visita: abrir o app não monta (nem baixa) as seis.
    readonly property var tabPages: [homePage, searchPage, radioPage, libraryPage, downloadsPage, settingsPage]
    function ensureTab(index) {
        const s = stacks.children[index];
        if (s.depth === 0) s.push(app.tabPages[index], {}, StackView.Immediate);
        return s;
    }
    Component.onCompleted: ensureTab(0)

    function push(component, properties) {
        const page = stack().push(component, properties);
        if (page) page.back.connect(app.back);
    }

    function back() {
        const s = stack();
        if (s.depth > 1) s.pop();
        else if (app.tab !== 0) app.tab = 0;
    }

    function goTo(index) {
        // Tocar de novo na aba em que já está volta ao começo dela.
        app.fullNow = false;
        const s = ensureTab(index);
        if (app.tab === index) s.pop(null);
        app.tab = index;
        if (index === 1) Qt.callLater(() => s.get(0).focusField());
    }

    /** Um cartão ou item de busca: música toca a prateleira a partir dela; o resto abre a página. */
    function activate(item, items) {
        if (item.kind === "song") {
            const songs = (items || []).filter(i => i.kind === "song");
            const index = Math.max(0, songs.findIndex(s => s.id === item.id));
            daemon.send({ cmd: "play", tracks: songs.length ? songs : [item], index: songs.length ? index : 0 });
        } else if (item.kind === "artist") {
            push(artistPage, { browseId: item.id });
        } else {
            push(collectionPage, { kind: item.kind, browseId: item.id });
        }
    }

    function confirm(title, detail, button, action) { confirmDialog.ask(title, detail, button, action); }

    // Controle de fora: `qs ipc -p <pasta> call mktunes <função>` (atalhos do Mk, a ilha, testes).
    IpcHandler {
        target: "mktunes"
        function tab(index: int): void { app.goTo(index); }
        function search(text: string): void { app.goTo(1); app.ensureTab(1).get(0).choose(text); }
        function open(kind: string, id: string): void { app.activate({ kind: kind, id: id }, []); }
        function back(): void { app.back(); }
        function toggle(): void { daemon.send({ cmd: "toggle" }); }
        function next(): void { daemon.send({ cmd: "next" }); }
        function previous(): void { daemon.send({ cmd: "previous" }); }
        function lyrics(): void { app.showPanel(1); }
        function queue(): void { app.showPanel(0); }
        function full(): void { app.fullNow = !app.fullNow; }
        function library(section: int): void { app.goTo(3); app.ensureTab(3).get(0).section = section; }
        // Joga a página atual como um gesto rápido de rolagem (negativo sobe).
        function scroll(velocity: int): void { const page = app.stack().currentItem; if (page && page.flick) page.flick(0, -velocity); }
    }

    Component { id: homePage; HomePage {} }
    Component { id: searchPage; SearchPage {} }
    Component { id: radioPage; RadioPage {} }
    Component { id: libraryPage; LibraryPage {} }
    Component { id: downloadsPage; DownloadsPage {} }
    Component { id: settingsPage; SettingsPage {} }
    Component { id: collectionPage; CollectionPage {} }
    Component { id: artistPage; ArtistPage {} }
    Component { id: trackListPage; TrackListPage {} }

    Shortcut { sequence: "Space"; enabled: !(app.Window.activeFocusItem instanceof TextInput); onActivated: daemon.send({ cmd: "toggle" }) }
    Shortcut { sequence: "Ctrl+F"; onActivated: { app.fullNow = false; app.goTo(1); } }
    Shortcut { sequence: "F11"; onActivated: app.fullNow = !app.fullNow }
    Shortcut { sequence: "Escape"; enabled: app.fullNow; onActivated: app.fullNow = false }
    Shortcut { sequences: ["Ctrl+Up"]; onActivated: daemon.send({ cmd: "volume", level: daemon.volume.level + 5 }) }
    Shortcut { sequences: ["Ctrl+Down"]; onActivated: daemon.send({ cmd: "volume", level: daemon.volume.level - 5 }) }
    Shortcut { sequence: "Ctrl+L"; onActivated: if (daemon.now.status !== "idle") daemon.send({ cmd: "like", track: daemon.now.track }) }
    Shortcut { sequence: "Ctrl+Right"; onActivated: daemon.send({ cmd: "next" }) }
    Shortcut { sequence: "Ctrl+Left"; onActivated: daemon.send({ cmd: "previous" }) }
    Shortcut { sequences: ["Alt+Left", "Backspace"]; enabled: !(app.Window.activeFocusItem instanceof TextInput); onActivated: app.back() }
    Repeater {
        model: 6
        Item {
            required property int index
            Shortcut { sequence: "Ctrl+" + (index + 1); onActivated: app.goTo(index) }
        }
    }
    // Botão "voltar" do mouse.
    TapHandler { acceptedButtons: Qt.BackButton; onTapped: app.back() }

    Ambient {
        x: 212
        width: parent.width - x
        height: 420
        source: daemon.now.artwork || ""
    }

    RowLayout {
        anchors.fill: parent
        anchors.bottomMargin: playerBar.height
        spacing: 0

        // Abas
        Rectangle {
            Layout.preferredWidth: 212
            Layout.fillHeight: true
            color: Theme.surface
            Rectangle { anchors.right: parent.right; width: 1; height: parent.height; color: Theme.hairline }
            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 16
                anchors.topMargin: 24
                spacing: 4
                ColumnLayout {
                    Layout.leftMargin: 8
                    Layout.bottomMargin: 24
                    spacing: 2
                    Text { text: "MKTUNES"; color: Theme.text; font.family: Theme.sans; font.pixelSize: 16; font.weight: Font.Bold; font.letterSpacing: 3 }
                    Caption { text: "Mk // music" }
                }
                Repeater {
                    model: app.tabs
                    Rectangle {
                        id: navItem
                        required property var modelData
                        required property int index
                        readonly property bool selected: app.tab === index
                        Layout.fillWidth: true
                        height: 40
                        radius: 8
                        color: selected ? Theme.raised : (navHover.hovered ? Theme.wash : "transparent")
                        Accessible.role: Accessible.PageTab
                        Accessible.name: modelData.title
                        Accessible.selected: selected
                        HoverHandler { id: navHover; cursorShape: Qt.PointingHandCursor }
                        TapHandler { onTapped: app.goTo(navItem.index) }
                        Rectangle { visible: navItem.selected; width: 3; height: 18; radius: 2; color: Theme.copper; anchors.verticalCenter: parent.verticalCenter }
                        Row {
                            anchors.verticalCenter: parent.verticalCenter
                            x: 14
                            spacing: 12
                            Text { text: navItem.modelData.glyph; color: navItem.selected ? Theme.copper : Theme.textDim; font.family: Theme.symbols; font.pixelSize: 17; anchors.verticalCenter: parent.verticalCenter }
                            Text { text: navItem.modelData.title; color: navItem.selected ? Theme.text : Theme.textSoft; font.family: Theme.sans; font.pixelSize: 14; font.weight: navItem.selected ? Font.DemiBold : Font.Normal; anchors.verticalCenter: parent.verticalCenter }
                        }
                    }
                }
                Item { Layout.fillHeight: true }
                Text {
                    Layout.leftMargin: 8
                    visible: !daemon.connected
                    text: "Iniciando o mktunesd…"
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
            }
        }

        // Conteúdo: uma pilha por aba, para cada aba lembrar onde estava.
        StackLayout {
            id: stacks
            Layout.fillWidth: true
            Layout.fillHeight: true
            currentIndex: app.tab
            TabStack {}
            TabStack {}
            TabStack {}
            TabStack {}
            TabStack {}
            TabStack {}
        }

        NowPanel {
            id: nowPanel
            Layout.preferredWidth: 340
            Layout.fillHeight: true
            Layout.margins: 16
            visible: app.panelShown
        }
    }

    Loader {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: playerBar.top
        active: app.fullNow
        visible: active
        sourceComponent: NowFull { onClose: app.fullNow = false }
    }

    PlayerBar {
        id: playerBar
        nowPanelTab: nowPanel.tab
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 92
    }

    TrackMenu { id: trackMenuItem; onNewPlaylist: track => nameDialogItem.askCreate(track) }

    NameDialog {
        id: nameDialogItem
        property var pendingTrack: null
        property string renaming: ""
        function askCreate(track) { pendingTrack = track; renaming = ""; ask("Nova playlist", "Criar", ""); }
        function askRename(id, current) { pendingTrack = null; renaming = id; ask("Renomear playlist", "Renomear", current); }
        onConfirmed: name => {
            if (renaming !== "") daemon.send({ cmd: "renamePlaylist", id: renaming, title: name });
            else daemon.send(pendingTrack ? { cmd: "createPlaylist", title: name, track: pendingTrack } : { cmd: "createPlaylist", title: name });
        }
    }

    ConfirmDialog { id: confirmDialog }
    Toast { id: toast; anchors.horizontalCenter: parent.horizontalCenter; anchors.bottom: playerBar.top; anchors.bottomMargin: 16 }

    component TabStack: StackView {
        clip: true
        pushEnter: Transition { NumberAnimation { property: "opacity"; from: 0; to: 1; duration: 140 } }
        pushExit: Transition { NumberAnimation { property: "opacity"; from: 1; to: 0; duration: 100 } }
        popEnter: Transition { NumberAnimation { property: "opacity"; from: 0; to: 1; duration: 140 } }
        popExit: Transition { NumberAnimation { property: "opacity"; from: 1; to: 0; duration: 100 } }
    }
}
