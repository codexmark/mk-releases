pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "SettingsSchema.js" as Schema

// Os ajustes do Mk: os valores que o painel Ajustes muda (SettingsWindow.qml).
// Ficam em ~/.config/mk/ajustes.json, ao lado do shell e fora das pastas que
// a instalação substitui. O Maré lê com get(); o que é do Hyprland também vai
// para ~/.config/mk/ajustes.lua (carregado no login) e é aplicado na hora.
// O que não pega aparece em lastError, e o valor volta ao que era.
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

    // O que não deu certo, para o painel dizer (uma linha em cobre).
    property string lastError: ""
    // O ajustes.json existe mas não dá para ler: nada é gravado por cima dele
    // até a pessoa escolher começar do zero. A cópia fica em .quebrado.
    property bool broken: false

    // O que a máquina tem (scripts/mare-caps); null enquanto não respondeu.
    property var caps: null

    // Valores de antes, por chave, das mudanças que o Hyprland ainda não
    // aceitou: `undo` espera o eval, `flight` é o eval que está rodando.
    property var undo: ({})
    property var flight: ({})
    property string flightLua: ""
    // Chaves do Hyprland mexidas nesta sessão (veja Schema.ours).
    property var touched: ({})
    // O ajustes.lua já foi comparado com o que o painel geraria; `silent` é
    // o eval dessa comparação, que não vira aviso se for recusado.
    property bool synced: false
    property bool silent: false

    function get(key) {
        return values[key] !== undefined ? values[key] : defaults[key];
    }

    function isDefault(key) {
        return values[key] === undefined || values[key] === defaults[key];
    }

    function set(key, value) {
        if (broken || get(key) === value)
            return;
        lastError = "";
        if (Schema.touchesHypr(key))
            remember(key);
        const next = Object.assign({}, values);
        if (value === defaults[key])
            delete next[key];
        else
            next[key] = value;
        values = next;
        saveTimer.restart();
        if (Schema.touchesHypr(key))
            hyprTimer.restart();
        if (key.startsWith("idle."))
            idleDirty = true;
        // O tamanho da interface é a escala do Qt: só vale num Maré novo.
        if (key === "ui.scale")
            restartTimer.restart();
    }

    function reset(page) {
        if (broken)
            return;
        lastError = "";
        const next = Object.assign({}, values);
        let hypr = false;
        for (const row of Schema.pageRows(page)) {
            if (row.hypr !== undefined) {
                hypr = true;
                remember(row.key);
            }
            idleDirty = idleDirty || row.key.startsWith("idle.");
            delete next[row.key];
        }
        values = next;
        saveTimer.restart();
        if (hypr)
            hyprTimer.restart();
    }

    // Guarda o valor de agora (undefined = de fábrica) para o caso de o
    // Hyprland recusar a mudança.
    function remember(key) {
        touched[key] = true;
        if (key in undo)
            return;
        const next = Object.assign({}, undo);
        next[key] = values[key];
        undo = next;
    }

    // A resposta do hyprctl eval. Aceito: o ajustes.lua recebe o que foi
    // aplicado. Recusado: as chaves daquele eval voltam ao que eram, e o
    // ajustes.lua fica como estava.
    function answered(text) {
        const keys = Object.keys(flight);
        const old = flight;
        const quiet = silent && keys.length === 0;
        flight = ({});
        silent = false;
        if (text === "" || text === "ok") {
            lua.setText(flightLua);
            return;
        }
        console.warn("ajustes: hyprctl eval: " + text);
        if (quiet)
            return;
        const next = Object.assign({}, values);
        const later = Object.assign({}, undo);
        const back = [];
        for (const key of keys) {
            // Já há um valor mais novo esperando o eval: ele fica, e o ponto
            // de volta passa a ser o último que o Hyprland aceitou.
            if (key in later) {
                later[key] = old[key];
                continue;
            }
            if (old[key] === undefined)
                delete next[key];
            else
                next[key] = old[key];
            back.push(key);
        }
        values = next;
        undo = later;
        saveTimer.restart();
        const names = back.map(k => Schema.rowOf(k).label.toLowerCase()).join(", ");
        if (back.length > 0 || (keys.length === 0 && lastError === ""))
            lastError = "o hyprland recusou" + (names !== "" ? " (" + names + ")" : "") + ": " + text.split("\n")[0];
        // Mais um eval com o que ficou valendo: um Lua recusado pode ter sido
        // aplicado pela metade. Sem chaves a devolver, não insiste.
        if (keys.length > 0 || Object.keys(later).length > 0)
            hyprTimer.restart();
    }

    // Uma vez por sessão: um ajustes.lua de um gerador antigo (sem kb_options,
    // sem as animações na Velocidade) é refeito por um eval. Sem arquivo, nada.
    function syncLua() {
        if (synced || !loaded || broken || !lua.loaded)
            return;
        synced = true;
        if (lua.text() !== Schema.hyprLua(values)) {
            silent = true;
            hyprTimer.restart();
        }
    }

    function idleText() {
        return Schema.hypridleConf(values, dir + "/hypr", caps);
    }

    // Sem brilho, a linha de escurecer some: o passo sai do arquivo também.
    function syncIdle() {
        if (!loaded || broken || !caps || caps.brilho !== false || !idle.loaded)
            return;
        if (idle.text().indexOf("brightnessctl") < 0 || idle.text() === idleText())
            return;
        idle.setText(idleText());
        idleRestart.running = true;
    }

    // Some com o aviso (ao trocar de página); o do arquivo quebrado fica.
    function dismiss() {
        if (!broken)
            lastError = "";
    }

    // Desiste do ajustes.json ilegível (a cópia .quebrado fica) e volta à fábrica.
    function startFresh() {
        if (!broken)
            return;
        broken = false;
        lastError = "";
        values = ({});
        // De fábrica de verdade: o eval fala de tudo.
        for (const row of Schema.rows)
            if (row.hypr !== undefined)
                touched[row.key] = true;
        idleDirty = true;
        saveTimer.restart();
        hyprTimer.restart();
    }

    FileView {
        id: json
        path: root.jsonPath
        watchChanges: true
        printErrors: false
        onLoaded: {
            // Vazio é como ausente: não é erro.
            const raw = text();
            let parsed = {};
            let bad = false;
            if (raw.trim() !== "") {
                try {
                    parsed = JSON.parse(raw);
                } catch (e) {
                    bad = true;
                }
                bad = bad || parsed === null || typeof parsed !== "object" || Array.isArray(parsed);
            }
            if (bad) {
                if (!root.broken)
                    Quickshell.execDetached(["cp", "--", root.jsonPath, root.jsonPath + ".quebrado"]);
                root.broken = true;
                root.values = {};
                root.lastError = "o ajustes.json não pôde ser lido; uma cópia ficou em ajustes.json.quebrado";
            } else {
                if (root.broken)
                    root.lastError = "";
                root.broken = false;
                root.values = parsed;
            }
            root.loaded = true;
            root.syncLua();
            root.syncIdle();
        }
        onLoadFailed: {
            root.loaded = true;
            root.syncIdle();
        }
        onFileChanged: reload()
    }

    FileView {
        id: lua
        path: root.luaPath
        printErrors: false
        onLoaded: root.syncLua()
    }

    FileView {
        id: idle
        path: root.idlePath
        printErrors: false
        onLoaded: root.syncIdle()
    }

    // Escreve um pouco depois do último toque, para um slider arrastado não
    // gravar o arquivo a cada pixel.
    Timer {
        id: saveTimer
        interval: 400
        onTriggered: {
            if (root.broken)
                return;
            json.setText(JSON.stringify(root.values, null, 2) + "\n");
            if (root.idleDirty) {
                root.idleDirty = false;
                idle.setText(root.idleText());
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

    // O Hyprland muda na hora: o mesmo Lua do ajustes.lua, por eval. O arquivo
    // só é gravado quando o Hyprland aceita (veja answered).
    Timer {
        id: hyprTimer
        interval: 120
        onTriggered: {
            if (apply.running) {
                restart();
                return;
            }
            root.flight = root.undo;
            root.undo = ({});
            root.flightLua = Schema.hyprLua(root.values);
            apply.command = ["hyprctl", "eval", Schema.hyprCode(root.values, root.touched)];
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
            onStreamFinished: root.answered(text.trim())
        }
    }

    Process {
        running: true
        command: ["bash", Quickshell.shellPath("scripts/mare-caps")]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    root.caps = JSON.parse(text);
                } catch (e) {}
                root.syncIdle();
            }
        }
    }
}
