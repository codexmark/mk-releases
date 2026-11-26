import QtQuick
import QtQuick.Shapes
import Quickshell
import Quickshell.Wayland
import Quickshell.Bluetooth
import Quickshell.Hyprland
import Quickshell.Networking
import Quickshell.Services.Pipewire
import Quickshell.Services.SystemTray
import Quickshell.Services.UPower
import "Island.js" as Island

// A ilha, no molde da do Ryoku (orpheus): uma peça só, pendurada no meio da
// borda de cima, com filetes côncavos onde encosta na tela. Esquerda: marca,
// workspaces e sistema; centro: relógio e clima; direita: bandeja e
// indicadores. Translúcida em repouso, sólida enquanto um painel está aberto.
// Os painéis descem soldados a ela, embaixo do item clicado (Tide.qml).
PanelWindow {
    id: bar

    anchors {
        top: true
        left: true
        right: true
    }
    // A janela tem uma folga embaixo da ilha para a onda do som; o espaço
    // reservado para as janelas continua sendo só a altura da ilha.
    // O que o Monitor.qml liga ao painel deste monitor: de que largura o
    // painel aberto precisa (a ilha alarga), para onde a ilha vai (o painel se
    // encaixa onde ela vai estar, não onde está a cada quadro: seguindo a
    // ilha em movimento, as molas dele recomeçavam sem sair do lugar).
    property real sheetW: 0
    readonly property real islandX: island.goalX
    readonly property real islandW: island.goalW

    readonly property int waveRoom: 34
    implicitHeight: barH + waveRoom
    exclusiveZone: barH
    color: "transparent"
    WlrLayershell.namespace: "mare-island"

    // Só a ilha recebe cliques; o resto da faixa deixa passar.
    mask: Region {
        item: island
    }

    readonly property int barH: 34
    readonly property int fillet: 12
    readonly property int corner: 12
    readonly property int pad: 12
    readonly property int gap: 4

    readonly property var sink: Pipewire.defaultAudioSink
    readonly property var battery: UPower.displayDevice
    readonly property bool online: Networking.devices.values.some(d => d.connected)
    readonly property int batteryPct: battery ? Math.round(battery.percentage > 1 ? battery.percentage : battery.percentage * 100) : 0
    readonly property bool panelOpen: Mare.mode !== "rest"

    PwObjectTracker {
        objects: [Pipewire.defaultAudioSink]
    }

    SystemClock {
        id: clock
        precision: SystemClock.Minutes
    }

    // OSD de volume: qualquer mudança (teclas, outro app) aparece sob a ilha.
    // Espera o PipeWire assentar para não piscar ao iniciar.
    property bool osdArmed: false

    Timer {
        interval: 2000
        running: true
        onTriggered: bar.osdArmed = true
    }

    Connections {
        target: bar.sink && bar.sink.audio ? bar.sink.audio : null

        function onVolumeChanged() {
            if (bar.osdArmed && Mare.mode !== "volume")
                Mare.osd("volume", bar.sink.audio.volume, bar.sink.audio.muted);
        }

        function onMutedChanged() {
            if (bar.osdArmed && Mare.mode !== "volume")
                Mare.osd("volume", bar.sink.audio.volume, bar.sink.audio.muted);
        }
    }

    Item {
        id: island

        // A ilha abraça o conteúdo, sem sobra de nenhum lado: cresce sozinha
        // quando aparece um item (bandeja, Bluetooth, música...) e encolhe
        // quando ele some. Para o relógio continuar no meio da tela, ela se
        // desloca para o lado em vez de completar o lado mais curto.
        readonly property real natural: bar.fillet * 2 + bar.pad * 4
            + left.implicitWidth + center.implicitWidth + right.implicitWidth
        // Um painel mais largo que a ilha a alarga (dos dois lados) para caber
        // inteiro dentro da borda de baixo (ombros + cantos de cada lado).
        readonly property real extra: Math.max(0,
            (bar.sheetW > 0 ? bar.sheetW + 2 * (bar.fillet + bar.corner) : 0) - natural)
        // Distância da borda esquerda da ilha até o centro do relógio.
        readonly property real clockOffset: bar.fillet + bar.pad + extra / 2
            + left.implicitWidth + bar.pad + clockBox.implicitWidth / 2

        // Para onde a ilha vai (a largura e o x abaixo chegam lá animando).
        readonly property real goalW: Math.min(bar.width, natural + extra)
        readonly property real goalX: Math.max(0, Math.min(bar.width - goalW, bar.width / 2 - clockOffset))

        anchors.top: parent.top
        width: goalW
        x: goalX
        height: bar.barH

        Behavior on width {
            NumberAnimation {
                duration: Motion.spatial
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.spatialCurve
            }
        }
        Behavior on x {
            NumberAnimation {
                duration: Motion.spatial
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.spatialCurve
            }
        }


        // Translúcida em repouso, sólida com um painel aberto (como no Ryoku).
        property real solidity: bar.panelOpen ? 1 : Settings.get("island.restOpacity")

        Behavior on solidity {
            NumberAnimation { duration: Motion.base }
        }
        onSolidityChanged: shape.requestPaint()

        Canvas {
            id: shape
            anchors.fill: parent
            onWidthChanged: requestPaint()

            onPaint: {
                const ctx = getContext("2d");
                ctx.reset();
                const f = bar.fillet, r = bar.corner, w = width, h = height - 1;
                const path = () => {
                    ctx.moveTo(0, 0);
                    ctx.arc(0, f, f, -Math.PI / 2, 0, false);
                    ctx.lineTo(f, h - r);
                    ctx.arc(f + r, h - r, r, Math.PI, Math.PI / 2, true);
                    ctx.lineTo(w - f - r, h);
                    ctx.arc(w - f - r, h - r, r, Math.PI / 2, 0, true);
                    ctx.lineTo(w - f, f);
                    ctx.arc(w, f, f, Math.PI, Math.PI * 1.5, false);
                };
                const c = Theme.surface;
                ctx.beginPath();
                path();
                ctx.closePath();
                ctx.fillStyle = Qt.rgba(c.r, c.g, c.b, island.solidity);
                ctx.fill();
                // A linha fina: os dois lados. O trecho reto de baixo é da
                // onda (logo abaixo), que o desenha reto quando há silêncio.
                ctx.strokeStyle = Theme.hairline;
                ctx.lineWidth = 1;
                ctx.beginPath();
                ctx.moveTo(0, 0);
                ctx.arc(0, f, f, -Math.PI / 2, 0, false);
                ctx.lineTo(f, h - r);
                ctx.arc(f + r, h - r, r, Math.PI, Math.PI / 2, true);
                ctx.stroke();
                ctx.beginPath();
                ctx.moveTo(w - f - r, h);
                ctx.arc(w - f - r, h - r, r, Math.PI / 2, 0, true);
                ctx.lineTo(w - f, f);
                ctx.arc(w, f, f, Math.PI, Math.PI * 1.5, false);
                ctx.stroke();
            }
        }

        // A ilha responde ao som: a borda de baixo ondula com o espectro do
        // que está tocando (Sound.qml), graves à esquerda e agudos à direita,
        // como se a música mexesse num líquido. Com um painel aberto, ou em
        // silêncio, é uma linha reta.
        Shape {
            id: wave
            x: bar.fillet + bar.corner
            y: island.height - 1
            width: island.width - 2 * (bar.fillet + bar.corner)
            height: bar.waveRoom
            preferredRendererType: Shape.CurveRenderer

            readonly property real amp: (bar.waveRoom - 3) * Sound.presence * (bar.panelOpen ? 0 : 1)
            // A curva: três pontos por faixa, suavizados, presos em zero nas pontas.
            readonly property var curve: {
                const pts = [Qt.point(0, 0)];
                const lv = Sound.levels, n = Sound.bands, steps = n * 3;
                if (wave.amp > 0.05 && lv.length === n && wave.width > 0) {
                    // Contraste: a faixa mais fraca do momento fica perto de
                    // zero e a mais forte usa a altura toda. Sem isso, com
                    // todas as faixas soando, a borda só descia por igual.
                    let lo = 1, hi = 0;
                    for (let i = 0; i < n; i++) {
                        lo = Math.min(lo, lv[i]);
                        hi = Math.max(hi, lv[i]);
                    }
                    const floor = lo * 0.85, span = Math.max(0.08, hi - floor);
                    const gain = 0.55 + 0.45 * Math.min(1, hi * 1.6);
                    for (let i = 1; i < steps; i++) {
                        const t = i / steps, p = t * (n - 1), a = Math.floor(p), b = Math.min(n - 1, a + 1);
                        const k = (1 - Math.cos((p - a) * Math.PI)) / 2;
                        const raw = lv[a] * (1 - k) + lv[b] * k;
                        const v = Math.pow(Math.max(0, (raw - floor) / span), 0.6) * gain;
                        const edge = Math.sin(t * Math.PI);
                        pts.push(Qt.point(t * wave.width, v * wave.amp * Math.pow(edge, 0.6)));
                    }
                }
                pts.push(Qt.point(wave.width, 0));
                return pts;
            }

            ShapePath {
                fillColor: Qt.rgba(Theme.surface.r, Theme.surface.g, Theme.surface.b, island.solidity)
                strokeColor: "transparent"
                PathPolyline { path: wave.curve }
            }
            ShapePath {
                fillColor: "transparent"
                // A linha esquenta para o cobre conforme o som enche.
                strokeColor: Qt.tint(Theme.hairline, Qt.rgba(Theme.copper.r, Theme.copper.g, Theme.copper.b,
                    Math.min(0.95, 0.45 + Sound.energy * 2.5) * Sound.presence * (bar.panelOpen ? 0 : 1)))
                strokeWidth: 1 + 0.5 * Sound.presence
                PathPolyline { path: wave.curve }
            }
        }

        // Grão também na ilha, bem leve.
        Image {
            x: bar.fillet
            width: parent.width - 2 * bar.fillet
            height: parent.height
            source: Qt.resolvedUrl("assets/grain.png")
            fillMode: Image.Tile
            opacity: 0.035
        }

        Item {
            anchors.fill: parent
            anchors.leftMargin: bar.fillet + bar.pad
            anchors.rightMargin: bar.fillet + bar.pad
            clip: true

            // Esquerda: marca, workspaces, sistema.
            Row {
                id: left
                x: island.extra / 2
                anchors.verticalCenter: parent.verticalCenter
                spacing: bar.gap

                Behavior on x {
                    NumberAnimation {
                        duration: Motion.spatial
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.spatialCurve
                    }
                }

                // Marca "Mk": o M em osso, o k em cobre e os eixos do logo (a
                // forma compacta de system/mk/art/make-logo.py, em vetor).
                // Abre o launcher (ou um toque no Super).
                Item {
                    id: brand
                    anchors.verticalCenter: parent.verticalCenter
                    implicitWidth: mark.implicitWidth + 14
                    implicitHeight: 24

                    Image {
                        id: mark
                        anchors.centerIn: parent
                        source: Qt.resolvedUrl("assets/mark.svg")
                        sourceSize.height: 20
                        smooth: true
                    }

                    HoverHandler { cursorShape: Qt.PointingHandCursor }
                    TapHandler {
                        onTapped: Mare.openAt("launcher", brand.mapToItem(null, brand.width / 2, 0).x)
                    }
                }

                // Workspaces por letra grega (α β γ...). Sempre pelo menos três;
                // passa disso quando um workspace mais alto existe. O atual fica
                // sob um círculo de osso que desliza de uma letra para outra.
                Item {
                    id: spaces

                    readonly property int slot: 24
                    readonly property var existing: Hyprland.workspaces.values.filter(w => w.id > 0)
                    readonly property int count: Math.max(3, ...existing.map(w => w.id))
                    readonly property int current: Hyprland.focusedWorkspace ? Hyprland.focusedWorkspace.id : 1

                    anchors.verticalCenter: parent.verticalCenter
                    implicitWidth: count * slot + (count - 1) * bar.gap
                    implicitHeight: slot

                    Rectangle {
                        visible: spaces.current >= 1 && spaces.current <= spaces.count
                        x: (spaces.current - 1) * (spaces.slot + bar.gap) + 1
                        y: 1
                        width: spaces.slot - 2
                        height: width
                        radius: width / 2
                        color: Theme.text

                        Behavior on x {
                            NumberAnimation {
                                duration: Motion.spatial
                                easing.type: Easing.BezierSpline
                                easing.bezierCurve: Motion.spatialCurve
                            }
                        }
                    }

                    Row {
                        spacing: bar.gap

                        Repeater {
                            model: spaces.count

                            Item {
                                id: space
                                required property int index
                                readonly property int wsId: index + 1
                                readonly property bool active: wsId === spaces.current
                                readonly property bool used: spaces.existing.some(w => w.id === wsId)

                                width: spaces.slot
                                height: spaces.slot

                                Rectangle {
                                    anchors.fill: parent
                                    anchors.margins: 1
                                    radius: width / 2
                                    color: Theme.wash
                                    visible: spaceHover.hovered && !space.active
                                }

                                // Centro visual da letra, não da linha: o α
                                // de serifa não tem ascendente e cairia baixo.
                                TextMetrics {
                                    id: glyph
                                    font: label.font
                                    text: label.text
                                }

                                Text {
                                    id: label
                                    anchors.centerIn: parent
                                    anchors.verticalCenterOffset: label.height / 2
                                        - (label.baselineOffset + glyph.tightBoundingRect.y + glyph.tightBoundingRect.height / 2)
                                    anchors.horizontalCenterOffset: label.width / 2
                                        - (glyph.tightBoundingRect.x + glyph.tightBoundingRect.width / 2)
                                    text: Theme.wsName(space.wsId)
                                    color: space.active ? Theme.ground : space.used ? Theme.textSoft : Theme.textFaint
                                    font.family: Theme.serif
                                    font.pixelSize: 15

                                    Behavior on color {
                                        ColorAnimation { duration: Motion.fast }
                                    }
                                }

                                HoverHandler {
                                    id: spaceHover
                                    cursorShape: Qt.PointingHandCursor
                                }

                                TapHandler {
                                    onTapped: Hypr.workspace(space.wsId)
                                }
                            }
                        }
                    }
                }

                Item { visible: Settings.get("island.showSystem"); width: bar.pad - bar.gap; height: 1 }

                // Sistema, com ícone e valor; fica cobre quando passa do limite.
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Stats.temp > 0 && Settings.get("island.showSystem")
                    icon: "󰔏"
                    text: Island.degrees(Stats.temp)
                    reserve: Island.reserve.degrees
                    iconColor: Stats.temp > 90 ? Theme.copper : Theme.textDim
                    panel: "system"
                }
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Settings.get("island.showSystem")
                    icon: "󰇚"
                    text: Island.rate(Stats.rxRate)
                    reserve: Island.reserve.rate
                    panel: "system"
                }
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Settings.get("island.showSystem")
                    icon: "󰍛"
                    text: Island.percent(Stats.cpu)
                    reserve: Island.reserve.percent
                    iconColor: Stats.cpu > 0.9 ? Theme.copper : Theme.textDim
                    panel: "system"
                }
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Settings.get("island.showSystem")
                    icon: "󰘚"
                    text: Island.percent(Stats.mem)
                    reserve: Island.reserve.percent
                    iconColor: Stats.mem > 0.9 ? Theme.copper : Theme.textDim
                    panel: "system"
                }
            }

            // Centro: relógio e data; o clima ao lado.
            Row {
                id: center
                x: left.x + left.implicitWidth + bar.pad
                anchors.verticalCenter: parent.verticalCenter
                spacing: bar.pad

                Item {
                    id: clockBox
                    anchors.verticalCenter: parent.verticalCenter
                    implicitWidth: Math.max(time.implicitWidth, date.implicitWidth) + 10
                    implicitHeight: 30

                    Text {
                        id: time
                        anchors.horizontalCenter: parent.horizontalCenter
                        y: 1
                        text: Qt.formatDateTime(clock.date, "hh:mm")
                        color: Theme.text
                        font.family: Theme.sans
                        font.pixelSize: 13
                        font.weight: Font.Medium
                        font.features: ({ "tnum": 1 })
                    }

                    Text {
                        id: date
                        anchors.horizontalCenter: parent.horizontalCenter
                        anchors.top: time.bottom
                        anchors.topMargin: -1
                        text: Qt.locale("pt_BR").toString(clock.date, "ddd d MMM").replace(/\./g, "")
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 9
                        font.letterSpacing: 0.6
                    }

                    HoverHandler { cursorShape: Qt.PointingHandCursor }
                    TapHandler {
                        onTapped: Mare.openAt("calendar", clockBox.mapToItem(null, clockBox.width / 2, 0).x)
                    }
                }

                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Weather.ready && Settings.get("island.showWeather")
                    icon: Weather.ready ? Weather.icon(Weather.data.code, Weather.data.day) : ""
                    text: Weather.ready ? Weather.data.temp + "°" : ""
                    textColor: Theme.text
                    panel: "calendar"
                }
            }

            // Direita: bandeja e indicadores.
            Row {
                id: right
                x: center.x + center.implicitWidth + bar.pad
                anchors.verticalCenter: parent.verticalCenter
                spacing: bar.gap

                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Media.active
                    icon: Media.player && Media.player.isPlaying ? "󰎈" : "󰏤"
                    iconColor: Media.player && Media.player.isPlaying ? Theme.copper : Theme.textDim
                    panel: "media"
                }

                Repeater {
                    model: SystemTray.items.values

                    TrayIcon {
                        required property var modelData
                        anchors.verticalCenter: parent.verticalCenter
                        item: modelData
                        barWindow: bar
                    }
                }

                // Gravando a tela: o tempo, em cobre. Um clique para.
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Capture.recording
                    icon: "󰑊"
                    text: Capture.clock()
                    iconColor: Theme.copper
                    textColor: Theme.copper
                    onClicked: Capture.toggleRecording()
                }

                // Modo apresentação ou jogo ligado (Modes.qml): o ícone do
                // modo, em cobre. Um clique desliga.
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Modes.on
                    icon: String.fromCodePoint(Modes.game ? 0xF0297 : 0xF0428)
                    iconColor: Theme.copper
                    onClicked: Modes.off()
                }

                // Atualizações: só aparece pelo que importa (Updates.badge).
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Updates.badge && Settings.get("updates.island")
                    icon: "󰚰"
                    text: String(Updates.count)
                    iconColor: Updates.mkAvailable ? Theme.copper : Theme.textDim
                    onClicked: {
                        Mare.settingsPage = "atualizacoes";
                        Mare.open("settings");
                    }
                }
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    icon: Notifs.dnd ? "󰂛" : "󰂚"
                    dot: Notifs.count > 0 && !Notifs.dnd
                    panel: "notifications"
                }
                // VPN: o escudo só aparece enquanto há um túnel aberto.
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Vpn.up
                    icon: "󰦝"
                    iconColor: Theme.copper
                    onClicked: {
                        Mare.settingsPage = "vpn";
                        Mare.open("settings");
                    }
                }
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    icon: !Networking.wifiEnabled ? "󰖪" : bar.online ? "󰖩" : "󰤯"
                    iconColor: bar.online ? Theme.textDim : Theme.copper
                    panel: "network"
                }
                IslandItem {
                    readonly property var adapter: Bluetooth.defaultAdapter
                    readonly property bool connected: adapter !== null && adapter.devices.values.some(d => d.connected)
                    anchors.verticalCenter: parent.verticalCenter
                    visible: adapter !== null
                    icon: adapter && !adapter.enabled ? "󰂲" : connected ? "󰂱" : "󰂯"
                    iconColor: connected ? Theme.text : Theme.textDim
                    panel: "bluetooth"
                }
                IslandItem {
                    readonly property bool ready: bar.sink !== null && bar.sink.audio !== null
                    readonly property bool muted: !ready || bar.sink.audio.muted
                    anchors.verticalCenter: parent.verticalCenter
                    icon: muted ? "󰖁" : "󰕾"
                    panel: "volume"
                    onScrolled: delta => Mare.changeVolume(delta > 0 ? 0.05 : -0.05)
                }
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Display.hasBacklight
                    icon: Display.nightLight ? "󰖔" : "󰃠"
                    iconColor: Display.nightLight ? Theme.copper : Theme.textDim
                    panel: "display"
                    onScrolled: delta => Display.set(Display.brightness + (delta > 0 ? 0.05 : -0.05))
                }
                IslandItem {
                    readonly property bool charging: bar.battery && bar.battery.state === UPowerDeviceState.Charging
                    anchors.verticalCenter: parent.verticalCenter
                    visible: bar.battery && bar.battery.isLaptopBattery
                    icon: charging ? "󰂄" : bar.batteryPct > 60 ? "󰁹" : bar.batteryPct > 25 ? "󰁾" : "󰁻"
                    iconColor: bar.batteryPct <= 15 ? Theme.copper : Theme.textDim
                    text: Island.percent(bar.batteryPct / 100)
                    reserve: Island.reserve.percent
                    textColor: Theme.text
                    panel: "power"
                }
                IslandItem {
                    anchors.verticalCenter: parent.verticalCenter
                    icon: "󰐥"
                    panel: "session"
                }
            }
        }
    }
}
