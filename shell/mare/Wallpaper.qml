pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// Papel de parede e a cor que sai dele. O Maré desenha o papel de parede
// (Backdrop.qml) e tinge a interface com o matiz da cor principal da imagem.
Singleton {
    id: root

    readonly property string dir: Quickshell.env("HOME") + "/Pictures/Wallpapers"
    readonly property string storePath: Quickshell.statePath("wallpaper.json")

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
        list.running = true;
    }

    function save() {
        store.setText(JSON.stringify({ path: current, primary: primary }, null, 2) + "\n");
    }

    Component.onCompleted: refresh()

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
        command: ["find", root.dir, "-maxdepth", "1", "-type", "f",
            "(", "-iname", "*.jpg", "-o", "-iname", "*.jpeg", "-o", "-iname", "*.png", "-o", "-iname", "*.webp", ")"]
        stdout: StdioCollector {
            onStreamFinished: root.files = text.split("\n").filter(s => s.length > 0).sort()
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
