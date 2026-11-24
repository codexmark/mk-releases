import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import Quickshell.Bluetooth

// Bluetooth: liga/desliga o adaptador, mostra os aparelhos pareados (clique
// conecta ou desconecta, com a bateria quando o aparelho informa), procura
// aparelhos novos para parear e esquece os que não servem mais.
//
// O Quickshell fala com o BlueZ mas não registra um agente de pareamento:
// aparelhos que pareiam sozinhos (fones, mouses, caixas de som, controles)
// funcionam daqui; os que pedem código ou confirmação na tela falham, e para
// eles fica o atalho do bluetoothctl no rodapé.
Item {
    id: panel

    readonly property var adapter: Bluetooth.defaultAdapter
    readonly property bool blocked: adapter !== null && adapter.state === BluetoothAdapterState.Blocked
    readonly property bool on: adapter !== null && adapter.enabled
    readonly property bool scanning: on && adapter.discovering
    readonly property var devices: adapter ? adapter.devices.values.slice() : []
    readonly property var paired: devices.filter(d => d.paired)
        .sort((a, b) => (b.connected - a.connected) || a.name.localeCompare(b.name))
    // Aparelhos novos: só os que dizem o nome (sem isso a lista enche de
    // endereços de sensores e etiquetas que ninguém quer parear).
    readonly property var found: devices.filter(d => !d.paired && d.deviceName !== "")
        .sort((a, b) => a.name.localeCompare(b.name))
    // As listas não são cortadas: passando desta altura, elas rolam.
    readonly property int listMax: 320
    readonly property int connectedCount: paired.filter(d => d.connected).length

    // Só para de procurar ao fechar se foi este painel que começou.
    property bool startedScan: false
    // O pareamento em andamento e o aviso moram no BluetoothPair, que
    // continua depois de o painel fechar.
    readonly property string notice: BluetoothPair.notice
    // "esquecer" pede um segundo clique: o endereço que está esperando por ele.
    property string forgetAddress: ""
    readonly property string unblockFailed: "não deu para desbloquear: pode ser a chave física ou a tecla de modo avião."
    // Sem adaptador: o serviço do BlueZ está de pé?
    property bool serviceActive: true

    implicitWidth: 440
    implicitHeight: column.implicitHeight + 42

    // --- puro: início ---
    // Daqui até "puro: fim" nada toca no BlueZ nem no painel: são funções de
    // um aparelho para texto, testadas fora do shell com node
    // (tests/js/BluetoothPanel.test.mjs).
    function icon(d) {
        const i = d.icon || "";
        if (i.indexOf("headset") >= 0)
            return "󰋎";
        if (i.indexOf("headphone") >= 0)
            return "󰋋";
        if (i.indexOf("audio") >= 0 || i.indexOf("multimedia") >= 0)
            return "󰓃";
        if (i.indexOf("mouse") >= 0)
            return "󰍽";
        if (i.indexOf("keyboard") >= 0)
            return "󰌌";
        if (i.indexOf("gaming") >= 0 || i.indexOf("joystick") >= 0)
            return "󰊴";
        if (i.indexOf("tablet") >= 0)
            return "󰓶";
        if (i.indexOf("phone") >= 0)
            return "󰄜";
        if (i.indexOf("computer") >= 0)
            return "󰌢";
        if (i.indexOf("printer") >= 0)
            return "󰐪";
        if (i.indexOf("camera") >= 0)
            return "󰄀";
        if (i.indexOf("watch") >= 0)
            return "󰖉";
        if (i.indexOf("display") >= 0)
            return "󰍹";
        return "󰂯";
    }

    // A bateria vem de 0 a 1; se um dia vier em porcentagem, não estoura.
    function batteryText(d) {
        if (!d.connected || !d.batteryAvailable)
            return "";
        return Math.round(d.battery <= 1 ? d.battery * 100 : d.battery) + "%";
    }
    // --- puro: fim ---

    function status(d) {
        if (d.pairing)
            return "pareando…";
        if (d.state === BluetoothDeviceState.Connecting)
            return "conectando…";
        if (d.state === BluetoothDeviceState.Disconnecting)
            return "desconectando…";
        if (d.connected)
            return "conectado";
        return d.paired ? "pareado" : "parear";
    }

    function busy(d) {
        return BluetoothPair.busy(d);
    }

    function click(d) {
        BluetoothPair.notice = "";
        forgetAddress = "";
        if (BluetoothPair.watchedDevice === d)
            BluetoothPair.unwatch();
        if (d.pairing) {
            // Segundo clique num aparelho sendo pareado: desiste.
            BluetoothPair.cancelPair(d);
        } else if (!d.paired) {
            // Um pareamento por vez: trocar de aparelho no meio deixaria o
            // primeiro sem confiança, sem conexão e sem aviso de falha.
            const p = BluetoothPair.pairingDevice;
            if (p !== null && p !== d && (p.pairing || BluetoothPair.settling)) {
                BluetoothPair.notice = "espere o pareamento com " + p.name + " terminar.";
                return;
            }
            BluetoothPair.pair(d);
        } else if (d.connected) {
            BluetoothPair.unlink(d);
        } else if (!busy(d)) {
            BluetoothPair.link(d);
        }
    }

    function forget(d) {
        BluetoothPair.notice = "";
        if (forgetAddress !== d.address) {
            forgetAddress = d.address;
            forgetReset.restart();
            return;
        }
        forgetAddress = "";
        BluetoothPair.drop(d);
        d.forget();
    }

    function toggleScan() {
        if (!on)
            return;
        BluetoothPair.notice = "";
        if (adapter.discovering) {
            adapter.discovering = false;
            startedScan = false;
        } else {
            adapter.discovering = true;
            startedScan = true;
            scanLimit.restart();
        }
    }

    function stopScan() {
        if (startedScan && adapter && adapter.discovering)
            adapter.discovering = false;
        startedScan = false;
    }

    Component.onCompleted: if (adapter === null) serviceProc.running = true
    // Para de procurar ao fechar o painel. O aviso já foi visto; só fica
    // para a próxima abertura o de um pareamento que ainda está andando.
    Component.onDestruction: {
        stopScan();
        if (!BluetoothPair.active)
            BluetoothPair.notice = "";
    }

    onAdapterChanged: {
        startedScan = false;
        if (adapter === null)
            serviceProc.running = true;
    }
    onOnChanged: {
        if (!on) {
            startedScan = false;
            forgetAddress = "";
        }
    }

    // Procurar gasta bateria e rádio: para sozinho depois de um minuto.
    Timer {
        id: scanLimit
        interval: 60000
        onTriggered: panel.stopScan()
    }

    Timer {
        id: forgetReset
        interval: 4000
        onTriggered: panel.forgetAddress = ""
    }

    Process {
        id: serviceProc
        // O bluetooth.service só sobe se existir /sys/class/bluetooth
        // (ConditionPathIsDirectory): sem rádio ele fica parado e isso é
        // "sem adaptador", não "serviço parado".
        command: ["bash", "-c", "[ -d /sys/class/bluetooth ] || exit 0; systemctl is-active --quiet bluetooth.service"]
        onExited: (code, status) => panel.serviceActive = code === 0
    }

    // rfkill: /dev/rfkill é do usuário da sessão, não precisa de senha.
    Process {
        id: unblockProc
        command: ["rfkill", "unblock", "bluetooth"]
        onExited: (code, status) => {
            if (code !== 0)
                BluetoothPair.notice = panel.unblockFailed;
            else
                unblockCheck.restart();
        }
    }

    // O rfkill sai com 0 mesmo quando a chave física segura o rádio: dá um
    // instante para o adaptador reagir e confere se continua bloqueado.
    Timer {
        id: unblockCheck
        interval: 1200
        onTriggered: {
            if (panel.blocked)
                BluetoothPair.notice = panel.unblockFailed;
        }
    }

    component SectionLabel: RowLayout {
        property string text: ""
        property bool pulsing: false

        Layout.fillWidth: true
        Layout.topMargin: 12
        Layout.bottomMargin: 4
        spacing: 8

        Text {
            text: parent.text
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 9
            font.capitalization: Font.AllUppercase
            font.letterSpacing: 1.6
        }

        // Procurando: um ponto de cobre que respira.
        Rectangle {
            id: dot
            visible: parent.pulsing
            Layout.alignment: Qt.AlignVCenter
            implicitWidth: 5
            implicitHeight: 5
            color: Theme.copper

            SequentialAnimation on opacity {
                running: dot.visible
                loops: Animation.Infinite
                NumberAnimation { from: 1; to: 0.2; duration: 700; easing.type: Easing.InOutSine }
                NumberAnimation { from: 0.2; to: 1; duration: 700; easing.type: Easing.InOutSine }
            }
        }

        Item { Layout.fillWidth: true }
    }

    component Hint: Text {
        Layout.fillWidth: true
        Layout.topMargin: 10
        wrapMode: Text.WordWrap
        color: Theme.textDim
        font.family: Theme.serif
        font.pixelSize: 15
        font.italic: true
        font.weight: Font.Light
    }

    component Detail: Text {
        Layout.fillWidth: true
        Layout.topMargin: 6
        wrapMode: Text.WordWrap
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 11
    }

    // Uma linha de aparelho. A área de clique e o "esquecer" ficam lado a
    // lado (não um sobre o outro), senão o clique chegaria nos dois.
    component DeviceRow: Item {
        id: row

        required property var modelData
        readonly property var dev: modelData
        readonly property bool confirming: panel.forgetAddress !== "" && dev.address === panel.forgetAddress
        readonly property string batt: panel.batteryText(dev)

        Layout.fillWidth: true
        implicitHeight: 44

        Rectangle {
            anchors.fill: parent
            color: row.dev.connected || hover.hovered || forgetHover.hovered ? Theme.wash : "transparent"

            Behavior on color {
                ColorAnimation { duration: Motion.fast }
            }
        }
        Rectangle {
            visible: row.dev.connected
            width: 2
            height: parent.height
            color: Theme.copper
        }
        Rectangle {
            anchors.bottom: parent.bottom
            width: parent.width
            height: 1
            color: Theme.hairline
        }

        RowLayout {
            anchors.fill: parent
            anchors.leftMargin: 12
            anchors.rightMargin: 4
            spacing: 0

            Item {
                Layout.fillWidth: true
                Layout.fillHeight: true

                RowLayout {
                    anchors.fill: parent
                    spacing: 12

                    Text {
                        text: panel.icon(row.dev)
                        color: row.dev.connected ? Theme.copper : Theme.textSoft
                        font.family: Theme.icons
                        font.pixelSize: 16
                    }

                    Text {
                        Layout.fillWidth: true
                        text: row.dev.name
                        color: row.dev.connected ? Theme.text : Theme.textSoft
                        font.family: Theme.sans
                        font.pixelSize: 13
                        elide: Text.ElideRight
                    }

                    Text {
                        visible: row.batt !== ""
                        text: row.batt
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }

                    Text {
                        text: panel.status(row.dev)
                        color: panel.busy(row.dev) || (!row.dev.paired && hover.hovered) ? Theme.copper : Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: 1.2
                    }
                }

                HoverHandler {
                    id: hover
                    cursorShape: Qt.PointingHandCursor
                }
                TapHandler {
                    onTapped: panel.click(row.dev)
                }
            }

            // Esquecer: o primeiro clique pede a confirmação, o segundo apaga.
            Item {
                visible: row.dev.paired
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
                    onTapped: panel.forget(row.dev)
                }
            }
        }
    }

    ColumnLayout {
        id: column
        x: 22
        y: 20
        width: parent.width - 44
        spacing: 0

        RowLayout {
            Layout.fillWidth: true
            spacing: 14

            PanelHeader {
                Layout.fillWidth: true
                index: "01"
                title: "Bluetooth"
                meta: {
                    if (panel.adapter === null)
                        return "sem adaptador";
                    switch (panel.adapter.state) {
                    case BluetoothAdapterState.Blocked: return "bloqueado";
                    case BluetoothAdapterState.Enabling: return "ligando…";
                    case BluetoothAdapterState.Disabling: return "desligando…";
                    }
                    if (!panel.on)
                        return "desligado";
                    if (panel.scanning)
                        return "procurando…";
                    return panel.connectedCount === 0 ? "ligado"
                        : panel.connectedCount === 1 ? "1 conectado" : panel.connectedCount + " conectados";
                }
                metaColor: panel.on && (panel.scanning || panel.connectedCount > 0) ? Theme.copper : Theme.textDim
            }

            MkToggle {
                visible: panel.adapter !== null && !panel.blocked
                checked: panel.on
                onToggled: value => {
                    BluetoothPair.notice = "";
                    // Desligar cancela o que estava em andamento, sem aviso de falha.
                    if (!value)
                        BluetoothPair.reset();
                    panel.adapter.enabled = value;
                }
            }
        }

        // Sem adaptador (ou sem o serviço do BlueZ).
        Hint {
            visible: panel.adapter === null
            text: panel.serviceActive ? "Nenhum adaptador bluetooth por aqui." : "O serviço do bluetooth está parado."
        }
        Detail {
            visible: panel.adapter === null
            text: panel.serviceActive
                ? "se o computador tem bluetooth, confira se ele não está desligado na bios; um adaptador usb aparece aqui assim que for plugado."
                : "ligue com: sudo systemctl enable --now bluetooth.service"
        }

        // Bloqueado pelo rfkill (modo avião ou chave física).
        Hint {
            visible: panel.blocked
            text: "O bluetooth está bloqueado, como no modo avião."
        }
        RowLayout {
            visible: panel.blocked
            Layout.fillWidth: true
            Layout.topMargin: 12
            spacing: 8

            PillButton {
                primary: true
                text: unblockProc.running || unblockCheck.running ? "desbloqueando…" : "desbloquear"
                onClicked: {
                    if (unblockProc.running || unblockCheck.running)
                        return;
                    BluetoothPair.notice = "";
                    unblockProc.running = true;
                }
            }
            Item { Layout.fillWidth: true }
        }

        // Desligado.
        Hint {
            visible: panel.adapter !== null && !panel.blocked && !panel.on
            text: "Desligado. Ligue para ver os seus aparelhos."
        }

        // As duas listas, sem corte: num lugar cheio de aparelhos elas rolam.
        // ScriptModel compara os valores e mantém as linhas que não mudaram
        // (com um array novo a cada mudança, a linha sob o cursor seria
        // recriada entre o apertar e o soltar do clique).
        Flickable {
            id: listView
            visible: panel.on
            Layout.fillWidth: true
            Layout.preferredHeight: Math.min(list.implicitHeight, panel.listMax)
            contentHeight: list.implicitHeight
            clip: true
            interactive: contentHeight > height
            boundsBehavior: Flickable.StopAtBounds

            ColumnLayout {
                id: list
                width: listView.width
                spacing: 0

                // Aparelhos pareados.
                SectionLabel {
                    text: "seus aparelhos"
                }
                Repeater {
                    model: ScriptModel {
                        values: panel.on ? panel.paired : []
                    }
                    DeviceRow {}
                }
                Hint {
                    visible: panel.paired.length === 0
                    Layout.topMargin: 4
                    text: "Nenhum aparelho pareado ainda."
                }
                Detail {
                    visible: panel.paired.length === 0 && !panel.scanning && panel.found.length === 0
                    text: "deixe o aparelho em modo de pareamento e toque em procurar."
                }

                // Aparelhos novos, enquanto procura (e os que ficaram da última busca).
                SectionLabel {
                    visible: panel.scanning || panel.found.length > 0
                    text: "por perto"
                    pulsing: panel.scanning
                }
                Repeater {
                    model: ScriptModel {
                        values: panel.on ? panel.found : []
                    }
                    DeviceRow {}
                }
                Detail {
                    visible: panel.scanning && panel.found.length === 0
                    Layout.topMargin: 2
                    text: "procurando… o aparelho precisa estar em modo de pareamento."
                }
            }

            // Tem mais para baixo ou para cima: um fio na borda marca onde se está.
            Rectangle {
                visible: listView.interactive
                parent: listView
                x: listView.width - width
                y: listView.height * listView.contentY / listView.contentHeight
                width: 2
                height: Math.max(16, listView.height * listView.height / listView.contentHeight)
                color: Theme.line
            }
        }

        // O que deu errado por último.
        Text {
            visible: panel.notice !== ""
            Layout.fillWidth: true
            Layout.topMargin: 10
            wrapMode: Text.WordWrap
            text: panel.notice
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
        }

        RowLayout {
            visible: panel.adapter !== null && !panel.blocked
            Layout.fillWidth: true
            Layout.topMargin: 14
            spacing: 8

            PillButton {
                visible: panel.on
                text: panel.scanning ? "parar" : "procurar"
                selected: panel.scanning
                onClicked: panel.toggleScan()
            }

            Text {
                visible: panel.on
                Layout.fillWidth: true
                Layout.leftMargin: 6
                text: "pede código? →"
                horizontalAlignment: Text.AlignRight
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 10
                elide: Text.ElideRight
            }
            Item {
                visible: !panel.on
                Layout.fillWidth: true
            }

            // Pareamento com código/confirmação e casos difíceis: o
            // bluetoothctl tem o agente que o painel não tem.
            PillButton {
                text: "bluetoothctl"
                onClicked: {
                    Projects.openIn("", "bluetoothctl");
                    Mare.close();
                }
            }
        }
    }
}
