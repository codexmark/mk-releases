pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Hyprland
import Quickshell.Io

// Capturas e gravação da tela. As capturas rodam scripts/mare-shot depois de
// o painel fechar (para ele não sair na foto). A gravação usa o
// gpu-screen-recorder, com o som do sistema; enquanto grava, a ilha mostra o
// tempo e um clique nela para. Antes de tudo, uma sondagem (só a ajuda do
// gravador) diz se dá para gravar; se não der, `reason` diz por quê.
Singleton {
    id: root

    readonly property bool recording: recorder.running
    property int seconds: 0
    property string file: ""

    property bool available: true
    property string reason: ""
    property string lastLine: ""
    property string errLine: ""
    property string target: "screen"

    function shot(mode) {
        pending = mode;
        delay.restart();
    }

    function toggleRecording() {
        if (recorder.running) {
            stop.running = true;
        } else if (!available) {
            Quickshell.execDetached(["notify-send", "-a", "Mk", "Não dá para gravar", reason]);
        } else if (!where.running) {
            where.running = true;
        }
    }

    function clock() {
        const m = Math.floor(seconds / 60), s = seconds % 60;
        return String(m).padStart(2, "0") + ":" + String(s).padStart(2, "0");
    }

    property string pending: ""

    function start(target) {
        root.target = target;
        root.seconds = 0;
        root.lastLine = "";
        root.errLine = "";
        recorder.command = ["sh", "-c", "mkdir -p \"$(dirname \"$1\")\" && exec gpu-screen-recorder -w \"$2\" -f 60 -a default_output -o \"$1\"",
            "sh", root.file, target];
        recorder.running = true;
    }

    Timer {
        id: delay
        interval: Math.max(120, Motion.exit + 80)
        onTriggered: Quickshell.execDetached(["bash", Quickshell.shellPath("scripts/mare-shot"), root.pending])
    }

    Process {
        id: probe
        running: true
        command: ["bash", Quickshell.shellPath("scripts/mare-shot"), "gravacao"]
        stdout: StdioCollector {
            onStreamFinished: {
                root.reason = text.trim().split("\n")[0];
                root.available = root.reason === "";
            }
        }
    }

    // A pasta das capturas (a mesma dos prints), depois começa a gravar.
    Process {
        id: where
        command: ["bash", Quickshell.shellPath("scripts/mare-shot"), "pasta"]
        stdout: StdioCollector {
            onStreamFinished: {
                const dir = text.trim();
                const d = new Date();
                const p = n => String(n).padStart(2, "0");
                root.file = dir + "/" + d.getFullYear() + "-" + p(d.getMonth() + 1) + "-" + p(d.getDate())
                    + "_" + p(d.getHours()) + "-" + p(d.getMinutes()) + "-" + p(d.getSeconds()) + ".mp4";
                // O monitor em foco, se o Hyprland disser qual é; senão o primeiro.
                const mon = Hyprland.focusedMonitor;
                root.start(mon !== null && /^[\w-]+$/.test(mon.name) ? mon.name : "screen");
            }
        }
    }

    Process {
        id: recorder
        // O gravador fala pelo stderr. Numa morte vale a última linha com cara
        // de erro (depois dela podem vir listas e o uso); sem nenhuma, a última.
        stderr: SplitParser {
            onRead: line => {
                const l = line.trim();
                if (l === "" || l.startsWith("update fps"))
                    return;
                if (/error|not found|not a valid|failed|invalid|unable|can't|cannot/i.test(l))
                    root.errLine = l;
                root.lastLine = l;
            }
        }
        onExited: (code, status) => {
            const path = root.file.replace(Quickshell.env("HOME"), "~");
            if (code === 0)
                Quickshell.execDetached(["notify-send", "-a", "Mk", "Gravação salva", path]);
            else if (root.target !== "screen" && root.seconds < 2)
                // O gravador recusou o monitor em foco: uma vez só, a tela padrão.
                Qt.callLater(root.start, "screen");
            else
                Quickshell.execDetached(["notify-send", "-a", "Mk", "A gravação parou com erro", root.errLine || root.lastLine || path]);
        }
    }

    // O gpu-screen-recorder fecha o arquivo direito com SIGINT.
    Process {
        id: stop
        command: ["pkill", "-INT", "-x", "gpu-screen-reco"]
    }

    Timer {
        interval: 1000
        repeat: true
        running: recorder.running
        onTriggered: root.seconds += 1
    }
}
