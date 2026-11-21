pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland

// Liga workspaces do Hyprland a projetos em ~/Work e coleta o estado do
// projeto em foco (branch, containers, portas, runtime).
Singleton {
    id: root

    readonly property string home: Quickshell.env("HOME")
    readonly property string workDir: home + "/Work"
    readonly property string storePath: Quickshell.statePath("workspaces.json")
    readonly property var emptyInfo: ({ branch: "", dirty: 0, containers: [], ports: [], runtime: "" })

    // { "1": { "name": "loja-api", "dir": "/home/u/Work/loja-api" }, "4": { "name": "livre", "dir": "" } }
    property var bindings: ({})
    // Repositórios git em ~/Work (até 3 níveis): [{ name, dir }]
    property var dirs: []
    property var info: emptyInfo

    readonly property var focusedWs: Hyprland.focusedWorkspace
    readonly property int focusedId: focusedWs ? focusedWs.id : 0
    readonly property var current: bindings[String(focusedId)] || null
    readonly property string name: current ? current.name : ""
    readonly property string dir: current ? current.dir : ""
    readonly property bool free: current !== null && current.dir === ""
    readonly property color accent: current ? Theme.accentFor(current.name) : Theme.neutral
    readonly property int windowCount: focusedWs ? focusedWs.toplevels.values.length : 0
    readonly property bool needsPicker: focusedId > 0 && current === null && windowCount === 0
    readonly property int containersUp: info.containers.filter(c => c.running).length

    function bind(id, projectName, projectDir) {
        const next = Object.assign({}, bindings);
        next[String(id)] = { name: projectName, dir: projectDir };
        bindings = next;
        store.setText(JSON.stringify(next, null, 2) + "\n");
    }

    // Config do ghostty do Mk (ao lado do shell instalado). O "?" deixa a
    // config opcional e config-default-files=false ignora o ~/.config/ghostty
    // de outros desktops. Sai de Settings.dir porque o shellPath não sobe
    // acima da pasta do shell ("../ghostty" voltava como caminho relativo, e
    // o ghostty abria sem a config do Mk).
    readonly property string terminalConfig: Settings.dir + "/ghostty/config"

    // Os terminais que o Mk sabe abrir "neste diretório, rodando isto". Não há
    // padrão do XDG para terminal: a escolha fica no ajustes.json
    // ("apps.terminal", página Apps padrão dos Ajustes). O de fábrica é o
    // ghostty com a config do Mk.
    readonly property var terminals: ["ghostty", "kitty", "foot", "alacritty", "wezterm"]
    readonly property string terminal: {
        const t = Settings.get("apps.terminal");
        return terminals.indexOf(t) >= 0 ? t : "ghostty";
    }

    // Texto -> um argumento só para o shell.
    function shellQuote(s) {
        return "'" + String(s).replace(/'/g, "'\\''") + "'";
    }

    // A linha de shell que abre o terminal `name` em `dir`. `cmd` é texto de
    // shell (vazio = só o shell do usuário) e `title`, o título da janela.
    // Cada terminal tem as suas flags; só o ghostty leva a config do Mk.
    function terminalLineFor(name, dir, cmd, title) {
        const d = shellQuote(dir);
        const t = title ? shellQuote(title) : "";
        switch (name) {
        case "kitty":
            return `kitty --directory=${d}` + (t ? ` --title=${t}` : "") + (cmd ? ` ${cmd}` : "");
        case "foot":
            return `foot --working-directory=${d}` + (t ? ` --title=${t}` : "") + (cmd ? ` ${cmd}` : "");
        case "alacritty":
            return `alacritty --working-directory ${d}` + (t ? ` --title ${t}` : "") + (cmd ? ` -e ${cmd}` : "");
        case "wezterm":
            // Sem --always-new-process a janela nasce num wezterm já aberto e
            // este comando volta na hora (quem espera o fim, como as
            // atualizações, acharia que acabou). Não tem flag de título.
            return `wezterm start --always-new-process --cwd ${d}` + (cmd ? ` -- ${cmd}` : "");
        default:
            return `ghostty --config-default-files=false --config-file=${shellQuote("?" + terminalConfig)} --working-directory=${d}`
                + (t ? ` --title=${t}` : "") + (cmd ? ` -e ${cmd}` : "");
        }
    }

    // O mesmo, para o terminal escolhido. Se ele foi desinstalado depois da
    // escolha, abre o ghostty: Super + Enter nunca fica sem terminal.
    function terminalLine(dir, cmd, title) {
        const d = dir && dir !== "" ? dir : home;
        const fallback = terminalLineFor("ghostty", d, cmd, title);
        if (terminal === "ghostty")
            return fallback;
        return `if command -v ${terminal} >/dev/null 2>&1; then ${terminalLineFor(terminal, d, cmd, title)}; else ${fallback}; fi`;
    }

    // Abre o terminal no diretório do projeto; `cmd` roda dentro dele (lazygit, lazydocker).
    function openIn(projectDir, cmd) {
        Hypr.exec(terminalLine(projectDir, cmd, ""));
    }

    function openHere(cmd) {
        openIn(dir, cmd);
    }

    function refreshDirs() {
        listDirs.running = true;
    }

    onDirChanged: {
        info = emptyInfo;
        if (dir !== "")
            infoProc.running = true;
    }

    onNeedsPickerChanged: pickerCheck.restart()

    // Espera um pouco antes de decidir: logo depois de trocar de workspace o
    // Hyprland ainda pode estar reportando as janelas.
    Timer {
        id: pickerCheck
        interval: 250
        onTriggered: {
            // A folha do workspace vazio (Backdrop.qml) já convida a começar;
            // o seletor só abre quando chamado. Aqui só se fecha um seletor que
            // ficou aberto num workspace que ganhou janela.
            if (!root.needsPicker && Mare.mode === "picker")
                Mare.close();
        }
    }

    Timer {
        interval: 5000
        repeat: true
        running: root.dir !== ""
        onTriggered: infoProc.running = true
    }

    Process {
        id: infoProc
        command: ["bash", Quickshell.shellPath("scripts/mare-project"), root.dir]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    root.info = JSON.parse(text);
                } catch (e) {
                    root.info = root.emptyInfo;
                }
            }
        }
    }

    Process {
        id: listDirs
        command: ["find", root.workDir, "-mindepth", "2", "-maxdepth", "4", "-name", ".git", "-prune", "-printf", "%h\\n"]
        stdout: StdioCollector {
            onStreamFinished: {
                // Só troca a lista se mudou: quem depende dela (a busca) refaz tudo.
                const next = text.split("\n")
                    .filter(s => s.length > 0)
                    .map(d => ({ name: d.substring(d.lastIndexOf("/") + 1), dir: d }))
                    .sort((a, b) => a.name.localeCompare(b.name));
                if (JSON.stringify(next) !== JSON.stringify(root.dirs))
                    root.dirs = next;
            }
        }
    }

    Process {
        id: ensureStateDir
        running: true
        command: ["mkdir", "-p", root.storePath.substring(0, root.storePath.lastIndexOf("/"))]
    }

    FileView {
        id: store
        path: root.storePath
        watchChanges: true
        printErrors: false
        onFileChanged: reload()
        onLoaded: {
            try {
                root.bindings = JSON.parse(text());
            } catch (e) {
                root.bindings = {};
            }
            pickerCheck.restart();
        }
        onLoadFailed: {
            root.bindings = {};
            pickerCheck.restart();
        }
    }
}
