pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// Papel de parede e a cor que sai dele. O Maré desenha o papel de parede
// (Backdrop.qml) e tinge a interface com o matiz da cor principal da imagem.
Singleton {
    id: root

    // <pasta de imagens do sistema>/Wallpapers: ~/Imagens em português,
    // ~/Pictures em inglês (xdg-user-dir). Vazio até a pasta ser achada.
    property string dir: ""
    // Os que vêm com o Mk (system/mk/art/make-wallpapers.py), ao lado do shell.
    readonly property string bundled: Quickshell.shellPath("wallpapers")
    readonly property string storePath: Quickshell.statePath("wallpaper.json")

    // O papel de parede de fábrica: uma instalação nova nasce com ele. Vale
    // uma vez só, e nunca por cima de uma escolha. Quem já usava o Mk não o
    // recebe: a migração de usuário abaixo grava a marca para essa gente. Como
    // o mk-update reinicia o Maré antes de rodar as migrações, o de fábrica só
    // é aplicado com a migração já constando como feita (uma instalação nova
    // nasce com todas marcadas), do mesmo jeito que as boas-vindas.
    readonly property string factory: bundled + "/00-6-mare.jpg"
    readonly property string factoryMark: Settings.dir + "/.papel-de-fabrica"
    readonly property string migration: "20261128-papel-de-fabrica.sh"
    readonly property string gatePath: (Quickshell.env("XDG_STATE_HOME") || (Quickshell.env("HOME") + "/.local/state"))
        + "/mk/migrations/" + migration
    property bool storeChecked: false
    property bool markChecked: false
    property bool marked: false
    property bool gateChecked: false
    property bool fresh: false

    function factoryCheck() {
        if (!storeChecked || !markChecked || !gateChecked)
            return;
        if (fresh && !marked) {
            marked = true;
            mark.setText("papel de parede de fábrica aplicado\n");
            if (current === "")
                set(factory);
        }
    }

    property string current: ""
    property string primary: ""
    property var files: []
    // Matiz de 0 a 1, ou -1 sem papel de parede (cores padrão do Maré).
    readonly property real hue: primary !== "" ? Math.max(0, Qt.color(primary).hslHue) : -1

    function set(path) {
        if (path === current)
            return;
        current = path;
        palette.command = ["bash", Quickshell.shellPath("scripts/mare-palette"), path];
        palette.running = true;
    }

    function random() {
        if (files.length > 0)
            set(files[Math.floor(Math.random() * files.length)]);
    }

    function refresh() {
        if (dir !== "")
            list.running = true;
    }

    function save() {
        store.setText(JSON.stringify({ path: current, primary: primary }, null, 2) + "\n");
    }

    Component.onCompleted: where.running = true

    Process {
        id: where
        command: ["sh", "-c", "d=$(xdg-user-dir PICTURES 2>/dev/null); "
            + "{ [ -n \"$d\" ] && [ \"$d\" != \"$HOME\" ]; } || d=$HOME/Pictures; "
            + "mkdir -p \"$d/Wallpapers\"; printf %s \"$d/Wallpapers\""]
        stdout: StdioCollector {
            onStreamFinished: {
                root.dir = text.trim();
                root.refresh();
            }
        }
    }

    Process {
        id: palette
        stdout: StdioCollector {
            onStreamFinished: {
                const c = text.trim();
                root.primary = /^#[0-9a-fA-F]{6}$/.test(c) ? c : "";
                root.save();
            }
        }
    }

    Process {
        id: list
        // Primeiro os seus, depois os do Mk.
        command: ["find", root.dir, root.bundled, "-maxdepth", "1", "-type", "f",
            "(", "-iname", "*.jpg", "-o", "-iname", "*.jpeg", "-o", "-iname", "*.png", "-o", "-iname", "*.webp", ")"]
        stdout: StdioCollector {
            onStreamFinished: {
                const all = text.split("\n").filter(s => s.length > 0);
                const mine = all.filter(f => !f.startsWith(root.bundled + "/")).sort();
                root.files = mine.concat(all.filter(f => f.startsWith(root.bundled + "/")).sort());
            }
        }
    }

    FileView {
        id: store
        path: root.storePath
        printErrors: false
        onLoaded: {
            try {
                const s = JSON.parse(text());
                root.current = s.path || "";
                root.primary = s.primary || "";
            } catch (e) {}
            root.storeChecked = true;
            root.factoryCheck();
        }
        onLoadFailed: {
            root.storeChecked = true;
            root.factoryCheck();
        }
    }

    FileView {
        id: mark
        path: root.factoryMark
        printErrors: false
        onLoaded: {
            root.marked = true;
            root.markChecked = true;
            root.factoryCheck();
        }
        onLoadFailed: {
            root.markChecked = true;
            root.factoryCheck();
        }
    }

    FileView {
        path: root.gatePath
        printErrors: false
        onLoaded: {
            root.fresh = true;
            root.gateChecked = true;
            root.factoryCheck();
        }
        onLoadFailed: {
            root.gateChecked = true;
            root.factoryCheck();
        }
    }

    IpcHandler {
        target: "wallpaper"

        function set(path: string): void {
            root.set(path);
        }

        function random(): void {
            root.random();
        }

        function get(): string {
            return root.current;
        }
    }
}
