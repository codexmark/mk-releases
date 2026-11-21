import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

// Página "VPN" dos Ajustes: as conexões privadas do NetworkManager (WireGuard,
// que já vem no kernel, e as de plugin, como o OpenVPN). Quem fala com o
// nmcli é o scripts/mare-vpn, que responde em JSON. Cada conexão tem a chave
// de ligar, o estado, a interface e o IP do túnel; clicar no nome abre os
// detalhes (conectar ao iniciar, só no WireGuard; esquecer). Uma conexão nova entra por um
// arquivo .conf ou .ovpn: solto na página ou pelo caminho escrito.
// Senha: o Mk não tem um agente gráfico de segredos, então a VPN que pede
// usuário e senha conecta num terminal (`nmcli --ask`), nunca por argumento.
ColumnLayout {
    id: page

    spacing: 0

    property var info: ({ plugins: { wireguard: true, openvpn: false, openconnect: false }, nm: true, connections: [] })
    property bool loaded: false
    readonly property var connections: info.connections || []
    readonly property var active: connections.filter(c => c.state === "on")

    // Uma ação por vez: a conexão e o verbo (up, down, delete, autoconnect).
    property string busy: ""
    property string busyVerb: ""
    // O que deu errado, por conexão, e quais falharam por falta de senha.
    property var errors: ({})
    property var needsTerminal: ({})

    property string expanded: ""
    property string forgetting: ""

    // Importar: o que o campo tem, o que o script achou dele e o resultado.
    property var check: ({ ok: false, kind: "", name: "", path: "", error: "" })
    property string checkedText: ""
    property bool importing: false
    property string importNote: ""
    property string importError: ""

    readonly property string script: Quickshell.shellPath("scripts/mare-vpn")
    readonly property string openvpnPackage: "networkmanager-vpn-plugin-openvpn"

    function refresh() {
        if (listProc.running)
            listAgain = true;
        else
            listProc.running = true;
    }
    property bool listAgain: false
    // A última resposta do `list`, crua: a lista só é trocada quando muda.
    property string lastList: ""

    function patch(map, key, value) {
        const next = Object.assign({}, map);
        if (value === undefined)
            delete next[key];
        else
            next[key] = value;
        return next;
    }

    function run(verb, uuid, extra) {
        if (busy !== "" || actProc.running)
            return;
        busy = uuid;
        busyVerb = verb;
        errors = patch(errors, uuid, undefined);
        actProc.command = ["bash", script, verb, uuid].concat(extra !== undefined ? [extra] : []);
        actProc.running = true;
    }

    // Liga ou desliga. A que pede senha vai direto para o terminal.
    function toggle(c, on) {
        if (busy !== "")
            return;
        if (!on) {
            run("down", c.uuid);
        } else if (c.secrets || needsTerminal[c.uuid]) {
            openTerminal(c);
        } else {
            run("up", c.uuid);
        }
    }

    // O `nmcli --ask` pergunta usuário e senha dentro do terminal do Mk. A
    // folha dos Ajustes cobre a tela, então ela sai da frente (e esta página
    // some com ela: o estado novo vem do nmcli quando ela reabrir).
    function openTerminal(c) {
        Projects.openIn("", "bash " + Projects.shellQuote(script) + " ask " + c.uuid);
        Mare.close();
    }

    function forget(c) {
        if (forgetting !== c.uuid) {
            forgetting = c.uuid;
            forgetTimer.restart();
            return;
        }
        forgetting = "";
        run("delete", c.uuid);
    }

    // Com a lista de pacotes velha no disco o pacman pede uma versão que o
    // espelho já não tem; a saída é atualizar tudo (mk-update) e tentar de novo.
    function installOpenvpn() {
        const line = "sudo pacman -S --needed " + openvpnPackage
            + " || { echo; echo 'não instalou. se o erro foi de download (404), a lista de pacotes está velha:';"
            + " echo 'rode mk-update e tente de novo.'; }"
            + "; echo; read -r -n 1 -s -p 'aperte uma tecla para fechar'";
        Projects.openIn("", "bash -c " + Projects.shellQuote(line));
        Mare.close();
    }

    function kindName(c) {
        return c.kind === "wireguard" ? "wireguard" : c.kind === "openvpn" ? "openvpn" : c.kind;
    }

    function isOn(c) {
        if (busy === c.uuid && busyVerb === "up")
            return true;
        if (busy === c.uuid && (busyVerb === "down" || busyVerb === "delete"))
            return false;
        return c.state === "on" || c.state === "connecting";
    }

    function stateText(c) {
        if (busy === c.uuid && busyVerb === "up" || c.state === "connecting")
            return "conectando…";
        if (busy === c.uuid && busyVerb === "down" || c.state === "disconnecting")
            return "desconectando…";
        if (busy === c.uuid && busyVerb === "delete")
            return "esquecendo…";
        if (c.state === "on")
            return "conectada";
        if (errors[c.uuid])
            return "falhou: " + errors[c.uuid];
        if (c.secrets || needsTerminal[c.uuid])
            return "pede senha: conecta pelo terminal";
        return "desligada";
    }

    // Um arquivo solto chega como URL (file:///home/...%20...).
    function pathFromUrl(url) {
        const s = String(url);
        if (!s.startsWith("file://"))
            return s;
        try {
            return decodeURIComponent(s.slice(7));
        } catch (e) {
            return s.slice(7);
        }
    }

    function runCheck() {
        const text = pathInput.text.trim();
        if (text === "") {
            checkedText = "";
            check = ({ ok: false, kind: "", name: "", path: "", error: "" });
            return;
        }
        if (checkProc.running)
            return;
        checkProc.asked = text;
        checkProc.command = ["bash", script, "check", text];
        checkProc.running = true;
    }

    function doImport() {
        const text = pathInput.text.trim();
        if (importing || text === "")
            return;
        importNote = "";
        importError = "";
        importing = true;
        importProc.command = ["bash", script, "import", text];
        importProc.running = true;
    }

    Component.onCompleted: refresh()

    // O NetworkManager avisa quando algo muda (`nmcli monitor`); a lista é
    // relida um instante depois. O relógio lento cobre o que o aviso não diz
    // (o IP que chega depois do túnel subir).
    Process {
        id: watchProc
        running: true
        command: ["bash", page.script, "watch"]
        stdout: SplitParser {
            onRead: line => refreshTimer.restart()
        }
    }

    Timer {
        id: refreshTimer
        interval: 350
        onTriggered: page.refresh()
    }

    Timer {
        interval: 6000
        repeat: true
        running: true
        onTriggered: page.refresh()
    }

    Timer {
        id: forgetTimer
        interval: 3500
        onTriggered: page.forgetting = ""
    }

    Timer {
        id: checkTimer
        interval: 280
        onTriggered: page.runCheck()
    }

    Process {
        id: listProc
        command: ["bash", page.script, "list"]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    // Trocar `info` refaz todas as linhas da lista (o modelo é
                    // um array): só quando a resposta mudou, senão um clique
                    // que cruza a releitura se perde.
                    const parsed = JSON.parse(text);
                    if (parsed && parsed.connections && text !== page.lastList) {
                        page.lastList = text;
                        page.info = parsed;
                    }
                } catch (e) {}
                page.loaded = true;
            }
        }
        onExited: {
            if (page.listAgain) {
                page.listAgain = false;
                running = true;
            }
        }
    }

    Process {
        id: actProc
        stdout: StdioCollector {
            onStreamFinished: {
                let result = ({ ok: false, error: "sem resposta do mare-vpn", needsSecrets: false });
                try {
                    result = JSON.parse(text);
                } catch (e) {}
                const uuid = page.busy;
                if (!result.ok) {
                    page.errors = page.patch(page.errors, uuid, result.error);
                    if (result.needsSecrets)
                        page.needsTerminal = page.patch(page.needsTerminal, uuid, true);
                }
                if (page.busyVerb === "delete" && result.ok && page.expanded === uuid)
                    page.expanded = "";
                page.busy = "";
                page.busyVerb = "";
                page.refresh();
            }
        }
    }

    Process {
        id: checkProc
        property string asked: ""
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    page.check = JSON.parse(text);
                    page.checkedText = checkProc.asked;
                } catch (e) {}
                // O campo mudou enquanto o script conferia: confere de novo.
                if (pathInput.text.trim() !== checkProc.asked)
                    checkTimer.restart();
            }
        }
    }

    Process {
        id: importProc
        stdout: StdioCollector {
            onStreamFinished: {
                let result = ({ ok: false, error: "sem resposta do mare-vpn", name: "", kind: "", uuid: "" });
                try {
                    result = JSON.parse(text);
                } catch (e) {}
                page.importing = false;
                if (result.ok) {
                    page.importNote = "importada: " + result.name + " (" + result.kind + "). "
                        + (result.kind === "wireguard"
                            ? "o networkmanager a liga ao importar; já foi desligada. ligue na lista acima."
                            : "está desligada; ligue na lista acima.");
                    pathInput.text = "";
                    page.expanded = result.uuid;
                } else {
                    page.importError = result.error;
                }
                page.refresh();
            }
        }
    }

    component Label: Text {
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.capitalization: Font.AllUppercase
        font.letterSpacing: Theme.labelSpacing
    }

    component InfoPair: ColumnLayout {
        property string label: ""
        property string value: ""

        spacing: 3
        Layout.fillWidth: true

        Label { text: parent.label }

        Text {
            Layout.fillWidth: true
            text: parent.value !== "" ? parent.value : "—"
            color: Theme.text
            font.family: Theme.mono
            font.pixelSize: 12
            elide: Text.ElideRight
        }
    }

    component Hairline: Rectangle {
        Layout.fillWidth: true
        implicitHeight: 1
        color: Theme.hairline
    }

    // ── o túnel aberto ─────────────────────────────────────────────────

    ColumnLayout {
        visible: page.active.length > 0
        Layout.fillWidth: true
        Layout.topMargin: 18
        spacing: 12

        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Text {
                text: "󰦝"
                color: Theme.copper
                font.family: Theme.icons
                font.pixelSize: 16
            }

            Text {
                Layout.fillWidth: true
                text: page.active.length > 1 ? page.active.length + " túneis abertos" : "túnel aberto"
                color: Theme.textSoft
                font.family: Theme.serif
                font.pixelSize: 16
                font.italic: true
                font.weight: Font.Light
            }
        }

        Repeater {
            model: page.active

            RowLayout {
                required property var modelData
                Layout.fillWidth: true
                spacing: 16

                InfoPair { label: "vpn"; value: modelData.name }
                InfoPair { label: "interface"; value: modelData.iface }
                InfoPair { label: "ip no túnel"; value: modelData.ip }
            }
        }
    }

    // ── as conexões ────────────────────────────────────────────────────

    RowLayout {
        Layout.fillWidth: true
        Layout.topMargin: 22
        Layout.bottomMargin: 8

        Label {
            Layout.fillWidth: true
            text: "conexões"
        }

        Label {
            visible: page.connections.length > 0
            text: page.connections.length + (page.connections.length === 1 ? " guardada" : " guardadas")
        }
    }

    Hairline {}

    // Nenhuma ainda: o convite a importar.
    ColumnLayout {
        visible: page.loaded && page.connections.length === 0
        Layout.fillWidth: true
        Layout.topMargin: 20
        Layout.bottomMargin: 20
        spacing: 6

        Text {
            text: page.info.nm === false ? "o networkmanager não respondeu." : "nenhuma vpn guardada."
            color: Theme.textSoft
            font.family: Theme.serif
            font.pixelSize: 18
            font.italic: true
            font.weight: Font.Light
        }

        Text {
            Layout.fillWidth: true
            visible: page.info.nm !== false
            wrapMode: Text.WordWrap
            text: "Traga o arquivo que o provedor ou o seu servidor entrega: .conf para WireGuard, .ovpn para OpenVPN. Ele entra logo abaixo e a conexão fica aqui, desligada, até você ligar."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 12
            lineHeight: 1.35
        }
    }

    Hairline {
        visible: page.loaded && page.connections.length === 0
    }

    Repeater {
        model: page.connections

        ColumnLayout {
            id: entry
            required property var modelData
            readonly property bool on: modelData.state === "on"
            readonly property bool open: page.expanded === modelData.uuid
            readonly property bool failed: page.errors[modelData.uuid] !== undefined && !on

            Layout.fillWidth: true
            spacing: 0

            Item {
                Layout.fillWidth: true
                implicitHeight: 58

                Rectangle {
                    anchors.fill: parent
                    color: entry.on || rowHover.hovered ? Theme.wash : "transparent"

                    Behavior on color {
                        ColorAnimation { duration: Motion.fast }
                    }
                }

                Rectangle {
                    visible: entry.on
                    width: 2
                    height: parent.height
                    color: Theme.copper
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 14
                    anchors.rightMargin: 12
                    spacing: 14

                    // Nome e estado: clicar aqui abre os detalhes. A chave fica
                    // fora desta área, para o clique nela não abrir também.
                    Item {
                        Layout.fillWidth: true
                        Layout.fillHeight: true

                        Column {
                            anchors.verticalCenter: parent.verticalCenter
                            width: parent.width
                            spacing: 5

                            Text {
                                width: parent.width
                                text: entry.modelData.name
                                color: entry.on ? Theme.text : Theme.textSoft
                                font.family: Theme.sans
                                font.pixelSize: 14
                                elide: Text.ElideRight
                            }

                            Row {
                                width: parent.width
                                spacing: 8

                                Label {
                                    id: kindLabel
                                    text: page.kindName(entry.modelData)
                                }

                                Text {
                                    width: parent.width - kindLabel.width - 8
                                    text: page.stateText(entry.modelData)
                                        + (entry.on && entry.modelData.iface !== "" ? "  ·  " + entry.modelData.iface : "")
                                        + (entry.on && entry.modelData.ip !== "" ? "  ·  " + entry.modelData.ip : "")
                                    color: entry.on || entry.failed ? Theme.copper : Theme.textDim
                                    font.family: Theme.mono
                                    font.pixelSize: 10
                                    elide: Text.ElideRight
                                }
                            }
                        }

                        HoverHandler {
                            id: rowHover
                            cursorShape: Qt.PointingHandCursor
                        }

                        TapHandler {
                            onTapped: page.expanded = entry.open ? "" : entry.modelData.uuid
                        }
                    }

                    Text {
                        text: entry.open ? "󰅃" : "󰅀"
                        color: Theme.textFaint
                        font.family: Theme.icons
                        font.pixelSize: 14
                    }

                    MkToggle {
                        opacity: page.busy !== "" ? 0.5 : 1
                        checked: page.isOn(entry.modelData)
                        onToggled: value => page.toggle(entry.modelData, value)
                    }
                }
            }

            // Detalhes.
            ColumnLayout {
                visible: entry.open
                Layout.fillWidth: true
                Layout.leftMargin: 14
                Layout.rightMargin: 12
                Layout.topMargin: 6
                Layout.bottomMargin: 14
                spacing: 12

                RowLayout {
                    Layout.fillWidth: true
                    spacing: 16

                    InfoPair { label: "tipo"; value: page.kindName(entry.modelData) }
                    InfoPair { label: "interface"; value: entry.modelData.iface }
                    InfoPair { label: "ip no túnel"; value: entry.modelData.ip }
                    InfoPair {
                        visible: entry.modelData.type === "vpn"
                        label: "usuário"
                        value: entry.modelData.username
                    }
                }

                Text {
                    Layout.fillWidth: true
                    visible: entry.modelData.secrets || page.needsTerminal[entry.modelData.uuid] === true
                    wrapMode: Text.WordWrap
                    text: "Esta conexão pede usuário e senha. O Mk não guarda nem repassa a senha: ao ligar, abre um terminal e o NetworkManager pergunta ali."
                    color: Theme.textDim
                    font.family: Theme.serif
                    font.pixelSize: 13
                    font.italic: true
                    lineHeight: 1.3
                }

                Text {
                    Layout.fillWidth: true
                    visible: entry.failed
                    wrapMode: Text.WordWrap
                    text: page.errors[entry.modelData.uuid] || ""
                    color: Theme.copper
                    font.family: Theme.mono
                    font.pixelSize: 11
                }

                // O NetworkManager só liga sozinho o WireGuard; nas VPNs de
                // plugin (OpenVPN) o autoconnect existe no perfil e não faz nada.
                RowLayout {
                    visible: entry.modelData.type === "wireguard"
                    Layout.fillWidth: true
                    spacing: 12

                    Text {
                        Layout.fillWidth: true
                        text: "conectar ao iniciar"
                        color: Theme.textSoft
                        font.family: Theme.sans
                        font.pixelSize: 13
                    }

                    MkToggle {
                        opacity: page.busy !== "" ? 0.5 : 1
                        checked: page.busy === entry.modelData.uuid && page.busyVerb === "autoconnect"
                            ? !entry.modelData.autoconnect : entry.modelData.autoconnect
                        onToggled: value => page.run("autoconnect", entry.modelData.uuid, value ? "yes" : "no")
                    }
                }

                Text {
                    Layout.fillWidth: true
                    visible: entry.modelData.type !== "wireguard"
                    wrapMode: Text.WordWrap
                    text: "O NetworkManager não liga sozinho uma vpn de plugin: esta se liga à mão, aqui."
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 13
                    font.italic: true
                    lineHeight: 1.3
                }

                RowLayout {
                    Layout.fillWidth: true
                    spacing: 8

                    PillButton {
                        visible: entry.modelData.type === "vpn" && !entry.on
                        text: "conectar pelo terminal"
                        onClicked: page.openTerminal(entry.modelData)
                    }

                    Item { Layout.fillWidth: true }

                    PillButton {
                        primary: page.forgetting === entry.modelData.uuid
                        text: page.forgetting === entry.modelData.uuid ? "esquecer mesmo?" : "esquecer"
                        onClicked: page.forget(entry.modelData)
                    }
                }
            }

            Hairline {}
        }
    }

    // ── importar ───────────────────────────────────────────────────────

    Label {
        Layout.topMargin: 26
        Layout.bottomMargin: 8
        text: "importar arquivo .conf / .ovpn"
    }

    // Onde soltar o arquivo (arrastado do gerenciador de arquivos).
    Rectangle {
        Layout.fillWidth: true
        implicitHeight: 76
        radius: 3
        color: dropArea.containsDrag ? Theme.wash : "transparent"
        border.width: 1
        border.color: dropArea.containsDrag ? Theme.copper : Theme.hairline

        Behavior on border.color {
            ColorAnimation { duration: Motion.fast }
        }

        Column {
            anchors.centerIn: parent
            spacing: 5

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: dropArea.containsDrag ? "pode soltar" : "solte o arquivo aqui"
                color: dropArea.containsDrag ? Theme.copper : Theme.textSoft
                font.family: Theme.serif
                font.pixelSize: 15
                font.italic: true
                font.weight: Font.Light
            }

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "ou escreva o caminho abaixo"
            }
        }

        DropArea {
            id: dropArea
            anchors.fill: parent
            onDropped: drop => {
                let path = "";
                if (drop.hasUrls && drop.urls.length > 0)
                    path = page.pathFromUrl(drop.urls[0]);
                else if (drop.hasText)
                    path = page.pathFromUrl(drop.text.trim().split("\n")[0].trim());
                if (path === "")
                    return;
                drop.accept();
                pathInput.text = path;
                pathInput.forceActiveFocus();
            }
        }
    }

    RowLayout {
        Layout.fillWidth: true
        Layout.topMargin: 12
        spacing: 10

        Item {
            Layout.fillWidth: true
            implicitHeight: 32

            TextInput {
                id: pathInput
                anchors.fill: parent
                anchors.bottomMargin: 5
                verticalAlignment: TextInput.AlignVCenter
                color: Theme.text
                selectionColor: Theme.copper
                font.family: Theme.mono
                font.pixelSize: 12
                clip: true
                selectByMouse: true
                cursorDelegate: Rectangle {
                    width: 2
                    color: Theme.copper
                    visible: pathInput.activeFocus
                }
                onTextChanged: {
                    page.importError = "";
                    if (text !== "")
                        page.importNote = "";
                    checkTimer.restart();
                }
                Keys.onReturnPressed: page.doImport()
                Keys.onEnterPressed: page.doImport()

                Text {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: pathInput.text === ""
                    text: "~/Downloads/minha-vpn.conf"
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 12
                }
            }

            Rectangle {
                anchors.bottom: parent.bottom
                width: parent.width
                height: 1
                color: page.importError !== "" || (page.check.error !== "" && pathInput.text !== "") ? Theme.copper
                    : pathInput.activeFocus ? Theme.line : Theme.hairline
            }
        }

        PillButton {
            primary: page.check.ok && !page.importing
            opacity: pathInput.text.trim() === "" ? 0.5 : 1
            text: page.importing ? "importando…" : "importar"
            onClicked: page.doImport()
        }
    }

    // O que o script achou do arquivo, ou o que deu errado.
    Text {
        Layout.fillWidth: true
        Layout.topMargin: 8
        visible: text !== ""
        wrapMode: Text.WordWrap
        text: page.importError !== "" ? "não importou: " + page.importError
            : page.importNote !== "" ? page.importNote
            : pathInput.text.trim() === "" ? ""
            : page.check.error !== "" ? page.check.error
            : page.check.ok ? page.check.kind + " · pronto para importar como " + page.check.name
            : ""
        color: page.importError !== "" || (page.importNote === "" && page.check.error !== "") ? Theme.copper : Theme.textDim
        font.family: Theme.mono
        font.pixelSize: 11
    }

    // ── o que o sistema sabe falar ─────────────────────────────────────

    Label {
        Layout.topMargin: 26
        Layout.bottomMargin: 8
        text: "protocolos"
    }

    Hairline {}

    Repeater {
        model: [
            { name: "WireGuard", ok: true, note: "no kernel, sem plugin" },
            { name: "OpenVPN", ok: page.info.plugins.openvpn === true,
              note: page.info.plugins.openvpn === true ? "plugin instalado" : "falta o pacote " + page.openvpnPackage,
              install: true }
        ].concat(page.info.plugins.openconnect === true
            ? [{ name: "OpenConnect", ok: true, note: "plugin instalado" }] : [])

        ColumnLayout {
            id: proto
            required property var modelData
            Layout.fillWidth: true
            spacing: 0

            RowLayout {
                Layout.fillWidth: true
                Layout.preferredHeight: 42
                Layout.leftMargin: 14
                Layout.rightMargin: 12
                spacing: 12

                Text {
                    text: proto.modelData.name
                    color: proto.modelData.ok ? Theme.textSoft : Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 13
                }

                Item { Layout.fillWidth: true }

                Text {
                    text: proto.modelData.note
                    color: proto.modelData.ok ? Theme.textDim : Theme.copper
                    font.family: Theme.mono
                    font.pixelSize: 10
                }

                PillButton {
                    visible: !proto.modelData.ok && proto.modelData.install === true && page.loaded
                    implicitHeight: 28
                    text: "instalar"
                    onClicked: page.installOpenvpn()
                }
            }

            Hairline {}
        }
    }

    Text {
        Layout.fillWidth: true
        Layout.topMargin: 14
        wrapMode: Text.WordWrap
        text: "As conexões ficam no NetworkManager, não no Mk: o que entra aqui também aparece no nmtui e no nmcli."
        color: Theme.textFaint
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
    }
}
