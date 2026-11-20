pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "SettingsSchema.js" as Schema

// Os ajustes do Mk: os valores que o painel Ajustes muda (SettingsWindow.qml).
// Ficam em ~/.config/mk/ajustes.json, ao lado do shell e fora das pastas que
// a instalação substitui. O Maré lê com get(); o que é do Hyprland também vai
// para ~/.config/mk/ajustes.lua (carregado no login) e é aplicado na hora.
Singleton {
    id: root

    // ~/.config/mk, onde quer que o shell esteja instalado.
    readonly property string dir: Quickshell.shellPath("").replace(/\/+$/, "").replace(/\/[^\/]+$/, "")
    readonly property string jsonPath: dir + "/ajustes.json"
    readonly property string luaPath: dir + "/ajustes.lua"
    readonly property string idlePath: dir + "/hypridle.conf"
    property bool idleDirty: false

    readonly property var defaults: Schema.defaults()
    property var values: ({})
    property bool loaded: false

    function get(key) {
        return values[key] !== undefined ? values[key] : defaults[key];
    }

    function isDefault(key) {
        return values[key] === undefined || values[key] === defaults[key];
    }

    function set(key, value) {
        if (get(key) === value)
            return;
        const next = Object.assign({}, values);
        if (value === defaults[key])
            delete next[key];
        else
            next[key] = value;
        values = next;
        saveTimer.restart();
        if (key.startsWith("hypr."))
            hyprTimer.restart();
        if (key.startsWith("idle."))
            idleDirty = true;
        // O tamanho da interface é a escala do Qt: só vale num Maré novo.
        if (key === "ui.scale")
            restartTimer.restart();
    }

    function reset(page) {
        const next = Object.assign({}, values);
        let hypr = false;
        for (const row of Schema.pageRows(page)) {
            hypr = hypr || row.key.startsWith("hypr.");
            idleDirty = idleDirty || row.key.startsWith("idle.");
            delete next[row.key];
        }
        values = next;
        saveTimer.restart();
        if (hypr)
            hyprTimer.restart();
    }

    FileView {
        id: json
        path: root.jsonPath
        watchChanges: true
        printErrors: false
        onLoaded: {
            try {
                const parsed = JSON.parse(text());
                root.values = parsed && typeof parsed === "object" ? parsed : {};
            } catch (e) {
                root.values = {};
            }
            root.loaded = true;
        }
        onLoadFailed: root.loaded = true
        onFileChanged: reload()
    }

    FileView {
        id: lua
        path: root.luaPath
        printErrors: false
    }

    FileView {
        id: idle
        path: root.idlePath
        printErrors: false
    }

    // Escreve um pouco depois do último toque, para um slider arrastado não
    // gravar o arquivo a cada pixel.
    Timer {
        id: saveTimer
        interval: 400
        onTriggered: {
            json.setText(JSON.stringify(root.values, null, 2) + "\n");
            lua.setText(Schema.hyprLua(root.values));
            if (root.idleDirty) {
                root.idleDirty = false;
                idle.setText(Schema.hypridleConf(root.values, root.dir + "/hypr"));
                idleRestart.running = true;
            }
        }
    }

    // Reinicia o Maré depois de gravar o ajustes.json (o mare-start lê de lá).
    // Roda fora do Maré, porque o próprio Maré vai fechar.
    Timer {
        id: restartTimer
        interval: 700
        onTriggered: Quickshell.execDetached(["setsid", "bash", Quickshell.shellPath("scripts/mare-start"), "--restart"])
    }

    // O Hyprland muda na hora: o mesmo hl.config do ajustes.lua, por eval.
    Timer {
        id: hyprTimer
        interval: 120
        onTriggered: {
            apply.command = ["hyprctl", "eval", Schema.hyprCode(root.values)];
            apply.running = true;
        }
    }

    // Reinicia só o hypridle que o Mk iniciou (com uma das duas configs dele),
    // e só se ele estava rodando: numa sessão de teste não há nenhum.
    Process {
        id: idleRestart
        command: ["sh", "-c", "pkill -x -f \"hypridle -c (" + root.dir + "/hypridle.conf|" + root.dir
            + "/hypr/hypridle.conf)\" && sleep 0.3 && setsid -f hypridle -c '" + root.idlePath + "' >/dev/null 2>&1"]
    }

    Process {
        id: apply
        stdout: StdioCollector {
            onStreamFinished: if (text.trim() !== "" && text.trim() !== "ok") console.warn("ajustes: hyprctl eval: " + text.trim())
        }
    }
}
