import QtQuick
import QtQuick.Layouts

// Tocando agora: capa, faixa, progresso (clique para pular) e controles.
// Com mais de um player, os nomes deles aparecem em cima para trocar.
Item {
    id: panel

    readonly property var p: Media.player

    implicitWidth: 480
    implicitHeight: column.implicitHeight + 40

    ColumnLayout {
        id: column
        x: 20
        y: 20
        width: parent.width - 40
        spacing: 14

        Row {
            visible: Media.players.length > 1
            spacing: 6

            Repeater {
                model: Media.players

                PillButton {
                    required property var modelData
                    text: modelData.identity
                    selected: modelData === panel.p
                    onClicked: Media.pinned = modelData
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 16

            Rectangle {
                implicitWidth: 96
                implicitHeight: 96
                radius: 2
                color: Theme.surfaceLow
                border.width: 1
                border.color: Theme.line
                clip: true

                Image {
                    anchors.fill: parent
                    source: panel.p ? panel.p.trackArtUrl : ""
                    fillMode: Image.PreserveAspectCrop
                    sourceSize.width: 192
                    sourceSize.height: 192
                }

                Text {
                    anchors.centerIn: parent
                    visible: !panel.p || panel.p.trackArtUrl === ""
                    text: "󰎈"
                    color: Projects.accent
                    font.family: Theme.mono
                    font.pixelSize: 30
                }
            }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 4

                Text {
                    text: panel.p ? panel.p.identity : ""
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }

                Text {
                    text: panel.p ? panel.p.trackTitle : "Nada tocando"
                    color: Theme.text
                    font.family: Theme.sans
                    font.pixelSize: 17
                    font.weight: Font.DemiBold
                    Layout.fillWidth: true
                    elide: Text.ElideRight
                }

                Text {
                    text: panel.p ? panel.p.trackArtist : ""
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 13
                    Layout.fillWidth: true
                    elide: Text.ElideRight
                }

                Text {
                    visible: text !== ""
                    text: panel.p ? panel.p.trackAlbum : ""
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 12
                    Layout.fillWidth: true
                    elide: Text.ElideRight
                }
            }
        }

        ColumnLayout {
            visible: panel.p !== null && panel.p.lengthSupported && panel.p.length > 0
            Layout.fillWidth: true
            spacing: 6

            Item {
                Layout.fillWidth: true
                implicitHeight: 12

                Rectangle {
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width
                    height: 4
                    radius: 2
                    color: Theme.line

                    Rectangle {
                        width: panel.p && panel.p.length > 0 ? parent.width * Math.min(1, panel.p.position / panel.p.length) : 0
                        height: parent.height
                        radius: 2
                        color: Projects.accent
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    enabled: panel.p !== null && panel.p.canSeek
                    cursorShape: enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
                    onClicked: mouse => {
                        const target = panel.p.length * mouse.x / width;
                        panel.p.seek(target - panel.p.position);
                    }
                }
            }

            RowLayout {
                Layout.fillWidth: true

                Text {
                    text: panel.p ? Media.format(panel.p.position) : ""
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }

                Item { Layout.fillWidth: true }

                Text {
                    text: panel.p ? Media.format(panel.p.length) : ""
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
            }
        }

        Row {
            Layout.alignment: Qt.AlignHCenter
            spacing: 14

            Repeater {
                model: [
                    { icon: "󰒮", act: "prev", big: false },
                    { icon: panel.p && panel.p.isPlaying ? "󰏤" : "󰐊", act: "toggle", big: true },
                    { icon: "󰒭", act: "next", big: false }
                ]

                Rectangle {
                    id: control
                    required property var modelData
                    width: modelData.big ? 52 : 40
                    height: width
                    radius: width / 2
                    color: modelData.big ? Projects.accent : hover.hovered ? Theme.raised : "transparent"
                    border.width: modelData.big ? 0 : 1
                    border.color: Theme.line
                    anchors.verticalCenter: parent.verticalCenter

                    Text {
                        anchors.centerIn: parent
                        text: control.modelData.icon
                        color: control.modelData.big ? Theme.ground : Theme.text
                        font.family: Theme.mono
                        font.pixelSize: control.modelData.big ? 22 : 17
                    }

                    HoverHandler {
                        id: hover
                        cursorShape: Qt.PointingHandCursor
                    }

                    TapHandler {
                        onTapped: {
                            if (!panel.p)
                                return;
                            if (control.modelData.act === "prev")
                                panel.p.previous();
                            else if (control.modelData.act === "next")
                                panel.p.next();
                            else
                                panel.p.togglePlaying();
                        }
                    }
                }
            }
        }
    }
}
