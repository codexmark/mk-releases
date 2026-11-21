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
    readonly property var networks: wifi
        ? wifi.networks.values.slice()
            .sort((a, b) => (b.connected - a.connected) || (b.signalStrength - a.signalStrength))
            .slice(0, 6)
        : []

    // Entrar numa rede nova: a rede que está pedindo senha, e o que deu errado.
    property var asking: null
    property string wifiError: ""

    function pick(network) {
        wifiError = "";
        if (network.connected || network.stateChanging)
            return;
        if (network.known || network.security === WifiSecurityType.Open) {
            asking = null;
            network.connect();
        } else {
            asking = network;
            secret.text = "";
            secret.forceActiveFocus();
        }
    }

    function join() {
        if (!asking || secret.text.length < 8) {
            wifiError = "a senha tem pelo menos 8 caracteres";
            return;
        }
        wifiError = "";
        // A senha vai para o NetworkManager pelo D-Bus, não por linha de comando.
        asking.connectWithPsk(secret.text);
        secret.text = "";
    }

    Connections {
        target: panel.asking
        ignoreUnknownSignals: true
        function onConnectedChanged() {
            if (panel.asking && panel.asking.connected) {
                panel.asking = null;
                netProc.running = true;
            }
        }
        function onConnectionFailed(reason) {
            panel.wifiError = "não conectou: confira a senha";
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
    readonly property int loss: internetPings.length === 0 ? 0
        : Math.round(100 * internetPings.filter(p => p < 0).length / internetPings.length)

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

    function last(list) {
        return list.length > 0 ? list[list.length - 1] : -1;
    }

    function push(list, value) {
        const next = list.concat([value]);
        return next.length > 24 ? next.slice(next.length - 24) : next;
    }

    function ms(value) {
        return value < 0 ? "—" : value < 10 ? value.toFixed(1).replace(".", ",") + " ms" : Math.round(value) + " ms";
    }

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
    }
    Component.onDestruction: {
        if (wifi)
            wifi.scannerEnabled = false;
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
                const m = text.match(/time=([0-9.]+)/);
                panel.routerPings = panel.push(panel.routerPings, m ? Number(m[1]) : -1);
            }
        }
    }

    Process {
        id: internetPing
        command: ["env", "LC_ALL=C", "ping", "-n", "-c", "1", "-W", "1", "1.1.1.1"]
        stdout: StdioCollector {
            onStreamFinished: {
                const m = text.match(/time=([0-9.]+)/);
                panel.internetPings = panel.push(panel.internetPings, m ? Number(m[1]) : -1);
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
                Layout.fillWidth: true
                implicitHeight: 38

                Rectangle {
                    anchors.fill: parent
                    color: row.modelData.connected ? Theme.wash : hover.hovered ? Theme.wash : "transparent"
                }

                Rectangle {
                    visible: row.modelData.connected
                    width: 2
                    height: parent.height
                    color: Theme.copper
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 12
                    anchors.rightMargin: 10
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
                        Keys.onEscapePressed: panel.asking = null
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
