import QtQuick
import Quickshell

// A tela de bloqueio, na linha da skin Orbital (clockwork) do Ryoku, com os
// tokens do Mk. Dois anéis numerados (minutos e segundos) giram em volta de um
// ponto fora da tela à esquerda; o número que passa pela pílula acende em
// cobre. A hora grande em serifa, a data ao lado. Embaixo, à direita, quem
// entra e a senha. Ao desbloquear, os anéis dão corda, a tela treme, saem
// faíscas de cobre e um clarão de osso abre a sessão.
Item {
    id: root

    readonly property real s: Math.max(0.6, height / 1080)
    readonly property color bone: Theme.text
    readonly property color copper: Theme.copper

    // Relógio, amostrado a cada segundo; os anéis interpolam entre amostras.
    property int curH: 0
    property int curM: 0
    property date curDate: new Date()
    property real secTarget: 0
    property real minTarget: 0

    function sample() {
        const d = new Date();
        curH = d.getHours();
        curM = d.getMinutes();
        curDate = d;
        const ms = d.getMilliseconds() + 1000;
        secTarget = -((d.getSeconds() * 1000 + ms) / 60000) * 360;
        minTarget = -((d.getMinutes() * 60000 + d.getSeconds() * 1000 + ms) / 3600000) * 360;
    }

    Component.onCompleted: {
        sample();
        fadeIn.restart();
        input.forceActiveFocus();
    }

    Timer {
        interval: 1000
        repeat: true
        running: true
        onTriggered: root.sample()
    }

    // A corda soma voltas aos anéis: o de segundos gira o dobro.
    readonly property real spin: Lock.windup * 1440
    readonly property real secAngle: secondRing.rotation - spin * 2
    readonly property real minAngle: minuteRing.rotation - spin

    // Tremor durante a corda.
    property real jitterX: 0
    property real jitterY: 0
    Timer {
        interval: 16
        repeat: true
        running: Lock.windup > 0.05 && Lock.boom === 0
        onTriggered: {
            const k = Lock.windup * 26 * root.s;
            root.jitterX = (Math.random() - 0.5) * k;
            root.jitterY = (Math.random() - 0.5) * k;
        }
        onRunningChanged: if (!running) { root.jitterX = 0; root.jitterY = 0; }
    }
    readonly property real sparks: Lock.windup > 0.25 ? (Lock.windup - 0.25) * 1.6 : 0

    property real ui: 0
    NumberAnimation { id: fadeIn; target: root; property: "ui"; from: 0; to: 1; duration: 420; easing.type: Easing.OutCubic }

    // ── a folha: grão, moldura, marcas de corte ────────────────────────

    Canvas {
        anchors.fill: parent
        opacity: root.ui
        onWidthChanged: requestPaint()
        onHeightChanged: requestPaint()
        onPaint: {
            const ctx = getContext("2d");
            ctx.reset();
            const L = 28, T = 28, R = width - 28, B = height - 28;
            const line = (x1, y1, x2, y2, a) => {
                ctx.beginPath();
                ctx.moveTo(x1, y1);
                ctx.lineTo(x2, y2);
                ctx.strokeStyle = Qt.rgba(205 / 255, 196 / 255, 186 / 255, a);
                ctx.lineWidth = 1;
                ctx.stroke();
            };
            ctx.strokeStyle = Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.1);
            ctx.strokeRect(L + 0.5, T + 0.5, R - L, B - T);
            for (const [x, y, dx, dy] of [[L - 10, T - 10, -1, -1], [R + 10, T - 10, 1, -1], [L - 10, B + 10, -1, 1], [R + 10, B + 10, 1, 1]]) {
                line(x + dx * 6, y, x + dx * 20, y, 0.26);
                line(x, y + dy * 6, x, y + dy * 20, 0.26);
            }
            for (let x = L + 24, i = 1; x < R; x += 24, i++)
                line(x + 0.5, T, x + 0.5, T + (i % 5 === 0 ? 10 : 5), i % 5 === 0 ? 0.22 : 0.1);
        }
    }

    // ── o relógio ──────────────────────────────────────────────────────

    Item {
        id: stage
        anchors.fill: parent
        opacity: root.ui
        x: root.jitterX
        y: root.jitterY
        transform: Scale {
            origin.x: 400 * root.s
            origin.y: stage.height / 2
            xScale: Lock.boomScale
            yScale: Lock.boomScale
        }

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
            color: Theme.surfaceLow
            border.width: 1
            border.color: Theme.line

            Rectangle {
                x: 170 * root.s
                anchors.verticalCenter: parent.verticalCenter
                width: 1
                height: 35 * root.s
                color: Theme.hairline
            }
        }

        // Faíscas de cobre na corda.
        Repeater {
            model: 48
            delegate: Rectangle {
                z: 50
                property real a: Math.random() * 6.28
                property real v: 400 * root.s + Math.random() * 900 * root.s
                visible: root.sparks > 0
                x: stage.cx + 400 * root.s + Math.cos(a) * v * root.sparks
                y: stage.cy + Math.sin(a) * v * root.sparks
                width: (1 + Math.random() * 2) * root.s
                height: (1 + 14 * root.sparks) * root.s
                radius: width / 2
                rotation: a * 180 / Math.PI + 90
                color: root.copper
                opacity: root.sparks * (Math.random() > 0.4 ? 1 : 0.25)
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
                            font.family: Theme.mono
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
            angle: root.minAngle
            extraSpin: root.spin
            tickLong: 18
            numSize: 22
            numInset: 35
        }

        Ring {
            id: secondRing
            radius: stage.secR
            rotation: root.secTarget
            angle: root.secAngle
            extraSpin: root.spin * 2
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
            color: root.bone
            font.family: Theme.serif
            font.pixelSize: Math.round(120 * root.s)
            font.weight: Font.Light
        }

        // Data: o dia em mono, o dia da semana em serifa itálica.
        Column {
            anchors.left: pill.right
            anchors.leftMargin: 110 * root.s
            anchors.verticalCenter: parent.verticalCenter
            spacing: 6 * root.s

            Text {
                text: {
                    const m = ["jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"];
                    return (String(root.curDate.getDate()).padStart(2, "0") + " " + m[root.curDate.getMonth()] + " " + root.curDate.getFullYear()).toUpperCase();
                }
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: Math.round(13 * root.s)
                font.letterSpacing: 4 * root.s
            }
            Text {
                text: ["domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"][root.curDate.getDay()]
                color: root.bone
                font.family: Theme.serif
                font.pixelSize: Math.round(26 * root.s)
                font.italic: true
                font.weight: Font.Light
            }
        }
    }

    // ── o que fica por cima: cabeçalho, ações, entrada ─────────────────

    Item {
        anchors.fill: parent
        opacity: root.ui * (Lock.boom > 0 ? 0 : 1)

        Row {
            x: 58
            y: 54
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
                text: "MK  ———  BLOQUEADO"
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 11
                font.letterSpacing: 2.4
            }
        }

        // Ações no alto, à direita: deslizam no hover, com a marca de cobre.
        component Action: Item {
            id: act
            property string label: ""
            signal clicked

            width: actText.implicitWidth + 24
            height: 18

            Text {
                id: actText
                anchors.right: parent.right
                anchors.rightMargin: actMouse.containsMouse ? 14 : 0
                anchors.verticalCenter: parent.verticalCenter
                text: act.label.toUpperCase()
                color: actMouse.containsMouse ? root.bone : Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 10
                font.letterSpacing: 3

                Behavior on anchors.rightMargin { NumberAnimation { duration: 200; easing.type: Easing.OutCubic } }
                Behavior on color { ColorAnimation { duration: 200 } }
            }
            Rectangle {
                anchors.left: actText.right
                anchors.leftMargin: 5
                anchors.verticalCenter: actText.verticalCenter
                width: 5
                height: 5
                color: root.copper
                opacity: actMouse.containsMouse ? 1 : 0

                Behavior on opacity { NumberAnimation { duration: 200 } }
            }
            MouseArea {
                id: actMouse
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: act.clicked()
            }
        }

        Row {
            anchors.right: parent.right
            anchors.rightMargin: 58
            y: 52
            spacing: 18

            Action {
                label: "suspender"
                onClicked: Quickshell.execDetached(["systemctl", "suspend"])
            }
            Action {
                label: "reiniciar"
                onClicked: Quickshell.execDetached(["systemctl", "reboot"])
            }
            Action {
                label: "desligar"
                onClicked: Quickshell.execDetached(["systemctl", "poweroff"])
            }
        }

        // Quem entra e a senha, embaixo à direita.
        Column {
            id: panel
            anchors.right: parent.right
            anchors.rightMargin: 58 + shakeX
            anchors.bottom: parent.bottom
            anchors.bottomMargin: 80 * root.s
            width: 360 * root.s
            spacing: 10 * root.s

            property real shakeX: 0

            Image {
                anchors.right: parent.right
                source: Qt.resolvedUrl("assets/mark.svg")
                sourceSize.height: Math.round(30 * root.s)
                height: Math.round(30 * root.s)
                fillMode: Image.PreserveAspectFit
                mipmap: true
            }

            Text {
                anchors.right: parent.right
                text: Lock.realName !== "" ? Lock.realName : Lock.user
                color: root.bone
                font.family: Theme.serif
                font.pixelSize: Math.round(24 * root.s)
            }

            Item {
                width: parent.width
                height: 38 * root.s

                TextInput {
                    id: input
                    anchors.fill: parent
                    horizontalAlignment: TextInput.AlignRight
                    verticalAlignment: TextInput.AlignVCenter
                    echoMode: TextInput.Password
                    passwordCharacter: "•"
                    color: root.bone
                    font.family: Theme.mono
                    font.pixelSize: Math.round(16 * root.s)
                    font.letterSpacing: 6 * root.s
                    focus: true
                    readOnly: Lock.busy
                    cursorDelegate: Item {}
                    onTextChanged: {
                        Lock.password = text;
                        if (text !== "")
                            Lock.failed = false;
                    }
                    Keys.onReturnPressed: Lock.submit()
                    Keys.onEnterPressed: Lock.submit()
                    Keys.onEscapePressed: Lock.password = ""

                    // Agulha de cobre no lugar do cursor.
                    Rectangle {
                        id: needle
                        x: input.cursorRectangle.x
                        anchors.verticalCenter: parent.verticalCenter
                        width: 2
                        height: 16 * root.s
                        color: root.copper
                        visible: input.activeFocus && !Lock.busy

                        SequentialAnimation on opacity {
                            loops: Animation.Infinite
                            running: needle.visible
                            NumberAnimation { from: 1; to: 0.1; duration: 450 }
                            NumberAnimation { from: 0.1; to: 1; duration: 450 }
                        }
                    }

                    Text {
                        anchors.right: parent.right
                        anchors.rightMargin: 10
                        anchors.verticalCenter: parent.verticalCenter
                        text: Lock.busy ? "conferindo…" : "senha"
                        color: Theme.textFaint
                        font.family: Theme.serif
                        font.pixelSize: Math.round(17 * root.s)
                        font.italic: true
                        opacity: input.text === "" ? 0.8 : 0

                        Behavior on opacity { NumberAnimation { duration: 300 } }
                    }
                }

                Rectangle {
                    anchors.bottom: parent.bottom
                    width: parent.width
                    height: 1
                    color: Lock.failed ? root.copper : input.activeFocus ? Theme.line : Theme.hairline
                }

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.IBeamCursor
                    onClicked: input.forceActiveFocus()
                }
            }

            Item {
                width: parent.width
                height: 18

                Text {
                    anchors.left: parent.left
                    anchors.verticalCenter: parent.verticalCenter
                    text: Lock.failed ? Lock.message : ""
                    color: root.copper
                    font.family: Theme.mono
                    font.pixelSize: 10
                    font.letterSpacing: 1.5
                }

                Action {
                    anchors.right: parent.right
                    label: input.text !== "" ? "desbloquear" : ""
                    visible: input.text !== ""
                    onClicked: Lock.submit()
                }
            }
        }

        SequentialAnimation {
            id: shakeAnim
            NumberAnimation { target: panel; property: "shakeX"; to: 12; duration: 50; easing.type: Easing.InOutSine }
            NumberAnimation { target: panel; property: "shakeX"; to: -12; duration: 60; easing.type: Easing.InOutSine }
            NumberAnimation { target: panel; property: "shakeX"; to: 6; duration: 60; easing.type: Easing.InOutSine }
            NumberAnimation { target: panel; property: "shakeX"; to: 0; duration: 60; easing.type: Easing.InOutSine }
        }
        Connections {
            target: Lock
            // A senha mora no Lock (todas as telas veem a mesma); limpar lá limpa aqui.
            function onPasswordChanged() {
                if (input.text !== Lock.password)
                    input.text = Lock.password;
            }
            function onLockedChanged() {
                if (Lock.locked) {
                    fadeIn.restart();
                    input.forceActiveFocus();
                }
            }
            function onShake() {
                shakeAnim.restart();
                input.forceActiveFocus();
            }
        }
    }

    // Grão por cima de tudo.
    Image {
        anchors.fill: parent
        source: Qt.resolvedUrl("assets/grain.png")
        fillMode: Image.Tile
        opacity: 0.06
        smooth: false
    }

    // O clarão de osso que abre a sessão.
    Rectangle {
        anchors.fill: parent
        color: root.bone
        opacity: Lock.boom
    }
}
