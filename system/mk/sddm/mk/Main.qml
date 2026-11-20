import QtQuick
import SddmComponents 2.0

// A entrada do Mk, na linguagem do Maré: a folha de desenho técnico (moldura,
// régua, arco com o sol de cobre), o relógio grande em serifa à esquerda e,
// embaixo dele, quem entra e a senha. No canto, o carimbo da folha.
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
        }
        function onLoginSucceeded() {
            root.busy = false;
        }
    }

    // ── a folha ────────────────────────────────────────────────────────

    readonly property real frameL: 28
    readonly property real frameR: width - 28
    readonly property real frameT: 28
    readonly property real frameB: height - 28
    readonly property real archR: Math.min(height * 0.31, width * 0.2)
    readonly property real archX: width * 0.7
    readonly property real horizon: height * 0.76

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

            const cx = root.archX, hy = root.horizon, r1 = root.archR;
            const r2 = r1 * 0.8, r3 = r1 * 0.46;
            for (let a = 15; a < 180; a += 15) {
                const t = a * Math.PI / 180;
                line(cx + Math.cos(t) * r3, hy - Math.sin(t) * r3,
                     cx + Math.cos(t) * r1 * 1.18, hy - Math.sin(t) * r1 * 1.18, 0.05);
            }
            ctx.beginPath();
            ctx.arc(cx, hy, r1 * 1.18, 0, Math.PI * 2);
            ctx.strokeStyle = bone(0.06);
            ctx.stroke();
            const arch = (r, a) => {
                ctx.beginPath();
                ctx.arc(cx, hy, r, Math.PI, 0, false);
                ctx.strokeStyle = bone(a);
                ctx.lineWidth = 1;
                ctx.stroke();
            };
            arch(r1, 0.3);
            arch(r2, 0.16);
            arch(r3, 0.13);
            for (const x of [cx - r1, cx - r2, cx + r2, cx + r1])
                line(x + 0.5, hy, x + 0.5, B, x === cx - r1 || x === cx + r1 ? 0.2 : 0.1);
            line(cx - r1 * 1.7, hy + 0.5, R, hy + 0.5, 0.26);

            ctx.beginPath();
            ctx.moveTo(cx - r1 * 1.45, hy - r1 * 0.9);
            ctx.lineTo(cx - r1 * 0.2, hy - r1 * 0.9);
            ctx.strokeStyle = Qt.rgba(206 / 255, 94 / 255, 50 / 255, 0.85);
            ctx.lineWidth = 1.5;
            ctx.stroke();

            ctx.beginPath();
            ctx.arc(cx, hy - r1, 7, 0, Math.PI * 2);
            ctx.fillStyle = root.copper;
            ctx.fill();
            ctx.beginPath();
            ctx.arc(cx, hy - r1, 16, 0, Math.PI * 2);
            ctx.strokeStyle = Qt.rgba(206 / 255, 94 / 255, 50 / 255, 0.35);
            ctx.lineWidth = 1;
            ctx.stroke();
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

    // ── relógio e entrada ──────────────────────────────────────────────

    Timer {
        interval: 1000
        repeat: true
        running: true
        triggeredOnStart: true
        onTriggered: {
            const now = new Date();
            clock.text = Qt.formatTime(now, "HH:mm");
            const days = ["domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"];
            const months = ["janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho",
                "agosto", "setembro", "outubro", "novembro", "dezembro"];
            date.text = days[now.getDay()] + ", " + now.getDate() + " de " + months[now.getMonth()];
        }
    }

    Column {
        id: panel
        x: root.frameL + 30
        y: root.height * 0.2
        width: Math.min(520, root.width * 0.36)
        spacing: 0

        Text {
            id: clock
            color: root.text
            font.family: root.serif
            font.pixelSize: Math.round(root.height * 0.2)
            font.weight: Font.Light
        }
        Text {
            id: date
            topPadding: 2
            color: root.soft
            font.family: root.serif
            font.pixelSize: 22
            font.italic: true
            font.weight: Font.Light
        }

        Item { width: 1; height: root.height * 0.08 }

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

        Item { width: 1; height: 18 }

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
                        running: password.activeFocus
                        NumberAnimation { to: 0; duration: 520 }
                        NumberAnimation { to: 1; duration: 520 }
                    }
                }
                Keys.onReturnPressed: root.login()
                Keys.onEnterPressed: root.login()
                onTextChanged: if (text !== "") root.failed = false

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

    // ── carimbo: sessão, teclado, energia ──────────────────────────────

    Row {
        anchors.right: parent.right
        anchors.rightMargin: root.width - root.frameR + 30
        anchors.bottom: parent.bottom
        anchors.bottomMargin: root.height - root.frameB + 24
        spacing: 34

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

    // O Mk no canto de cima, à direita.
    Image {
        anchors.right: parent.right
        anchors.rightMargin: root.width - root.frameR + 30
        y: root.frameT + 22
        source: "mark.png"
        height: 30
        fillMode: Image.PreserveAspectFit
        smooth: true
        mipmap: true
    }

    // Grão por cima de tudo: o preto nunca é preto.
    Image {
        anchors.fill: parent
        source: "grain.png"
        fillMode: Image.Tile
        opacity: 0.06
        smooth: false
    }

    Component.onCompleted: password.forceActiveFocus()
}
