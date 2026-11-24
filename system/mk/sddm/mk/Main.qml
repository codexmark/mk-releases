import QtQuick
import SddmComponents 2.0
import "geometria.js" as Geo

// A entrada do Mk, na linguagem do Maré: a folha de desenho técnico (moldura e
// régua), à esquerda o relógio de anéis da tela de bloqueio
// (shell/mare/LockSurface.qml: os minutos e os segundos giram em volta de um
// ponto fora da tela e o número que passa pela pílula acende em cobre) e, à
// direita, a marca, com quem entra e a senha embaixo dela. A marca não aparece
// pronta: a folha a constrói, como no
// logo (system/mk/art/make-logo.py): os eixos, as guias, as diagonais, as
// letras e os nós nas quinas, nessa ordem, em pouco mais de dois segundos.
// Depois fica quase parada: um cursor de cobre anda pela cota com os segundos,
// um nó de cada vez pisca, e cada tecla da senha acende o próximo. Enquanto a
// senha é conferida, uma guia varre a marca.
// Só as cores do Mk (fundo, osso, cobre). motion=nao no theme.conf desliga
// todo o movimento e mostra a marca pronta.
Rectangle {
    id: root

    width: 1920
    height: 1080
    color: ground

    readonly property color ground: "#0b0b0b"
    readonly property color text: "#cdc4ba"
    readonly property color soft: "#b0a9a0"
    readonly property color dim: "#958f87"
    readonly property color faint: "#716b69"
    readonly property color copper: "#ce5e32"
    readonly property color hairline: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.13)
    readonly property color line: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.26)
    readonly property string serif: "Fraunces"
    readonly property string sans: "Space Grotesk"
    readonly property string mono: "Space Mono"
    readonly property string icons: "SpaceMono Nerd Font"

    // Usuários e sessões vêm dos modelos do SDDM; guardamos só os nomes.
    property var users: []
    property int userIndex: 0
    property var sessions: []
    property int sessionIndex: sessionModel.lastIndex
    property bool failed: false
    property bool busy: false

    // O interruptor do movimento (theme.conf). Sem ele, tudo nasce pronto.
    readonly property bool motion: config.motion !== "nao" && config.motion !== "false"
    // A construção da marca, de 0 (folha em branco) a 1 (pronta).
    property real t: motion ? 0 : 1
    readonly property bool built: t >= 1
    // Os segundos, para o cursor da cota.
    property int sec: 0

    // O relógio de anéis, amostrado a cada segundo; os anéis interpolam entre
    // as amostras (com o movimento desligado, pulam de segundo em segundo).
    readonly property real s: Math.max(0.6, height / 1080)
    property int curH: 0
    property date curDate: new Date()
    property real secTarget: 0
    property real minTarget: 0
    // Ao abrir, os anéis chegam girando até a hora certa.
    readonly property real windIn: (1 - seg(0.05, 0.75)) * 48

    function sample() {
        const d = new Date();
        curH = d.getHours();
        curDate = d;
        sec = d.getSeconds();
        const ms = motion ? d.getMilliseconds() + 1000 : 0;
        secTarget = -((d.getSeconds() * 1000 + ms) / 60000) * 360;
        minTarget = -((d.getMinutes() * 60000 + d.getSeconds() * 1000 + ms) / 3600000) * 360;
    }
    // O nó que pisca agora (-1 = nenhum).
    property int ping: -1
    property int pingNext: 0
    // A sessão abrindo: tudo some para o fundo.
    property real leaving: 0

    // Um trecho da construção: 0 antes de `from`, 1 depois de `to`, suave no meio.
    function seg(from, to) {
        const v = Math.max(0, Math.min(1, (t - from) / (to - from)));
        return 1 - Math.pow(1 - v, 3);
    }
    function pingNode() {
        if (!motion || !built)
            return;
        ping = pingNext;
        pingNext = (pingNext + 1) % Geo.nodes.length;
        pingAnim.restart();
    }

    NumberAnimation {
        id: build
        target: root
        property: "t"
        from: 0
        to: 1
        duration: 2400
    }

    readonly property var user: users.length > 0 ? users[userIndex] : ({ name: userModel.lastUser, real: "" })

    function login() {
        if (busy)
            return;
        busy = true;
        failed = false;
        sddm.login(user.name, password.text, sessionIndex);
    }

    Repeater {
        model: userModel
        Item {
            required property string name
            required property string realName
            Component.onCompleted: {
                const list = root.users.concat([{ name: name, real: realName }]);
                root.users = list;
                if (name === userModel.lastUser)
                    root.userIndex = list.length - 1;
            }
        }
    }

    Repeater {
        model: sessionModel
        Item {
            required property string name
            Component.onCompleted: root.sessions = root.sessions.concat([name])
        }
    }

    Connections {
        target: sddm
        function onLoginFailed() {
            root.busy = false;
            root.failed = true;
            password.text = "";
            password.forceActiveFocus();
            shake.restart();
            if (root.motion)
                dip.restart();
        }
        function onLoginSucceeded() {
            root.busy = false;
            if (root.motion)
                leave.restart();
        }
    }

    // ── a folha ────────────────────────────────────────────────────────

    readonly property real frameL: 28
    readonly property real frameR: width - 28
    readonly property real frameT: 28
    readonly property real frameB: height - 28
    // O plano da marca: o quadro das imagens das letras (Geo.vb), à direita.
    readonly property real planeH: Math.min(height * 0.52, width * 0.44 * Geo.vb[3] / Geo.vb[2])
    readonly property real planeW: planeH * Geo.vb[2] / Geo.vb[3]
    readonly property real planeX: Math.min(width * 0.71 - planeW / 2, frameR - 24 - planeW)
    readonly property real planeY: height * 0.45 - planeH * 0.54
    // O eixo e a cota da marca, em pontos da tela: o painel da senha se alinha a eles.
    readonly property real axisDX: (Geo.ax - Geo.vb[0]) / Geo.vb[2] * planeW
    readonly property real cotaW: (Geo.right - Geo.ax) / Geo.vb[2] * planeW
    readonly property real cotaY: planeY + (Geo.ay - Geo.vb[1]) / Geo.vb[3] * planeH

    Canvas {
        id: drawing
        anchors.fill: parent
        onWidthChanged: requestPaint()
        onHeightChanged: requestPaint()

        onPaint: {
            const ctx = getContext("2d");
            ctx.reset();
            const L = root.frameL, R = root.frameR, T = root.frameT, B = root.frameB;
            const bone = a => Qt.rgba(205 / 255, 196 / 255, 186 / 255, a);
            const line = (x1, y1, x2, y2, a, w) => {
                ctx.beginPath();
                ctx.moveTo(x1, y1);
                ctx.lineTo(x2, y2);
                ctx.strokeStyle = bone(a);
                ctx.lineWidth = w || 1;
                ctx.stroke();
            };

            ctx.fillStyle = bone(0.07);
            for (let x = L + 48; x < R; x += 48)
                for (let y = T + 48; y < B; y += 48)
                    ctx.fillRect(x, y, 1, 1);

            ctx.strokeStyle = bone(0.13);
            ctx.lineWidth = 1;
            ctx.strokeRect(L + 0.5, T + 0.5, R - L, B - T);
            const crop = (x, y, dx, dy) => {
                line(x + dx * 6, y, x + dx * 20, y, 0.26);
                line(x, y + dy * 6, x, y + dy * 20, 0.26);
            };
            crop(L - 10, T - 10, -1, -1);
            crop(R + 10, T - 10, 1, -1);
            crop(L - 10, B + 10, -1, 1);
            crop(R + 10, B + 10, 1, 1);

            for (let x = L + 24, i = 1; x < R; x += 24, i++)
                line(x + 0.5, T, x + 0.5, T + (i % 5 === 0 ? 10 : 5), i % 5 === 0 ? 0.26 : 0.13);
            for (let y = T + 24, i = 1; y < B; y += 24, i++)
                line(L, y + 0.5, L + (i % 5 === 0 ? 10 : 5), y + 0.5, i % 5 === 0 ? 0.26 : 0.13);

        }
    }

    // Cabeçalho da folha.
    Row {
        x: root.frameL + 30
        y: root.frameT + 26
        spacing: 12

        Rectangle {
            anchors.verticalCenter: parent.verticalCenter
            width: 6
            height: 6
            radius: 3
            color: root.copper
        }
        Text {
            anchors.verticalCenter: parent.verticalCenter
            text: "MK  ———  ENTRADA"
            color: root.dim
            font.family: root.mono
            font.pixelSize: 11
            font.letterSpacing: 2.4
        }
    }

    // ── o tempo e a entrada ────────────────────────────────────────────

    Timer {
        interval: 1000
        repeat: true // clockTick: o relógio anda mesmo com o movimento desligado
        running: true
        triggeredOnStart: true
        onTriggered: root.sample()
    }

    // Quem entra e a senha, embaixo da marca, na largura da cota.
    Column {
        id: panel
        x: root.planeX + root.axisDX
        y: root.cotaY + 40 * root.s
        width: root.cotaW
        spacing: 0
        opacity: root.seg(0.55, 0.9) * (1 - root.leaving)
        transform: Translate { y: (1 - root.seg(0.55, 0.9)) * 12 }

        // Quem entra. Com mais de um usuário, as setas trocam.
        Row {
            spacing: 14

            Text {
                text: "01"
                color: root.copper
                font.family: root.mono
                font.pixelSize: 11
                anchors.baseline: who.baseline
            }
            Text {
                id: who
                text: root.user.real !== "" ? root.user.real : root.user.name
                color: root.text
                font.family: root.serif
                font.pixelSize: 26
            }
            Text {
                visible: root.users.length > 1
                anchors.baseline: who.baseline
                text: "‹  ›"
                color: root.faint
                font.family: root.mono
                font.pixelSize: 13

                MouseArea {
                    anchors.fill: parent
                    anchors.margins: -8
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        root.userIndex = (root.userIndex + 1) % root.users.length;
                        password.forceActiveFocus();
                    }
                }
            }
        }

        Item { width: 1; height: 10 }

        // A senha: uma linha fina, cursor de cobre, pontos no lugar das letras.
        Item {
            id: field
            width: panel.width
            height: 44

            TextInput {
                id: password
                anchors.left: parent.left
                anchors.right: go.left
                anchors.rightMargin: 12
                anchors.verticalCenter: parent.verticalCenter
                echoMode: TextInput.Password
                passwordCharacter: "•"
                color: root.text
                selectionColor: root.copper
                font.family: root.mono
                font.pixelSize: 18
                font.letterSpacing: 3
                focus: true
                clip: true
                cursorDelegate: Rectangle {
                    width: 2
                    color: root.copper
                    visible: password.activeFocus

                    SequentialAnimation on opacity {
                        loops: Animation.Infinite
                        running: root.motion && password.activeFocus
                        NumberAnimation { to: 0; duration: 520 }
                        NumberAnimation { to: 1; duration: 520 }
                    }
                }
                Keys.onReturnPressed: root.login()
                Keys.onEnterPressed: root.login()
                onTextChanged: {
                    if (text !== "") {
                        root.failed = false;
                        root.pingNode();
                    }
                }

                Text {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: password.text === ""
                    text: root.busy ? "entrando…" : "senha"
                    color: root.faint
                    font.family: root.serif
                    font.pixelSize: 18
                    font.italic: true
                    font.letterSpacing: 0
                }
            }

            Text {
                id: go
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                text: "entrar  →"
                color: password.text !== "" ? root.copper : root.faint
                font.family: root.mono
                font.pixelSize: 12
                font.letterSpacing: 1.2

                MouseArea {
                    anchors.fill: parent
                    anchors.margins: -10
                    cursorShape: Qt.PointingHandCursor
                    onClicked: root.login()
                }
            }

            Rectangle {
                anchors.bottom: parent.bottom
                width: parent.width
                height: 1
                color: root.failed ? root.copper : password.activeFocus ? root.line : root.hairline
            }

            SequentialAnimation {
                id: shake
                NumberAnimation { target: field; property: "x"; to: 10; duration: 60 }
                NumberAnimation { target: field; property: "x"; to: -8; duration: 70 }
                NumberAnimation { target: field; property: "x"; to: 4; duration: 70 }
                NumberAnimation { target: field; property: "x"; to: 0; duration: 80 }
            }
        }

        Text {
            topPadding: 10
            text: root.failed ? "senha errada. tente de novo." : keyboard.capsLock ? "caps lock ligado" : " "
            color: root.failed || keyboard.capsLock ? root.copper : root.faint
            font.family: root.mono
            font.pixelSize: 11
        }
    }

    // ── no alto, à direita: sessão, teclado, energia ───────────────────

    Row {
        anchors.right: parent.right
        anchors.rightMargin: root.width - root.frameR + 30
        y: root.frameT + 20
        spacing: 34
        opacity: root.seg(0.5, 0.9) * (1 - root.leaving)

        Repeater {
            model: [
                { k: "sessão", v: root.sessions.length > 0 ? (root.sessions[root.sessionIndex] || "") : "", act: "session" },
                // A lista do sddm diz "us" mesmo com o X em outro layout; com
                // um layout só, vale o do sistema (theme.conf.user, do instalador).
                { k: "teclado", v: keyboard.layouts.length <= 1 && config.layout ? config.layout
                    : keyboard.layouts.length > 0 ? keyboard.layouts[keyboard.currentLayout].shortName : "", act: "layout" },
                { k: "reiniciar", v: "󰜉", act: "reboot", icon: true },
                { k: "desligar", v: "󰐥", act: "poweroff", icon: true }
            ]

            Column {
                id: cell
                required property var modelData
                visible: modelData.v !== ""
                spacing: 4

                Text {
                    text: cell.modelData.k
                    color: root.faint
                    font.family: root.mono
                    font.pixelSize: 9
                    font.capitalization: Font.AllUppercase
                    font.letterSpacing: 1.6
                }
                Text {
                    text: cell.modelData.v
                    color: cellHover.containsMouse ? root.copper : root.soft
                    font.family: cell.modelData.icon ? root.icons : root.mono
                    font.pixelSize: cell.modelData.icon ? 15 : 13

                    MouseArea {
                        id: cellHover
                        anchors.fill: parent
                        anchors.margins: -8
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: {
                            switch (cell.modelData.act) {
                            case "session":
                                if (root.sessions.length > 0)
                                    root.sessionIndex = (root.sessionIndex + 1) % root.sessions.length;
                                break;
                            case "layout":
                                if (keyboard.layouts.length > 0)
                                    keyboard.currentLayout = (keyboard.currentLayout + 1) % keyboard.layouts.length;
                                break;
                            case "reboot":
                                if (sddm.canReboot)
                                    sddm.reboot();
                                break;
                            case "poweroff":
                                if (sddm.canPowerOff)
                                    sddm.powerOff();
                                break;
                            }
                            password.forceActiveFocus();
                        }
                    }
                }
            }
        }
    }

    // ── o relógio de anéis ─────────────────────────────────────────────

    Item {
        id: stage
        anchors.fill: parent
        opacity: root.seg(0.05, 0.5) * (1 - root.leaving)

        readonly property real cx: 40 * root.s
        readonly property real cy: height / 2
        readonly property real minR: 320 * root.s
        readonly property real secR: 480 * root.s

        // A pílula onde os números acendem.
        Rectangle {
            id: pill
            z: 1
            x: stage.cx + 230 * root.s
            anchors.verticalCenter: parent.verticalCenter
            width: 330 * root.s
            height: 90 * root.s
            radius: height / 2
            color: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.04)
            border.width: 1
            border.color: root.line

            Rectangle {
                x: 170 * root.s
                anchors.verticalCenter: parent.verticalCenter
                width: 1
                height: 35 * root.s
                color: root.hairline
            }
        }

        component Ring: Item {
            id: ring
            property real radius: 300
            property real angle: 0
            property real tickLong: 18
            property real numSize: 22
            property real numInset: 35
            property real extraSpin: 0

            x: stage.cx - radius
            y: stage.cy - radius
            width: radius * 2
            height: width
            z: 10

            Behavior on rotation {
                enabled: root.motion
                RotationAnimation { duration: 1000; direction: RotationAnimation.Shortest; easing.type: Easing.Linear }
            }

            Item {
                anchors.fill: parent
                rotation: -ring.extraSpin

                Repeater {
                    model: 60
                    delegate: Item {
                        required property int index
                        readonly property real base: index * 6
                        readonly property real rel: {
                            let a = (base + ring.angle) % 360;
                            if (a > 180) a -= 360;
                            if (a < -180) a += 360;
                            return a;
                        }
                        // 1 quando o número está na pílula (ângulo 0, à direita).
                        readonly property real spot: Math.max(0, 1 - Math.abs(rel) / 4)
                        readonly property bool major: index % 5 === 0
                        readonly property real t: base * Math.PI / 180

                        Rectangle {
                            x: ring.width / 2 + ring.radius * Math.cos(parent.t) - width / 2
                            y: ring.height / 2 + ring.radius * Math.sin(parent.t) - height / 2
                            width: parent.major ? 2 * root.s : 1 * root.s
                            height: (parent.major ? ring.tickLong : ring.tickLong * 0.55) * root.s
                            rotation: parent.base + 90
                            color: parent.spot > 0 ? root.copper
                                : Qt.rgba(205 / 255, 196 / 255, 186 / 255, parent.major ? 0.3 : 0.14)
                        }
                        Text {
                            visible: parent.major
                            readonly property real r: ring.radius - ring.numInset * root.s
                            x: ring.width / 2 + r * Math.cos(parent.t) - width / 2
                            y: ring.height / 2 + r * Math.sin(parent.t) - height / 2
                            rotation: parent.base
                            text: String(parent.index).padStart(2, "0")
                            color: parent.spot > 0 ? Qt.tint(Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.35), Qt.rgba(206 / 255, 94 / 255, 50 / 255, parent.spot))
                                : Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.28)
                            font.family: root.mono
                            font.pixelSize: Math.round(ring.numSize * root.s)
                            font.bold: parent.spot > 0.5
                        }
                    }
                }
            }
        }

        Ring {
            id: minuteRing
            radius: stage.minR
            rotation: root.minTarget
            angle: rotation - extraSpin
            extraSpin: root.windIn
            tickLong: 18
            numSize: 22
            numInset: 35
        }

        Ring {
            id: secondRing
            radius: stage.secR
            rotation: root.secTarget
            angle: rotation - extraSpin
            extraSpin: root.windIn * 1.6
            tickLong: 13
            numSize: 16
            numInset: 30
        }

        // A hora grande, em serifa.
        Text {
            anchors.right: pill.left
            anchors.rightMargin: 40 * root.s
            anchors.verticalCenter: parent.verticalCenter
            text: String(root.curH).padStart(2, "0")
            color: root.text
            font.family: root.serif
            font.pixelSize: Math.round(120 * root.s)
            font.weight: Font.Light
        }

        // Data: o dia em mono, o dia da semana em serifa itálica.
        Column {
            anchors.left: pill.right
            anchors.leftMargin: 70 * root.s
            anchors.verticalCenter: parent.verticalCenter
            spacing: 6 * root.s

            Text {
                text: {
                    const m = ["jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"];
                    return (String(root.curDate.getDate()).padStart(2, "0") + " " + m[root.curDate.getMonth()] + " " + root.curDate.getFullYear()).toUpperCase();
                }
                color: root.dim
                font.family: root.mono
                font.pixelSize: Math.round(13 * root.s)
                font.letterSpacing: 4 * root.s
            }
            Text {
                text: ["domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"][root.curDate.getDay()]
                color: root.text
                font.family: root.serif
                font.pixelSize: Math.round(26 * root.s)
                font.italic: true
                font.weight: Font.Light
            }
        }
    }

    // ── a marca, construída ────────────────────────────────────────────

    Item {
        id: plane
        x: root.planeX
        y: root.planeY
        width: root.planeW
        height: root.planeH
        opacity: 1 - root.leaving

        // Unidades da fonte -> pontos do plano (y da fonte sobe; o da tela desce).
        function ux(x) { return (x - Geo.vb[0]) / Geo.vb[2] * width; }
        function uy(y) { return (-y - Geo.vb[1]) / Geo.vb[3] * height; }
        function dy(v) { return (v - Geo.vb[1]) / Geo.vb[3] * height; }

        // Eixos, guias e diagonais: riscados conforme root.t avança.
        Canvas {
            id: construction
            anchors.fill: parent
            onWidthChanged: requestPaint()
            onHeightChanged: requestPaint()

            Connections {
                target: root
                function onTChanged() { construction.requestPaint(); }
            }

            onPaint: {
                const ctx = getContext("2d");
                ctx.reset();
                const bone = a => Qt.rgba(205 / 255, 196 / 255, 186 / 255, a);
                const copper = a => Qt.rgba(206 / 255, 94 / 255, 50 / 255, a);
                const X = plane.ux, Y = plane.uy, D = plane.dy;
                // Um traço de (x1,y1) até a fração k do caminho para (x2,y2).
                const stroke = (x1, y1, x2, y2, k, style, w) => {
                    if (k <= 0)
                        return;
                    ctx.beginPath();
                    ctx.moveTo(x1, y1);
                    ctx.lineTo(x1 + (x2 - x1) * k, y1 + (y2 - y1) * k);
                    ctx.strokeStyle = style;
                    ctx.lineWidth = w || 1;
                    ctx.stroke();
                };
                const left = X(Geo.ax), right = X(Geo.right), base = D(Geo.ay), top = D(Geo.top);

                // Guias: as alturas e as hastes, quase invisíveis, uma depois da outra.
                for (let i = 0; i < Geo.heights.length; i++) {
                    const y = Math.round(Y(Geo.heights[i])) + 0.5;
                    stroke(left, y, right, y, root.seg(0.16 + i * 0.03, 0.42 + i * 0.03), bone(0.13));
                }
                for (let i = 0; i < Geo.verts.length; i++) {
                    const x = Math.round(X(Geo.verts[i])) + 0.5;
                    stroke(x, base, x, top, root.seg(0.22 + i * 0.03, 0.5 + i * 0.03), bone(0.13));
                }

                // Diagonais de construção, um pouco além das letras: as do M
                // em osso, o braço e a perna do k em cobre.
                for (let i = 0; i < Geo.diags.length; i++) {
                    const d = Geo.diags[i], ex = (d[2] - d[0]) * 0.35, ey = (d[3] - d[1]) * 0.35;
                    stroke(X(d[0] - ex), Y(d[1] - ey), X(d[2] + ex), Y(d[3] + ey),
                        root.seg(0.38 + i * 0.05, 0.66 + i * 0.05), i < 2 ? bone(0.22) : copper(0.3));
                }

                // Eixos: o vertical sobe, a cota corre para a direita.
                const a = root.seg(0, 0.3);
                stroke(left, base + 0.5, right, base + 0.5, a, bone(0.6), 1.5);
                stroke(left + 0.5, base, left + 0.5, top, a, bone(0.6), 1.5);
                if (a >= 1) {
                    stroke(left - 5, top + 0.5, left + 5, top + 0.5, 1, bone(0.6), 1.5);
                    stroke(right + 0.5, base - 5, right + 0.5, base + 5, 1, bone(0.6), 1.5);
                }
                // A cota sob o k, em cobre.
                stroke(X(Geo.kx - 20), base + 0.5, right, base + 0.5, root.seg(0.74, 1), copper(1), 2.5);
            }
        }

        // As letras: o M assenta de cima, o k chega da direita.
        Image {
            anchors.fill: parent
            source: "letra-m.png"
            smooth: true
            mipmap: true
            opacity: root.seg(0.48, 0.8)
            transform: Translate { y: (root.seg(0.48, 0.8) - 1) * 12 }
        }
        Image {
            id: letterK
            anchors.fill: parent
            source: "letra-k.png"
            smooth: true
            mipmap: true
            property real dim: 1
            opacity: root.seg(0.6, 0.92) * dim
            transform: Translate { x: (1 - root.seg(0.6, 0.92)) * 16 }

            // Senha errada: o cobre apaga e volta.
            SequentialAnimation {
                id: dip
                NumberAnimation { target: letterK; property: "dim"; to: 0.3; duration: 120 }
                NumberAnimation { target: letterK; property: "dim"; to: 1; duration: 520; easing.type: Easing.OutCubic }
            }
        }

        // Nós nas quinas: um furo escuro com aro, na cor da letra.
        Repeater {
            model: Geo.nodes.length

            Item {
                id: node
                required property int index
                readonly property var at: Geo.nodes[index]
                readonly property color ink: at[2] === 1 ? root.copper : root.text
                readonly property real on: root.seg(0.8 + index * 0.02, 0.9 + index * 0.02)
                x: plane.ux(at[0])
                y: plane.uy(at[1])
                opacity: on
                scale: 0.4 + 0.6 * on

                Rectangle {
                    anchors.centerIn: parent
                    width: Math.max(7, plane.width * 0.014)
                    height: width
                    radius: width / 2
                    color: root.ground
                    border.width: 1.5
                    border.color: node.ink
                }
                Rectangle {
                    anchors.centerIn: parent
                    width: 2.5
                    height: 2.5
                    radius: 1.25
                    color: node.ink
                }
            }
        }

        // O nó da vez: um aro que abre e some.
        Rectangle {
            id: ring
            visible: root.ping >= 0
            readonly property var at: Geo.nodes[Math.max(0, root.ping)]
            x: plane.ux(at[0]) - width / 2
            y: plane.uy(at[1]) - height / 2
            width: Math.max(7, plane.width * 0.014) * grow
            height: width
            radius: width / 2
            color: Qt.rgba(206 / 255, 94 / 255, 50 / 255, 0)
            border.width: 1
            border.color: at[2] === 1 ? root.copper : root.text
            property real grow: 1

            ParallelAnimation {
                id: pingAnim
                NumberAnimation { target: ring; property: "grow"; from: 1; to: 4.2; duration: 900; easing.type: Easing.OutCubic }
                NumberAnimation { target: ring; property: "opacity"; from: 0.7; to: 0; duration: 900; easing.type: Easing.OutCubic }
            }
        }
        Timer {
            interval: 4200
            repeat: true
            running: root.motion && root.built && !root.busy && root.leaving === 0
            onTriggered: root.pingNode()
        }

        // O cursor da cota: atravessa o plano uma vez por minuto.
        Item {
            id: marker
            visible: root.built
            opacity: root.sec === 0 ? 0 : 0.9
            x: plane.ux(Geo.ax + (Geo.right - Geo.ax) * root.sec / 60)
            y: plane.dy(Geo.ay)

            Behavior on x {
                enabled: root.motion && root.sec > 0
                NumberAnimation { duration: 1000 }
            }
            Behavior on opacity {
                enabled: root.motion
                NumberAnimation { duration: 300 }
            }

            Rectangle {
                x: -0.5
                y: -7
                width: 1.5
                height: 14
                color: root.copper
            }
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                y: 12
                text: (root.sec < 10 ? "0" : "") + root.sec
                color: root.faint
                font.family: root.mono
                font.pixelSize: 9
                font.letterSpacing: 1
            }
        }

        // Conferindo a senha: uma guia de cobre varre a marca.
        Rectangle {
            id: scan
            visible: root.motion && root.busy
            y: plane.dy(Geo.top)
            width: 1
            height: plane.dy(Geo.ay) - y
            color: Qt.rgba(206 / 255, 94 / 255, 50 / 255, 0.55)

            SequentialAnimation on x {
                loops: Animation.Infinite
                running: root.motion && root.busy
                NumberAnimation { from: plane.ux(Geo.ax); to: plane.ux(Geo.right); duration: 760; easing.type: Easing.InOutSine }
                NumberAnimation { from: plane.ux(Geo.right); to: plane.ux(Geo.ax); duration: 760; easing.type: Easing.InOutSine }
            }
        }
    }

    // A sessão abrindo: a folha recolhe para o fundo.
    NumberAnimation {
        id: leave
        target: root
        property: "leaving"
        to: 1
        duration: 260
        easing.type: Easing.InCubic
    }

    // Grão por cima de tudo: o preto nunca é preto.
    Image {
        anchors.fill: parent
        source: "grain.png"
        fillMode: Image.Tile
        opacity: 0.06
        smooth: false
    }

    Component.onCompleted: {
        password.forceActiveFocus();
        if (motion)
            build.start();
    }
}
