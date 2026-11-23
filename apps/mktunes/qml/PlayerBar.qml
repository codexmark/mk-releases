import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// A barra de baixo: o que toca, os controles, a posição e os extras (repetir, timer, velocidade).
Rectangle {
    id: bar
    readonly property var now: daemon.now
    readonly property var progress: daemon.progress
    readonly property bool hasTrack: now.status !== "idle"
    readonly property bool playing: now.status === "playing"
    property int nowPanelTab: 0
    color: Theme.surface
    Rectangle { anchors.top: parent.top; width: parent.width; height: 1; color: Theme.hairline }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: 20
        anchors.rightMargin: 20
        spacing: 16

        // O que toca
        RowLayout {
            Layout.preferredWidth: 280
            Layout.maximumWidth: 280
            spacing: 12
            Cover { Layout.preferredWidth: 52; Layout.preferredHeight: 52; source: bar.now.artwork }
            ColumnLayout {
                Layout.fillWidth: true
                spacing: 2
                Text { Layout.fillWidth: true; text: bar.now.title; color: Theme.text; font.family: Theme.sans; font.pixelSize: 14; font.weight: Font.Medium; elide: Text.ElideRight }
                Text { Layout.fillWidth: true; text: bar.now.artist; color: Theme.textDim; font.family: Theme.sans; font.pixelSize: 12; elide: Text.ElideRight }
            }
            Glyph {
                visible: bar.hasTrack
                glyph: bar.now.liked ? Theme.icon.heart : Theme.icon.heartOutline
                active: bar.now.liked
                label: bar.now.liked ? "Descurtir (Ctrl+L)" : "Curtir (Ctrl+L)"
                onClicked: daemon.send({ cmd: "like", track: bar.now.track })
            }
        }

        // Controles e posição
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2
            RowLayout {
                Layout.alignment: Qt.AlignHCenter
                spacing: 10
                Glyph { glyph: Theme.icon.previous; label: "Anterior (Ctrl+←)"; enabled: bar.hasTrack; onClicked: daemon.send({ cmd: "previous" }) }
                Glyph {
                    glyph: bar.playing ? Theme.icon.pause : Theme.icon.play
                    label: bar.playing ? "Pausar" : "Tocar"
                    primary: true
                    size: 44
                    enabled: bar.hasTrack
                    onClicked: daemon.send({ cmd: "toggle" })
                    BusyIndicator { anchors.fill: parent; running: bar.now.status === "loading"; visible: running }
                }
                Glyph { glyph: Theme.icon.next; label: "Próxima (Ctrl+→)"; enabled: bar.hasTrack; onClicked: daemon.send({ cmd: "next" }) }
            }
            RowLayout {
                Layout.fillWidth: true
                spacing: 10
                Text { text: bar.now.live ? "" : Theme.clock(seek.pressed ? seek.value : bar.progress.position); color: Theme.textDim; font.family: Theme.mono; font.pixelSize: 11; Layout.preferredWidth: 40; horizontalAlignment: Text.AlignRight }
                Slider {
                    id: seek
                    Layout.fillWidth: true
                    enabled: bar.hasTrack && bar.progress.duration > 0 && !bar.now.live
                    from: 0
                    to: Math.max(1, bar.progress.duration)
                    Binding on value { value: bar.progress.position; when: !seek.pressed }
                    onPressedChanged: if (!pressed) daemon.send({ cmd: "seek", seconds: value })
                    Accessible.name: "Posição da música"
                    background: Rectangle {
                        x: seek.leftPadding
                        y: seek.topPadding + seek.availableHeight / 2 - height / 2
                        width: seek.availableWidth
                        height: seekHover.hovered || seek.pressed ? 5 : 3
                        radius: 2
                        color: Theme.line
                        Rectangle { width: seek.visualPosition * parent.width; height: parent.height; radius: 2; color: Theme.copper }
                    }
                    handle: Rectangle {
                        x: seek.leftPadding + seek.visualPosition * (seek.availableWidth - width)
                        y: seek.topPadding + seek.availableHeight / 2 - height / 2
                        width: 12; height: 12; radius: 6
                        color: Theme.copper
                        visible: seek.enabled && (seekHover.hovered || seek.pressed)
                    }
                    HoverHandler { id: seekHover }
                }
                Text { text: bar.now.live ? "AO VIVO" : Theme.clock(bar.progress.duration); color: bar.now.live ? Theme.copper : Theme.textDim; font.family: Theme.mono; font.pixelSize: 11; Layout.preferredWidth: 52 }
            }
        }

        // Extras
        RowLayout {
            Layout.preferredWidth: 420
            Layout.maximumWidth: 420
            Layout.alignment: Qt.AlignRight
            spacing: 2
            Item { Layout.fillWidth: true }
            Glyph {
                glyph: bar.now.repeat === "one" ? Theme.icon.repeatOne : bar.now.repeat === "all" ? Theme.icon.repeat : Theme.icon.repeatOff
                active: bar.now.repeat !== "off"
                label: bar.now.repeat === "one" ? "Repetindo esta música" : bar.now.repeat === "all" ? "Repetindo a fila" : "Repetir"
                onClicked: daemon.send({ cmd: "cycleRepeat" })
            }
            Glyph {
                glyph: Theme.icon.shuffle
                label: "Embaralhar a fila"
                enabled: bar.now.upNext.length > 1
                onClicked: daemon.send({ cmd: "shuffleUpNext" })
            }
            Glyph {
                glyph: Theme.icon.queue
                label: "Fila"
                active: app.panelShown && nowPanelTab === 0
                onClicked: app.showPanel(0)
            }
            Glyph {
                glyph: Theme.icon.lyrics
                label: "Letra"
                active: app.panelShown && nowPanelTab === 1
                onClicked: app.showPanel(1)
            }
            Glyph {
                id: sleepButton
                glyph: Theme.icon.timer
                active: bar.now.sleep !== ""
                label: bar.now.sleep !== "" ? "Para em " + bar.now.sleep : "Timer para dormir"
                onClicked: sleepMenu.popup(sleepButton, 0, -sleepMenu.implicitHeight)
            }
            Glyph {
                id: speedButton
                glyph: Theme.icon.speed
                active: bar.now.speed !== 1
                label: "Velocidade: " + bar.now.speed + "×"
                onClicked: speedMenu.popup(speedButton, 0, -speedMenu.implicitHeight)
            }
            Glyph {
                glyph: bar.now.download === "completed" ? Theme.icon.check : Theme.icon.download
                active: bar.now.download !== ""
                enabled: bar.hasTrack && !bar.now.live
                label: bar.now.download === "completed" ? "Baixada" : bar.now.download === "downloading" ? "Baixando…" : "Baixar"
                onClicked: if (bar.now.download === "") daemon.send({ cmd: "download", track: bar.now.track })
            }
            Glyph {
                glyph: Theme.icon.fullscreen
                label: "Tela cheia (F11)"
                active: app.fullNow
                onClicked: app.fullNow = !app.fullNow
            }
            // Volume: clicar no ícone silencia e volta; a roda do mouse ajusta de 5 em 5.
            Glyph {
                id: volumeIcon
                property int beforeMute: 100
                readonly property int level: daemon.volume.level
                glyph: level === 0 ? Theme.icon.volumeOff : level < 34 ? Theme.icon.volumeLow : level < 67 ? Theme.icon.volumeMedium : Theme.icon.volumeHigh
                label: level === 0 ? "Volume: mudo" : "Volume: " + level + "%"
                onClicked: {
                    if (level > 0) { beforeMute = level; daemon.send({ cmd: "volume", level: 0 }); }
                    else daemon.send({ cmd: "volume", level: Math.max(beforeMute, 10) });
                }
            }
            Slider {
                id: volume
                Layout.preferredWidth: 88
                from: 0
                to: 100
                stepSize: 1
                Binding on value { value: daemon.volume.level; when: !volume.pressed }
                onMoved: if (Math.round(value) !== daemon.volume.level) daemon.send({ cmd: "volume", level: Math.round(value) })
                Accessible.name: "Volume"
                WheelHandler {
                    acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
                    onWheel: event => daemon.send({ cmd: "volume", level: daemon.volume.level + (event.angleDelta.y > 0 ? 5 : -5) })
                }
                background: Rectangle {
                    x: volume.leftPadding
                    y: volume.topPadding + volume.availableHeight / 2 - height / 2
                    width: volume.availableWidth
                    height: 3
                    radius: 2
                    color: Theme.line
                    Rectangle { width: volume.visualPosition * parent.width; height: parent.height; radius: 2; color: Theme.textSoft }
                }
                handle: Rectangle {
                    x: volume.leftPadding + volume.visualPosition * (volume.availableWidth - width)
                    y: volume.topPadding + volume.availableHeight / 2 - height / 2
                    width: 10; height: 10; radius: 5
                    color: Theme.text
                }
            }
            Text {
                visible: bar.now.sleep !== ""
                text: bar.now.sleep
                color: Theme.copper
                font.family: Theme.mono
                font.pixelSize: 11
            }
        }
    }

    MkMenu {
        id: sleepMenu
        MenuEntry { text: "Em 15 minutos"; glyph: Theme.icon.timer; onTriggered: daemon.send({ cmd: "sleep", minutes: 15 }) }
        MenuEntry { text: "Em 30 minutos"; glyph: Theme.icon.timer; onTriggered: daemon.send({ cmd: "sleep", minutes: 30 }) }
        MenuEntry { text: "Em 1 hora"; glyph: Theme.icon.timer; onTriggered: daemon.send({ cmd: "sleep", minutes: 60 }) }
        MenuEntry { text: "No fim desta música"; glyph: Theme.icon.timer; onTriggered: daemon.send({ cmd: "sleep", endOfTrack: true }) }
        MenuEntry {
            text: bar.now.stopAfterCurrent ? "Não parar depois desta" : "Parar depois desta música"
            glyph: Theme.icon.pause
            onTriggered: daemon.send({ cmd: "stopAfterCurrent" })
        }
        MenuEntry { text: "Desligar o timer"; glyph: Theme.icon.close; enabled: bar.now.sleep !== ""; onTriggered: daemon.send({ cmd: "sleep", cancel: true }) }
    }
    MkMenu {
        id: speedMenu
        implicitWidth: 160
        MenuEntry { text: "0,75×"; glyph: bar.now.speed === 0.75 ? Theme.icon.check : ""; onTriggered: daemon.send({ cmd: "speed", value: 0.75 }) }
        MenuEntry { text: "Normal"; glyph: bar.now.speed === 1 ? Theme.icon.check : ""; onTriggered: daemon.send({ cmd: "speed", value: 1 }) }
        MenuEntry { text: "1,25×"; glyph: bar.now.speed === 1.25 ? Theme.icon.check : ""; onTriggered: daemon.send({ cmd: "speed", value: 1.25 }) }
        MenuEntry { text: "1,5×"; glyph: bar.now.speed === 1.5 ? Theme.icon.check : ""; onTriggered: daemon.send({ cmd: "speed", value: 1.5 }) }
    }
}
