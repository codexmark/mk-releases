pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// Atualizações do Mk, do Arch e do AUR (scripts/mare-updates, sem sudo).
// Procura sozinho a cada 3 horas, se os Ajustes deixarem; quem aplica é o
// mk-update, num terminal, porque o pacman pede a senha.
Singleton {
    id: root

    property var info: null
    property bool checking: false
    readonly property int systemCount: info ? info.system.length : 0
    readonly property int aurCount: info ? info.aur.length : 0
    readonly property bool mkAvailable: info ? info.mk.available : false
    readonly property int migrations: info && info.migrations ? info.migrations : 0
    readonly property int count: systemCount + aurCount + (mkAvailable ? 1 : 0) + migrations
    readonly property string error: info ? info.error : ""

    function check() {
        if (!checking) {
            checking = true;
            proc.running = true;
        }
    }

    // Abre o mk-update num terminal; procura de novo quando ele fecha.
    function run() {
        runner.running = true;
    }

    function ago() {
        if (!info)
            return "ainda não procurou";
        const min = Math.round((Date.now() / 1000 - info.checked) / 60);
        return min < 1 ? "agora há pouco" : min < 60 ? "há " + min + " min"
            : "há " + Math.round(min / 60) + " h";
    }

    Process {
        id: proc
        command: ["bash", Quickshell.shellPath("scripts/mare-updates")]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    root.info = JSON.parse(text);
                } catch (e) {}
                root.checking = false;
            }
        }
    }

    // O mk-update instalado no sistema; sem ele (sessão de teste), o do repositório.
    Process {
        id: runner
        command: ["sh", "-c", "u=$(command -v mk-update || echo /usr/share/mk/bin/mk-update); "
            + "exec ghostty --config-default-files=false --config-file='?" + Projects.terminalConfig + "' "
            + "--title='Mk · atualizações' -e \"$u\""]
        onExited: root.check()
    }

    // Primeira procura 2 min depois do login, para não pesar na entrada.
    Timer {
        interval: root.info ? 3 * 3600 * 1000 : 120 * 1000
        repeat: true
        running: Settings.get("updates.auto")
        onTriggered: root.check()
    }
}
