import QtQuick
import QtQuick.Layouts
import Quickshell.Services.Pipewire

// Volume da saída padrão: arraste na barra, clique no ícone para silenciar.
Item {
    id: panel

    readonly property var sink: Pipewire.defaultAudioSink
    readonly property bool ready: sink !== null && sink.audio !== null
    readonly property real volume: ready ? sink.audio.volume : 0

    // O que está tocando agora, app por app (os fluxos de saída do PipeWire).
    readonly property var streams: Pipewire.nodes.values.filter(n => n.isStream && n.isSink && n.audio)

    PwObjectTracker {
        objects: panel.streams
    }

    implicitWidth: 380
    implicitHeight: 72 + (streams.length > 0 ? 14 + streams.length * 40 : 0)

    function setFrom(x, width) {
        if (ready)
            sink.audio.volume = Math.max(0, Math.min(1, x / width));
    }

    // Volume de cada app.
    Column {
        x: 22
        y: 72
        width: parent.width - 44
        visible: panel.streams.length > 0

        Rectangle {
            width: parent.width
            height: 1
            color: Theme.hairline
        }
        Item { width: 1; height: 8 }

        Repeater {
            model: panel.streams

            Item {
                id: app
                required property var modelData
                readonly property real vol: modelData.audio ? modelData.audio.volume : 0
                readonly property string label: modelData.properties["application.name"] || modelData.description || modelData.name
                readonly property string detail: modelData.properties["media.name"] || ""
                width: parent.width
                height: 40

                function setFrom(x, w) {
                    if (modelData.audio)
                        modelData.audio.volume = Math.max(0, Math.min(1, x / w));
                }

                Text {
                    id: mute
                    anchors.verticalCenter: parent.verticalCenter
                    text: app.modelData.audio && app.modelData.audio.muted ? "󰖁" : "󰕾"
                    color: app.modelData.audio && app.modelData.audio.muted ? Theme.textFaint : Theme.textSoft
                    font.family: Theme.mono
                    font.pixelSize: 14

                    TapHandler {
                        onTapped: if (app.modelData.audio) app.modelData.audio.muted = !app.modelData.audio.muted
                    }
                    HoverHandler { cursorShape: Qt.PointingHandCursor }
                }

                Text {
                    anchors.left: mute.right
                    anchors.leftMargin: 14
                    anchors.right: pct.left
                    anchors.rightMargin: 10
                    y: 4
                    text: app.label + (app.detail !== "" && app.detail !== app.label ? "  ·  " + app.detail : "")
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 11
                    elide: Text.ElideRight
                }

                Text {
                    id: pct
                    anchors.right: parent.right
                    y: 4
                    text: Math.round(app.vol * 100) + "%"
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }

                Item {
                    anchors.left: mute.right
                    anchors.leftMargin: 14
                    anchors.right: parent.right
                    y: 22
                    height: 12

                    Rectangle {
                        anchors.verticalCenter: parent.verticalCenter
                        width: parent.width
                        height: 2
                        color: Theme.line

                        Rectangle {
                            width: parent.width * Math.min(1, app.vol)
                            height: parent.height
                            color: Theme.textSoft
                        }
                    }

                    MouseArea {
                        anchors.fill: parent
                        cursorShape: Qt.PointingHandCursor
                        onPressed: mouse => app.setFrom(mouse.x, width)
                        onPositionChanged: mouse => {
                            if (pressed)
                                app.setFrom(mouse.x, width);
                        }
                    }
                }
            }
        }
    }

    RowLayout {
        x: 22
        width: parent.width - 44
        height: 72
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
