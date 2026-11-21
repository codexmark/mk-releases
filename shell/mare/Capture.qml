pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// Capturas e gravação da tela. As capturas rodam scripts/mare-shot depois de
// o painel fechar (para ele não sair na foto). A gravação usa o
// gpu-screen-recorder, com o som do sistema; enquanto grava, a ilha mostra o
// tempo e um clique nela para.
Singleton {
    id: root

    readonly property bool recording: recorder.running
    property int seconds: 0
    property string file: ""

    function shot(mode) {
        pending = mode;
        delay.restart();
    }

    function toggleRecording() {
        if (recorder.running) {
            stop.running = true;
        } else {
            where.running = true;
        }
    }

    function clock() {
        const m = Math.floor(seconds / 60), s = seconds % 60;
        return String(m).padStart(2, "0") + ":" + String(s).padStart(2, "0");
    }

    property string pending: ""

    Timer {
        id: delay
        interval: Math.max(120, Motion.exit + 80)
        onTriggered: Quickshell.execDetached(["bash", Quickshell.shellPath("scripts/mare-shot"), root.pending])
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
                root.seconds = 0;
                recorder.command = ["sh", "-c", "mkdir -p \"$1\" && exec gpu-screen-recorder -w screen -f 60 -a default_output -o \"$2\"",
                    "sh", dir, root.file];
                recorder.running = true;
            }
        }
    }

    Process {
        id: recorder
        onExited: (code, status) => {
            Quickshell.execDetached(["notify-send", "-a", "Mk", code === 0 ? "Gravação salva" : "A gravação parou com erro",
                root.file.replace(Quickshell.env("HOME"), "~")]);
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
