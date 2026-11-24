import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Bluetooth
import Quickshell.Networking
import Quickshell.Services.Pipewire
import Quickshell.Services.UPower
import "SessionActions.js" as SessionActions

// Controles (Super + Esc): o que se mexe várias vezes por dia, de um relance e
// num lugar só. As leituras da máquina, Wi-Fi e Bluetooth, o perfil de
// energia, volume e brilho, a sessão e os alternadores (luz noturna, cafeína,
// não perturbe, microfone). Nada aqui é novo por baixo: cada bloco usa o mesmo
// singleton do painel dele (Stats, Display, Notifs…), e quem quer o detalhe
// abre o painel inteiro pelo título do bloco. Os quatro blocos do meio ligam
// e desligam em Ajustes › Ilha; um bloco some sozinho quando a máquina não
// tem aquilo (brilho, Bluetooth, perfis de energia).
Item {
    id: panel

    readonly property var sink: Pipewire.defaultAudioSink
    readonly property var mic: Pipewire.defaultAudioSource
    readonly property bool sinkReady: sink !== null && sink.audio !== null
    readonly property bool micReady: mic !== null && mic.audio !== null
    readonly property var wifi: Networking.devices.values.find(d => d.type === DeviceType.Wifi) ?? null
    readonly property var adapter: Bluetooth.defaultAdapter
    readonly property var battery: UPower.displayDevice
    readonly property bool hasBattery: battery !== null && battery.isLaptopBattery
    readonly property var actions: SessionActions.actions(Settings.sleepMode)
    // O botão de sessão à espera do segundo clique.
    property string armed: ""

    PwObjectTracker {
        objects: [panel.sink, panel.mic].filter(o => o !== null)
    }

    implicitWidth: 560
    implicitHeight: column.implicitHeight + 40

    function go(mode) {
        Mare.mode = mode;
    }

    function run(action) {
        if (action.confirm && armed !== action.id) {
            armed = action.id;
            disarm.restart();
            return;
        }
        Mare.close();
        Quickshell.execDetached(action.cmd);
    }

    Timer {
        id: disarm
        interval: 3000
        onTriggered: panel.armed = ""
    }

    // Um rótulo de bloco: clicável quando leva ao painel inteiro.
    component Label: Text {
        property string target: ""
        color: hover.hovered && target !== "" ? Theme.copper : Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.capitalization: Font.AllUppercase
        font.letterSpacing: Theme.labelSpacing

        HoverHandler {
            id: hover
            enabled: parent.target !== ""
            cursorShape: Qt.PointingHandCursor
        }
        TapHandler {
            enabled: parent.target !== ""
            onTapped: panel.go(parent.target)
        }
    }

    // Uma leitura: rótulo, valor e a barra fina de quanto está cheio.
    component Reading: Item {
        id: reading
        property string label: ""
        property string value: ""
        property real ratio: 0
        property bool hot: false
        Layout.fillWidth: true
        implicitHeight: 30

        Label { text: reading.label }
        Text {
            anchors.right: parent.right
            text: reading.value
            color: Theme.textSoft
            font.family: Theme.mono
            font.pixelSize: 11
        }
        Rectangle {
            y: 22
            width: parent.width
            height: 2
            color: Theme.wash
            Rectangle {
                width: parent.width * Math.max(0, Math.min(1, reading.ratio))
                height: parent.height
                color: reading.hot ? Theme.copper : Theme.textDim
            }
        }
    }

    // Wi-Fi e Bluetooth: um cartão com o estado e a chave.
    component Radio: Rectangle {
        id: radio
        property string title: ""
        property string detail: ""
        property string glyph: ""
        property bool on: false
        property string target: ""
        signal toggled()
        Layout.fillWidth: true
        implicitHeight: 54
        color: on ? Theme.wash : Qt.rgba(0, 0, 0, 0)
        border.width: 1
        border.color: on ? Theme.line : Theme.hairline

        Text {
            x: 14
            anchors.verticalCenter: parent.verticalCenter
            text: radio.glyph
            color: radio.on ? Theme.copper : Theme.textFaint
            font.family: Theme.icons
            font.pixelSize: 17
        }
        Column {
            x: 44
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width - 44 - 60
            spacing: 2
            Text {
                text: radio.title
                color: Theme.text
                font.family: Theme.sans
                font.pixelSize: 13
            }
            Text {
                width: parent.width
                text: radio.detail
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 10
                elide: Text.ElideRight
            }
        }
        // O cartão abre o painel inteiro; a chave só liga e desliga.
        TapHandler { onTapped: panel.go(radio.target) }
        HoverHandler { cursorShape: Qt.PointingHandCursor }
        MkToggle {
            anchors.right: parent.right
            anchors.rightMargin: 14
            anchors.verticalCenter: parent.verticalCenter
            checked: radio.on
            onToggled: radio.toggled()
        }
    }

    // Um nível para arrastar (volume, brilho), com o ícone que silencia.
    component Level: RowLayout {
        id: level
        property string glyph: ""
        property string label: ""
        property real value: 0
        property bool dimmed: false
        property string target: ""
        signal moved(real value)
        signal iconTapped()
        Layout.fillWidth: true
        spacing: 12

        Text {
            Layout.preferredWidth: 22
            horizontalAlignment: Text.AlignHCenter
            text: level.glyph
            color: level.dimmed ? Theme.textFaint : Theme.textSoft
            font.family: Theme.icons
            font.pixelSize: 16
            TapHandler { onTapped: level.iconTapped() }
            HoverHandler { cursorShape: Qt.PointingHandCursor }
        }
        Label {
            Layout.preferredWidth: 58
            text: level.label
            target: level.target
        }
        MkSlider {
            Layout.fillWidth: true
            from: 0
            to: 1
            step: 0.01
            value: level.value
            label: Math.round(level.value * 100) + "%"
            opacity: level.dimmed ? 0.45 : 1
            onMoved: v => level.moved(v)
        }
    }

    // Um botão quadrado de ícone: ação de sessão ou alternador.
    component IconButton: Rectangle {
        id: button
        property string glyph: ""
        property string hint: ""
        property bool active: false
        property bool armed: false
        signal tapped()
        implicitWidth: 46
        implicitHeight: 40
        color: armed ? Theme.copper : active ? Theme.wash : hover.hovered ? Theme.wash : Qt.rgba(0, 0, 0, 0)
        border.width: 1
        border.color: active || armed ? Theme.copper : Theme.line

        Text {
            anchors.centerIn: parent
            text: button.glyph
            color: button.armed ? Theme.ground : button.active ? Theme.copper : Theme.textSoft
            font.family: Theme.icons
            font.pixelSize: 16
        }
        HoverHandler {
            id: hover
            cursorShape: Qt.PointingHandCursor
        }
        TapHandler { onTapped: button.tapped() }

        // O que o botão faz, por extenso, embaixo dele enquanto o ponteiro está em cima.
        Text {
            visible: hover.hovered || button.armed
            anchors.horizontalCenter: parent.horizontalCenter
            y: parent.height + 5
            text: button.armed ? "de novo confirma" : button.hint
            color: button.armed ? Theme.copper : Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 9
        }
    }

    ColumnLayout {
        id: column
        x: 22
        y: 20
        width: parent.width - 44
        spacing: 14

        // Quem, há quanto tempo, e a hora.
        RowLayout {
            Layout.fillWidth: true
            Text {
                text: "Controles"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
                Layout.fillWidth: true
            }
            Label {
                text: hasBattery ? Math.round(panel.battery.percentage * 100) + "% · " + (panel.battery.state === UPowerDeviceState.Discharging ? "na bateria" : "na tomada") : ""
                visible: panel.hasBattery
                target: "power"
            }
            PillButton {
                text: "MkPulso"
                onClicked: {
                    Hypr.exec("mkpulso");
                    Mare.close();
                }
            }
        }

        // ── leituras ──────────────────────────────────────────────────
        RowLayout {
            visible: Settings.get("controls.showStats")
            Layout.fillWidth: true
            spacing: 18

            ColumnLayout {
                Layout.preferredWidth: 150
                spacing: 2
                Label { text: "cpu"; target: "system" }
                Text {
                    text: Math.round(Stats.cpu * 100) + "%"
                    color: Stats.cpu > 0.9 ? Theme.copper : Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 30
                    font.weight: Font.Light
                }
                Text {
                    text: Stats.temp > 0 ? Math.round(Stats.temp) + "°" : ""
                    color: Stats.temp > 90 ? Theme.copper : Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }
            Sparkline {
                Layout.preferredWidth: 130
                Layout.preferredHeight: 62
                values: Stats.cpuHistory
            }
            ColumnLayout {
                Layout.fillWidth: true
                spacing: 4
                Reading {
                    label: "memória"
                    value: Stats.memUsedGb.toFixed(1).replace(".", ",") + " / " + Stats.memTotalGb.toFixed(1).replace(".", ",") + " GB"
                    ratio: Stats.mem
                    hot: Stats.mem > 0.9
                }
                Reading {
                    label: "disco"
                    value: Math.round(Stats.diskUsedGb) + " / " + Math.round(Stats.diskTotalGb) + " GB"
                    ratio: Stats.disk
                    hot: Stats.disk > 0.95
                }
                Reading {
                    label: "rede"
                    value: "↓ " + Stats.rate(Stats.rxRate) + "   ↑ " + Stats.rate(Stats.txRate)
                    ratio: 0
                }
            }
        }

        // ── Wi-Fi e Bluetooth ─────────────────────────────────────────
        RowLayout {
            visible: Settings.get("controls.showRadios")
            Layout.fillWidth: true
            spacing: 10

            Radio {
                title: "Wi-Fi"
                glyph: String.fromCodePoint(0xF05A9)
                on: Networking.wifiEnabled
                target: "network"
                detail: {
                    if (!Networking.wifiEnabled)
                        return "desligado";
                    const net = panel.wifi ? panel.wifi.networks.values.find(n => n.connected) : null;
                    return net ? net.name + " · " + Math.round(net.signalStrength * 100) + "%" : "sem rede";
                }
                onToggled: Networking.wifiEnabled = !Networking.wifiEnabled
            }
            Radio {
                visible: panel.adapter !== null
                title: "Bluetooth"
                glyph: String.fromCodePoint(0xF00AF)
                on: panel.adapter !== null && panel.adapter.enabled
                target: "bluetooth"
                detail: {
                    if (panel.adapter === null || !panel.adapter.enabled)
                        return "desligado";
                    const linked = panel.adapter.devices.values.filter(d => d.connected);
                    return linked.length === 0 ? "nada conectado" : linked.map(d => d.name).join(", ");
                }
                onToggled: panel.adapter.enabled = !panel.adapter.enabled
            }
        }

        // ── perfil de energia ─────────────────────────────────────────
        RowLayout {
            // Sem o power-profiles-daemon não há o que escolher.
            visible: Settings.get("controls.showPower") && PowerProfiles.profile !== undefined
            Layout.fillWidth: true
            spacing: 8

            Label {
                Layout.preferredWidth: 92
                text: "energia"
                target: "power"
            }
            Repeater {
                model: [
                    { label: "economia", profile: PowerProfile.PowerSaver },
                    { label: "balanceado", profile: PowerProfile.Balanced },
                    { label: "desempenho", profile: PowerProfile.Performance }
                ].filter(p => p.profile !== PowerProfile.Performance || PowerProfiles.hasPerformanceProfile)

                PillButton {
                    required property var modelData
                    text: modelData.label
                    selected: PowerProfiles.profile === modelData.profile
                    onClicked: PowerProfiles.profile = modelData.profile
                }
            }
            Item { Layout.fillWidth: true }
        }

        // ── volume e brilho ───────────────────────────────────────────
        ColumnLayout {
            visible: Settings.get("controls.showLevels")
            Layout.fillWidth: true
            spacing: 10

            Level {
                glyph: String.fromCodePoint(panel.sinkReady && panel.sink.audio.muted ? 0xF075F : 0xF057E)
                label: "volume"
                target: "volume"
                value: panel.sinkReady ? panel.sink.audio.volume : 0
                dimmed: !panel.sinkReady || panel.sink.audio.muted
                onMoved: v => {
                    if (panel.sinkReady)
                        panel.sink.audio.volume = v;
                }
                onIconTapped: {
                    if (panel.sinkReady)
                        panel.sink.audio.muted = !panel.sink.audio.muted;
                }
            }
            Level {
                visible: Display.hasBacklight
                glyph: String.fromCodePoint(0xF00E0)
                label: "brilho"
                target: "display"
                value: Display.brightness
                onMoved: v => Display.set(v)
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 1
            color: Theme.hairline
        }

        // ── sessão e alternadores ─────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            // Espaço para a legenda que aparece embaixo do botão.
            Layout.bottomMargin: 20
            spacing: 8

            Repeater {
                model: panel.actions
                IconButton {
                    required property var modelData
                    glyph: modelData.icon
                    hint: modelData.label
                    armed: panel.armed === modelData.id
                    onTapped: panel.run(modelData)
                }
            }

            Item { Layout.fillWidth: true }

            IconButton {
                glyph: String.fromCodePoint(0xF0594)
                hint: "luz noturna"
                active: Display.nightLight
                onTapped: Display.toggleNight()
            }
            IconButton {
                glyph: String.fromCodePoint(0xF0176)
                hint: Caffeine.on ? "cafeína ligada" : "cafeína"
                active: Caffeine.on
                onTapped: Caffeine.toggle()
            }
            IconButton {
                glyph: String.fromCodePoint(Notifs.dnd ? 0xF009B : 0xF009A)
                hint: "não perturbe"
                active: Notifs.dnd
                onTapped: Notifs.dnd = !Notifs.dnd
            }
            IconButton {
                visible: panel.micReady
                glyph: String.fromCodePoint(panel.micReady && panel.mic.audio.muted ? 0xF036D : 0xF036C)
                hint: panel.micReady && panel.mic.audio.muted ? "microfone mudo" : "microfone"
                active: panel.micReady && panel.mic.audio.muted
                onTapped: panel.mic.audio.muted = !panel.mic.audio.muted
            }
            IconButton {
                glyph: String.fromCodePoint(0xF0493)
                hint: "ajustes"
                onTapped: Mare.mode = "settings"
            }
        }
    }
}
