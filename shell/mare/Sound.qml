pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Wave.js" as Wave

// O som que está tocando, em faixas dos graves aos agudos (scripts/mare-cava).
// A ilha usa para ondular a borda de baixo com a música, com um rastro atrás
// (as contas estão no Wave.js). Em silêncio o cava
// repete a mesma linha, que é ignorada: nada é redesenhado.
Singleton {
    id: root

    readonly property int bands: 28
    readonly property bool enabled: Settings.get("island.sound")

    // 0 a 1 por faixa, e a média.
    property var levels: []
    // O mesmo, com atraso: a segunda linha da ilha.
    property var trail: []
    property real energy: 0
    // Sobe quando há som e desce devagar quando para (a onda não some de estalo).
    property real presence: energy > 0.015 ? 1 : 0

    Behavior on presence {
        NumberAnimation { duration: 600; easing.type: Easing.OutCubic }
    }

    property string last: ""

    function feed(line) {
        if (line === last)
            return;
        last = line;
        const parts = line.split(";");
        const raw = [];
        for (let i = 0; i < bands; i++)
            raw.push((parseInt(parts[i]) || 0) / 100);
        levels = Wave.smooth(levels, raw, bands);
        trail = Wave.trail(trail, levels, bands);
        let sum = 0;
        for (let i = 0; i < bands; i++)
            sum += levels[i];
        energy = sum / bands;
    }

    onEnabledChanged: {
        if (!enabled) {
            levels = [];
            trail = [];
            energy = 0;
            last = "";
        }
    }

    Process {
        id: cava
        running: root.enabled && !retry.running
        command: ["bash", Quickshell.shellPath("scripts/mare-cava"), String(root.bands)]
        stdout: SplitParser {
            onRead: line => root.feed(line)
        }
        // Sem o cava instalado (127), tenta de novo de vez em quando; se ele
        // caiu por outro motivo, logo.
        onExited: (code, status) => {
            root.levels = [];
            root.trail = [];
            root.energy = 0;
            retry.interval = code === 127 ? 60000 : 4000;
            if (root.enabled)
                retry.restart();
        }
    }

    Timer {
        id: retry
    }
}
