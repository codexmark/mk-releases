import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Wayland
import "Keybinds.js" as KB

// Boas-vindas: uma folha no meio da tela, feita como a dos Ajustes, que
// aparece no primeiro login (Welcome.qml decide quando) e apresenta o Mk em
// quatro passos: a ilha, a tecla Super, os projetos e o cuidado com o sistema.
// À esquerda, uma figura desenhada; à direita, o texto e as teclas; embaixo, o
// carimbo com "pular", os passos e os botões.
//
// Quem abre um painel no meio do passeio (a busca, os Ajustes) recolhe a
// folha: ela não volta sozinha por cima do que foi aberto. Fica uma etiqueta
// no pé da tela, sem teclado e sem cobrir nada, que a traz de volta.
//
// Os atalhos mostrados vêm do Keybinds.js (o espelho do binds.lua) mais o que
// o usuário trocou nos Ajustes, para a folha nunca ensinar uma tecla que não
// existe.
PanelWindow {
    id: win

    property bool active: true
    // Dá a vez a qualquer painel da ilha e aos Ajustes. Os passageiros (aviso,
    // OSD) não a escondem. Com a tela bloqueada, nunca.
    readonly property bool clear: active && Welcome.shown && !Lock.locked && !Welcome.panelUp
    readonly property bool open: clear && !Welcome.paused
    // Recolhida: só a etiqueta.
    readonly property bool tagOpen: clear && Welcome.paused

    // Teclas só valem um instante depois de a folha subir (quem estava
    // digitando quando ela apareceu não a atravessa sem ler) e nunca por
    // repetição de tecla segurada.
    property double openedAt: 0
    function keyOk(event) {
        return !event.isAutoRepeat && Date.now() - openedAt > 700;
    }

    readonly property int step: Welcome.step
    readonly property var current: steps[Math.max(0, Math.min(steps.length - 1, step))]
    readonly property string home: Quickshell.env("HOME") || ""

    // O combo de uma ação hoje: o de fábrica ou o que foi trocado nos Ajustes
    // ("" = desligado). `fallback` vale se o Keybinds.js não conhecer o id.
    function combo(id, fallback) {
        try {
            const action = KB.byId(id);
            if (!action)
                return fallback;
            const changed = KB.clean(Settings.get("keybinds"))[id];
            return typeof changed === "string" ? changed : action.combo;
        } catch (e) {
            return fallback;
        }
    }

    // O combo como teclas para desenhar: ["Super", "Enter"]. `tail` é a tecla
    // de uma família ("1…9"). Vazio = a ação está sem atalho.
    function caps(id, fallback, tail) {
        const c = combo(id, fallback);
        if (c === "")
            return [];
        const parsed = KB.parse(c);
        const out = parsed.mods.map(m => KB.modLabels[m] || m);
        if (parsed.key !== "") {
            const k = parsed.key.toLowerCase();
            out.push(KB.keyLabels[k] || (parsed.key.length === 1 ? parsed.key.toUpperCase() : parsed.key));
        }
        if (tail)
            out.push(tail);
        return out;
    }

    // Os quatro passos. Cada linha tem `what` e, à direita, teclas (`keys`,
    // com uma `note` opcional) ou um texto (`value`).
    readonly property var steps: [
        {
            fig: "a ilha",
            title: "bem-vindo ao Mk",
            lead: "No alto da tela fica a ilha. Tudo no Mk sai dela: clique num item e o painel dele desce dali, soldado à ilha. Esc ou um clique fora o recolhe.",
            rows: [
                { what: "à esquerda", value: "marca, workspaces, sistema" },
                { what: "no centro", value: "relógio e clima" },
                { what: "à direita", value: "bandeja e indicadores" }
            ]
        },
        {
            fig: "a tecla super",
            title: "a tecla Super",
            lead: "Quase tudo começa nela. Um toque sozinho abre a busca, que acha apps, projetos e comandos. A lista inteira, e como trocar cada atalho, está em Ajustes › Atalhos.",
            rows: [
                { what: "busca", keys: caps("mare.launcher.tap", "SUPER", ""), note: "um toque" },
                { what: "ajustes", keys: caps("mare.settings", "SUPER + COMMA", "") },
                { what: "terminal", keys: caps("app.terminal", "SUPER + RETURN", "") },
                // As letras vão na nota: o rótulo é em maiúsculas, e α β
                // maiúsculos viram "A B".
                { what: "workspaces", keys: caps("ws.focus", "SUPER", "1…9"), note: "α β γ" }
            ]
        },
        {
            fig: "workspace = projeto",
            title: "projetos",
            lead: "Um workspace pode ser ligado a uma pasta de projeto: um repositório git dentro de ~/Work. Escolha o projeto na busca e o Mk abre um workspace novo para ele, com um terminal na pasta. Dali em diante, naquele workspace, o terminal e o lazygit abrem na pasta dele.",
            rows: [
                { what: "ligar", value: "na busca, o nome do projeto" },
                { what: "painel do projeto", keys: caps("mare.project", "SUPER + P", "") },
                { what: "terminal na pasta", keys: caps("app.terminal", "SUPER + RETURN", "") },
                { what: "lazygit", keys: caps("app.lazygit", "SUPER + G", "") }
            ]
        },
        {
            fig: "snapshots",
            title: "cuidando do sistema",
            lead: "Um comando atualiza tudo: o Mk, o Arch e o AUR. Antes e depois de cada leva de pacotes o sistema guarda um snapshot, e o menu de boot lista os últimos: se algo quebrar, dá para voltar.",
            rows: [
                { what: "no terminal", value: "mk atualizar" },
                { what: "ou em", value: "Ajustes › Atualizações" },
                { what: "se quebrar", value: "snapshots no menu de boot" },
                { what: "seus ajustes", value: Settings.jsonPath.replace(win.home, "~") }
            ]
        }
    ]

    anchors {
        top: true
        bottom: true
        left: true
        right: true
    }
    // A tela inteira, também atrás da ilha (veja Mare.ignoreZone).
    exclusionMode: ExclusionMode.Normal
    exclusiveZone: Mare.ignoreZone
    color: "transparent"
    visible: open || sheet.opacity > 0 || tagOpen || tag.opacity > 0
    // Recolhida, a janela só recebe cliques na etiqueta; o resto da tela passa.
    mask: Welcome.paused ? tagRegion : null
    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "mare-welcome"
    // Sem o grab do Monitor.qml (ele é dos painéis): a folha pede o teclado
    // por conta própria enquanto está na frente, e o solta na hora em que dá a
    // vez a um painel. Os atalhos do Hyprland continuam valendo.
    WlrLayershell.keyboardFocus: open ? WlrKeyboardFocus.Exclusive : WlrKeyboardFocus.None

    onOpenChanged: {
        if (open) {
            openedAt = Date.now();
            sheet.forceActiveFocus();
        }
    }

    Region {
        id: tagRegion
        item: tag
    }
    onStepChanged: turn.restart()

    // Fundo escurecido: clicar fora da folha fecha. Um TapHandler não para o
    // clique no item de cima, então este recebe também os cliques dados dentro
    // da folha: só fecha se o ponto estiver fora dela. Com a folha recolhida
    // não vale: o clique na etiqueta também chega aqui, e "continuar" não pode
    // virar "fechar".
    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: win.open ? 0.55 : 0

        Behavior on opacity {
            NumberAnimation { duration: Motion.base }
        }

        TapHandler {
            enabled: win.open
            onTapped: (point, button) => {
                const p = point.position;
                const inside = p.x >= sheet.x && p.x <= sheet.x + sheet.width
                    && p.y >= sheet.y && p.y <= sheet.y + sheet.height;
                const onTag = p.x >= tag.x && p.x <= tag.x + tag.width
                    && p.y >= tag.y && p.y <= tag.y + tag.height;
                if (!inside && !onTag)
                    Welcome.finish();
            }
        }
    }

    // ── peças ──────────────────────────────────────────────────────────

    // Uma tecla: a mesma em todas as figuras e linhas.
    component KeyCap: Rectangle {
        id: cap
        property string label: ""
        property bool lit: false

        implicitWidth: Math.max(28, capText.implicitWidth + 18)
        implicitHeight: 26
        radius: 3
        color: Theme.raised
        border.width: 1
        border.color: lit ? Theme.copper : Theme.line

        // A borda de baixo mais grossa: o corpo da tecla.
        Rectangle {
            x: 1
            y: parent.height - 3
            width: parent.width - 2
            height: 2
            color: cap.lit ? Theme.copper : Theme.hairline
        }

        Text {
            id: capText
            anchors.centerIn: parent
            anchors.verticalCenterOffset: -1
            text: cap.label
            color: cap.lit ? Theme.copper : Theme.text
            font.family: Theme.mono
            font.pixelSize: 11
        }
    }

    // Link de texto do carimbo ("pular").
    component TextLink: Text {
        id: link
        signal clicked

        color: linkHover.hovered ? Theme.text : Theme.textDim
        font.family: Theme.serif
        font.pixelSize: 14
        font.italic: true
        font.underline: linkHover.hovered

        HoverHandler {
            id: linkHover
            cursorShape: Qt.PointingHandCursor
        }
        TapHandler {
            onTapped: link.clicked()
        }
    }

    // ── a folha ────────────────────────────────────────────────────────

    Rectangle {
        id: sheet

        width: Math.min(860, win.width - 60)
        height: Math.min(540, win.height - 60)
        x: (win.width - width) / 2
        y: (win.height - height) / 2 + (win.open ? 0 : 14)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        opacity: win.open ? 1 : 0
        clip: true
        focus: true

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

        Keys.onPressed: event => {
            const k = event.key;
            const known = k === Qt.Key_Escape || k === Qt.Key_Left || k === Qt.Key_Right
                || k === Qt.Key_Return || k === Qt.Key_Enter || k === Qt.Key_Space;
            if (!known)
                return;
            event.accepted = true;
            if (!win.keyOk(event))
                return;
            if (k === Qt.Key_Escape)
                Welcome.finish();
            else if (k === Qt.Key_Left)
                Welcome.back();
            else
                Welcome.next();
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
            y: 0
            width: 28
            height: 2
            color: Theme.copper
        }

        // ── a prancha: a figura do passo ──
        Item {
            id: plate
            width: 300
            height: parent.height - 54

            Rectangle {
                anchors.right: parent.right
                width: 1
                height: parent.height
                color: Theme.hairline
            }

            Row {
                x: 26
                y: 26
                spacing: 10

                Image {
                    id: brand
                    source: Qt.resolvedUrl("assets/mark.svg")
                    sourceSize.height: 26
                }
                Text {
                    anchors.bottom: brand.bottom
                    anchors.bottomMargin: 2
                    text: "boas-vindas"
                    color: Theme.textDim
                    font.family: Theme.serif
                    font.pixelSize: 17
                    font.italic: true
                    font.weight: Font.Light
                }
            }

            // As quatro figuras ocupam o mesmo lugar; só a do passo aparece.
            Item {
                id: figure
                anchors.centerIn: parent
                anchors.verticalCenterOffset: 6
                width: 240
                height: 200

                // 1 · a ilha: uma tela, a ilha no alto e um painel descendo dela.
                Item {
                    id: figIsland
                    anchors.fill: parent
                    visible: win.step === 0

                    Rectangle {
                        anchors.fill: parent
                        anchors.topMargin: 16
                        anchors.bottomMargin: 16
                        color: "transparent"
                        border.width: 1
                        border.color: Theme.hairline
                        radius: 2
                    }

                    // O painel, saindo da borda de baixo da ilha.
                    Rectangle {
                        id: figPanel
                        x: (parent.width - width) / 2
                        y: 16 + 13
                        width: 84
                        height: 74
                        color: Theme.raised
                        border.width: 1
                        border.color: Theme.line
                        radius: 2
                        clip: true

                        Column {
                            x: 10
                            y: 14
                            width: parent.width - 20
                            spacing: 9

                            Rectangle { width: 16; height: 2; color: Theme.copper }
                            Rectangle { width: parent.width; height: 1; color: Theme.line }
                            Rectangle { width: parent.width * 0.7; height: 1; color: Theme.hairline }
                            Rectangle { width: parent.width * 0.85; height: 1; color: Theme.hairline }
                            Rectangle { width: parent.width * 0.5; height: 1; color: Theme.hairline }
                        }

                        // Desce e recolhe, devagar, enquanto o passo está à vista.
                        SequentialAnimation on height {
                            running: win.open && win.step === 0 && Motion.speed > 0
                            loops: Animation.Infinite
                            NumberAnimation { from: 0; to: 74; duration: 700; easing.type: Easing.OutCubic }
                            PauseAnimation { duration: 2200 }
                            NumberAnimation { from: 74; to: 0; duration: 360; easing.type: Easing.InCubic }
                            PauseAnimation { duration: 700 }
                        }
                    }

                    // A ilha: pendurada no meio da borda de cima.
                    Rectangle {
                        x: (parent.width - width) / 2
                        y: 16
                        width: 112
                        height: 14
                        color: Theme.raised
                        border.width: 1
                        border.color: Theme.line
                        radius: 2

                        Row {
                            x: 8
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 4
                            Rectangle { width: 4; height: 4; radius: 2; color: Theme.copper }
                            Rectangle { width: 4; height: 4; radius: 2; color: Theme.textFaint }
                            Rectangle { width: 4; height: 4; radius: 2; color: Theme.textFaint }
                        }
                        Rectangle {
                            anchors.centerIn: parent
                            width: 22
                            height: 2
                            color: Theme.textSoft
                        }
                        Row {
                            anchors.right: parent.right
                            anchors.rightMargin: 8
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 4
                            Rectangle { width: 6; height: 2; color: Theme.textFaint }
                            Rectangle { width: 6; height: 2; color: Theme.textFaint }
                        }
                    }

                    // Uma janela, ao fundo, para dar a escala.
                    Rectangle {
                        x: 14
                        y: parent.height - 16 - 60
                        width: 70
                        height: 48
                        color: "transparent"
                        border.width: 1
                        border.color: Theme.hairline
                    }
                    Rectangle {
                        x: parent.width - 14 - 70
                        y: parent.height - 16 - 60
                        width: 70
                        height: 48
                        color: "transparent"
                        border.width: 1
                        border.color: Theme.hairline
                    }
                }

                // 2 · a tecla Super: a fileira de baixo do teclado, com ela acesa.
                Item {
                    id: figSuper
                    anchors.fill: parent
                    visible: win.step === 1

                    Column {
                        anchors.centerIn: parent
                        spacing: 6

                        Row {
                            spacing: 6
                            KeyCap { label: "shift"; implicitWidth: 76; implicitHeight: 34 }
                            KeyCap { label: "z"; implicitWidth: 34; implicitHeight: 34 }
                            KeyCap { label: "x"; implicitWidth: 34; implicitHeight: 34 }
                            KeyCap { label: "c"; implicitWidth: 34; implicitHeight: 34 }
                            KeyCap { label: "v"; implicitWidth: 34; implicitHeight: 34 }
                        }
                        Row {
                            spacing: 6
                            KeyCap { label: "ctrl"; implicitWidth: 46; implicitHeight: 34 }
                            KeyCap { label: "super"; lit: true; implicitWidth: 56; implicitHeight: 34 }
                            KeyCap { label: "alt"; implicitWidth: 46; implicitHeight: 34 }
                            KeyCap { label: ""; implicitWidth: 74; implicitHeight: 34 }
                        }
                    }
                }

                // 3 · projetos: três workspaces, cada um com (ou sem) a sua pasta.
                Item {
                    id: figProjects
                    anchors.fill: parent
                    visible: win.step === 2

                    Column {
                        anchors.centerIn: parent
                        width: parent.width
                        spacing: 0

                        Repeater {
                            model: [
                                { ws: 1, name: "loja-api", dir: "~/Work/loja-api" },
                                { ws: 2, name: "site", dir: "~/Work/site" },
                                { ws: 3, name: "livre", dir: "" }
                            ]

                            Item {
                                id: tile
                                required property var modelData
                                required property int index
                                width: parent.width
                                height: 56

                                Rectangle { width: parent.width; height: 1; color: Theme.hairline }
                                Rectangle {
                                    visible: tile.index === 2
                                    y: parent.height - 1
                                    width: parent.width
                                    height: 1
                                    color: Theme.hairline
                                }
                                Rectangle {
                                    visible: tile.index === 0
                                    width: 2
                                    height: parent.height
                                    color: Theme.copper
                                }

                                Text {
                                    id: letter
                                    x: 14
                                    anchors.verticalCenter: parent.verticalCenter
                                    width: 26
                                    text: Theme.wsName(tile.modelData.ws)
                                    color: tile.index === 0 ? Theme.copper : Theme.textDim
                                    font.family: Theme.serif
                                    font.pixelSize: 26
                                    font.weight: Font.Light
                                }

                                Column {
                                    anchors.left: letter.right
                                    anchors.leftMargin: 12
                                    anchors.verticalCenter: parent.verticalCenter
                                    spacing: 4

                                    Text {
                                        text: tile.modelData.name
                                        color: tile.modelData.dir !== "" ? Theme.text : Theme.textDim
                                        font.family: Theme.sans
                                        font.pixelSize: 13
                                    }
                                    Text {
                                        text: tile.modelData.dir !== "" ? tile.modelData.dir : "sem pasta"
                                        color: Theme.textFaint
                                        font.family: Theme.mono
                                        font.pixelSize: 9
                                    }
                                }
                            }
                        }
                    }
                }

                // 4 · snapshots: uma linha do tempo, com a volta desenhada.
                Item {
                    id: figCare
                    anchors.fill: parent
                    visible: win.step === 3

                    Canvas {
                        id: careCanvas
                        anchors.fill: parent

                        readonly property color lineColor: Theme.line
                        readonly property color faintColor: Theme.hairline
                        readonly property color accentColor: Theme.copper
                        readonly property color dotColor: Theme.textDim

                        // Escondido, o Canvas não desenha: pinta ao aparecer.
                        onVisibleChanged: if (visible) requestPaint()

                        onPaint: {
                            const ctx = getContext("2d");
                            ctx.reset();
                            const y = 110;
                            const xs = [24, 72, 120, 168, 216];

                            // A linha do tempo.
                            ctx.strokeStyle = lineColor;
                            ctx.lineWidth = 1;
                            ctx.beginPath();
                            ctx.moveTo(8, y + 0.5);
                            ctx.lineTo(width - 8, y + 0.5);
                            ctx.stroke();

                            // A volta: do agora até um snapshot de antes.
                            ctx.strokeStyle = accentColor;
                            ctx.beginPath();
                            ctx.moveTo(xs[4], y - 8);
                            ctx.bezierCurveTo(xs[4] - 10, y - 62, xs[2] + 10, y - 62, xs[2], y - 9);
                            ctx.stroke();
                            ctx.beginPath();
                            ctx.moveTo(xs[2] - 4, y - 16);
                            ctx.lineTo(xs[2], y - 8);
                            ctx.lineTo(xs[2] + 5, y - 15);
                            ctx.stroke();

                            // Os snapshots, e o agora em cobre.
                            for (let i = 0; i < xs.length; i++) {
                                ctx.beginPath();
                                ctx.arc(xs[i], y + 0.5, i === 4 ? 5 : 3.5, 0, Math.PI * 2);
                                ctx.fillStyle = i === 4 ? accentColor : dotColor;
                                ctx.fill();
                            }

                            // Cada atualização fica entre dois snapshots.
                            ctx.strokeStyle = faintColor;
                            for (const pair of [[0, 1], [2, 3]]) {
                                ctx.beginPath();
                                ctx.moveTo(xs[pair[0]], y + 14);
                                ctx.lineTo(xs[pair[0]], y + 20.5);
                                ctx.lineTo(xs[pair[1]], y + 20.5);
                                ctx.lineTo(xs[pair[1]], y + 14);
                                ctx.stroke();
                            }
                        }
                    }

                    Repeater {
                        model: [
                            { x: 48, text: "atualização" },
                            { x: 144, text: "atualização" },
                            { x: 216, text: "agora" }
                        ]

                        Text {
                            required property var modelData
                            x: modelData.x - implicitWidth / 2
                            y: 110 + 26
                            text: modelData.text
                            color: modelData.text === "agora" ? Theme.copper : Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 8
                            font.capitalization: Font.AllUppercase
                            font.letterSpacing: 1.2
                        }
                    }

                    Text {
                        x: 168 - implicitWidth / 2
                        y: 34
                        text: "voltar"
                        color: Theme.textDim
                        font.family: Theme.serif
                        font.pixelSize: 13
                        font.italic: true
                    }
                }
            }

            // Legenda da figura, como numa prancha.
            Text {
                x: 26
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 20
                width: parent.width - 52
                text: "fig. " + String(win.step + 1).padStart(2, "0") + "  ·  " + win.current.fig
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 9
                font.capitalization: Font.AllUppercase
                font.letterSpacing: Theme.labelSpacing
                elide: Text.ElideRight
            }
        }

        // ── o passo: título, texto e linhas ──
        ColumnLayout {
            id: body
            anchors.left: plate.right
            anchors.leftMargin: 40
            anchors.right: parent.right
            anchors.rightMargin: 40
            y: 34
            spacing: 0

            // Trocar de passo: o texto novo aparece aos poucos.
            NumberAnimation {
                id: turn
                target: body
                property: "opacity"
                from: 0
                to: 1
                duration: Motion.base
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.curve
            }

            PanelHeader {
                Layout.fillWidth: true
                index: String(win.step + 1).padStart(2, "0")
                title: win.current.title
                meta: "passo " + (win.step + 1) + " de " + win.steps.length
            }

            Text {
                Layout.fillWidth: true
                Layout.topMargin: 14
                Layout.bottomMargin: 18
                text: win.current.lead
                wrapMode: Text.WordWrap
                color: Theme.textSoft
                font.family: Theme.serif
                font.pixelSize: 15
                font.weight: Font.Light
                lineHeight: 1.25
            }

            Repeater {
                model: win.current.rows

                ColumnLayout {
                    id: line
                    required property var modelData
                    readonly property bool hasKeys: modelData.keys !== undefined
                    Layout.fillWidth: true
                    spacing: 0

                    Rectangle {
                        Layout.fillWidth: true
                        implicitHeight: 1
                        color: Theme.hairline
                    }

                    RowLayout {
                        Layout.fillWidth: true
                        Layout.preferredHeight: 42
                        spacing: 10

                        Text {
                            text: line.modelData.what
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                            font.capitalization: Font.AllUppercase
                            font.letterSpacing: Theme.labelSpacing
                        }

                        Item { Layout.fillWidth: true }

                        Text {
                            visible: line.hasKeys && line.modelData.note !== undefined
                            text: line.modelData.note || ""
                            color: Theme.textDim
                            font.family: Theme.serif
                            font.pixelSize: 13
                            font.italic: true
                        }

                        // As teclas, com um "+" entre elas.
                        Row {
                            visible: line.hasKeys && line.modelData.keys.length > 0
                            spacing: 6

                            Repeater {
                                model: line.hasKeys ? line.modelData.keys : []

                                Row {
                                    id: key
                                    required property var modelData
                                    required property int index
                                    spacing: 6

                                    Text {
                                        visible: key.index > 0
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: "+"
                                        color: Theme.textFaint
                                        font.family: Theme.mono
                                        font.pixelSize: 11
                                    }
                                    KeyCap {
                                        label: key.modelData
                                        lit: key.index === 0 && key.modelData === "Super"
                                    }
                                }
                            }
                        }

                        Text {
                            visible: line.hasKeys && line.modelData.keys.length === 0
                            text: "sem atalho"
                            color: Theme.textFaint
                            font.family: Theme.serif
                            font.pixelSize: 13
                            font.italic: true
                        }

                        Text {
                            visible: !line.hasKeys
                            Layout.maximumWidth: 300
                            text: line.modelData.value || ""
                            color: Theme.text
                            font.family: Theme.mono
                            font.pixelSize: 12
                            elide: Text.ElideMiddle
                        }
                    }
                }
            }

            Rectangle {
                Layout.fillWidth: true
                implicitHeight: 1
                color: Theme.hairline
            }
        }

        // ── carimbo da folha: pular, os passos, voltar e seguir ──
        Item {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            height: 54

            // Fundo próprio: numa tela baixa demais para o passo inteiro, o
            // texto some atrás do carimbo em vez de se misturar com ele.
            Rectangle {
                anchors.fill: parent
                anchors.margins: 1
                color: Theme.surface
            }
            Rectangle {
                width: parent.width
                height: 1
                color: Theme.hairline
            }

            TextLink {
                x: 26
                anchors.verticalCenter: parent.verticalCenter
                visible: win.step < win.steps.length - 1
                text: "pular"
                onClicked: Welcome.finish()
            }

            // Um traço por passo; o atual em cobre. Clicar leva até ele.
            Row {
                anchors.centerIn: parent
                spacing: 6

                Repeater {
                    model: win.steps.length

                    Item {
                        id: tick
                        required property int index
                        width: 22
                        height: 20

                        Rectangle {
                            anchors.verticalCenter: parent.verticalCenter
                            width: parent.width
                            height: tick.index === win.step ? 2 : 1
                            color: tick.index === win.step ? Theme.copper
                                : tick.index < win.step ? Theme.textDim : Theme.line
                        }

                        HoverHandler { cursorShape: Qt.PointingHandCursor }
                        TapHandler { onTapped: Welcome.step = tick.index }
                    }
                }
            }

            Row {
                anchors.right: parent.right
                anchors.rightMargin: 26
                anchors.verticalCenter: parent.verticalCenter
                spacing: 8

                PillButton {
                    visible: win.step > 0
                    text: "voltar"
                    onClicked: Welcome.back()
                }
                PillButton {
                    selected: true
                    text: win.step < win.steps.length - 1 ? "seguir" : "começar"
                    onClicked: Welcome.next()
                }
            }
        }
    }

    // ── a etiqueta: a folha recolhida ──────────────────────────────────
    // Duas áreas lado a lado (e não uma dentro da outra: um TapHandler não
    // para o clique no item de baixo): continuar e dispensar.
    Rectangle {
        id: tag

        width: tagRow.implicitWidth
        height: 38
        x: Math.round((win.width - width) / 2)
        y: win.height - height - 28 + (win.tagOpen ? 0 : 10)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        radius: 2
        opacity: win.tagOpen ? 1 : 0

        Behavior on opacity {
            NumberAnimation { duration: win.tagOpen ? Motion.enter : Motion.exit }
        }
        Behavior on y {
            NumberAnimation {
                duration: win.tagOpen ? Motion.enter : Motion.exit
                easing.type: Easing.BezierSpline
                easing.bezierCurve: win.tagOpen ? Motion.enterCurve : Motion.exitCurve
            }
        }

        Rectangle {
            x: 14
            y: 0
            width: 18
            height: 2
            color: Theme.copper
        }

        Row {
            id: tagRow
            height: parent.height

            Item {
                width: tagText.implicitWidth + 28
                height: parent.height

                Rectangle {
                    anchors.fill: parent
                    anchors.margins: 1
                    color: Theme.wash
                    visible: tagHover.hovered
                }
                Row {
                    id: tagText
                    anchors.centerIn: parent
                    spacing: 10

                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        text: "continuar as boas-vindas"
                        color: Theme.text
                        font.family: Theme.serif
                        font.pixelSize: 14
                        font.italic: true
                    }
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        text: "passo " + (win.step + 1) + " de " + win.steps.length
                        color: Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 9
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: Theme.labelSpacing
                    }
                }
                HoverHandler {
                    id: tagHover
                    cursorShape: Qt.PointingHandCursor
                }
                TapHandler {
                    onTapped: Welcome.resume()
                }
            }

            Rectangle {
                anchors.verticalCenter: parent.verticalCenter
                width: 1
                height: parent.height - 16
                color: Theme.hairline
            }

            Item {
                width: tagDismiss.implicitWidth + 28
                height: parent.height

                Rectangle {
                    anchors.fill: parent
                    anchors.margins: 1
                    color: Theme.wash
                    visible: dismissHover.hovered
                }
                Text {
                    id: tagDismiss
                    anchors.centerIn: parent
                    text: "dispensar"
                    color: dismissHover.hovered ? Theme.text : Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
                HoverHandler {
                    id: dismissHover
                    cursorShape: Qt.PointingHandCursor
                }
                TapHandler {
                    onTapped: Welcome.finish()
                }
            }
        }
    }
}
