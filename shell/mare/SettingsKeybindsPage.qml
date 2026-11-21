import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "Keybinds.js" as KB

// Página "Atalhos" dos Ajustes: a lista de Keybinds.js (o espelho de
// config/hypr/mare/binds.lua), por grupo, com o combo que vale agora. Clicar
// numa linha abre o editor: os modificadores são botões e a tecla é um campo.
// O Hyprland engole os atalhos que já existem antes de o Maré vê-los, então
// "aperte as teclas" não serve sozinho: a tecla pode ser digitada pelo nome
// (k, f5, enter, print) e os modificadores marcados à mão; e o "capturar"
// põe os atalhos do Mk para dormir (suspend() do binds.lua, que os acorda
// sozinho em 8 s) enquanto espera o combo inteiro. As trocas vão para
// ajustes.json ("keybinds"), viram <raiz>/atalhos.lua e valem na hora, por
// `hyprctl eval` do apply() do binds.lua (se a sessão não o tiver, um
// `hyprctl reload`). Depois de aplicar, a página confere com `hyprctl -j
// binds` se o Hyprland ficou mesmo com o que foi pedido.
ColumnLayout {
    id: page

    spacing: 0

    // Linha a destacar, vinda da busca dos Ajustes ("atalho:<id>"). A lista é
    // longa: a folha rola até a linha (o Flickable é de quem carrega a página).
    property string flashKey: ""
    property var lines: ({})
    onFlashKeyChanged: if (flashKey.startsWith("atalho:")) revealTimer.restart()

    Timer {
        id: revealTimer
        interval: 80
        onTriggered: {
            const row = page.lines[page.flashKey.slice(7)];
            let flick = page.parent;
            while (flick && flick.contentY === undefined)
                flick = flick.parent;
            if (!row || !flick)
                return;
            const y = row.mapToItem(flick.contentItem, 0, 0).y;
            flick.contentY = Math.max(0, Math.min(y - 120, flick.contentHeight - flick.height));
        }
    }

    readonly property var overrides: KB.clean(Settings.get("keybinds"))
    readonly property var combos: KB.effective(overrides)
    readonly property int changed: Object.keys(overrides).length

    // O editor: uma ação por vez.
    property string editing: ""
    property var draftMods: []
    property string draftKey: ""
    property bool capturing: false
    // A sessão suspendeu os atalhos: dá para apertar o combo inteiro. Se não
    // (binds.lua antigo), a captura vale só para a tecla.
    property bool captureFull: false
    property string status: ""
    // Ações que o Hyprland não tem no combo pedido (veja KB.diverged).
    property var failed: []
    // Quem tinha o foco antes do editor (a busca dos Ajustes), para devolver.
    property var focusBefore: null

    readonly property var draftAction: editing !== "" ? KB.byId(editing) : null
    readonly property var draft: draftAction ? KB.validate(draftAction, draftMods, draftKey) : ({ combo: "", error: "" })
    // Todos os que já usam o combo do editor: uma família pode bater em vários.
    readonly property var clashes: draftAction && draft.error === "" ? KB.conflicts(editing, draft.combo, overrides) : []
    readonly property var clash: clashes.length > 0 ? clashes[0] : null
    readonly property bool draftSame: draftAction !== null && draft.error === "" && KB.same(draft.combo, combos[editing])
    readonly property bool canApply: draftAction !== null && draft.error === "" && clash === null && !draftSame

    function begin(id) {
        const action = KB.byId(id);
        if (!action || action.fixed)
            return;
        const now = KB.parse(combos[id] !== "" ? combos[id] : action.combo);
        draftMods = now.mods;
        draftKey = action.family ? "" : now.key;
        stopCapture();
        if (editing === "")
            focusBefore = page.Window.activeFocusItem;
        editing = id;
    }

    function cancel() {
        stopCapture();
        if (editing === "")
            return;
        editing = "";
        // O editor some com o foco dentro: devolve a quem o tinha.
        const before = focusBefore;
        focusBefore = null;
        Qt.callLater(() => {
            if (page.editing === "" && before && before.visible)
                before.forceActiveFocus();
        });
    }

    // Captura: os atalhos do Mk dormem até a tecla chegar (ou 8 s, contados
    // pelo próprio Hyprland), senão Super+Q fecharia uma janela em vez de
    // aparecer aqui.
    function startCapture() {
        captureFull = false;
        capturing = true;
        suspend.running = true;
        captureTimer.restart();
    }

    function stopCapture() {
        if (!capturing)
            return;
        capturing = false;
        captureFull = false;
        captureTimer.stop();
        Quickshell.execDetached(["hyprctl", "eval", KB.resumeCode()]);
    }

    Process {
        id: suspend
        command: ["hyprctl", "eval", KB.suspendCode(8)]
        stdout: StdioCollector {
            onStreamFinished: page.captureFull = page.capturing && text.trim() === "ok"
        }
    }

    // Um pouco antes dos 8 s do Hyprland, para a tela não prometer o que acabou.
    Timer {
        id: captureTimer
        interval: 7000
        onTriggered: page.stopCapture()
    }

    // A página pode fechar no meio de uma captura ou antes de o applyTimer
    // disparar: os atalhos acordam e a troca vai mesmo assim.
    Component.onDestruction: {
        if (capturing)
            Quickshell.execDetached(["hyprctl", "eval", KB.resumeCode()]);
        if (applyTimer.running)
            Quickshell.execDetached(["hyprctl", "eval", KB.evalCode(pending)]);
    }

    function toggleMod(mod) {
        draftMods = draftMods.indexOf(mod) >= 0 ? draftMods.filter(m => m !== mod) : draftMods.concat([mod]);
    }

    // Grava as trocas (só as que diferem da fábrica) e manda aplicar.
    function commit(next) {
        const cleaned = KB.clean(next);
        checked = true;
        // O arquivo antes do ajuste: quando `overrides` mudar, sync() já vê os dois iguais.
        fileText = KB.lua(cleaned);
        file.setText(fileText);
        Settings.set("keybinds", Object.keys(cleaned).length > 0 ? cleaned : undefined);
        pending = cleaned;
        status = "";
        failed = [];
        applyTimer.restart();
    }

    // ajustes.json é o que a página mostra e atalhos.lua é o que o Hyprland
    // lê; os dois só se encontram no commit(). Se um mudou sem o outro (backup
    // restaurado, arquivo apagado ou editado à mão), acerta aqui: sem a chave
    // em ajustes.json, as trocas do atalhos.lua são adotadas; no resto,
    // ajustes.json manda e o arquivo é refeito e aplicado.
    function sync() {
        if (!fileReady || !Settings.loaded)
            return;
        const disk = KB.fromLua(fileText);
        const saved = Settings.values["keybinds"];
        if (saved === undefined && Object.keys(disk).length > 0) {
            if (KB.lua(disk) !== fileText)
                commit(disk);
            else
                Settings.set("keybinds", disk);
            status = "trocas lidas do atalhos.lua";
        } else if (JSON.stringify(disk) !== JSON.stringify(overrides)
                || (saved !== undefined && JSON.stringify(saved) !== JSON.stringify(overrides))) {
            commit(overrides);
        } else if (!checked) {
            // Tudo igual: uma conferida com o Hyprland, ao abrir a página.
            checked = true;
            pending = overrides;
            verify.running = true;
        }
    }

    // Depois, e não dentro da troca: sync() pode gravar o ajuste de novo.
    onOverridesChanged: Qt.callLater(page.sync)

    // `free`: ids que ficam sem atalho para dar lugar ao combo.
    function setCombo(id, combo, free) {
        const next = Object.assign({}, overrides);
        next[id] = combo;
        for (const other of free)
            next[other] = "";
        commit(next);
        cancel();
    }

    function restore(id) {
        const next = Object.assign({}, overrides);
        delete next[id];
        commit(next);
        if (editing === id)
            cancel();
    }

    function restoreAll() {
        commit({});
        cancel();
    }

    // Voltar ao combo de fábrica pode bater em quem tomou o lugar dele.
    function restoreClash(id) {
        const action = KB.byId(id);
        if (!action)
            return null;
        // Contra o que vale agora (page.combos), sem limpar de novo o mapa.
        const mine = KB.expand(action, action.combo);
        for (const other of KB.actions) {
            if (other.id !== id && KB.expand(other, page.combos[other.id]).some(k => mine.indexOf(k) >= 0))
                return other;
        }
        return null;
    }

    property var pending: ({})
    // O atalhos.lua como está no disco (ou como acabou de ser gravado).
    property string fileText: ""
    property bool fileReady: false
    property bool checked: false

    FileView {
        id: file
        path: Settings.dir + "/atalhos.lua"
        printErrors: false
        onLoaded: {
            page.fileText = text();
            page.fileReady = true;
            page.sync();
        }
        onLoadFailed: {
            page.fileText = "";
            page.fileReady = true;
            page.sync();
        }
    }

    Connections {
        target: Settings
        function onLoadedChanged() { page.sync(); }
    }

    // Um pouco depois da troca; se o hyprctl anterior ainda roda, espera.
    Timer {
        id: applyTimer
        interval: 120
        onTriggered: {
            if (apply.running || reload.running) {
                restart();
                return;
            }
            apply.command = ["hyprctl", "eval", KB.evalCode(page.pending)];
            apply.running = true;
        }
    }

    // Na hora: tira os binds do Mk e cria de novo, sem recarregar a config.
    Process {
        id: apply
        stdout: StdioCollector {
            onStreamFinished: {
                if (text.trim() === "ok") {
                    page.status = "aplicado";
                    verify.running = true;
                } else {
                    reload.running = true;
                }
            }
        }
    }

    // Reserva: uma sessão cuja config ainda não tem o apply() relê tudo.
    Process {
        id: reload
        command: ["hyprctl", "reload"]
        stdout: StdioCollector {
            onStreamFinished: {
                page.status = text.trim() === "ok" ? "aplicado (config recarregada)" : "gravado; vale no próximo login";
                if (text.trim() === "ok")
                    verify.running = true;
            }
        }
    }

    // "ok" só diz que o Lua rodou: o apply() pode ter recusado um combo e
    // voltado ao de fábrica. O que vale é a lista de binds do Hyprland.
    Process {
        id: verify
        command: ["hyprctl", "-j", "binds"]
        stdout: StdioCollector {
            onStreamFinished: {
                let binds = null;
                try {
                    binds = JSON.parse(text);
                } catch (e) {
                    return;
                }
                page.failed = KB.diverged(binds, page.pending);
                if (page.failed.length > 0 && page.status.startsWith("aplicado"))
                    page.status = page.failed.length === 1 ? "1 não aplicado" : page.failed.length + " não aplicados";
            }
        }
    }

    // ── resumo ─────────────────────────────────────────────────────────

    RowLayout {
        Layout.fillWidth: true
        Layout.topMargin: 6
        Layout.preferredHeight: 34
        spacing: 12

        Text {
            Layout.fillWidth: true
            text: (page.changed === 0 ? "todos de fábrica" : page.changed === 1 ? "1 trocado" : page.changed + " trocados")
                + (page.status !== "" ? "  ·  " + page.status : "")
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 10
            font.capitalization: Font.AllUppercase
            font.letterSpacing: 1.2
            elide: Text.ElideRight
        }

        PillButton {
            visible: page.changed > 0
            text: "voltar todos ao padrão"
            onClicked: page.restoreAll()
        }
    }

    // ── a lista ────────────────────────────────────────────────────────

    Repeater {
        model: KB.groups()

        ColumnLayout {
            id: group
            required property var modelData
            Layout.fillWidth: true
            spacing: 0

            Text {
                Layout.topMargin: 18
                Layout.bottomMargin: 4
                text: group.modelData.name
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 9
                font.letterSpacing: 1.8
            }

            Repeater {
                model: group.modelData.items

                ColumnLayout {
                    id: line
                    required property var modelData
                    readonly property string combo: page.combos[modelData.id]
                    readonly property bool isChanged: page.overrides[modelData.id] !== undefined
                    readonly property bool open: page.editing === modelData.id
                    readonly property bool isFailed: page.failed.indexOf(modelData.id) >= 0
                    readonly property var backClash: isChanged ? page.restoreClash(modelData.id) : null

                    Layout.fillWidth: true
                    spacing: 0

                    Component.onCompleted: page.lines[modelData.id] = line

                    Rectangle {
                        Layout.fillWidth: true
                        implicitHeight: 1
                        color: Theme.hairline
                    }

                    Item {
                        Layout.fillWidth: true
                        implicitHeight: 38

                        Rectangle {
                            anchors.fill: parent
                            anchors.leftMargin: -12
                            anchors.rightMargin: -12
                            color: line.open || (rowHover.hovered && !line.modelData.fixed) ? Theme.wash : "transparent"
                        }

                        // Vindo da busca.
                        Rectangle {
                            anchors.fill: parent
                            anchors.leftMargin: -12
                            anchors.rightMargin: -12
                            color: Theme.copper
                            opacity: page.flashKey === "atalho:" + line.modelData.id ? 0.12 : 0

                            Behavior on opacity {
                                NumberAnimation { duration: Motion.slow }
                            }
                        }

                        // A parte que abre o editor: tudo menos o "restaurar".
                        Item {
                            id: rowBody
                            anchors.left: parent.left
                            anchors.right: back.visible ? back.left : parent.right
                            anchors.rightMargin: back.visible ? 12 : 0
                            height: parent.height

                            Text {
                                id: comboText
                                anchors.verticalCenter: parent.verticalCenter
                                width: 220
                                text: KB.pretty(line.modelData, line.combo)
                                color: line.combo === "" ? Theme.textFaint : line.modelData.fixed ? Theme.textDim : Theme.text
                                font.family: Theme.mono
                                font.pixelSize: 12
                                font.italic: line.combo === ""
                                elide: Text.ElideRight
                            }

                            // Mudado em relação à fábrica.
                            Rectangle {
                                id: dot
                                visible: line.isChanged
                                anchors.left: comboText.right
                                anchors.leftMargin: 4
                                anchors.verticalCenter: parent.verticalCenter
                                width: 5
                                height: 5
                                radius: 3
                                color: Theme.copper
                            }

                            Text {
                                anchors.left: comboText.right
                                anchors.leftMargin: 22
                                anchors.right: mark.left
                                anchors.rightMargin: 12
                                anchors.verticalCenter: parent.verticalCenter
                                text: line.modelData.desc
                                color: line.modelData.fixed ? Theme.textDim : Theme.textSoft
                                font.family: Theme.sans
                                font.pixelSize: 13
                                elide: Text.ElideRight
                            }

                            Text {
                                id: mark
                                anchors.right: parent.right
                                anchors.verticalCenter: parent.verticalCenter
                                text: line.modelData.fixed ? "fixo" : line.isFailed ? "não aplicado"
                                    : line.modelData.required && !line.isChanged ? "emergência"
                                    : rowHover.hovered && !line.open && !line.isChanged ? "trocar" : ""
                                color: line.isFailed ? Theme.copper : Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 9
                                font.capitalization: Font.AllUppercase
                                font.letterSpacing: 1.2
                            }

                            HoverHandler {
                                id: rowHover
                                cursorShape: line.modelData.fixed ? Qt.ArrowCursor : Qt.PointingHandCursor
                            }
                            TapHandler {
                                enabled: !line.modelData.fixed
                                onTapped: line.open ? page.cancel() : page.begin(line.modelData.id)
                            }
                        }

                        Text {
                            id: back
                            visible: line.isChanged
                            anchors.right: parent.right
                            anchors.verticalCenter: parent.verticalCenter
                            text: line.backClash ? "padrão em uso" : "restaurar"
                            color: line.backClash ? Theme.textFaint : backHover.hovered ? Theme.copper : Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 9
                            font.capitalization: Font.AllUppercase
                            font.letterSpacing: 1.2

                            HoverHandler {
                                id: backHover
                                cursorShape: line.backClash ? Qt.ArrowCursor : Qt.PointingHandCursor
                            }
                            TapHandler {
                                enabled: line.backClash === null
                                onTapped: page.restore(line.modelData.id)
                            }
                        }
                    }

                    // ── o editor ──
                    Loader {
                        Layout.fillWidth: true
                        active: line.open
                        visible: active
                        sourceComponent: editor
                    }
                }
            }
        }
    }

    Text {
        Layout.fillWidth: true
        Layout.topMargin: 18
        text: "As trocas ficam em " + file.path.replace(Quickshell.env("HOME"), "~") + " e valem na hora, sem sair da sessão."
        wrapMode: Text.WordWrap
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
    }

    Component {
        id: editor

        ColumnLayout {
            id: form
            spacing: 0

            // O foco vem para cá quando o editor abre (senão o que se digita
            // cai na busca dos Ajustes): no campo da tecla, ou, numa família,
            // no próprio editor, para Esc fechar só ele e Enter aplicar. Só
            // depois da carga: no onCompleted o Loader ainda não é visível.
            function grab() {
                if (!page.draftAction)
                    return;
                if (page.draftAction.family) {
                    form.forceActiveFocus();
                } else {
                    keyInput.forceActiveFocus();
                    keyInput.selectAll();
                }
            }
            Component.onCompleted: Qt.callLater(form.grab)

            Keys.onEscapePressed: page.cancel()
            Keys.onReturnPressed: form.accept()
            Keys.onEnterPressed: form.accept()
            function accept() {
                if (page.canApply)
                    page.setCombo(page.editing, page.draft.combo, []);
            }

            RowLayout {
                Layout.fillWidth: true
                Layout.topMargin: 6
                spacing: 6

                Repeater {
                    model: KB.mods

                    PillButton {
                        required property string modelData
                        text: KB.modLabels[modelData].toLowerCase()
                        selected: page.draftMods.indexOf(modelData) >= 0
                        onClicked: page.toggleMod(modelData)
                    }
                }

                Text {
                    Layout.leftMargin: 6
                    Layout.rightMargin: 6
                    text: "+"
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 12
                }

                // Família: a tecla não muda, só os modificadores.
                Text {
                    visible: page.draftAction !== null && page.draftAction.family !== undefined
                    text: page.draftAction && page.draftAction.family ? KB.familyLabels[page.draftAction.family] : ""
                    color: Theme.text
                    font.family: Theme.mono
                    font.pixelSize: 12
                }

                // A tecla: digitar o nome, ou capturar um toque.
                Rectangle {
                    visible: page.draftAction !== null && page.draftAction.family === undefined
                    implicitWidth: 230
                    implicitHeight: 34
                    radius: 2
                    color: "transparent"
                    border.width: 1
                    border.color: page.capturing ? Theme.copper : keyInput.activeFocus ? Theme.line : Theme.hairline

                    TextInput {
                        id: keyInput
                        anchors.fill: parent
                        anchors.leftMargin: 10
                        anchors.rightMargin: 10
                        verticalAlignment: TextInput.AlignVCenter
                        clip: true
                        color: page.capturing ? "transparent" : Theme.text
                        selectionColor: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 12
                        maximumLength: 24
                        cursorDelegate: Rectangle {
                            width: 2
                            color: Theme.copper
                            visible: keyInput.activeFocus && !page.capturing
                        }
                        onTextEdited: page.draftKey = text

                        Component.onCompleted: text = page.draftKey
                        // Sem o foco não chega tecla nenhuma: a captura acaba.
                        onActiveFocusChanged: if (!activeFocus) page.stopCapture()

                        // Capturando, a próxima tecla (que não seja só um
                        // modificador) vira a tecla do atalho. Com os atalhos
                        // suspensos, os modificadores segurados com ela marcam
                        // os botões; sem isso o Hyprland ficaria com os combos
                        // que já existem, então só a tecla conta. Esc sozinho
                        // desiste da captura. Fora dela, Esc fecha o editor e
                        // Enter aplica.
                        Keys.onPressed: event => {
                            if (!page.capturing) {
                                if (event.key === Qt.Key_Escape) {
                                    event.accepted = true;
                                    page.cancel();
                                } else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                                    event.accepted = true;
                                    form.accept();
                                }
                                return;
                            }
                            event.accepted = true;
                            const held = KB.modsFromQt(event.modifiers);
                            if (event.key === Qt.Key_Escape && held.length === 0) {
                                page.stopCapture();
                                return;
                            }
                            const name = KB.fromQt(event.key, event.nativeScanCode, event.modifiers);
                            if (name === "")
                                return;
                            if (page.captureFull)
                                page.draftMods = held;
                            page.draftKey = name;
                            text = name;
                            page.stopCapture();
                        }

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: page.capturing || keyInput.text === ""
                            text: !page.capturing ? "tecla: k, f5, enter…"
                                : page.captureFull ? "aperte o atalho inteiro…" : "só a tecla, sem Super/Ctrl/Alt…"
                            color: page.capturing ? Theme.copper : Theme.textFaint
                            font.family: Theme.serif
                            font.pixelSize: 13
                            font.italic: true
                        }
                    }
                }

                PillButton {
                    visible: page.draftAction !== null && page.draftAction.family === undefined
                    text: page.capturing ? "cancelar captura" : "capturar"
                    selected: page.capturing
                    onClicked: {
                        if (page.capturing) {
                            page.stopCapture();
                        } else {
                            // O foco primeiro: perdê-lo encerra a captura.
                            keyInput.forceActiveFocus();
                            page.startCapture();
                        }
                    }
                }

                Item { Layout.fillWidth: true }
            }

            RowLayout {
                Layout.fillWidth: true
                Layout.topMargin: 8
                Layout.bottomMargin: 12
                spacing: 6

                Text {
                    Layout.fillWidth: true
                    text: page.draft.error !== "" ? page.draft.error
                        : page.clash ? "já usado por «" + page.clash.desc + "»"
                            + (page.clashes.length > 1 ? " e mais " + (page.clashes.length - 1) : "")
                        : page.draftSame ? "é o atalho de agora"
                        : page.draftAction ? "vai ficar " + KB.pretty(page.draftAction, page.draft.combo) : ""
                    color: page.draft.error !== "" || page.clash ? Theme.copper : Theme.textDim
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                    elide: Text.ElideRight
                }

                // Tira o combo de quem o usa (se der para deixar todos eles sem atalho).
                PillButton {
                    visible: page.clashes.length > 0 && page.clashes.every(c => !c.fixed && !c.required && !c.family)
                    text: "tirar de lá e usar"
                    onClicked: page.setCombo(page.editing, page.draft.combo, page.clashes.map(c => c.id))
                }
                PillButton {
                    visible: page.draftAction !== null && !page.draftAction.required && !page.draftAction.family
                        && page.combos[page.editing] !== ""
                    text: "sem atalho"
                    onClicked: page.setCombo(page.editing, "", [])
                }
                PillButton {
                    text: "cancelar"
                    onClicked: page.cancel()
                }
                PillButton {
                    primary: true
                    opacity: page.canApply ? 1 : 0.35
                    text: "aplicar"
                    onClicked: form.accept()
                }
            }
        }
    }
}
