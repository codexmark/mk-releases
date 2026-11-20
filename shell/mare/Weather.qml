pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// Clima via scripts/mare-weather, renovado a cada 30 min.
Singleton {
    id: root

    property var data: null
    readonly property bool ready: data !== null && data.error === undefined

    // Códigos WMO -> ícone (Nerd Font) e descrição.
    function icon(code, day) {
        if (code === 0)
            return day === false ? "󰖔" : "󰖙";
        if (code <= 2)
            return day === false ? "󰼱" : "󰖕";
        if (code === 3)
            return "󰖐";
        if (code <= 48)
            return "󰖑";
        if (code <= 67 || (code >= 80 && code <= 82))
            return "󰖗";
        if (code <= 77 || code === 85 || code === 86)
            return "󰖘";
        return "󰖓";
    }

    function describe(code) {
        if (code === 0)
            return "céu limpo";
        if (code <= 2)
            return "poucas nuvens";
        if (code === 3)
            return "nublado";
        if (code <= 48)
            return "neblina";
        if (code <= 57)
            return "garoa";
        if (code <= 67)
            return "chuva";
        if (code <= 77)
            return "neve";
        if (code <= 82)
            return "pancadas de chuva";
        if (code <= 86)
            return "neve";
        return "tempestade";
    }

    function refresh() {
        proc.running = true;
    }

    Timer {
        interval: 30 * 60 * 1000
        repeat: true
        running: true
        triggeredOnStart: true
        onTriggered: root.refresh()
    }

    Process {
        id: proc
        command: ["bash", Quickshell.shellPath("scripts/mare-weather")]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    root.data = JSON.parse(text);
                } catch (e) {}
            }
        }
    }
}
