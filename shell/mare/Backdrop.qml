import QtQuick
import Quickshell
import Quickshell.Wayland

// O fundo, desenhado pelo próprio Maré na camada de baixo: uma folha de
// desenho técnico na linguagem do perfil. Moldura com marcas de corte e régua,
// grade de pontos, um arco arquitetônico com colunas e linhas de construção,
// o sol de cobre na chave do arco, a letra do workspace em serifa e o carimbo
// da folha no canto. Num workspace vazio, a folha convida a começar. Com papel
// de parede escolhido (Super+W), a imagem entra por cima, escurecida.
PanelWindow {
    id: backdrop

    anchors {
        top: true
        bottom: true
        left: true
        right: true
    }
    exclusionMode: ExclusionMode.Ignore
    color: Theme.ground
    WlrLayershell.layer: WlrLayer.Background
    WlrLayershell.namespace: "mare-backdrop"

    readonly property int wsId: Projects.focusedId
    readonly property bool empty: Projects.windowCount === 0
    readonly property string letter: Theme.wsName(wsId)

    // Moldura da folha (a ilha ocupa o topo).
    readonly property real frameL: 28
    readonly property real frameR: width - 28
    readonly property real frameT: 58
    readonly property real frameB: height - 28

    // Arco: centro na linha do horizonte, à direita.
    readonly property real archR: Math.min(height * 0.31, width * 0.2)
    readonly property real archX: width * 0.7
    readonly property real horizon: height * 0.76

    SystemClock {
        id: clock
        precision: SystemClock.Hours
    }

    Canvas {
        id: drawing
        anchors.fill: parent
        onWidthChanged: requestPaint()
        onHeightChanged: requestPaint()

        onPaint: {
            const ctx = getContext("2d");
            ctx.reset();
            const L = backdrop.frameL, R = backdrop.frameR, T = backdrop.frameT, B = backdrop.frameB;
            const bone = (a) => Qt.rgba(205 / 255, 196 / 255, 186 / 255, a);
            const line = (x1, y1, x2, y2, a, w) => {
                ctx.beginPath();
                ctx.moveTo(x1, y1);
                ctx.lineTo(x2, y2);
                ctx.strokeStyle = bone(a);
                ctx.lineWidth = w || 1;
                ctx.stroke();
            };

            // Grade de pontos, bem leve.
            ctx.fillStyle = bone(0.07);
            for (let x = L + 48; x < R; x += 48)
                for (let y = T + 48; y < B; y += 48)
                    ctx.fillRect(x, y, 1, 1);

            // Moldura e marcas de corte nos cantos.
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

            // Régua: um traço a cada 24 px, maior a cada 120.
            for (let x = L + 24, i = 1; x < R; x += 24, i++)
                line(x + 0.5, T, x + 0.5, T + (i % 5 === 0 ? 10 : 5), i % 5 === 0 ? 0.26 : 0.13);
            for (let y = T + 24, i = 1; y < B; y += 24, i++)
                line(L, y + 0.5, L + (i % 5 === 0 ? 10 : 5), y + 0.5, i % 5 === 0 ? 0.26 : 0.13);

            const cx = backdrop.archX, hy = backdrop.horizon, r1 = backdrop.archR;
            const r2 = r1 * 0.8, r3 = r1 * 0.46;

            // Linhas de construção: raios do centro do arco, a cada 15°.
            for (let a = 15; a < 180; a += 15) {
                const t = a * Math.PI / 180;
                line(cx + Math.cos(t) * r3, hy - Math.sin(t) * r3,
                     cx + Math.cos(t) * r1 * 1.18, hy - Math.sin(t) * r1 * 1.18, 0.05);
            }
            // Círculo de construção completo, por trás de tudo.
            ctx.beginPath();
            ctx.arc(cx, hy, r1 * 1.18, 0, Math.PI * 2);
            ctx.strokeStyle = bone(0.06);
            ctx.stroke();

            // Arcos: o externo mais marcado, os internos finos.
            const arch = (r, a, w) => {
                ctx.beginPath();
                ctx.arc(cx, hy, r, Math.PI, 0, false);
                ctx.strokeStyle = bone(a);
                ctx.lineWidth = w;
                ctx.stroke();
            };
            arch(r1, 0.3, 1);
            arch(r2, 0.16, 1);
            arch(r3, 0.13, 1);

            // Colunas: descem dos pés dos arcos até a moldura.
            for (const x of [cx - r1, cx - r2, cx + r2, cx + r1])
                line(x + 0.5, hy, x + 0.5, B, x === cx - r1 || x === cx + r1 ? 0.2 : 0.1);

            // Horizonte, de lado a lado do arco e além.
            line(cx - r1 * 1.7, hy + 0.5, R, hy + 0.5, 0.26);

            // Linha de cobre cruzando o alto do arco, como no perfil.
            ctx.beginPath();
            ctx.moveTo(cx - r1 * 1.45, hy - r1 * 0.9);
            ctx.lineTo(cx - r1 * 0.2, hy - r1 * 0.9);
            ctx.strokeStyle = Qt.rgba(206 / 255, 94 / 255, 50 / 255, 0.85);
            ctx.lineWidth = 1.5;
            ctx.stroke();

            // Sol de cobre na chave do arco.
            ctx.beginPath();
            ctx.arc(cx, hy - r1, 7, 0, Math.PI * 2);
            ctx.fillStyle = Theme.copper;
            ctx.fill();
            ctx.beginPath();
            ctx.arc(cx, hy - r1, 16, 0, Math.PI * 2);
            ctx.strokeStyle = Qt.rgba(206 / 255, 94 / 255, 50 / 255, 0.35);
            ctx.stroke();

            // Cotas: uma medida do vão do arco, como num desenho técnico.
            const dy = hy + 22;
            line(cx - r1, dy, cx + r1, dy, 0.13);
            line(cx - r1, dy - 4, cx - r1, dy + 4, 0.26);
            line(cx + r1, dy - 4, cx + r1, dy + 4, 0.26);
        }
    }

    // Cota do vão e coordenadas de Porto Velho, junto ao horizonte.
    Text {
        x: backdrop.archX - implicitWidth / 2
        y: backdrop.horizon + 28
        text: "vão " + Math.round(backdrop.archR * 2)
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 10
        font.letterSpacing: 2
        font.capitalization: Font.AllUppercase
    }

    Text {
        x: backdrop.frameR - implicitWidth - 18
        y: backdrop.horizon - implicitHeight - 8
        text: "8°45′ s · 63°54′ o"
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 10
        font.letterSpacing: 2
        font.capitalization: Font.AllUppercase
    }

    // Cabeçalho da folha.
    Row {
        x: backdrop.frameL + 36
        y: backdrop.frameT + 30
        spacing: 14

        Rectangle {
            anchors.verticalCenter: parent.verticalCenter
            width: 7
            height: 7
            radius: 4
            color: Theme.copper
        }

        Text {
            anchors.verticalCenter: parent.verticalCenter
            text: "Mk"
            color: Theme.text
            font.family: Theme.mono
            font.pixelSize: 11
            font.capitalization: Font.AllUppercase
            font.letterSpacing: 2.4
        }

        Rectangle {
            anchors.verticalCenter: parent.verticalCenter
            width: 120
            height: 1
            color: Theme.line
        }

        Text {
            anchors.verticalCenter: parent.verticalCenter
            // Só a parte latina em maiúsculas: a maiúscula de α é Α, que vira "A".
            text: "FOLHA " + backdrop.letter
                + (Projects.current && !Projects.free ? " · " + Projects.name.toUpperCase() : "")
            color: Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 11
            font.letterSpacing: 2.4
        }
    }

    // A letra do workspace, grande, em serifa; o índice em cobre ao lado.
    Row {
        id: bigLetter
        x: backdrop.frameL + 30
        y: backdrop.frameT + backdrop.height * 0.08
        spacing: 14
        visible: Settings.get("backdrop.letter")
        opacity: backdrop.empty ? 1 : 0.4

        Behavior on opacity {
            NumberAnimation { duration: Motion.slow }
        }

        Text {
            text: backdrop.letter
            color: Theme.text
            font.family: Theme.serif
            font.pixelSize: backdrop.height * 0.3
            font.weight: Font.Light
        }

        Text {
            y: parent.height * 0.4
            text: String(backdrop.wsId).padStart(2, "0")
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 12
            font.letterSpacing: 2
        }
    }

    // Convite, só num workspace vazio.
    Column {
        x: backdrop.frameL + 36
        y: bigLetter.y + bigLetter.height + 4
        width: Math.min(520, backdrop.width * 0.36)
        spacing: 0
        opacity: backdrop.empty ? 1 : 0
        visible: opacity > 0 && Settings.get("backdrop.letter")

        Behavior on opacity {
            NumberAnimation { duration: Motion.slow }
        }

        Text {
            text: "o que vai acontecer aqui?"
            color: Theme.textSoft
            font.family: Theme.serif
            font.pixelSize: 24
            font.italic: true
            font.weight: Font.Light
            bottomPadding: 24
        }

        Repeater {
            model: [
                { what: "abrir", key: "toque no super" },
                { what: "terminal aqui", key: "super + enter" },
                { what: "papel de parede", key: "super + w" }
            ]

            Item {
                id: hint
                required property var modelData
                required property int index
                width: parent.width
                height: 40

                Rectangle { width: parent.width; height: 1; color: Theme.hairline }
                Rectangle {
                    visible: hint.index === 2
                    y: parent.height - 1
                    width: parent.width
                    height: 1
                    color: Theme.hairline
                }

                Text {
                    id: what
                    anchors.verticalCenter: parent.verticalCenter
                    text: hint.modelData.what
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                    font.capitalization: Font.AllUppercase
                    font.letterSpacing: 2
                }

                Rectangle {
                    anchors.verticalCenter: parent.verticalCenter
                    x: what.implicitWidth + 16
                    width: parent.width - what.implicitWidth - key.implicitWidth - 32
                    height: 1
                    color: Theme.hairline
                }

                Text {
                    id: key
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.right: parent.right
                    text: hint.modelData.key
                    color: Theme.text
                    font.family: Theme.mono
                    font.pixelSize: 11
                    font.capitalization: Font.AllUppercase
                    font.letterSpacing: 2
                }
            }
        }
    }

    // Carimbo da folha, no canto de baixo à direita.
    Rectangle {
        id: stamp
        x: backdrop.frameR - width
        y: backdrop.frameB - height
        width: 300
        height: 96
        color: Theme.ground
        border.width: 1
        border.color: Theme.line

        Rectangle { width: 2; height: parent.height; color: Theme.copper }
        Rectangle { x: 0; y: parent.height / 2; width: parent.width; height: 1; color: Theme.hairline }
        Rectangle { x: parent.width * 0.55; y: 0; width: 1; height: parent.height; color: Theme.hairline }

        Grid {
            anchors.fill: parent
            columns: 2

            Repeater {
                model: [
                    { k: "projeto", v: (Projects.current && !Projects.free ? Projects.name : "livre").toUpperCase() },
                    { k: "folha", v: backdrop.letter + " · " + String(backdrop.wsId).padStart(2, "0") },
                    { k: "Mk", v: "PORTO VELHO · RO" },
                    { k: "data", v: Qt.formatDateTime(clock.date, "dd.MM.yyyy") }
                ]

                Item {
                    required property var modelData
                    required property int index
                    width: index % 2 === 0 ? stamp.width * 0.55 : stamp.width * 0.45
                    height: stamp.height / 2

                    Column {
                        anchors.verticalCenter: parent.verticalCenter
                        x: 16
                        spacing: 4

                        Text {
                            text: parent.parent.modelData.k
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 9
                            font.capitalization: Font.AllUppercase
                            font.letterSpacing: 1.8
                        }

                        Text {
                            text: parent.parent.modelData.v
                            color: Theme.text
                            font.family: Theme.mono
                            font.pixelSize: 11
                            font.letterSpacing: 1.4
                        }
                    }
                }
            }
        }
    }

    // Papel de parede escolhido: entra por cima da folha, escurecido.
    Image {
        id: shown
        anchors.fill: parent
        fillMode: Image.PreserveAspectCrop
        asynchronous: true
        sourceSize: Qt.size(backdrop.width, backdrop.height)
        visible: Wallpaper.current !== ""
    }

    Image {
        id: incoming
        anchors.fill: parent
        fillMode: Image.PreserveAspectCrop
        asynchronous: true
        sourceSize: Qt.size(backdrop.width, backdrop.height)
        opacity: 0
        source: Wallpaper.current !== "" ? "file://" + Wallpaper.current : ""

        // A imagem nova carrega por baixo e entra por cima da antiga.
        onStatusChanged: {
            if (status === Image.Ready)
                crossfade.restart();
        }
    }

    Rectangle {
        anchors.fill: parent
        visible: Wallpaper.current !== ""
        color: Theme.ground
        opacity: Settings.get("backdrop.dim")
    }

    SequentialAnimation {
        id: crossfade

        NumberAnimation {
            target: incoming
            property: "opacity"
            from: 0
            to: 1
            duration: 700
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Motion.curve
        }
        ScriptAction {
            script: {
                shown.source = incoming.source;
                incoming.opacity = 0;
            }
        }
    }

    // Grão por cima de tudo: o preto nunca é preto.
    Image {
        anchors.fill: parent
        source: Qt.resolvedUrl("assets/grain.png")
        fillMode: Image.Tile
        opacity: Settings.get("backdrop.grain")
        smooth: false
    }
}
