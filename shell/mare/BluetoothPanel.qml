import QtQuick
import QtQuick.Layouts
import Quickshell.Bluetooth

// Bluetooth: liga/desliga, aparelhos pareados (clique conecta ou desconecta,
// com a bateria quando o aparelho informa) e "procurar" para parear um novo.
Item {
    id: panel

    readonly property var adapter: Bluetooth.defaultAdapter
    readonly property bool on: adapter !== null && adapter.enabled
    readonly property var devices: adapter ? adapter.devices.values.slice() : []
    readonly property var paired: devices.filter(d => d.paired)
        .sort((a, b) => (b.connected - a.connected) || a.name.localeCompare(b.name))
        .slice(0, 6)
    readonly property var found: devices.filter(d => !d.paired && d.name !== "").slice(0, 4)

    implicitWidth: 420
    implicitHeight: column.implicitHeight + 40

    // Para de procurar ao fechar o painel.
    Component.onDestruction: if (adapter && adapter.discovering) adapter.discovering = false

    function icon(d) {
        const i = d.icon || "";
        if (i.indexOf("headset") >= 0 || i.indexOf("headphone") >= 0 || i.indexOf("audio") >= 0)
            return "󰋋";
        if (i.indexOf("mouse") >= 0)
            return "󰍽";
        if (i.indexOf("keyboard") >= 0)
            return "󰌌";
        if (i.indexOf("phone") >= 0)
            return "󰏲";
        if (i.indexOf("gaming") >= 0 || i.indexOf("joystick") >= 0)
            return "󰊴";
        return "󰂯";
    }

    function status(d) {
        if (d.state === BluetoothDeviceState.Connecting)
            return "conectando…";
        if (d.state === BluetoothDeviceState.Disconnecting)
            return "desconectando…";
        if (d.pairing)
            return "pareando…";
        if (d.connected)
            return d.batteryAvailable ? "conectado · " + Math.round(d.battery * 100) + "%" : "conectado";
        return d.paired ? "pareado" : "novo";
    }

    function click(d) {
        if (!d.paired) {
            d.trusted = true;
            d.pair();
        } else if (d.connected) {
            d.disconnect();
        } else {
            d.connect();
        }
    }

    ColumnLayout {
        id: column
        x: 22
        y: 20
        width: parent.width - 44
        spacing: 10

        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Text {
                text: "Bluetooth"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }

            Text {
                text: panel.adapter === null ? "sem adaptador" : !panel.on ? "desligado"
                    : panel.paired.some(d => d.connected) ? "conectado" : "ligado"
                color: panel.on ? Projects.accent : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
            }

            Item { Layout.fillWidth: true }

            Rectangle {
                visible: panel.adapter !== null
                width: 36
                height: 20
                radius: 2
                color: panel.on ? Projects.accent : Theme.line

                Rectangle {
                    x: panel.on ? parent.width - width - 2 : 2
                    y: 2
                    width: 16
                    height: 16
                    radius: 2
                    color: Theme.ground

                    Behavior on x {
                        NumberAnimation {
                            duration: Motion.fast
                            easing.type: Easing.BezierSpline
                            easing.bezierCurve: Motion.curve
                        }
                    }
                }

                TapHandler {
                    onTapped: panel.adapter.enabled = !panel.adapter.enabled
                }
            }
        }

        Repeater {
            model: panel.on ? panel.paired.concat(panel.found) : []

            Rectangle {
                id: row
                required property var modelData
                Layout.fillWidth: true
                implicitHeight: 42
                radius: 2
                color: modelData.connected ? Theme.raised : hover.hovered ? Theme.surfaceLow : "transparent"

                Behavior on color {
                    ColorAnimation { duration: Motion.fast }
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 12
                    anchors.rightMargin: 12
                    spacing: 12

                    Text {
                        text: panel.icon(row.modelData)
                        color: row.modelData.connected ? Projects.accent : Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 16
                    }

                    Text {
                        text: row.modelData.name
                        color: Theme.text
                        font.family: Theme.sans
                        font.pixelSize: 14
                        elide: Text.ElideRight
                        Layout.fillWidth: true
                    }

                    Text {
                        text: panel.status(row.modelData)
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 12
                    }
                }

                HoverHandler {
                    id: hover
                    cursorShape: Qt.PointingHandCursor
                }

                TapHandler {
                    onTapped: panel.click(row.modelData)
                }
            }
        }

        Text {
            visible: panel.on && panel.paired.length === 0 && panel.found.length === 0
            text: panel.adapter && panel.adapter.discovering ? "Procurando aparelhos…" : "Nenhum aparelho pareado."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 13
        }

        RowLayout {
            visible: panel.on
            Layout.fillWidth: true

            Text {
                text: panel.found.length > 0 ? "clique num aparelho novo para parear" : ""
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 11
                Layout.fillWidth: true
            }

            PillButton {
                text: panel.adapter && panel.adapter.discovering ? "parar" : "procurar"
                selected: panel.adapter !== null && panel.adapter.discovering
                onClicked: panel.adapter.discovering = !panel.adapter.discovering
            }
        }
    }
}
