import QtQuick
import Quickshell
import Quickshell.Io

// Fala com o mktunesd: uma linha JSON por mensagem. Guarda o último estado de cada área e
// repassa as mensagens avulsas (páginas, avisos). Sem daemon, tenta de novo a cada 2 s.
Item {
    id: root

    property string socketPath: (Quickshell.env("XDG_RUNTIME_DIR") || "/tmp") + "/mktunes/mktunes.sock"
    readonly property bool connected: sock !== null && sock.connected

    property var now: ({ status: "idle", track: {}, title: "", artist: "", artwork: "", upNext: [], repeat: "off", liked: false, download: "", sleep: "", speed: 1, stopAfterCurrent: false, live: false })
    property var progress: ({ position: 0, duration: 0 })
    property var search: ({ status: "idle", query: "", filter: "all", tracks: [], items: [], top: null, sections: [], recent: [], suggestions: [] })
    property var home: ({ status: "loading", chips: [], chip: "", shelves: [], hasMore: false, loadingMore: false, moreFailed: false, recent: [] })
    property var library: ({ liked: [], likedIds: [], playlists: [], saved: [], smart: [], period: "week", stats: {} })
    property var downloads: ({ jobs: [], folder: "" })
    property var radio: ({ status: "idle", query: "", stations: [] })
    property var lyrics: ({ status: "none" })
    property var volume: ({ level: 100 })
    property var local: ({ status: "idle", tracks: [] })
    property var settings: ({ autoplay: true, normalizeVolume: true, audioQuality: "high", downloadWorkers: 1, maxWorkers: 4, backupsDir: "", version: "" })

    // Ids curtidos num objeto, para cada linha saber se é curtida sem varrer a lista.
    property var likedSet: ({})

    signal message(var data)
    signal notice(string text)

    property var sock: null
    // Pedidos feitos antes de o socket abrir (a janela costuma abrir antes do daemon): saem na conexão.
    property var waiting: []

    function send(command) {
        const line = JSON.stringify(command);
        if (!root.connected) {
            if (root.waiting.indexOf(line) < 0) root.waiting.push(line);
            return false;
        }
        sock.write(line + "\n");
        sock.flush();
        return true;
    }

    onConnectedChanged: {
        if (!connected) return;
        const pending = root.waiting;
        root.waiting = [];
        for (const line of pending) sock.write(line + "\n");
        sock.flush();
    }

    function isLiked(id) { return root.likedSet[id] === true; }

    function handleLine(line) {
        let m;
        try { m = JSON.parse(line); } catch (e) { return; }
        switch (m.type) {
        case "now": root.now = m; break;
        case "progress": root.progress = m; break;
        case "search": root.search = m; break;
        case "home": root.home = m; break;
        case "library": {
            const set = {};
            for (const id of m.likedIds) set[id] = true;
            root.likedSet = set;
            root.library = m;
            break;
        }
        case "downloads": root.downloads = m; break;
        case "radio": root.radio = m; break;
        case "lyrics": root.lyrics = m; break;
        case "settings": root.settings = m; break;
        case "volume": root.volume = m; break;
        case "local": root.local = m; break;
        case "notice": root.notice(m.text); break;
        default: root.message(m);
        }
    }

    function reconnect() {
        if (root.connected) return;
        // Um Socket que falhou não reconecta: cria um novo.
        if (root.sock) root.sock.destroy();
        root.sock = socketComponent.createObject(root);
        root.sock.connected = true;
    }

    Component.onCompleted: reconnect()

    Component {
        id: socketComponent
        Socket {
            path: root.socketPath
            parser: SplitParser { onRead: line => root.handleLine(line) }
        }
    }

    Timer {
        interval: 2000
        running: !root.connected
        repeat: true
        onTriggered: root.reconnect()
    }
}
