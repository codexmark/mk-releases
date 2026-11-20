import QtQuick
import QtQuick.Layouts
import Quickshell.Services.Pipewire

// Volume da saída padrão: arraste na barra, clique no ícone para silenciar.
Item {
    id: panel

    readonly property var sink: Pipewire.defaultAudioSink
    readonly property bool ready: sink !== null && sink.audio !== null
    readonly property real volume: ready ? sink.audio.volume : 0

    implicitWidth: 380
    implicitHeight: 72

    function setFrom(x, width) {
        if (ready)
            sink.audio.volume = Math.max(0, Math.min(1, x / width));
    }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: 22
        anchors.rightMargin: 22
        spacing: 16

        Text {
            text: panel.ready && panel.sink.audio.muted ? "󰖁" : "󰕾"
            color: Projects.accent
            font.family: Theme.mono
            font.pixelSize: 18

            TapHandler {
                onTapped: if (panel.ready) panel.sink.audio.muted = !panel.sink.audio.muted
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 8

            RowLayout {
                Layout.fillWidth: true

                Text {
                    text: panel.sink ? (panel.sink.description || panel.sink.name) : "Sem saída de áudio"
                    color: Theme.text
                    font.family: Theme.sans
                    font.pixelSize: 12
                    elide: Text.ElideRight
                    Layout.fillWidth: true
                }

                Text {
                    text: Math.round(panel.volume * 100) + "%"
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 12
                }
            }

            Item {
                id: track
                Layout.fillWidth: true
                implicitHeight: 14

                Rectangle {
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width
                    height: 4
                    radius: 2
                    color: Theme.line

                    Rectangle {
                        width: parent.width * Math.min(1, panel.volume)
                        height: parent.height
                        radius: 2
                        color: Projects.accent

                        Behavior on width {
                            NumberAnimation { duration: Motion.fast }
                        }
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onPressed: mouse => panel.setFrom(mouse.x, width)
                    onPositionChanged: mouse => {
                        if (pressed)
                            panel.setFrom(mouse.x, width);
                    }
                }
            }
        }
    }
}
