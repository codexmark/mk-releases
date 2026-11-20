import QtQuick
import QtQuick.Shapes
import Quickshell
import Quickshell.Wayland

// O painel que desce da ilha, como os cards da ilha do Ryoku (orpheus): sai
// soldado à borda de baixo dela (cobre a linha da emenda),
// com filetes côncavos na junção, embaixo do item clicado. Abre rápido e
// assenta, fecha acelerando, e desliza entre itens passando um pouco do ponto.
PanelWindow {
    id: tide

    property bool active: true
    // O painel do projeto só existe num workspace ligado a um projeto.
    readonly property string mode: {
        const m = active ? Mare.mode : "rest";
        if (m === "settings")
            return "rest";
        return m === "project" && (!Projects.current || Projects.free) ? "rest" : m;
    }
    readonly property bool open: mode !== "rest"

    readonly property int barH: 34
    // Sobreposição com a ilha: só cobre a linha fina da emenda. (No Ryoku a
    // ilha e o painel são uma forma só; aqui são duas camadas, e uma sobra
    // maior cobriria a data do relógio.)
    readonly property int weld: 1
    readonly property int fillet: 12
    readonly property int corner: 14
    readonly property int edge: 8
    // Trecho reto da borda de baixo da ilha: depois do ombro (12) e do canto (12).
    readonly property int islandInset: 24

    anchors {
        top: true
        left: true
        right: true
    }
    // Por cima da ilha, colado nela (veja Mare.ignoreZone).
    exclusionMode: ExclusionMode.Normal
    exclusiveZone: Mare.ignoreZone
    implicitHeight: 720
    color: "transparent"
    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "mare-tide"
    // Launcher, Alt+Tab e área de transferência precisam do teclado. OnDemand
    // (e não Exclusive): quem entrega o teclado é o HyprlandFocusGrab do
    // Monitor.qml; Exclusive brigava com ele e o grab fechava o painel na hora.
    WlrLayershell.keyboardFocus: tide.active && Mare.wantsKeyboard
        ? WlrKeyboardFocus.OnDemand : WlrKeyboardFocus.None

    // Só o painel recebe cliques; o resto da janela é transparente ao mouse.
    mask: Region {
        item: hit
    }

    // ── o painel como líquido ────────────────────────────────────────────
    // A ilha escorre para o painel. Em vez de um retângulo que desliza, a
    // forma tem partes que se movem em tempos diferentes, cada uma numa mola:
    //   pescoço (onde ela sai da ilha): rápido, quase sem balanço;
    //   corpo (os lados lá embaixo):    mais mole, passa do ponto e volta;
    //   fundo: o meio desce antes dos cantos, como uma gota, e balança.
    // Ao abrir, nasce como um fio embaixo do item clicado e engorda. Ao trocar
    // de item, o pescoço vai na frente e o corpo é puxado atrás, inclinado.
    // Ao fechar, é sugada de volta para um ponto. Com o movimento desligado
    // nos Ajustes, tudo vai direto para o lugar.
    Item {
        id: sheet

        // Largura do conteúdo + os filetes dos dois lados.
        readonly property real targetW: content.item ? content.item.implicitWidth + 2 * tide.fillet : 0
        readonly property real targetH: tide.open && content.item ? content.item.implicitHeight + tide.weld : 0
        // Embaixo do item clicado (centro em Mare.originX; aberto por atalho,
        // no meio), mas sempre dentro da borda de baixo da ilha: o corpo do
        // painel não passa dos cantos arredondados dela.
        readonly property real targetX: {
            const center = Mare.originX >= 0 ? Mare.originX : tide.width / 2;
            const lo = Mare.islandW > 0 ? Mare.islandX + tide.islandInset : tide.edge;
            const hi = Mare.islandW > 0 ? Mare.islandX + Mare.islandW - tide.islandInset - targetW : tide.width - targetW - tide.edge;
            return Math.max(lo, Math.min(Math.max(lo, hi), center - targetW / 2));
        }

        Binding { target: Mare; property: "sheetW"; value: tide.open ? sheet.targetW : 0 }

        // Para onde o corpo vai: os lados de dentro (sem os filetes).
        readonly property real half: Math.max(0, targetW / 2 - tide.fillet)
        readonly property real cx: targetX + targetW / 2
        // `pin` junta tudo num ponto e `snap` desliga as molas: é como o
        // painel nasce num fio embaixo do item, e não voando de onde fechou.
        property bool pin: false
        property bool snap: false
        // Recolhido (fechado ou ainda carregando), os lados ficam onde estão.
        property real lastL: 0
        property real lastR: 0
        readonly property real goL: pin ? cx : ready ? cx - half : lastL
        readonly property real goR: pin ? cx : ready ? cx + half : lastR
        onGoLChanged: if (ready || pin) lastL = goL
        onGoRChanged: if (ready || pin) lastR = goR

        readonly property bool moving: Motion.speed > 0 && !snap
        // Velocidade dos Ajustes: mola mais dura = mais rápido.
        readonly property real k: Motion.speed > 0 ? 1 / (Motion.speed * Motion.speed) : 1

        property real neckL: goL
        property real neckR: goR
        property real bodyL: goL
        property real bodyR: goR
        // A altura em duas molas, uma mais rápida que a outra. O meio do
        // fundo é sempre a mais baixa das duas e os cantos, a mais alta:
        // crescendo, o meio desce na frente; encolhendo (ao fechar ou ao
        // trocar para um painel menor), os cantos sobem primeiro e o meio
        // fica pendurado. Nos dois casos o fundo é uma gota, nunca um arco.
        property real fastH: ready ? targetH : 0
        property real slowH: ready ? targetH : 0
        readonly property real edgeH: Math.min(fastH, slowH)   // os cantos de baixo
        readonly property real midH: Math.max(fastH, slowH)    // o meio do fundo

        Behavior on neckL { enabled: sheet.moving; SpringAnimation { spring: 7.5 * sheet.k; damping: 0.62; epsilon: 0.2 } }
        Behavior on neckR { enabled: sheet.moving; SpringAnimation { spring: 7.5 * sheet.k; damping: 0.62; epsilon: 0.2 } }
        // O corpo alarga quase junto com a descida: se ficasse muito atrás,
        // a altura chegava ao fim com a largura ainda no começo, e o painel
        // virava um fio comprido antes de engordar.
        Behavior on bodyL { enabled: sheet.moving; SpringAnimation { spring: 4.6 * sheet.k; damping: 0.36; epsilon: 0.2 } }
        Behavior on bodyR { enabled: sheet.moving; SpringAnimation { spring: 4.6 * sheet.k; damping: 0.36; epsilon: 0.2 } }
        Behavior on fastH { enabled: sheet.moving; SpringAnimation { spring: 4.2 * sheet.k; damping: 0.36; epsilon: 0.2 } }
        Behavior on slowH { enabled: sheet.moving; SpringAnimation { spring: 3.0 * sheet.k; damping: 0.44; epsilon: 0.2 } }

        // O conteúdo do painel demora um instante para carregar, e até lá a
        // largura-alvo é 0: só há o que desenhar quando ele tem tamanho.
        readonly property bool loaded: tide.open && targetW > 2 * tide.fillet && targetH > tide.weld
        // Numa máquina lenta, carregar a busca (a lista de apps e os ícones)
        // trava a tela por meio segundo. Se o painel começasse a escorrer
        // antes, ficava congelado no primeiro quadro: um fio pendurado na
        // ilha. Abrindo do repouso, ele só começa quando os quadros voltam a
        // fluir (ou depois de meio segundo, de qualquer jeito). Trocando de
        // painel não espera: o painel já está na tela.
        property bool ready: false
        onLoadedChanged: {
            if (!loaded)
                ready = false;
            else if (slowH >= 1 || fastH >= 1 || Motion.speed <= 0)
                ready = true;
        }

        FrameAnimation {
            property int frames: 0
            running: sheet.loaded && !sheet.ready
            onRunningChanged: {
                if (running) {
                    frames = 0;
                    reset();
                }
            }
            onTriggered: {
                frames += 1;
                if ((frames >= 3 && frameTime < 0.05) || elapsedTime > 0.5)
                    sheet.ready = true;
            }
        }

        // Abrindo com o painel recolhido: nasce num ponto embaixo do item (sem
        // voar de onde fechou da última vez) e escorre a partir dali.
        onReadyChanged: {
            if (ready && slowH < 1 && fastH < 1) {
                snap = true;
                pin = true;
                snap = false;
                pin = false;
            }
        }

        // A forma, já com as medidas seguras para desenhar. Se os lados do
        // corpo se cruzarem num balanço, viram um ponto no meio.
        readonly property real bl: Math.min(bodyL, (bodyL + bodyR) / 2)
        readonly property real br: Math.max(bodyR, (bodyL + bodyR) / 2)
        // Quanto do corpo já abriu (0 a 1). A altura desenhada acompanha: um
        // corpo estreito é uma gota curta, nunca um fio até lá embaixo.
        readonly property real grown: half > 0 ? Math.max(0, Math.min(1, (br - bl) / (2 * half))) : 1
        // Começa do zero: com o corpo ainda sem largura não se desenha nada.
        // (Com um mínimo de altura, o primeiro quadro era um fio; e é nesse
        // quadro que uma máquina lenta trava, desenhando o conteúdo pela
        // primeira vez. O fio ficava parado na tela.)
        readonly property real reach: Math.sqrt(grown)
        readonly property real h: Math.max(0, edgeH) * reach
        readonly property real hm: Math.max(0, midH) * reach
        readonly property real fe: Math.min(tide.fillet, h / 2)
        readonly property real re: Math.min(tide.corner, Math.max(0, (h - fe) / 2), (br - bl) / 2)
        readonly property real sideY: fe + (h - re - fe) / 2

        y: tide.barH - tide.weld
        width: tide.width
        height: tide.height - y
        visible: h > 0.5 || hm > 0.5

        // Onde o painel pousa: é o que recebe cliques (a máscara da janela).
        Item {
            id: hit
            x: sheet.targetX
            width: sheet.targetW
            height: sheet.targetH
        }

        Shape {
            id: shape
            anchors.fill: parent
            preferredRendererType: Shape.CurveRenderer

            // O corpo, preenchido.
            ShapePath {
                fillColor: Theme.surface
                strokeColor: "transparent"
                startX: sheet.neckL - sheet.fe
                startY: 0
                PathQuad { x: sheet.neckL; y: sheet.fe; controlX: sheet.neckL; controlY: 0 }
                PathCubic { x: sheet.bl; y: sheet.h - sheet.re; control1X: sheet.neckL; control1Y: sheet.sideY; control2X: sheet.bl; control2Y: sheet.sideY }
                PathQuad { x: sheet.bl + sheet.re; y: sheet.h; controlX: sheet.bl; controlY: sheet.h }
                PathQuad { x: sheet.br - sheet.re; y: sheet.h; controlX: (sheet.bl + sheet.br) / 2; controlY: 2 * sheet.hm - sheet.h }
                PathQuad { x: sheet.br; y: sheet.h - sheet.re; controlX: sheet.br; controlY: sheet.h }
                PathCubic { x: sheet.neckR; y: sheet.fe; control1X: sheet.br; control1Y: sheet.sideY; control2X: sheet.neckR; control2Y: sheet.sideY }
                PathQuad { x: sheet.neckR + sheet.fe; y: 0; controlX: sheet.neckR; controlY: 0 }
            }

            // A linha fina nos lados e embaixo (em cima fica a ilha).
            ShapePath {
                fillColor: "transparent"
                strokeColor: Theme.hairline
                strokeWidth: 1
                startX: sheet.neckL - sheet.fe
                startY: 0
                PathQuad { x: sheet.neckL; y: sheet.fe; controlX: sheet.neckL; controlY: 0 }
                PathCubic { x: sheet.bl; y: sheet.h - sheet.re; control1X: sheet.neckL; control1Y: sheet.sideY; control2X: sheet.bl; control2Y: sheet.sideY }
                PathQuad { x: sheet.bl + sheet.re; y: sheet.h; controlX: sheet.bl; controlY: sheet.h }
                PathQuad { x: sheet.br - sheet.re; y: sheet.h; controlX: (sheet.bl + sheet.br) / 2; controlY: 2 * sheet.hm - sheet.h }
                PathQuad { x: sheet.br; y: sheet.h - sheet.re; controlX: sheet.br; controlY: sheet.h }
                PathCubic { x: sheet.neckR; y: sheet.fe; control1X: sheet.br; control1Y: sheet.sideY; control2X: sheet.neckR; control2Y: sheet.sideY }
                PathQuad { x: sheet.neckR + sheet.fe; y: 0; controlX: sheet.neckR; controlY: 0 }
            }
        }

        // O conteúdo anda com o corpo e é cortado por ele.
        Item {
            id: body
            x: sheet.bl
            width: sheet.br - sheet.bl
            height: Math.min(sheet.h, sheet.hm)
            clip: true

            Image {
                anchors.fill: parent
                source: Qt.resolvedUrl("assets/grain.png")
                fillMode: Image.Tile
                opacity: 0.035
            }

            // Marca de cobre no canto, como os destaques das folhas do perfil.
            Rectangle {
                x: 18
                y: tide.weld + 6
                width: 28
                height: 2
                color: Theme.copper
                z: 2
            }

            Loader {
                id: content

                y: tide.weld
                focus: true
                sourceComponent: {
                    switch (tide.mode) {
                    case "project": return projectPanel;
                    case "volume": return volumePanel;
                    case "power": return powerPanel;
                    case "network": return networkPanel;
                    case "calendar": return calendarPanel;
                    case "picker": return pickerPanel;
                    case "launcher": return launcherPanel;
                    case "switcher": return switcherPanel;
                    case "notifications": return notificationsPanel;
                    case "session": return sessionPanel;
                    case "toast": return toastView;
                    case "osd": return osdView;
                    case "media": return mediaPanel;
                    case "bluetooth": return bluetoothPanel;
                    case "display": return displayPanel;
                    case "system": return systemPanel;
                    case "clipboard": return clipboardPanel;
                    case "wallpaper": return wallpaperPanel;
                    default: return null;
                    }
                }
                onLoaded: {
                    item.opacity = 0;
                    fadeIn.restart();
                }
            }

            SequentialAnimation {
                id: fadeIn

                PauseAnimation { duration: Motion.contentDelay }
                NumberAnimation {
                    target: content.item
                    property: "opacity"
                    to: 1
                    duration: Motion.content
                    easing.type: Easing.BezierSpline
                    easing.bezierCurve: Motion.curve
                }
            }
        }
    }

    Component { id: projectPanel; ProjectPanel {} }
    Component { id: volumePanel; VolumePanel {} }
    Component { id: powerPanel; PowerPanel {} }
    Component { id: networkPanel; NetworkPanel {} }
    Component { id: calendarPanel; CalendarPanel {} }
    Component { id: pickerPanel; PickerPanel {} }
    Component { id: launcherPanel; LauncherPanel {} }
    Component { id: switcherPanel; SwitcherPanel {} }
    Component { id: notificationsPanel; NotificationsPanel {} }
    Component { id: sessionPanel; SessionPanel {} }
    Component { id: toastView; ToastView {} }
    Component { id: osdView; OsdView {} }
    Component { id: mediaPanel; MediaPanel {} }
    Component { id: bluetoothPanel; BluetoothPanel {} }
    Component { id: displayPanel; DisplayPanel {} }
    Component { id: systemPanel; SystemPanel {} }
    Component { id: clipboardPanel; ClipboardPanel {} }
    Component { id: wallpaperPanel; WallpaperPanel {} }
}
