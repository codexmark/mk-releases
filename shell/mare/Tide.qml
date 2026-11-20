import QtQuick
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
        item: sheet
    }

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

        x: targetX
        y: tide.barH - tide.weld
        width: targetW
        height: targetH
        visible: height > 0.5
        clip: true

        Behavior on x {
            NumberAnimation {
                duration: Motion.spatial
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.spatialCurve
            }
        }
        Behavior on width {
            NumberAnimation {
                duration: Motion.spatial
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Motion.spatialCurve
            }
        }
        Behavior on height {
            NumberAnimation {
                duration: tide.open ? Motion.enter : Motion.exit
                easing.type: Easing.BezierSpline
                easing.bezierCurve: tide.open ? Motion.enterCurve : Motion.exitCurve
            }
        }

        onWidthChanged: shape.requestPaint()
        onHeightChanged: shape.requestPaint()

        // A forma: o mesmo desenho da ilha (filetes côncavos em cima, cantos
        // arredondados embaixo), sólida, com a linha fina nos lados e embaixo.
        Canvas {
            id: shape
            anchors.fill: parent

            onPaint: {
                const ctx = getContext("2d");
                ctx.reset();
                if (width < 2 * tide.fillet + 2 || height < 2)
                    return;
                const f = tide.fillet, w = width, h = height - 1;
                const r = Math.min(tide.corner, Math.max(0, (h - f) / 2));
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
                ctx.beginPath();
                path();
                ctx.closePath();
                ctx.fillStyle = Theme.surface;
                ctx.fill();
                ctx.beginPath();
                path();
                ctx.strokeStyle = Theme.hairline;
                ctx.lineWidth = 1;
                ctx.stroke();
            }
        }

        Image {
            x: tide.fillet
            width: parent.width - 2 * tide.fillet
            height: parent.height
            source: Qt.resolvedUrl("assets/grain.png")
            fillMode: Image.Tile
            opacity: 0.035
        }

        // Marca de cobre no canto, como os destaques das folhas do perfil.
        Rectangle {
            x: tide.fillet + 18
            y: tide.weld + 6
            width: 28
            height: 2
            color: Theme.copper
            z: 2
        }

        Loader {
            id: content

            x: tide.fillet
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
