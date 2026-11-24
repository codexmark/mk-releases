import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

// Página "Impressoras" dos Ajustes, em cima do CUPS (scripts/mare-printers):
// o serviço, as impressoras configuradas (a padrão em cobre, o estado, a fila)
// e a procura das que entram sem driver, na rede e no USB. Nada muda sem um
// clique; "remover" pede um segundo.
//
// Quando uma ação precisa da senha de administrador, quem a pede é a janela
// do polkit, e ela ficaria atrás desta folha (que cobre a tela inteira). Por
// isso a página fecha a folha nessa hora; a ação segue sozinha no script, que
// reabre a folha aqui com o resultado.
ColumnLayout {
    id: page
    spacing: 0

    readonly property string script: Quickshell.shellPath("scripts/mare-printers")

    // O que o mare-printers contou por último (null = ainda lendo). `seen` é
    // o texto dele: a lista só é refeita quando algo mudou de verdade.
    property var info: null
    property string seen: ""
    readonly property bool installed: info !== null && info.installed
    readonly property bool running: info !== null && info.running
    readonly property var printers: running ? info.printers : []

    // A ação que esta página pediu ("add:<uri>", "remove:<fila>", "enable"...)
    // e a ação em curso, que pode ser uma pedida antes de a folha reabrir.
    property string acting: ""
    readonly property string busy: acting !== "" ? acting : info !== null ? (info.pending || "") : ""
    // O recado da última ação, se foi um erro, e a quem se refere (uma fila,
    // o endereço de uma impressora achada, ou vazio).
    property string note: ""
    property bool noteError: false
    property string noteFor: ""
    property double noteAt: 0
    // O recado aparece na linha a que se refere; sem ela, no alto da página.
    readonly property bool noteInRow: noteFor !== ""
        && (printers.some(p => p.name === noteFor) || (found !== null && found.some(f => f.uri === noteFor)))
    // A fila esperando o segundo clique de "remover".
    property string confirming: ""

    // A procura: null = ainda não procurou.
    property var found: null
    property var legacy: []
    property bool mdns: true
    property bool legacyUnknown: false
    property bool searching: false

    readonly property var stateNames: ({ "idle": "pronta", "printing": "imprimindo", "paused": "em pausa" })

    function refresh() {
        if (!statProc.running)
            statProc.running = true;
    }

    // Uma ação pedida por clique: `key` marca o botão que fica esperando.
    function act(key, args) {
        if (busy !== "" || actProc.running)
            return;
        acting = key;
        note = "";
        confirming = "";
        actProc.command = ["bash", script].concat(args);
        actProc.running = true;
    }

    function search() {
        if (searching)
            return;
        searching = true;
        findProc.running = true;
    }

    function remove(name) {
        if (confirming !== name) {
            confirming = name;
            confirmTimer.restart();
            return;
        }
        act("remove:" + name, ["remove", name]);
    }

    // O estado que o script contou. Só troca `info` se algo mudou: trocar
    // refaz todas as linhas e os botões, e isso a cada leitura perderia o
    // clique que estivesse no meio.
    function apply(data) {
        const state = stateOf(data);
        const text = JSON.stringify(state);
        if (text !== seen) {
            seen = text;
            info = state;
        }
    }

    // O resultado de uma ação: { key, ok, message, at }.
    function report(result) {
        if (!result || !result.key || (result.at > 0 && result.at === noteAt))
            return;
        noteAt = result.at;
        const told = noteOf(result);
        noteError = told.error;
        note = told.text;
        noteFor = told.target;
        noteTimer.restart();
    }

    function isAdded(uri) {
        return addedIn(uri, printers);
    }

    // --- puro: início ---
    // Daqui até "puro: fim" nada toca na página nem no CUPS: são funções de
    // dado para dado e de dado para texto sobre o que o mare-printers
    // respondeu, testadas fora do shell com node
    // (tests/js/SettingsPrintersPage.test.mjs).
    // O estado que o script contou, com cada campo no seu tipo (o que faltar
    // vira falso, vazio ou lista vazia).
    function stateOf(data) {
        return {
            installed: data.installed === true, running: data.running === true, enabled: data.enabled === true,
            "default": data["default"] || "", pending: data.pending || "", printers: data.printers || []
        };
    }

    // O recado de uma ação ({ key: "<verbo>:<alvo>", ok, message }): é erro?,
    // o texto e a linha a que se refere ({ error, text, target }).
    function noteOf(result) {
        const what = result.key.split(":")[0];
        const target = result.key.slice(what.length + 1);
        const failed = result.ok !== true;
        return {
            error: failed,
            text: result.message || (failed ? "não deu certo" : ""),
            // Quem sumiu (removida) ou acabou de entrar (adicionada) não tem linha.
            target: (what === "remove" || what === "add") && !failed ? "" : target
        };
    }

    // A impressora achada já tem fila? A mesma costuma se anunciar em ipp e
    // em ipps: valem os dois endereços. `queues`: as filas ([{ uri, ... }]).
    function addedIn(uri, queues) {
        const bare = uri.replace(/^ipps?:/, "");
        return queues.some(p => p.uri.replace(/^ipps?:/, "") === bare);
    }

    function queueText(count) {
        return count === 0 ? "fila vazia" : count === 1 ? "1 na fila" : count + " na fila";
    }
    // --- puro: fim ---

    Component.onCompleted: refresh()

    // A fila e o estado mudam sozinhos: relê enquanto a página está aberta
    // (mais depressa enquanto uma ação pedida antes ainda não terminou).
    Timer {
        interval: page.busy !== "" ? 1500 : 6000
        repeat: true
        running: page.info !== null && page.acting === "" && (page.running || page.busy !== "")
        onTriggered: page.refresh()
    }

    Timer {
        id: confirmTimer
        interval: 4000
        onTriggered: page.confirming = ""
    }

    // O recado some sozinho; um erro fica mais tempo.
    Timer {
        id: noteTimer
        interval: page.noteError ? 14000 : 7000
        onTriggered: page.note = ""
    }

    // A leitura periódica: um processo só dela, para nunca engolir um clique.
    Process {
        id: statProc
        command: ["bash", page.script, "status"]
        stdout: StdioCollector {
            onStreamFinished: {
                let data = null;
                try {
                    data = JSON.parse(text);
                } catch (e) {
                    data = null;
                }
                if (data === null) {
                    if (page.info === null)
                        page.apply({});
                    return;
                }
                page.apply(data);
                // Uma ação que terminou com a folha fechada (a da janela de senha).
                if (page.acting === "")
                    page.report(data.last);
            }
        }
    }

    // As ações. O script responde uma linha de JSON por vez: {"elevating":true}
    // quando a janela de senha vai aparecer, e no fim o estado com o resultado.
    Process {
        id: actProc
        environment: ({ "MARE_PRINTERS_SHELL": page.script.replace(/\/scripts\/mare-printers$/, "") })
        stdout: SplitParser {
            onRead: line => {
                let data = null;
                try {
                    data = JSON.parse(line);
                } catch (e) {
                    return;
                }
                if (data === null || typeof data !== "object")
                    return;
                if (data.elevating === true) {
                    // Sai da frente da janela de senha. A folha (e esta página)
                    // some; o script termina a ação e reabre a folha.
                    Qt.callLater(() => Mare.close());
                    return;
                }
                page.acting = "";
                page.apply(data);
                page.report(data);
            }
        }
        onExited: lost.restart()
    }

    // O script saiu sem dizer o resultado.
    Timer {
        id: lost
        interval: 400
        onTriggered: {
            if (page.acting === "" || actProc.running)
                return;
            page.report({ key: page.acting, ok: false, message: "o mare-printers não respondeu", at: Date.now() });
            page.acting = "";
            page.refresh();
        }
    }

    Process {
        id: findProc
        command: ["bash", page.script, "find"]
        stdout: StdioCollector {
            onStreamFinished: {
                page.searching = false;
                let data = ({});
                try {
                    data = JSON.parse(text) || ({});
                } catch (e) {
                    data = ({});
                }
                page.found = data.found || [];
                page.legacy = data.legacy || [];
                page.mdns = data.mdns !== false;
                page.legacyUnknown = data.legacyUnknown === true;
            }
        }
    }

    // O recado de uma ação: serifa em itálico; o erro vai em cobre.
    component Note: Text {
        text: page.note
        wrapMode: Text.WordWrap
        color: page.noteError ? Theme.copper : Theme.textSoft
        font.family: Theme.serif
        font.pixelSize: 14
        font.italic: true
    }

    // Rótulo de seção: mono pequeno em maiúsculas espaçadas.
    component Section: Text {
        Layout.topMargin: 22
        Layout.bottomMargin: 6
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.letterSpacing: Theme.labelSpacing
    }

    // Uma linha de terminal para copiar, quando o conserto é por lá.
    component CommandLine: Rectangle {
        id: cmd
        property string command: ""
        property bool copied: false

        Layout.fillWidth: true
        implicitHeight: 34
        radius: 2
        color: Theme.ground
        border.width: 1
        border.color: Theme.hairline

        Text {
            anchors.left: parent.left
            anchors.leftMargin: 12
            anchors.right: copy.left
            anchors.rightMargin: 10
            anchors.verticalCenter: parent.verticalCenter
            text: cmd.command
            elide: Text.ElideRight
            color: Theme.textSoft
            font.family: Theme.mono
            font.pixelSize: 11
        }

        Text {
            id: copy
            anchors.right: parent.right
            anchors.rightMargin: 12
            anchors.verticalCenter: parent.verticalCenter
            text: cmd.copied ? "copiado" : "copiar"
            color: cmd.copied || copyHover.hovered ? Theme.copper : Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 10

            HoverHandler {
                id: copyHover
                cursorShape: Qt.PointingHandCursor
            }
            TapHandler {
                onTapped: {
                    Quickshell.execDetached(["wl-copy", cmd.command]);
                    cmd.copied = true;
                    copiedTimer.restart();
                }
            }
        }

        Timer {
            id: copiedTimer
            interval: 1800
            onTriggered: cmd.copied = false
        }
    }

    // ── Serviço ────────────────────────────────────────────────────────────
    Section { text: "SERVIÇO" }

    // Ligado: uma linha só, discreta.
    Item {
        visible: page.running
        Layout.fillWidth: true
        implicitHeight: 46

        Rectangle {
            width: parent.width
            height: 1
            color: Theme.hairline
        }

        Row {
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            spacing: 10

            Rectangle {
                anchors.verticalCenter: parent.verticalCenter
                width: 6
                height: 6
                radius: 3
                color: Theme.copper
            }
            Text {
                anchors.verticalCenter: parent.verticalCenter
                text: "impressão ligada"
                color: Theme.text
                font.family: Theme.sans
                font.pixelSize: 14
            }
        }

        Text {
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            text: page.running && !page.info.enabled ? "CUPS · SÓ ATÉ DESLIGAR" : "CUPS"
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 9
            font.letterSpacing: Theme.labelSpacing
        }
    }

    // Lendo pela primeira vez.
    Text {
        visible: page.info === null
        Layout.topMargin: 10
        text: "lendo as impressoras…"
        color: Theme.textDim
        font.family: Theme.serif
        font.pixelSize: 14
        font.italic: true
    }

    // Sem serviço: a folha explica o que falta e como resolver.
    Rectangle {
        visible: page.info !== null && !page.running
        Layout.fillWidth: true
        implicitHeight: off.implicitHeight + 44
        radius: 3
        color: Theme.surfaceLow
        border.width: 1
        border.color: Theme.hairline

        ColumnLayout {
            id: off
            x: 22
            y: 22
            width: parent.width - 44
            spacing: 0

            Text {
                text: page.installed ? "PARADO" : "NÃO INSTALADO"
                color: Theme.copper
                font.family: Theme.mono
                font.pixelSize: 9
                font.letterSpacing: Theme.labelSpacing
            }

            Text {
                Layout.fillWidth: true
                Layout.topMargin: 8
                text: page.installed ? "O serviço de impressão está desligado." : "Este computador ainda não sabe imprimir."
                wrapMode: Text.WordWrap
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 21
                font.weight: Font.Light
            }

            Text {
                Layout.fillWidth: true
                Layout.topMargin: 8
                text: page.installed
                    ? "O CUPS está instalado, mas não está rodando. Ligue aqui (esta folha fecha, o sistema pede a sua senha numa janela, e a folha volta) ou pelo terminal."
                    : "Falta o CUPS, o serviço que conversa com as impressoras. O Mk novo já vem com ele; aqui ele entra na próxima atualização do Mk, ou agora pelo terminal:"
                wrapMode: Text.WordWrap
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 13
                lineHeight: 1.25
            }

            CommandLine {
                visible: !page.installed
                Layout.topMargin: 16
                command: "sudo pacman -S --needed cups ipp-usb"
            }
            CommandLine {
                Layout.topMargin: page.installed ? 16 : 6
                command: "sudo systemctl enable --now cups.socket"
            }

            RowLayout {
                Layout.topMargin: 16
                spacing: 8

                PillButton {
                    visible: page.installed
                    text: page.busy === "enable" ? "ligando…" : "ligar a impressão"
                    onClicked: page.act("enable", ["enable"])
                }
                PillButton {
                    text: "verificar de novo"
                    onClicked: page.refresh()
                }
            }
        }
    }

    // O recado que não tem linha para morar (ligar, adicionar, remover).
    Note {
        visible: page.note !== "" && !page.noteInRow
        Layout.fillWidth: true
        Layout.topMargin: 14
    }

    // ── Impressoras configuradas ───────────────────────────────────────────
    Section {
        visible: page.running
        text: "IMPRESSORAS"
    }

    Text {
        visible: page.running && page.printers.length === 0
        Layout.fillWidth: true
        Layout.topMargin: 6
        Layout.bottomMargin: 6
        text: "Nenhuma impressora configurada. Procure logo abaixo: as de rede e as USB recentes entram sem driver."
        wrapMode: Text.WordWrap
        color: Theme.textDim
        font.family: Theme.serif
        font.pixelSize: 14
        font.italic: true
    }

    Repeater {
        model: page.printers

        Item {
            id: row
            required property var modelData
            readonly property string name: modelData.name
            readonly property int jobCount: modelData.jobs.length

            Layout.fillWidth: true
            implicitHeight: words.implicitHeight + 30

            Rectangle {
                width: parent.width
                height: 1
                color: Theme.hairline
            }

            // O nome e o estado em cima, as ações embaixo: são botões demais
            // para caber ao lado numa folha estreita.
            Column {
                id: words
                y: 15
                width: parent.width
                spacing: 5

                Row {
                    spacing: 10

                    Text {
                        id: title
                        width: Math.min(implicitWidth, words.width - (tag.visible ? tag.width + 10 : 0))
                        text: row.name
                        elide: Text.ElideRight
                        color: row.modelData.default ? Theme.copper : Theme.text
                        font.family: Theme.sans
                        font.pixelSize: 14
                    }
                    Text {
                        id: tag
                        visible: row.modelData.default
                        anchors.baseline: title.baseline
                        text: "PADRÃO"
                        color: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 9
                        font.letterSpacing: Theme.labelSpacing
                    }
                }

                Text {
                    width: parent.width
                    text: (page.stateNames[row.modelData.state] || row.modelData.state) + "  ·  " + page.queueText(row.jobCount)
                        + (row.modelData.uri !== "" ? "  ·  " + row.modelData.uri : "")
                    elide: Text.ElideRight
                    color: row.modelData.state === "paused" ? Theme.textSoft : Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 10
                }

                Item {
                    width: 1
                    height: 5
                }

                Flow {
                    id: actions
                    width: parent.width
                    spacing: 6

                    PillButton {
                        visible: !row.modelData.default
                        text: page.busy === "default:" + row.name ? "trocando…" : "tornar padrão"
                        onClicked: page.act("default:" + row.name, ["default", row.name])
                    }
                    PillButton {
                        visible: row.modelData.state === "paused"
                        text: page.busy === "resume:" + row.name ? "retomando…" : "retomar"
                        onClicked: page.act("resume:" + row.name, ["resume", row.name])
                    }
                    PillButton {
                        visible: row.jobCount > 0
                        text: page.busy === "cancel:" + row.name ? "cancelando…" : "cancelar fila"
                        onClicked: page.act("cancel:" + row.name, ["cancel", row.name])
                    }
                    PillButton {
                        text: page.busy === "test:" + row.name ? "enviando…" : "página de teste"
                        onClicked: page.act("test:" + row.name, ["test", row.name])
                    }
                    PillButton {
                        // O segundo clique é o que remove: a borda em cobre avisa.
                        border.color: page.confirming === row.name ? Theme.copper : Theme.line
                        text: page.busy === "remove:" + row.name ? "removendo…"
                            : page.confirming === row.name ? "remover mesmo?" : "remover"
                        onClicked: page.remove(row.name)
                    }
                }

                Note {
                    visible: page.note !== "" && page.noteFor === row.name
                    width: parent.width
                    topPadding: 4
                }
            }
        }
    }

    // ── Procurar ───────────────────────────────────────────────────────────
    Section {
        visible: page.running
        text: "NA REDE E NO USB"
    }

    Item {
        visible: page.running
        Layout.fillWidth: true
        implicitHeight: 62

        Rectangle {
            width: parent.width
            height: 1
            color: Theme.hairline
        }

        Text {
            anchors.left: parent.left
            anchors.right: searchButton.left
            anchors.rightMargin: 16
            anchors.verticalCenter: parent.verticalCenter
            text: page.searching ? "Ouvindo a rede por uns segundos…"
                : page.found === null ? "Impressoras ligadas, na mesma rede ou no cabo USB, que entram sem driver."
                : !page.mdns ? "A descoberta de rede (avahi) está desligada: só as impressoras no cabo USB aparecem aqui."
                : page.found.length === 0 ? "Nada respondeu. A impressora está ligada e na mesma rede?"
                : page.found.length === 1 ? "Uma impressora respondeu." : page.found.length + " impressoras responderam."
            wrapMode: Text.WordWrap
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 13
        }

        PillButton {
            id: searchButton
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            text: page.searching ? "procurando…" : page.found === null ? "procurar impressoras" : "procurar de novo"
            onClicked: page.search()
        }
    }

    Repeater {
        model: page.running && page.found !== null ? page.found : []

        Item {
            id: hit
            required property var modelData
            // Só o que vale agora: a fila pode ter sido removida depois da procura.
            readonly property bool added: page.isAdded(modelData.uri)
            readonly property bool noted: page.note !== "" && page.noteFor === modelData.uri

            Layout.fillWidth: true
            implicitHeight: 58 + (noted ? hitNote.implicitHeight + 12 : 0)

            Rectangle {
                width: parent.width
                height: 1
                color: Theme.hairline
            }

            Column {
                anchors.left: parent.left
                anchors.right: addSlot.left
                anchors.rightMargin: 16
                y: Math.round((58 - height) / 2)
                spacing: 5

                Text {
                    width: parent.width
                    text: hit.modelData.name
                    elide: Text.ElideRight
                    color: Theme.text
                    font.family: Theme.sans
                    font.pixelSize: 14
                }
                Text {
                    width: parent.width
                    text: (hit.modelData.kind === "usb" ? "usb" : "rede") + "  ·  "
                        + (hit.modelData.model !== "" && hit.modelData.model !== hit.modelData.name ? hit.modelData.model + "  ·  " : "")
                        + hit.modelData.uri
                    elide: Text.ElideRight
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }

            Item {
                id: addSlot
                anchors.right: parent.right
                y: 12
                width: hit.added ? addedLabel.implicitWidth : addButton.implicitWidth
                height: 34

                Text {
                    id: addedLabel
                    visible: hit.added
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    text: "JÁ ADICIONADA"
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 9
                    font.letterSpacing: Theme.labelSpacing
                }
                PillButton {
                    id: addButton
                    visible: !hit.added
                    anchors.right: parent.right
                    text: page.busy === "add:" + hit.modelData.uri ? "adicionando…" : "adicionar"
                    onClicked: page.act("add:" + hit.modelData.uri, ["add", hit.modelData.uri, hit.modelData.name])
                }
            }

            Note {
                id: hitNote
                visible: hit.noted
                y: 56
                width: parent.width
            }
        }
    }

    // USB que o CUPS enxerga mas que não fala IPP: não dá para adicionar daqui.
    Repeater {
        model: page.running && page.found !== null ? page.legacy : []

        Item {
            id: old
            required property var modelData

            Layout.fillWidth: true
            implicitHeight: 58

            Rectangle {
                width: parent.width
                height: 1
                color: Theme.hairline
            }

            Column {
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                spacing: 5

                Text {
                    width: parent.width
                    text: old.modelData.model !== "" ? old.modelData.model : old.modelData.uri
                    elide: Text.ElideRight
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 14
                }
                Text {
                    width: parent.width
                    text: "usb  ·  pede o driver do fabricante; o Mk só adiciona as que entram sem driver"
                    elide: Text.ElideRight
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }
        }
    }

    // O que a procura não pôde ver.
    Text {
        visible: page.running && page.found !== null && page.legacyUnknown
        Layout.fillWidth: true
        Layout.topMargin: 14
        text: "As USB que pedem o driver do fabricante não entraram na procura: o CUPS só as mostra a um administrador."
        wrapMode: Text.WordWrap
        color: Theme.textFaint
        font.family: Theme.sans
        font.pixelSize: 12
    }
}
