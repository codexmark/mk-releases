pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// O som que está tocando, em faixas dos graves aos agudos (scripts/mare-cava).
// A ilha usa para ondular a borda de baixo com a música. Em silêncio o cava
// repete a mesma linha, que é ignorada: nada é redesenhado.
Singleton {
    id: root

    readonly property int bands: 28
    readonly property bool enabled: Settings.get("island.sound")

    // 0 a 1 por faixa, e a média.
    property var levels: []
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
        const out = [];
        let sum = 0;
        for (let i = 0; i < bands; i++) {
            const v = Math.max(0, Math.min(1, (parseInt(parts[i]) || 0) / 100));
            // Meio caminho entre o quadro anterior e este: movimento de líquido.
            const prev = levels[i] || 0;
            const s = v > prev ? prev + (v - prev) * 0.7 : prev + (v - prev) * 0.35;
            out.push(s);
            sum += s;
        }
        levels = out;
        energy = sum / bands;
    }

    onEnabledChanged: {
        if (!enabled) {
            levels = [];
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
