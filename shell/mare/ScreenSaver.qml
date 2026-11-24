import QtQuick
import Quickshell
import Quickshell.Wayland
import "assets/geometria.js" as Geo

// O descanso de tela: no lugar de dormir, o Mk toma a tela com a própria
// marca. A folha fica preta e a marca é construída como no logo e na tela de
// entrada (eixos, guias, diagonais, as letras, os nós); fica um tempo, some, e
// volta a ser construída noutro canto, para não marcar a tela. Embaixo dela, a
// hora. Liga pelo hypridle (Ajustes › Energia › Descanso de tela) ou por
// `qs ipc call mare saver on`; qualquer tecla, clique ou movimento do mouse
// devolve a tela. Com a tela bloqueada não aparece: o bloqueio tem o desenho
// dele.
PanelWindow {
    id: win

    visible: Mare.saver && !Lock.locked
    color: Theme.ground
    anchors {
        top: true
        bottom: true
        left: true
        right: true
    }
    // A tela inteira, também por cima da ilha (veja Mare.ignoreZone).
    exclusionMode: ExclusionMode.Normal
    exclusiveZone: Mare.ignoreZone
    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "mare-saver"
    WlrLayershell.keyboardFocus: visible ? WlrKeyboardFocus.Exclusive : WlrKeyboardFocus.None

    // A construção da marca, de 0 a 1, e quanto dela aparece (some entre ciclos).
    property real t: 0
    property real shown: 0
    // Cada ciclo põe a marca num lugar diferente da tela.
    property int cycle: 0
    readonly property bool still: Motion.speed === 0
    readonly property var spots: [[0.5, 0.46], [0.27, 0.34], [0.7, 0.6], [0.3, 0.62], [0.68, 0.32], [0.5, 0.56]]
    readonly property var spot: spots[cycle % spots.length]

    function seg(from, to) {
        const v = Math.max(0, Math.min(1, (t - from) / (to - from)));
        return 1 - Math.pow(1 - v, 3);
    }

    onVisibleChanged: {
        if (visible) {
            cycle = Math.floor(Math.random() * spots.length);
            catcher.armed = false;
            keys.forceActiveFocus();
            loop.restart();
        } else {
            loop.stop();
        }
    }

    // Constrói, fica, some, muda de lugar, de novo. Sem movimento (Ajustes ›
    // Movimento em "sem"), a marca aparece pronta e só troca de lugar.
    SequentialAnimation {
        id: loop
        loops: Animation.Infinite
        PropertyAction { target: win; property: "t"; value: win.still ? 1 : 0 }
        PropertyAction { target: win; property: "shown"; value: 1 }
        NumberAnimation { target: win; property: "t"; to: 1; duration: win.still ? 0 : Math.round(4200 * Motion.speed) }
        PauseAnimation { duration: 38000 }
        NumberAnimation { target: win; property: "shown"; to: 0; duration: win.still ? 0 : Math.round(1400 * Motion.speed) }
        PropertyAction { target: win; property: "cycle"; value: win.cycle + 1 }
        PauseAnimation { duration: win.still ? 0 : 600 }
    }

    Item {
        id: keys
        anchors.fill: parent
        focus: true
        Keys.onPressed: event => {
            Mare.saver = false;
            event.accepted = true;
        }
    }

    // O mouse: ao aparecer, a janela recebe a posição do ponteiro sem ele ter
    // se mexido. Só um movimento de verdade, a partir dali, devolve a tela.
    MouseArea {
        id: catcher
        anchors.fill: parent
        hoverEnabled: true
        acceptedButtons: Qt.AllButtons
        cursorShape: Qt.BlankCursor
        property bool armed: false
        property real x0: 0
        property real y0: 0
        onPositionChanged: mouse => {
            if (!armed) {
                armed = true;
                x0 = mouse.x;
                y0 = mouse.y;
            } else if (Math.abs(mouse.x - x0) + Math.abs(mouse.y - y0) > 12) {
                Mare.saver = false;
            }
        }
        onPressed: Mare.saver = false
        onWheel: Mare.saver = false
    }

    // ── a marca ────────────────────────────────────────────────────────

    Item {
        id: plane
        height: Math.min(win.height * 0.44, win.width * 0.4 * Geo.vb[3] / Geo.vb[2])
        width: height * Geo.vb[2] / Geo.vb[3]
        x: Math.max(40, Math.min(win.width - width - 40, win.width * win.spot[0] - width / 2))
        y: Math.max(40, Math.min(win.height - height - 110, win.height * win.spot[1] - height / 2))
        opacity: win.shown

        // Unidades da fonte -> pontos do plano (y da fonte sobe; o da tela desce).
        function ux(x) { return (x - Geo.vb[0]) / Geo.vb[2] * width; }
        function uy(y) { return (-y - Geo.vb[1]) / Geo.vb[3] * height; }
        function dy(v) { return (v - Geo.vb[1]) / Geo.vb[3] * height; }

        Canvas {
            id: construction
            anchors.fill: parent
            onWidthChanged: requestPaint()
            onHeightChanged: requestPaint()

            Connections {
                target: win
                function onTChanged() { construction.requestPaint(); }
            }

            onPaint: {
                const ctx = getContext("2d");
                ctx.reset();
                const bone = a => Qt.rgba(205 / 255, 196 / 255, 186 / 255, a);
                const copper = a => Qt.rgba(206 / 255, 94 / 255, 50 / 255, a);
                const X = plane.ux, Y = plane.uy, D = plane.dy;
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

                for (let i = 0; i < Geo.heights.length; i++) {
                    const y = Math.round(Y(Geo.heights[i])) + 0.5;
                    stroke(left, y, right, y, win.seg(0.16 + i * 0.03, 0.42 + i * 0.03), bone(0.13));
                }
                for (let i = 0; i < Geo.verts.length; i++) {
                    const x = Math.round(X(Geo.verts[i])) + 0.5;
                    stroke(x, base, x, top, win.seg(0.22 + i * 0.03, 0.5 + i * 0.03), bone(0.13));
                }
                for (let i = 0; i < Geo.diags.length; i++) {
                    const d = Geo.diags[i], ex = (d[2] - d[0]) * 0.35, ey = (d[3] - d[1]) * 0.35;
                    stroke(X(d[0] - ex), Y(d[1] - ey), X(d[2] + ex), Y(d[3] + ey),
                        win.seg(0.38 + i * 0.05, 0.66 + i * 0.05), i < 2 ? bone(0.22) : copper(0.3));
                }
                const a = win.seg(0, 0.3);
                stroke(left, base + 0.5, right, base + 0.5, a, bone(0.6), 1.5);
                stroke(left + 0.5, base, left + 0.5, top, a, bone(0.6), 1.5);
                if (a >= 1) {
                    stroke(left - 5, top + 0.5, left + 5, top + 0.5, 1, bone(0.6), 1.5);
                    stroke(right + 0.5, base - 5, right + 0.5, base + 5, 1, bone(0.6), 1.5);
                }
                stroke(X(Geo.kx - 20), base + 0.5, right, base + 0.5, win.seg(0.74, 1), copper(1), 2.5);
            }
        }

        Image {
            anchors.fill: parent
            source: Qt.resolvedUrl("assets/letra-m.png")
            smooth: true
            mipmap: true
            opacity: win.seg(0.48, 0.8)
            transform: Translate { y: (win.seg(0.48, 0.8) - 1) * 12 }
        }
        Image {
            anchors.fill: parent
            source: Qt.resolvedUrl("assets/letra-k.png")
            smooth: true
            mipmap: true
            opacity: win.seg(0.6, 0.92)
            transform: Translate { x: (1 - win.seg(0.6, 0.92)) * 16 }
        }

        // Nós nas quinas: um furo escuro com aro, na cor da letra.
        Repeater {
            model: Geo.nodes.length

            Item {
                id: node
                required property int index
                readonly property var at: Geo.nodes[index]
                readonly property color ink: at[2] === 1 ? Theme.copper : Theme.text
                readonly property real on: win.seg(0.8 + index * 0.02, 0.9 + index * 0.02)
                x: plane.ux(at[0])
                y: plane.uy(at[1])
                opacity: on
                scale: 0.4 + 0.6 * on

                Rectangle {
                    anchors.centerIn: parent
                    width: Math.max(7, plane.width * 0.014)
                    height: width
                    radius: width / 2
                    color: Theme.ground
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

        // A hora, embaixo da cota, alinhada ao eixo.
        Row {
            x: plane.ux(Geo.ax)
            y: plane.dy(Geo.ay) + 22
            spacing: 16
            opacity: win.seg(0.7, 1)

            Text {
                id: hour
                text: Qt.formatTime(clock.date, "HH:mm")
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: Math.max(22, Math.round(plane.height * 0.12))
                font.weight: Font.Light
            }
            Text {
                anchors.baseline: hour.baseline
                text: ["domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"][clock.date.getDay()]
                color: Theme.textDim
                font.family: Theme.serif
                font.pixelSize: Math.max(13, Math.round(plane.height * 0.045))
                font.italic: true
                font.weight: Font.Light
            }
        }
    }

    SystemClock {
        id: clock
        precision: SystemClock.Minutes
    }

    // Grão por cima: o preto nunca é preto.
    Image {
        anchors.fill: parent
        source: Qt.resolvedUrl("assets/grain.png")
        fillMode: Image.Tile
        opacity: Settings.get("backdrop.grain")
        smooth: false
    }
}
