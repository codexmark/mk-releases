import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import Quickshell.Networking

// Rede: o que está conectado e como está indo. Detalhes da conexão (IP,
// gateway, DNS), tráfego ao vivo, ping até o roteador e até a internet,
// escolha de DNS, QR code para o celular entrar no Wi-Fi e as redes ao redor.
// Ideias vindas da barra do Omarchy, refeitas no estilo do Maré.
Item {
    id: panel

    readonly property var wifi: Networking.devices.values.find(d => d.type === DeviceType.Wifi) ?? null
    readonly property var networks: wifi ? nearest(wifi.networks.values, 6) : []

    // Entrar numa rede: a que está pedindo senha, e o que deu errado.
    property var asking: null
    property string wifiError: ""

    // A tentativa em curso: a rede, se foi com a senha guardada, e se é uma
    // rede nova (o perfil que ela deixar para trás ao falhar é apagado).
    // `freshSsid`: a rede nova escolhida, enquanto o campo de senha dela vive.
    property var trying: null
    property bool tryingStored: false
    property bool tryingFresh: false
    property string freshSsid: ""

    // Redes guardadas (scripts/mare-wifi): nome → reconecta sozinho?
    property var saved: ({})
    property bool savedLoaded: false
    property var wifiQueue: []
    // A linha de opções aberta, e o "esquecer" esperando o segundo clique.
    property string openSsid: ""
    property string forgetSsid: ""

    function pick(network) {
        wifiError = "";
        forgetSsid = "";
        if (network.connected || network.stateChanging)
            return;
        // Nova só se o NetworkManager não tem mesmo nenhum perfil com esse nome:
        // a faxina apaga pelo nome, e não pode levar um perfil que já existia.
        if (freshSsid !== network.name)
            freshSsid = network.known || !savedLoaded || saved[network.name] !== undefined ? "" : network.name;
        if (network.known || network.security === WifiSecurityType.Open) {
            asking = null;
            trying = network;
            tryingStored = network.known;
            tryingFresh = freshSsid === network.name;
            network.connect();
        } else {
            askFor(network);
        }
    }

    function askFor(network) {
        asking = network;
        secret.text = "";
        secret.forceActiveFocus();
    }

    function dismiss() {
        asking = null;
        freshSsid = "";
    }

    function join() {
        if (!asking || secret.text.length < 8) {
            wifiError = "a senha tem pelo menos 8 caracteres";
            return;
        }
        wifiError = "";
        trying = asking;
        tryingStored = false;
        tryingFresh = freshSsid === asking.name;
        // A senha vai para o NetworkManager pelo D-Bus, não por linha de comando.
        asking.connectWithPsk(secret.text);
        secret.text = "";
    }

    // Uma ação do mare-wifi por vez; o nome da rede vai como um argumento só.
    // `quiet`: a faxina depois de uma falha não mostra o erro dela.
    function wifiDo(args, quiet) {
        wifiQueue = wifiQueue.concat([{ args: args, quiet: quiet === true }]);
        wifiNext();
    }

    function wifiNext() {
        if (wifiProc.running || wifiQueue.length === 0)
            return;
        wifiProc.quiet = wifiQueue[0].quiet;
        wifiProc.command = ["bash", Quickshell.shellPath("scripts/mare-wifi")].concat(wifiQueue[0].args);
        wifiQueue = wifiQueue.slice(1);
        wifiProc.running = true;
    }

    function autoconnects(ssid) {
        return saved[ssid] !== false;
    }

    function setAutoconnect(ssid, on) {
        const next = Object.assign({}, saved);
        next[ssid] = on;
        saved = next;
        wifiDo(["autoconnect", ssid, on ? "on" : "off"]);
    }

    function forget(ssid) {
        wifiError = "";
        if (forgetSsid !== ssid) {
            forgetSsid = ssid;
            forgetReset.restart();
            return;
        }
        forgetSsid = "";
        openSsid = "";
        if (asking && asking.name === ssid)
            asking = null;
        wifiDo(["forget", ssid]);
    }

    Timer {
        id: forgetReset
        interval: 4000
        onTriggered: panel.forgetSsid = ""
    }

    Connections {
        target: panel.trying
        ignoreUnknownSignals: true
        function onConnectedChanged() {
            if (panel.trying && panel.trying.connected) {
                if (!panel.asking || panel.asking === panel.trying)
                    panel.dismiss();
                panel.trying = null;
                netProc.running = true;
                panel.wifiDo(["list-saved"]);
            }
        }
        function onConnectionFailed(reason) {
            const network = panel.trying;
            if (!network)
                return;
            panel.trying = null;
            if (panel.tryingFresh)
                panel.wifiDo(["forget", network.name], true);
            // O campo já é de outra rede: a falha desta não mexe nele.
            if (panel.asking && panel.asking !== network)
                return;
            if (!panel.asking)
                panel.freshSsid = "";
            // Só a recusa da senha pede outra (que substitui a guardada); rede
            // fora de alcance ou roteador reiniciando não é senha errada.
            if (panel.tryingStored && network.security !== WifiSecurityType.Open
                    && reason === ConnectionFailReason.NoSecrets) {
                panel.askFor(network);
                panel.wifiError = "a senha guardada não serve mais";
                return;
            }
            panel.wifiError = panel.asking ? "não conectou: confira a senha" : "não conectou";
            if (panel.asking)
                secret.forceActiveFocus();
        }
    }

    // Detalhes da conexão (scripts/mare-net).
    property var net: ({ iface: "", type: "none", conn: "", ip: "", gateway: "", dns: [], dnsMode: "auto" })
    readonly property bool connected: net.type !== "none"

    // Ping: as últimas 24 medidas (em ms; -1 = sem resposta).
    property var routerPings: []
    property var internetPings: []
    readonly property real routerMs: last(routerPings)
    readonly property real internetMs: last(internetPings)
    readonly property int loss: lossOf(internetPings)

    property string dnsBusy: ""
    property bool showQr: false
    property string qrSsid: ""
    property string qrError: ""
    readonly property string qrPath: Quickshell.env("XDG_RUNTIME_DIR") + "/mare-wifi-qr.png"

    // Frases que alternam enquanto há conexão, como na barra do Omarchy.
    readonly property var phrases: ["Carregando bits", "Empurrando pacotes", "Dobrando luz",
        "Contando quadros", "Roteando migalhas", "Despachando bytes", "Costurando sinais"]
    property int phraseIndex: 0

    implicitWidth: 460
    implicitHeight: column.implicitHeight + 44

    // --- puro: início ---
    // Daqui até "puro: fim" nada toca no Maré nem no NetworkManager: são
    // funções de lista para lista e de texto para número, testadas fora do
    // shell com node (tests/js/NetworkPanel.test.mjs).
    // As redes ao redor: a conectada primeiro, depois as de sinal mais forte.
    function nearest(list, max) {
        return list.slice()
            .sort((a, b) => (b.connected - a.connected) || (b.signalStrength - a.signalStrength))
            .slice(0, max);
    }

    function last(list) {
        return list.length > 0 ? list[list.length - 1] : -1;
    }

    function push(list, value) {
        const next = list.concat([value]);
        return next.length > 24 ? next.slice(next.length - 24) : next;
    }

    // Abaixo de 10 ms, uma casa (até 9,9: 9,96 já é "10 ms", não "10,0 ms").
    function ms(value) {
        return value < 0 ? "—" : value < 9.95 ? value.toFixed(1).replace(".", ",") + " ms" : Math.round(value) + " ms";
    }

    // Quantos por cento das medidas ficaram sem resposta (-1).
    function lossOf(pings) {
        return pings.length === 0 ? 0
            : Math.round(100 * pings.filter(p => p < 0).length / pings.length);
    }

    // A saída do `ping -c 1` -> o tempo em ms; -1 = sem resposta.
    function pingMs(text) {
        const m = text.match(/time=([0-9.]+)/);
        return m ? Number(m[1]) : -1;
    }

    // As redes guardadas que o mare-wifi listou ([{ ssid, autoconnect }]) ->
    // nome → reconecta sozinho? Dois perfis com o mesmo nome: vale o que
    // reconecta.
    function savedMap(list) {
        const next = {};
        for (const s of list)
            next[s.ssid] = next[s.ssid] === true || s.autoconnect;
        return next;
    }

    // A linha da VPN: o túnel aberto, quantos mais, a interface e o IP.
    function vpnLine(status) {
        return status.name + (status.count > 1 ? " +" + (status.count - 1) : "")
            + (status.iface !== "" ? "  ·  " + status.iface : "")
            + (status.ip !== "" ? "  ·  " + status.ip : "");
    }
    // --- puro: fim ---

    function setDns(mode) {
        if (dnsBusy !== "" || mode === net.dnsMode)
            return;
        dnsBusy = mode;
        dnsProc.command = ["bash", Quickshell.shellPath("scripts/mare-dns"), mode];
        dnsProc.running = true;
    }

    function toggleQr() {
        showQr = !showQr;
        if (showQr) {
            qrError = "";
            qrSsid = "";
            qrProc.running = true;
        } else {
            removeQr.running = true;
        }
    }

    // Procura redes só enquanto o painel está aberto.
    Component.onCompleted: {
        if (wifi)
            wifi.scannerEnabled = true;
        netProc.running = true;
        wifiDo(["list-saved"]);
    }
    Component.onDestruction: {
        if (wifi)
            wifi.scannerEnabled = false;
        // Fechar o painel não pode cortar um "esquecer" pela metade: o que
        // estava rodando ou na fila roda de novo, solto (repetir não faz mal).
        const pending = (wifiProc.running ? [wifiProc.command.slice(2)] : []).concat(wifiQueue.map(q => q.args));
        for (const args of pending) {
            if (args[0] !== "list-saved")
                Quickshell.execDetached(["bash", Quickshell.shellPath("scripts/mare-wifi")].concat(args));
        }
        Quickshell.execDetached(["rm", "-f", qrPath]);
    }

    Timer {
        interval: 2800
        repeat: true
        running: panel.connected
        onTriggered: panel.phraseIndex = (panel.phraseIndex + 1) % panel.phrases.length
    }

    Timer {
        interval: 2000
        repeat: true
        running: panel.connected
        triggeredOnStart: true
        onTriggered: {
            if (panel.net.gateway !== "")
                routerPing.running = true;
            internetPing.running = true;
        }
    }

    Process {
        id: netProc
        command: ["bash", Quickshell.shellPath("scripts/mare-net")]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    panel.net = JSON.parse(text);
                } catch (e) {}
            }
        }
    }

    // ping -c 1: "time=12.3 ms" (em LC_ALL=C; em português sai "tempo="); sem resposta conta como perda.
    Process {
        id: routerPing
        command: ["env", "LC_ALL=C", "ping", "-n", "-c", "1", "-W", "1", panel.net.gateway]
        stdout: StdioCollector {
            onStreamFinished: {
                panel.routerPings = panel.push(panel.routerPings, panel.pingMs(text));
            }
        }
    }

    Process {
        id: internetPing
        command: ["env", "LC_ALL=C", "ping", "-n", "-c", "1", "-W", "1", "1.1.1.1"]
        stdout: StdioCollector {
            onStreamFinished: {
                panel.internetPings = panel.push(panel.internetPings, panel.pingMs(text));
            }
        }
    }

    Process {
        id: dnsProc
        onExited: {
            panel.dnsBusy = "";
            netProc.running = true;
        }
    }

    Process {
        id: wifiProc
        property bool quiet: false
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    const reply = JSON.parse(text);
                    if (reply.saved) {
                        panel.saved = panel.savedMap(reply.saved);
                        panel.savedLoaded = reply.ok === true;
                    } else if (!reply.ok && reply.error && !wifiProc.quiet) {
                        panel.wifiError = reply.error;
                    }
                } catch (e) {}
            }
        }
        onExited: {
            if (command[2] !== "list-saved" && panel.wifiQueue.length === 0)
                panel.wifiQueue = [{ args: ["list-saved"], quiet: true }];
            Qt.callLater(panel.wifiNext);
        }
    }

    Process {
        id: qrProc
        command: ["bash", Quickshell.shellPath("scripts/mare-wifi-qr"), panel.qrPath]
        stdout: StdioCollector {
            onStreamFinished: panel.qrSsid = text.trim()
        }
        stderr: StdioCollector {
            onStreamFinished: if (text.trim() !== "") panel.qrError = text.trim()
        }
    }

    Process {
        id: removeQr
        command: ["rm", "-f", panel.qrPath]
    }

    component InfoPair: ColumnLayout {
        property string label: ""
        property string value: ""

        spacing: 3
        Layout.fillWidth: true

        Text {
            text: parent.label
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 9
            font.capitalization: Font.AllUppercase
            font.letterSpacing: 1.6
        }

        Text {
            Layout.fillWidth: true
            text: parent.value !== "" ? parent.value : "—"
            color: Theme.text
            font.family: Theme.mono
            font.pixelSize: 12
            elide: Text.ElideRight
        }
    }

    ColumnLayout {
        id: column
        x: 22
        y: 22
        width: parent.width - 44
        spacing: 12

        RowLayout {
            Layout.fillWidth: true
            spacing: 14

            PanelHeader {
                Layout.fillWidth: true
                index: "01"
                title: panel.net.type === "ethernet" ? "Ethernet" : "Wi-Fi"
                meta: !Networking.wifiEnabled && panel.net.type !== "ethernet" ? "desligado"
                    : panel.connected ? "conectado" : "sem conexão"
                metaColor: panel.connected ? Theme.copper : Theme.textDim
            }

            Rectangle {
                width: 36
                height: 20
                radius: 10
                color: Networking.wifiEnabled ? Theme.copper : Theme.line

                Rectangle {
                    x: Networking.wifiEnabled ? parent.width - width - 2 : 2
                    y: 2
                    width: 16
                    height: 16
                    radius: 8
                    color: Theme.ground

                    Behavior on x {
                        NumberAnimation {
                            duration: Motion.fast
                            easing.type: Easing.BezierSpline
                            easing.bezierCurve: Motion.curve
                        }
                    }
                }

                TapHandler {
                    onTapped: Networking.wifiEnabled = !Networking.wifiEnabled
                }
            }
        }

        // Frase do momento.
        Text {
            visible: panel.connected
            text: panel.phrases[panel.phraseIndex] + "…"
            color: Theme.textDim
            font.family: Theme.serif
            font.pixelSize: 15
            font.italic: true
            font.weight: Font.Light
        }

        // Detalhes da conexão.
        GridLayout {
            visible: panel.connected
            Layout.fillWidth: true
            columns: 3
            rowSpacing: 12
            columnSpacing: 16

            InfoPair { label: "rede"; value: panel.net.conn }
            InfoPair { label: "ip"; value: panel.net.ip }
            InfoPair { label: "gateway"; value: panel.net.gateway }
            InfoPair { label: "dns"; value: panel.net.dns.join(" · ") }
            InfoPair { label: "recebendo"; value: "↓ " + Stats.rate(Stats.rxRate) }
            InfoPair { label: "enviando"; value: "↑ " + Stats.rate(Stats.txRate) }
        }

        // Ping: roteador, internet e perda, com o último minuto em gráfico.
        RowLayout {
            visible: panel.connected
            Layout.fillWidth: true
            spacing: 16

            InfoPair { label: "ping roteador"; value: panel.ms(panel.routerMs) }
            InfoPair { label: "ping internet"; value: panel.ms(panel.internetMs) }
            InfoPair { label: "perda"; value: panel.loss + "%" }

            Sparkline {
                Layout.preferredWidth: 120
                Layout.preferredHeight: 30
                values: panel.internetPings.map(p => p < 0 ? 0 : p)
                normalize: true
                color: panel.loss > 10 ? Theme.copper : Theme.textSoft
            }
        }

        Rectangle {
            visible: panel.connected
            Layout.fillWidth: true
            implicitHeight: 1
            color: Theme.hairline
        }

        // DNS: o do roteador (automático), Cloudflare ou Google.
        RowLayout {
            visible: panel.connected
            Layout.fillWidth: true
            spacing: 6

            Text {
                text: "dns"
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 9
                font.capitalization: Font.AllUppercase
                font.letterSpacing: 1.6
                Layout.rightMargin: 8
            }

            Repeater {
                model: [
                    { mode: "auto", label: "automático" },
                    { mode: "cloudflare", label: "cloudflare" },
                    { mode: "google", label: "google" }
                ]

                PillButton {
                    required property var modelData
                    Layout.fillWidth: true
                    text: panel.dnsBusy === modelData.mode ? "aplicando…" : modelData.label
                    selected: panel.net.dnsMode === modelData.mode
                    onClicked: panel.setDns(modelData.mode)
                }
            }
        }

        // VPN: o túnel aberto, numa linha. O resto mora nos Ajustes.
        RowLayout {
            visible: !panel.showQr
            Layout.fillWidth: true
            spacing: 8

            Text {
                text: "vpn"
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 9
                font.capitalization: Font.AllUppercase
                font.letterSpacing: 1.6
                Layout.rightMargin: 8
            }

            Text {
                visible: Vpn.up
                text: "󰦝"
                color: Theme.copper
                font.family: Theme.icons
                font.pixelSize: 14
            }

            Text {
                Layout.fillWidth: true
                text: !Vpn.up ? "desligada" : panel.vpnLine(Vpn.status)
                color: Vpn.up ? Theme.text : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
                elide: Text.ElideRight
            }

            PillButton {
                implicitHeight: 28
                text: "ajustes"
                onClicked: {
                    Mare.settingsPage = "vpn";
                    Mare.open("settings");
                }
            }
        }

        // QR code para o celular entrar no Wi-Fi.
        ColumnLayout {
            visible: panel.showQr
            Layout.fillWidth: true
            spacing: 8

            Rectangle {
                Layout.alignment: Qt.AlignHCenter
                implicitWidth: 196
                implicitHeight: 196
                color: Theme.surfaceLow
                border.width: 1
                border.color: Theme.line

                Image {
                    anchors.fill: parent
                    anchors.margins: 6
                    visible: panel.qrSsid !== ""
                    source: panel.qrSsid !== "" ? "file://" + panel.qrPath : ""
                    cache: false
                    smooth: false
                    fillMode: Image.PreserveAspectFit
                }

                Text {
                    anchors.centerIn: parent
                    visible: panel.qrSsid === ""
                    width: parent.width - 24
                    horizontalAlignment: Text.AlignHCenter
                    wrapMode: Text.WordWrap
                    text: panel.qrError !== "" ? panel.qrError : "gerando…"
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
            }

            Text {
                Layout.alignment: Qt.AlignHCenter
                text: panel.qrSsid !== "" ? "aponte a câmera para entrar em " + panel.qrSsid : ""
                color: Theme.textSoft
                font.family: Theme.serif
                font.pixelSize: 14
                font.italic: true
            }
        }

        // Redes ao redor.
        Repeater {
            model: Networking.wifiEnabled && !panel.showQr ? panel.networks : []

            Item {
                id: row
                required property var modelData
                required property int index
                readonly property bool open: modelData.known && panel.openSsid === modelData.name
                readonly property bool confirming: panel.forgetSsid !== "" && panel.forgetSsid === modelData.name
                Layout.fillWidth: true
                implicitHeight: open ? 68 : 38

                Rectangle {
                    anchors.fill: parent
                    color: row.modelData.connected || row.open || hover.hovered || moreHover.hovered ? Theme.wash : "transparent"
                }

                Rectangle {
                    visible: row.modelData.connected
                    width: 2
                    height: parent.height
                    color: Theme.copper
                }

                // A área de clique da rede e a das opções ficam lado a lado, não
                // uma sobre a outra: senão o clique nas opções também conectaria.
                Item {
                    id: main
                    width: parent.width - (more.visible ? more.width : 0)
                    height: 38

                    RowLayout {
                        anchors.fill: parent
                        anchors.leftMargin: 12
                        anchors.rightMargin: more.visible ? 4 : 10
                        spacing: 12

                        Row {
                            spacing: 2
                            Layout.alignment: Qt.AlignVCenter

                            Repeater {
                                model: 3

                                Rectangle {
                                    required property int index
                                    anchors.bottom: parent.bottom
                                    width: 3
                                    height: 5 + index * 4.5
                                    color: row.modelData.signalStrength > index / 3
                                        ? (row.modelData.connected ? Theme.copper : Theme.textSoft) : Theme.line
                                }
                            }
                        }

                        Text {
                            text: row.modelData.name
                            color: row.modelData.connected ? Theme.text : Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 13
                            elide: Text.ElideRight
                            Layout.fillWidth: true
                        }

                        Text {
                            text: row.modelData.connected ? "conectado" : row.modelData.stateChanging ? "conectando…" : row.modelData.known ? "salva"
                                : row.modelData.security === WifiSecurityType.Open ? "aberta" : "󰌾"
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                            font.capitalization: Font.AllUppercase
                            font.letterSpacing: 1.2
                        }
                    }

                    HoverHandler {
                        id: hover
                        cursorShape: row.modelData.connected ? Qt.ArrowCursor : Qt.PointingHandCursor
                    }

                    TapHandler {
                        onTapped: panel.pick(row.modelData)
                    }
                }

                // Rede guardada: abre a linha de opções.
                Item {
                    id: more
                    visible: row.modelData.known
                    anchors.right: parent.right
                    width: 34
                    height: 38

                    Text {
                        anchors.centerIn: parent
                        text: row.open ? "\u{f0143}" : "\u{f0140}"
                        color: moreHover.hovered ? Theme.text : Theme.textFaint
                        font.family: Theme.icons
                        font.pixelSize: 14
                    }

                    HoverHandler {
                        id: moreHover
                        cursorShape: Qt.PointingHandCursor
                    }

                    TapHandler {
                        onTapped: {
                            panel.forgetSsid = "";
                            panel.openSsid = row.open ? "" : row.modelData.name;
                        }
                    }
                }

                RowLayout {
                    visible: row.open
                    y: 38
                    width: parent.width
                    height: 30
                    spacing: 10

                    Text {
                        Layout.leftMargin: 12
                        text: "reconectar sozinho"
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: 1.2
                    }

                    MkToggle {
                        scale: 0.8
                        checked: panel.autoconnects(row.modelData.name)
                        onToggled: value => panel.setAutoconnect(row.modelData.name, value)
                    }

                    Item { Layout.fillWidth: true }

                    // Esquecer: o primeiro clique pede a confirmação, o segundo apaga.
                    Item {
                        Layout.fillHeight: true
                        implicitWidth: forgetLabel.implicitWidth + 24

                        Text {
                            id: forgetLabel
                            anchors.centerIn: parent
                            text: row.confirming ? "esquecer mesmo?" : "esquecer"
                            color: row.confirming ? Theme.copper : forgetHover.hovered ? Theme.text : Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                            font.capitalization: Font.AllUppercase
                            font.letterSpacing: 1.2
                        }

                        HoverHandler {
                            id: forgetHover
                            cursorShape: Qt.PointingHandCursor
                        }

                        TapHandler {
                            onTapped: panel.forget(row.modelData.name)
                        }
                    }
                }
            }
        }

        // A senha da rede escolhida.
        ColumnLayout {
            visible: panel.asking !== null && !panel.showQr
            Layout.fillWidth: true
            spacing: 6

            Text {
                text: "senha de " + (panel.asking ? panel.asking.name : "")
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 9
                font.capitalization: Font.AllUppercase
                font.letterSpacing: 1.6
            }

            RowLayout {
                Layout.fillWidth: true
                spacing: 10

                Item {
                    Layout.fillWidth: true
                    implicitHeight: 32

                    TextInput {
                        id: secret
                        anchors.fill: parent
                        anchors.bottomMargin: 5
                        verticalAlignment: TextInput.AlignVCenter
                        echoMode: reveal.selected ? TextInput.Normal : TextInput.Password
                        passwordCharacter: "•"
                        color: Theme.text
                        selectionColor: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 14
                        clip: true
                        cursorDelegate: Rectangle {
                            width: 2
                            color: Theme.copper
                            visible: secret.activeFocus
                        }
                        Keys.onReturnPressed: panel.join()
                        Keys.onEnterPressed: panel.join()
                        Keys.onEscapePressed: panel.dismiss()
                    }

                    Rectangle {
                        anchors.bottom: parent.bottom
                        width: parent.width
                        height: 1
                        color: panel.wifiError !== "" ? Theme.copper : secret.activeFocus ? Theme.line : Theme.hairline
                    }
                }

                PillButton {
                    id: reveal
                    text: selected ? "ocultar" : "mostrar"
                    onClicked: selected = !selected
                }
                PillButton {
                    primary: true
                    text: "entrar"
                    onClicked: panel.join()
                }
            }
        }

        Text {
            visible: panel.wifiError !== ""
            text: panel.wifiError
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 8

            PillButton {
                visible: panel.net.type === "wifi"
                text: panel.showQr ? "voltar às redes" : "compartilhar wi-fi"
                selected: panel.showQr
                onClicked: panel.toggleQr()
            }

            Item { Layout.fillWidth: true }

            PillButton {
                text: "nmtui"
                onClicked: {
                    Projects.openIn("", "nmtui");
                    Mare.close();
                }
            }
        }
    }
}
