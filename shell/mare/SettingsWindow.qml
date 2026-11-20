import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import Quickshell.Wayland
import Quickshell.Services.Pipewire
import "SettingsSchema.js" as Schema

// Ajustes (Super + ,): uma folha inteira no meio da tela, não um painel da
// ilha. À esquerda, as páginas agrupadas por assunto (como o Hub do Ryoku),
// com o número da folha no lugar do selo; em cima delas, uma busca que acha
// qualquer opção. À direita, a página: linhas desenhadas a partir do
// SettingsSchema.js, que mudam o Mk na hora. No canto, o carimbo da folha.
PanelWindow {
    id: win

    property bool active: true
    readonly property bool open: active && Mare.mode === "settings"

    readonly property string page: Mare.settingsPage
    property string query: ""
    property string flashKey: ""
    readonly property var results: Schema.search(query)
    readonly property var pageInfo: Schema.pageName(page)
    readonly property var allPages: Schema.groups.reduce((a, g) => a.concat(g.items), [])

    // Detalhes da máquina para a página Sobre (scripts/mare-about).
    property var about: ({})

    anchors {
        top: true
        bottom: true
        left: true
        right: true
    }
    exclusionMode: ExclusionMode.Ignore
    color: "transparent"
    visible: open || sheet.opacity > 0
    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "mare-settings"
    WlrLayershell.keyboardFocus: open ? WlrKeyboardFocus.OnDemand : WlrKeyboardFocus.None

    function go(key, rowKey) {
        Mare.settingsPage = key;
        flashKey = rowKey || "";
        query = "";
        search.text = "";
        scroller.contentY = 0;
        if (flashKey !== "")
            flashTimer.restart();
    }

    function format(row, v) {
        switch (row.unit) {
        case "%": return Math.round(v * 100) + "%";
        case "px": return v + " px";
        case "ms": return v + " ms";
        case "/s": return v + "/s";
        case "s": return v + " s";
        case "K": return v + " K";
        case "h": return v + "h";
        case "min": return v === 0 ? "nunca" : v + " min";
        default: return (v > 0 ? "+" : "") + Number(v).toFixed(1).replace(".", ",");
        }
    }

    onOpenChanged: {
        if (open) {
            search.forceActiveFocus();
            if (page === "sobre")
                aboutProc.running = true;
            if (page === "atualizacoes" && (!Updates.info || Date.now() / 1000 - Updates.info.checked > 600))
                Updates.check();
        }
    }
    onPageChanged: {
        if (page === "sobre")
            aboutProc.running = true;
        if (page === "atualizacoes" && (!Updates.info || Date.now() / 1000 - Updates.info.checked > 600))
            Updates.check();
    }

    Timer {
        id: flashTimer
        interval: 1400
        onTriggered: win.flashKey = ""
    }

    Process {
        id: aboutProc
        command: ["bash", Quickshell.shellPath("scripts/mare-about")]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    win.about = JSON.parse(text);
                } catch (e) {}
            }
        }
    }

    // Fundo escurecido: clicar fora da folha fecha. Um TapHandler não para o
    // clique no item de cima, então este recebe também os cliques dados dentro
    // da folha: só fecha se o ponto estiver fora dela.
    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: win.open ? 0.55 : 0

        Behavior on opacity {
            NumberAnimation { duration: Motion.base }
        }

        TapHandler {
            onTapped: (point, button) => {
                const p = point.position;
                const inside = p.x >= sheet.x && p.x <= sheet.x + sheet.width
                    && p.y >= sheet.y && p.y <= sheet.y + sheet.height;
                if (!inside)
                    Mare.close();
            }
        }
    }

    // ── controles ──────────────────────────────────────────────────────

    component Toggle: Rectangle {
        id: toggle
        property bool checked: false
        signal toggled(bool value)

        implicitWidth: 36
        implicitHeight: 20
        radius: 10
        color: checked ? Theme.copper : Theme.line

        Behavior on color {
            ColorAnimation { duration: Motion.fast }
        }

        Rectangle {
            x: toggle.checked ? parent.width - width - 2 : 2
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

        HoverHandler { cursorShape: Qt.PointingHandCursor }
        TapHandler { onTapped: toggle.toggled(!toggle.checked) }
    }

    component Slider: Item {
        id: slider
        property real from: 0
        property real to: 1
        property real step: 0.1
        property real value: 0
        property string label: ""
        readonly property real ratio: (value - from) / (to - from)
        signal moved(real value)

        implicitWidth: 260
        implicitHeight: 24

        function pick(x) {
            const r = Math.max(0, Math.min(1, x / track.width));
            const raw = from + r * (to - from);
            const snapped = Math.round(raw / step) * step;
            moved(Math.round(snapped * 1000) / 1000);
        }

        Item {
            id: track
            anchors.left: parent.left
            anchors.right: valueText.left
            anchors.rightMargin: 14
            anchors.verticalCenter: parent.verticalCenter
            height: 14

            Rectangle {
                anchors.verticalCenter: parent.verticalCenter
                width: parent.width
                height: 1
                color: Theme.line
            }

            Rectangle {
                anchors.verticalCenter: parent.verticalCenter
                width: parent.width * slider.ratio
                height: 2
                color: Theme.copper
            }

            Rectangle {
                x: parent.width * slider.ratio - width / 2
                anchors.verticalCenter: parent.verticalCenter
                width: 12
                height: 12
                radius: 6
                color: Theme.text
                border.width: 2
                border.color: Theme.surface
            }

            MouseArea {
                anchors.fill: parent
                anchors.margins: -6
                cursorShape: Qt.PointingHandCursor
                onPressed: mouse => slider.pick(mouse.x - 6)
                onPositionChanged: mouse => {
                    if (pressed)
                        slider.pick(mouse.x - 6);
                }
            }
        }

        Text {
            id: valueText
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            width: 56
            horizontalAlignment: Text.AlignRight
            text: slider.label
            color: Theme.text
            font.family: Theme.mono
            font.pixelSize: 11
        }
    }

    // ── a folha ────────────────────────────────────────────────────────

    Rectangle {
        id: sheet

        width: Math.min(1060, win.width - (win.width < 1400 ? 60 : 120))
        height: Math.min(700, win.height - (win.height < 800 ? 60 : 140))
        x: (win.width - width) / 2
        y: (win.height - height) / 2 + (win.open ? 0 : 14)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        opacity: win.open ? 1 : 0

        Behavior on opacity {
            NumberAnimation {
                duration: win.open ? Motion.enter : Motion.exit
                easing.type: Easing.BezierSpline
                easing.bezierCurve: win.open ? Motion.enterCurve : Motion.exitCurve
            }
        }
        Behavior on y {
            NumberAnimation {
                duration: win.open ? Motion.enter : Motion.exit
                easing.type: Easing.BezierSpline
                easing.bezierCurve: win.open ? Motion.enterCurve : Motion.exitCurve
            }
        }

        Keys.onEscapePressed: {
            if (win.query !== "") {
                search.text = "";
            } else {
                Mare.close();
            }
        }

        Image {
            anchors.fill: parent
            source: Qt.resolvedUrl("assets/grain.png")
            fillMode: Image.Tile
            opacity: 0.05
            smooth: false
        }

        // Marca de cobre no alto, como nos painéis.
        Rectangle {
            x: 26
            y: -1
            width: 28
            height: 2
            color: Theme.copper
        }

        // ── trilho: busca e páginas ──
        Item {
            id: rail
            width: 250
            height: parent.height

            Rectangle {
                anchors.right: parent.right
                width: 1
                height: parent.height
                color: Theme.hairline
            }

            Column {
                id: railTop
                x: 26
                y: 26
                width: parent.width - 52
                spacing: 4

                Row {
                    spacing: 10

                    Image {
                        id: brand
                        source: Qt.resolvedUrl("assets/mark.svg")
                        sourceSize.height: 26
                    }
                    Text {
                        anchors.bottom: brand.bottom
                        anchors.bottomMargin: 2
                        text: "ajustes"
                        color: Theme.textDim
                        font.family: Theme.serif
                        font.pixelSize: 17
                        font.italic: true
                        font.weight: Font.Light
                    }
                }

                Item { width: 1; height: 14 }

                // Busca: acha páginas, opções e atalhos.
                Item {
                    width: parent.width
                    height: 30

                    TextInput {
                        id: search
                        anchors.fill: parent
                        anchors.bottomMargin: 6
                        verticalAlignment: TextInput.AlignVCenter
                        color: Theme.text
                        selectionColor: Theme.copper
                        font.family: Theme.sans
                        font.pixelSize: 14
                        cursorDelegate: Rectangle {
                            width: 2
                            color: Theme.copper
                            visible: search.activeFocus
                        }
                        onTextChanged: win.query = text
                        Keys.onReturnPressed: {
                            if (win.results.length > 0)
                                win.go(win.results[0].page, win.results[0].key);
                        }

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: search.text === ""
                            text: "buscar um ajuste…"
                            color: Theme.textFaint
                            font.family: Theme.serif
                            font.pixelSize: 14
                            font.italic: true
                        }
                    }

                    Rectangle {
                        anchors.bottom: parent.bottom
                        width: parent.width
                        height: 1
                        color: search.activeFocus ? Theme.line : Theme.hairline
                    }
                }

                Item { width: 1; height: 10 }
            }

            // As páginas rolam quando a tela é baixa; o caminho do arquivo fica embaixo.
            Flickable {
                x: 26
                y: railTop.y + railTop.height
                width: parent.width - 52
                height: parent.height - y - 52
                clip: true
                contentHeight: railItems.implicitHeight
                boundsBehavior: Flickable.StopAtBounds

            Column {
                id: railItems
                width: parent.width
                spacing: 4

                // Resultados da busca.
                Repeater {
                    model: win.query !== "" ? win.results : []

                    Item {
                        id: hit
                        required property var modelData
                        required property int index
                        width: parent.width
                        height: 44

                        Rectangle {
                            anchors.fill: parent
                            color: hitHover.hovered || hit.index === 0 ? Theme.wash : "transparent"
                        }

                        Column {
                            x: 10
                            anchors.verticalCenter: parent.verticalCenter
                            width: parent.width - 20
                            spacing: 3

                            Text {
                                width: parent.width
                                text: hit.modelData.label
                                color: Theme.text
                                font.family: Theme.sans
                                font.pixelSize: 13
                                elide: Text.ElideRight
                            }
                            Text {
                                width: parent.width
                                text: hit.modelData.where
                                color: Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 9
                                elide: Text.ElideRight
                            }
                        }

                        HoverHandler {
                            id: hitHover
                            cursorShape: Qt.PointingHandCursor
                        }
                        TapHandler {
                            onTapped: win.go(hit.modelData.page, hit.modelData.key)
                        }
                    }
                }

                Text {
                    visible: win.query !== "" && win.results.length === 0
                    text: "nada com esse nome"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }

                // Páginas, por grupo.
                Repeater {
                    model: win.query === "" ? Schema.groups : []

                    Column {
                        id: group
                        required property var modelData
                        width: parent.width
                        spacing: 0

                        Text {
                            topPadding: 12
                            bottomPadding: 6
                            text: group.modelData.name
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 9
                            font.letterSpacing: 1.8
                        }

                        Repeater {
                            model: group.modelData.items

                            Item {
                                id: entry
                                required property var modelData
                                readonly property bool current: win.page === modelData.key
                                width: group.width
                                height: 32

                                Rectangle {
                                    anchors.fill: parent
                                    color: entry.current || entryHover.hovered ? Theme.wash : "transparent"
                                }

                                Rectangle {
                                    visible: entry.current
                                    width: 2
                                    height: parent.height
                                    color: Theme.copper
                                }

                                Row {
                                    x: 12
                                    anchors.verticalCenter: parent.verticalCenter
                                    spacing: 12

                                    Text {
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: entry.modelData.num
                                        color: entry.current ? Theme.copper : Theme.textFaint
                                        font.family: Theme.mono
                                        font.pixelSize: 10
                                    }
                                    Text {
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: entry.modelData.name
                                        color: entry.current ? Theme.text : Theme.textSoft
                                        font.family: Theme.sans
                                        font.pixelSize: 13
                                    }
                                }

                                HoverHandler {
                                    id: entryHover
                                    cursorShape: Qt.PointingHandCursor
                                }
                                TapHandler {
                                    onTapped: win.go(entry.modelData.key, "")
                                }
                            }
                        }
                    }
                }
            }
            }

            Text {
                x: 26
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 22
                width: parent.width - 52
                text: Settings.jsonPath.replace(Quickshell.env("HOME"), "~")
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 9
                elide: Text.ElideMiddle
            }
        }

        // ── a página ──
        Flickable {
            id: scroller
            anchors.left: rail.right
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            anchors.bottomMargin: 54
            clip: true
            contentHeight: body.implicitHeight + 60
            boundsBehavior: Flickable.StopAtBounds

            ColumnLayout {
                id: body
                x: 40
                y: 30
                width: scroller.width - 80
                spacing: 0

                PanelHeader {
                    Layout.fillWidth: true
                    index: win.pageInfo ? win.pageInfo.num : ""
                    title: win.pageInfo ? win.pageInfo.name : ""
                    meta: Schema.groupOf(win.page).toLowerCase()
                }

                Text {
                    Layout.fillWidth: true
                    Layout.topMargin: 8
                    Layout.bottomMargin: 10
                    text: Schema.blurbs[win.page] || ""
                    wrapMode: Text.WordWrap
                    color: Theme.textDim
                    font.family: Theme.serif
                    font.pixelSize: 15
                    font.italic: true
                    font.weight: Font.Light
                }

                // Atualizações: o resumo, a lista e os botões, antes das opções.
                ColumnLayout {
                    visible: win.page === "atualizacoes"
                    Layout.fillWidth: true
                    Layout.topMargin: 8
                    spacing: 10

                    RowLayout {
                        Layout.fillWidth: true
                        spacing: 16

                        Text {
                            text: Updates.checking && !Updates.info ? "…" : Updates.count > 0 ? String(Updates.count)
                                : Updates.error !== "" ? "—" : "em dia"
                            color: Updates.count > 0 ? Theme.copper : Theme.text
                            font.family: Theme.serif
                            font.pixelSize: 44
                            font.weight: Font.Light
                        }

                        ColumnLayout {
                            spacing: 3
                            Layout.alignment: Qt.AlignVCenter

                            Text {
                                text: Updates.checking ? "procurando…"
                                    : Updates.count === 0 && Updates.error !== "" ? "não deu para procurar"
                                    : Updates.count === 0 ? "nada para atualizar"
                                    : Updates.count === 1 ? "atualização esperando" : "atualizações esperando"
                                color: Theme.textSoft
                                font.family: Theme.serif
                                font.pixelSize: 16
                                font.italic: true
                            }
                            Text {
                                text: Updates.ago()
                                color: Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 10
                            }
                        }

                        Item { Layout.fillWidth: true }

                        PillButton {
                            text: Updates.checking ? "procurando…" : "procurar agora"
                            onClicked: Updates.check()
                        }
                        PillButton {
                            visible: Updates.count > 0
                            primary: true
                            text: "atualizar tudo"
                            onClicked: {
                                Updates.run();
                                Mare.close();
                            }
                        }
                    }

                    Text {
                        visible: Updates.error !== ""
                        Layout.fillWidth: true
                        text: Updates.error
                        wrapMode: Text.WordWrap
                        color: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }

                    // Mk, sistema e AUR, cada um numa linha.
                    GridLayout {
                        Layout.fillWidth: true
                        Layout.topMargin: 6
                        columns: 3
                        columnSpacing: 24

                        Repeater {
                            model: [
                                ["mk", !Updates.info ? "—" : Updates.mkAvailable ? "nova versão " + Updates.info.mk.latest
                                    : Updates.info.mk.channel === "" ? "sem canal (" + (Updates.info.mk.version || "dev") + ")"
                                    : "em dia · " + Updates.info.mk.version],
                                ["sistema", Updates.info ? Updates.systemCount + " pacotes" : "—"],
                                ["aur", Updates.info ? Updates.aurCount + " pacotes" : "—"
                                    + (Updates.migrations > 0 ? " · " + Updates.migrations + " ajustes do Mk" : "")]
                            ]

                            ColumnLayout {
                                required property var modelData
                                Layout.fillWidth: true
                                Layout.preferredWidth: 1
                                spacing: 4

                                Rectangle {
                                    Layout.fillWidth: true
                                    implicitHeight: 1
                                    color: Theme.hairline
                                }
                                Text {
                                    Layout.topMargin: 6
                                    text: modelData[0]
                                    color: Theme.textFaint
                                    font.family: Theme.mono
                                    font.pixelSize: 9
                                    font.capitalization: Font.AllUppercase
                                    font.letterSpacing: 1.6
                                }
                                Text {
                                    Layout.fillWidth: true
                                    text: modelData[1]
                                    color: Theme.text
                                    font.family: Theme.mono
                                    font.pixelSize: 12
                                    elide: Text.ElideRight
                                }
                            }
                        }
                    }

                    // Os pacotes, os primeiros 12.
                    Repeater {
                        model: Updates.info ? Updates.info.system.concat(Updates.info.aur).slice(0, 12) : []

                        RowLayout {
                            required property var modelData
                            Layout.fillWidth: true
                            spacing: 12

                            Text {
                                Layout.preferredWidth: 220
                                text: modelData.name
                                color: Theme.textSoft
                                font.family: Theme.mono
                                font.pixelSize: 11
                                elide: Text.ElideRight
                            }
                            Text {
                                Layout.fillWidth: true
                                text: modelData.from + "  →  " + modelData.to
                                color: Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 11
                                elide: Text.ElideMiddle
                            }
                        }
                    }

                    Text {
                        visible: Updates.info !== null && Updates.systemCount + Updates.aurCount > 12
                        text: "… e mais " + (Updates.systemCount + Updates.aurCount - 12)
                        color: Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                }

                // Som: por onde sai e por onde entra.
                Repeater {
                    model: win.page === "som" ? [
                        { title: "SAÍDA", sink: true },
                        { title: "ENTRADA", sink: false }
                    ] : []

                    ColumnLayout {
                        id: devices
                        required property var modelData
                        readonly property var nodes: Pipewire.nodes.values.filter(n => n.audio && !n.isStream && n.isSink === devices.modelData.sink)
                        readonly property var current: devices.modelData.sink ? Pipewire.defaultAudioSink : Pipewire.defaultAudioSource
                        Layout.fillWidth: true
                        spacing: 0

                        PwObjectTracker {
                            objects: devices.nodes
                        }

                        Text {
                            Layout.topMargin: 18
                            Layout.bottomMargin: 4
                            text: devices.modelData.title
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 9
                            font.letterSpacing: 1.8
                        }

                        Text {
                            visible: devices.nodes.length === 0
                            text: "nenhum aparelho"
                            color: Theme.textDim
                            font.family: Theme.serif
                            font.pixelSize: 14
                            font.italic: true
                        }

                        Repeater {
                            model: devices.nodes

                            Item {
                                id: dev
                                required property var modelData
                                readonly property bool active: devices.current === modelData
                                Layout.fillWidth: true
                                implicitHeight: 44

                                Rectangle {
                                    anchors.fill: parent
                                    color: dev.active || devHover.hovered ? Theme.wash : "transparent"
                                }
                                Rectangle {
                                    visible: dev.active
                                    width: 2
                                    height: parent.height
                                    color: Theme.copper
                                }
                                Text {
                                    x: 14
                                    anchors.verticalCenter: parent.verticalCenter
                                    width: parent.width - 120
                                    text: dev.modelData.description || dev.modelData.nickname || dev.modelData.name
                                    color: dev.active ? Theme.text : Theme.textSoft
                                    font.family: Theme.sans
                                    font.pixelSize: 13
                                    elide: Text.ElideRight
                                }
                                Text {
                                    anchors.right: parent.right
                                    anchors.rightMargin: 12
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: dev.active ? "em uso" : ""
                                    color: Theme.copper
                                    font.family: Theme.mono
                                    font.pixelSize: 10
                                    font.capitalization: Font.AllUppercase
                                    font.letterSpacing: 1.2
                                }
                                HoverHandler {
                                    id: devHover
                                    cursorShape: Qt.PointingHandCursor
                                }
                                TapHandler {
                                    onTapped: {
                                        if (devices.modelData.sink)
                                            Pipewire.preferredDefaultAudioSink = dev.modelData;
                                        else
                                            Pipewire.preferredDefaultAudioSource = dev.modelData;
                                    }
                                }
                            }
                        }
                    }
                }

                // Notificações: o não perturbe (o mesmo do Super + Shift + N).
                RowLayout {
                    visible: win.page === "notificacoes"
                    Layout.fillWidth: true
                    Layout.topMargin: 14
                    spacing: 16

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 4

                        Text {
                            text: "Não perturbe"
                            color: Theme.text
                            font.family: Theme.sans
                            font.pixelSize: 14
                        }
                        Text {
                            text: Notifs.dnd ? "Os avisos vão direto para a lista, sem descer da ilha" : "Os avisos descem da ilha"
                            color: Theme.textDim
                            font.family: Theme.sans
                            font.pixelSize: 12
                        }
                    }

                    Toggle {
                        checked: Notifs.dnd
                        onToggled: v => Notifs.dnd = v
                    }
                }

                // Linhas da página.
                Repeater {
                    model: Schema.pageRows(win.page)

                    ColumnLayout {
                        id: line
                        required property var modelData
                        required property int index
                        readonly property var rowsHere: Schema.pageRows(win.page)
                        readonly property bool firstOfGroup: index === 0 || rowsHere[index - 1].group !== modelData.group
                        readonly property var value: Settings.get(modelData.key)

                        Layout.fillWidth: true
                        spacing: 0

                        Text {
                            visible: line.firstOfGroup
                            Layout.topMargin: 18
                            Layout.bottomMargin: 4
                            text: line.modelData.group
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 9
                            font.letterSpacing: 1.8
                        }

                        Rectangle {
                            Layout.fillWidth: true
                            implicitHeight: 1
                            color: Theme.hairline
                        }

                        Item {
                            Layout.fillWidth: true
                            implicitHeight: 62

                            Rectangle {
                                anchors.fill: parent
                                anchors.leftMargin: -12
                                anchors.rightMargin: -12
                                color: Theme.copper
                                opacity: win.flashKey === line.modelData.key ? 0.12 : 0

                                Behavior on opacity {
                                    NumberAnimation { duration: Motion.slow }
                                }
                            }

                            Column {
                                anchors.left: parent.left
                                anchors.right: control.left
                                anchors.rightMargin: 24
                                anchors.verticalCenter: parent.verticalCenter
                                spacing: 4

                                Row {
                                    spacing: 8

                                    Text {
                                        text: line.modelData.label
                                        color: Theme.text
                                        font.family: Theme.sans
                                        font.pixelSize: 14
                                    }
                                    // Mudado em relação à fábrica.
                                    Rectangle {
                                        visible: !Settings.isDefault(line.modelData.key)
                                        anchors.verticalCenter: parent.verticalCenter
                                        width: 5
                                        height: 5
                                        radius: 3
                                        color: Theme.copper
                                    }
                                }
                                Text {
                                    width: parent.width
                                    text: line.modelData.desc
                                    color: Theme.textDim
                                    font.family: Theme.sans
                                    font.pixelSize: 12
                                    wrapMode: Text.WordWrap
                                }
                            }

                            Item {
                                id: control
                                anchors.right: parent.right
                                anchors.verticalCenter: parent.verticalCenter
                                width: line.modelData.ctl === "switch" ? 36 : line.modelData.ctl === "slider" ? 280 : choiceRow.implicitWidth
                                height: 28

                                Toggle {
                                    visible: line.modelData.ctl === "switch"
                                    anchors.right: parent.right
                                    anchors.verticalCenter: parent.verticalCenter
                                    checked: line.value === true
                                    onToggled: v => Settings.set(line.modelData.key, v)
                                }

                                Slider {
                                    visible: line.modelData.ctl === "slider"
                                    anchors.fill: parent
                                    from: line.modelData.min ?? 0
                                    to: line.modelData.max ?? 1
                                    step: line.modelData.step ?? 0.1
                                    value: Number(line.value)
                                    label: line.modelData.ctl === "slider" ? win.format(line.modelData, Number(line.value)) : ""
                                    onMoved: v => Settings.set(line.modelData.key, v)
                                }

                                Row {
                                    id: choiceRow
                                    visible: line.modelData.ctl === "choice"
                                    anchors.right: parent.right
                                    anchors.verticalCenter: parent.verticalCenter
                                    spacing: 6

                                    Repeater {
                                        model: line.modelData.ctl === "choice" ? line.modelData.options : []

                                        PillButton {
                                            required property var modelData
                                            text: modelData.label
                                            selected: line.value === modelData.v
                                            onClicked: Settings.set(line.modelData.key, modelData.v)
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                // Extras de algumas páginas.
                RowLayout {
                    visible: win.page === "fundo"
                    Layout.topMargin: 20
                    spacing: 8

                    PillButton {
                        text: "escolher papel de parede"
                        onClicked: Mare.open("wallpaper")
                    }
                    PillButton {
                        visible: Wallpaper.current !== ""
                        text: "só a folha, sem imagem"
                        onClicked: Wallpaper.set("")
                    }
                }

                // Atalhos.
                Repeater {
                    model: win.page === "atalhos" ? Schema.shortcuts : []

                    ColumnLayout {
                        id: keys
                        required property var modelData
                        Layout.fillWidth: true
                        spacing: 0

                        Text {
                            Layout.topMargin: 18
                            Layout.bottomMargin: 4
                            text: keys.modelData.group
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 9
                            font.letterSpacing: 1.8
                        }

                        Repeater {
                            model: keys.modelData.items

                            ColumnLayout {
                                id: bind
                                required property var modelData
                                Layout.fillWidth: true
                                spacing: 0

                                Rectangle {
                                    Layout.fillWidth: true
                                    implicitHeight: 1
                                    color: Theme.hairline
                                }

                                RowLayout {
                                    Layout.fillWidth: true
                                    Layout.preferredHeight: 34
                                    spacing: 16

                                    Text {
                                        Layout.preferredWidth: 220
                                        text: bind.modelData[0]
                                        color: Theme.text
                                        font.family: Theme.mono
                                        font.pixelSize: 12
                                    }
                                    Text {
                                        Layout.fillWidth: true
                                        text: bind.modelData[1]
                                        color: Theme.textSoft
                                        font.family: Theme.sans
                                        font.pixelSize: 13
                                        elide: Text.ElideRight
                                    }
                                }
                            }
                        }
                    }
                }

                // Sobre.
                GridLayout {
                    visible: win.page === "sobre"
                    Layout.fillWidth: true
                    Layout.topMargin: 18
                    columns: 2
                    rowSpacing: 0
                    columnSpacing: 24

                    Repeater {
                        model: win.page === "sobre" ? [
                            ["máquina", win.about.host],
                            ["processador", win.about.cpu],
                            ["memória", win.about.mem],
                            ["sistema", win.about.kernel],
                            ["ligado há", win.about.uptime],
                            ["hyprland", win.about.hyprland],
                            ["quickshell", win.about.quickshell],
                            ["ajustes", Settings.jsonPath.replace(Quickshell.env("HOME"), "~")],
                            ["janelas", Settings.luaPath.replace(Quickshell.env("HOME"), "~")]
                        ] : []

                        ColumnLayout {
                            id: fact
                            required property var modelData
                            Layout.fillWidth: true
                            Layout.preferredWidth: 1
                            spacing: 4

                            Rectangle {
                                Layout.fillWidth: true
                                implicitHeight: 1
                                color: Theme.hairline
                            }
                            Text {
                                Layout.topMargin: 8
                                text: fact.modelData[0]
                                color: Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 9
                                font.capitalization: Font.AllUppercase
                                font.letterSpacing: 1.6
                            }
                            Text {
                                Layout.fillWidth: true
                                Layout.bottomMargin: 12
                                text: fact.modelData[1] || "—"
                                color: Theme.text
                                font.family: Theme.mono
                                font.pixelSize: 12
                                elide: Text.ElideMiddle
                            }
                        }
                    }
                }

                PillButton {
                    visible: Schema.pageRows(win.page).some(r => !Settings.isDefault(r.key))
                    Layout.topMargin: 24
                    text: "voltar ao padrão desta página"
                    onClicked: Settings.reset(win.page)
                }
            }
        }

        // ── carimbo da folha ──
        Item {
            anchors.left: rail.right
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            height: 54

            Rectangle {
                width: parent.width
                height: 1
                color: Theme.hairline
            }

            Row {
                anchors.right: parent.right
                anchors.rightMargin: 40
                anchors.verticalCenter: parent.verticalCenter
                spacing: 28

                Repeater {
                    model: [
                        ["projeto", "Mk"],
                        ["folha", (win.pageInfo ? win.pageInfo.num : "--") + " / " + String(win.allPages.length).padStart(2, "0")],
                        ["assunto", win.pageInfo ? win.pageInfo.name.toLowerCase() : ""],
                        ["fechar", "esc"]
                    ]

                    Column {
                        required property var modelData
                        spacing: 3

                        Text {
                            text: modelData[0]
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 8
                            font.capitalization: Font.AllUppercase
                            font.letterSpacing: 1.6
                        }
                        Text {
                            text: modelData[1]
                            color: modelData[0] === "projeto" ? Theme.copper : Theme.textSoft
                            font.family: Theme.mono
                            font.pixelSize: 11
                        }
                    }
                }
            }
        }
    }
}
