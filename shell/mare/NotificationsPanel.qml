import QtQuick
import QtQuick.Layouts

// Histórico de notificações, a mais nova em cima. Não perturbe segura os
// avisos (menos os críticos); o histórico continua registrando tudo.
Item {
    id: panel

    readonly property int maxRows: 6
    readonly property var shown: Notifs.list.slice(0, maxRows)

    implicitWidth: 460
    implicitHeight: column.implicitHeight + 40

    ColumnLayout {
        id: column
        x: 20
        y: 20
        width: parent.width - 40
        spacing: 10

        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Text {
                text: "Notificações"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }

            Text {
                text: Notifs.count > 0 ? Notifs.count : ""
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
            }

            Item { Layout.fillWidth: true }

            Text {
                text: "não perturbe"
                color: Notifs.dnd ? Projects.accent : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
            }

            Rectangle {
                width: 36
                height: 20
                radius: 2
                color: Notifs.dnd ? Projects.accent : Theme.line

                Rectangle {
                    x: Notifs.dnd ? parent.width - width - 2 : 2
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
                    onTapped: Notifs.dnd = !Notifs.dnd
                }
            }
        }

        Repeater {
            model: panel.shown

            Rectangle {
                id: row
                required property var modelData
                Layout.fillWidth: true
                implicitHeight: content.implicitHeight + 20
                radius: 2
                color: hover.hovered ? Theme.raised : Theme.surfaceLow

                Behavior on color {
                    ColorAnimation { duration: Motion.fast }
                }

                RowLayout {
                    id: content
                    x: 12
                    y: 10
                    width: parent.width - 24
                    spacing: 12

                    NotifIcon {
                        notification: row.modelData
                        Layout.alignment: Qt.AlignTop
                    }

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 2

                        RowLayout {
                            Layout.fillWidth: true

                            Text {
                                text: row.modelData.appName || "Notificação"
                                color: Theme.textDim
                                font.family: Theme.mono
                                font.pixelSize: 11
                                Layout.fillWidth: true
                                elide: Text.ElideRight
                            }

                            Text {
                                text: Notifs.ago(row.modelData)
                                color: Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 11
                            }

                            Text {
                                text: "×"
                                color: Theme.textDim
                                font.family: Theme.sans
                                font.pixelSize: 15
                                leftPadding: 6

                                HoverHandler { cursorShape: Qt.PointingHandCursor }
                                TapHandler { onTapped: row.modelData.dismiss() }
                            }
                        }

                        Text {
                            text: row.modelData.summary
                            color: Theme.text
                            font.family: Theme.sans
                            font.pixelSize: 13
                            font.weight: Font.Medium
                            Layout.fillWidth: true
                            elide: Text.ElideRight
                        }

                        Text {
                            visible: text !== ""
                            text: row.modelData.body
                            textFormat: Text.PlainText
                            color: Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 12
                            Layout.fillWidth: true
                            wrapMode: Text.WordWrap
                            maximumLineCount: 2
                            elide: Text.ElideRight
                        }
                    }
                }

                HoverHandler {
                    id: hover
                }

                TapHandler {
                    onTapped: {
                        Notifs.activate(row.modelData);
                        Mare.close();
                    }
                }
            }
        }

        Text {
            visible: Notifs.count === 0
            text: "Nada por aqui."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 13
            Layout.topMargin: 6
            Layout.bottomMargin: 6
        }

        RowLayout {
            visible: Notifs.count > 0
            Layout.fillWidth: true

            Text {
                text: Notifs.count > panel.maxRows ? "+" + (Notifs.count - panel.maxRows) + " mais antigas" : ""
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 11
                Layout.fillWidth: true
            }

            PillButton {
                text: "limpar tudo"
                onClicked: Notifs.clearAll()
            }
        }
    }
}
