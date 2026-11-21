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

    // A última rodada do mk-update (resumo que ele deixa), se o kernel em uso
    // já foi trocado e o que chega com a versão nova do Mk.
    readonly property var last: info && info.last ? info.last : null
    readonly property bool reboot: info ? info.reboot === true : false
    readonly property var news: mkAvailable && info.news ? info.news : []

    // A ilha só acende pelo que importa: versão nova do Mk, migração pendente
    // ou pacotes esperando há uma semana. No Arch há pacote novo todo dia.
    readonly property bool badge: {
        if (!info)
            return false;
        if (mkAvailable || migrations > 0)
            return true;
        if (systemCount + aurCount === 0)
            return false;
        return !info.applied || info.checked - info.applied >= 7 * 86400;
    }

    // O mk-update rodado num terminal qualquer grava o resumo ao terminar:
    // a página e a ilha se atualizam sem esperar a próxima procura.
    FileView {
        path: (Quickshell.env("XDG_STATE_HOME") || (Quickshell.env("HOME") + "/.local/state")) + "/mk/ultima-atualizacao.json"
        watchChanges: true
        printErrors: false
        onFileChanged: root.check()
    }

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

    // "hoje", "ontem", "há 5 dias"; depois de um mês, a data.
    function day(epoch) {
        const d = new Date(epoch * 1000);
        const n = new Date();
        const days = Math.round((new Date(n.getFullYear(), n.getMonth(), n.getDate())
            - new Date(d.getFullYear(), d.getMonth(), d.getDate())) / 86400000);
        return days <= 0 ? "hoje" : days === 1 ? "ontem" : days < 30 ? "há " + days + " dias"
            : Qt.formatDate(d, "dd/MM/yyyy");
    }

    function lastText() {
        if (!last)
            return "";
        const mk = last.to && last.to !== last.from ? " · mk " + last.to : "";
        return "última vez: " + day(last.when) + ", "
            + (last.ok ? "tudo certo" + mk : "parou em: " + (last.step || "início"));
    }

    // O registro da última rodada, no terminal (no fim, onde parou).
    function openLog() {
        if (last && last.log)
            Quickshell.execDetached(["sh", "-c", Projects.terminalLine("",
                "less -R +G " + Projects.shellQuote(last.log), "Mk · registro")]);
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
            + Projects.terminalLine("", "\"$u\"", "Mk · atualizações")]
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
