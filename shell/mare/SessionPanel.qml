import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

// Sessão: bloquear e dormir agem na hora; sair, reiniciar e desligar
// pedem um segundo clique (o botão vira "confirmar" por 3 segundos).
// Dormir é o que Ajustes › Energia escolheu (Settings.sleepMode): suspender
// ou nenhum botão.
Item {
    id: panel

    property string armed: ""

    readonly property var actions: [
        { id: "lock", icon: "󰌾", label: "bloquear", confirm: false, cmd: ["loginctl", "lock-session"] },
        { id: "suspend", icon: "󰤄", label: "suspender", confirm: false, cmd: ["systemctl", "suspend"], sleep: "suspender" },
        { id: "logout", icon: "󰍃", label: "sair", confirm: true, cmd: ["sh", "-c", "uwsm stop || hyprctl dispatch 'hl.dsp.exit()'"] },
        { id: "reboot", icon: "󰜉", label: "reiniciar", confirm: true, cmd: ["systemctl", "reboot"] },
        { id: "poweroff", icon: "󰐥", label: "desligar", confirm: true, cmd: ["systemctl", "poweroff"] }
    ].filter(a => a.sleep === undefined || a.sleep === Settings.sleepMode)

    implicitWidth: 520
    implicitHeight: column.implicitHeight + 40

    function run(action) {
        if (action.confirm && armed !== action.id) {
            armed = action.id;
            disarm.restart();
            return;
        }
        Mare.close();
        Quickshell.execDetached(action.cmd);
    }

    Timer {
        id: disarm
        interval: 3000
        onTriggered: panel.armed = ""
    }

    Process {
        id: uptime
        running: true
        command: ["uptime", "-p"]
        stdout: StdioCollector {
            onStreamFinished: uptimeText.text = text.trim()
                .replace("up ", "ligado há ")
                .replace(/weeks?/, "sem").replace(/days?/, "d").replace(/hours?/, "h").replace(/minutes?/, "min")
        }
    }

    ColumnLayout {
        id: column
        x: 20
        y: 20
        width: parent.width - 40
        spacing: 14

        RowLayout {
            Layout.fillWidth: true

            Text {
                text: Quickshell.env("USER")
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }

            Item { Layout.fillWidth: true }

            Text {
                id: uptimeText
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 8

            Repeater {
                model: panel.actions

                Rectangle {
                    id: button
                    required property var modelData
                    readonly property bool isArmed: panel.armed === modelData.id

                    Layout.fillWidth: true
                    implicitHeight: 82
                    radius: 2
                    color: isArmed ? Projects.accent : hover.hovered ? Theme.raised : Theme.surfaceLow
                    border.width: 1
                    border.color: isArmed ? Projects.accent : Theme.line

                    Behavior on color {
                        ColorAnimation { duration: Motion.fast }
                    }

                    ColumnLayout {
                        anchors.centerIn: parent
                        spacing: 8

                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: button.modelData.icon
                            color: button.isArmed ? Theme.ground : Theme.text
                            font.family: Theme.mono
                            font.pixelSize: 22
                        }

                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: button.isArmed ? "confirmar" : button.modelData.label
                            color: button.isArmed ? Theme.ground : Theme.textSoft
                            font.family: Theme.mono
                            font.pixelSize: 12
                        }
                    }

                    HoverHandler {
                        id: hover
                        cursorShape: Qt.PointingHandCursor
                    }

                    TapHandler {
                        onTapped: panel.run(button.modelData)
                    }
                }
            }
        }
    }
}
