pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland

// Layout do teclado principal. A barra só mostra o indicador quando há mais
// de um layout configurado (ex.: kb_layout = "br,us").
Singleton {
    id: root

    property var layouts: []
    property int index: 0
    readonly property bool multiple: layouts.length > 1
    readonly property string current: (layouts[index] || "").toUpperCase()

    function next() {
        Quickshell.execDetached(["hyprctl", "switchxkblayout", "all", "next"]);
    }

    Component.onCompleted: devices.running = true

    Connections {
        target: Hyprland

        function onRawEvent(event) {
            if (event.name === "activelayout" || event.name === "configreloaded")
                devices.running = true;
        }
    }

    Process {
        id: devices
        command: ["hyprctl", "devices", "-j"]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    const kb = JSON.parse(text).keyboards.find(k => k.main);
                    if (kb) {
                        root.layouts = kb.layout.split(",").map(s => s.trim()).filter(s => s.length > 0);
                        root.index = kb.active_layout_index;
                    }
                } catch (e) {}
            }
        }
    }
}
