import QtQuick
import QtQuick.Layouts

// Tela: brilho (arraste) e luz noturna (tela mais quente para a noite).
Item {
    id: panel

    implicitWidth: 400
    implicitHeight: column.implicitHeight + 40

    Component.onCompleted: Display.refresh()

    ColumnLayout {
        id: column
        x: 22
        y: 20
        width: parent.width - 44
        spacing: 16

        RowLayout {
            visible: Display.hasBacklight
            Layout.fillWidth: true
            spacing: 14

            Text {
                text: "󰃠"
                color: Projects.accent
                font.family: Theme.mono
                font.pixelSize: 18
            }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 8

                RowLayout {
                    Layout.fillWidth: true

                    Text {
                        text: "Brilho"
                        color: Theme.text
                        font.family: Theme.sans
                        font.pixelSize: 13
                        Layout.fillWidth: true
                    }

                    Text {
                        text: Math.round(Display.brightness * 100) + "%"
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 12
                    }
                }

                Item {
                    Layout.fillWidth: true
                    implicitHeight: 14

                    Rectangle {
                        anchors.verticalCenter: parent.verticalCenter
                        width: parent.width
                        height: 4
                        radius: 2
                        color: Theme.line

                        Rectangle {
                            width: parent.width * Display.brightness
                            height: parent.height
                            radius: 2
                            color: Projects.accent
                        }
                    }

                    MouseArea {
                        anchors.fill: parent
                        cursorShape: Qt.PointingHandCursor
                        onPressed: mouse => Display.set(mouse.x / width)
                        onPositionChanged: mouse => {
                            if (pressed)
                                Display.set(mouse.x / width);
                        }
                    }
                }
            }
        }

        Text {
            visible: !Display.hasBacklight
            text: "Esta tela não tem controle de brilho."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 13
        }

        Rectangle {
            Layout.fillWidth: true
            implicitHeight: 1
            color: Theme.hairline
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 14

            Text {
                text: "󰖔"
                color: Display.nightLight ? "#F2B84B" : Theme.textSoft
                font.family: Theme.mono
                font.pixelSize: 18
            }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 2

                Text {
                    text: "Luz noturna"
                    color: Theme.text
                    font.family: Theme.sans
                    font.pixelSize: 13
                }

                Text {
                    text: Display.nightLight ? Display.nightTemperature + " K · tela mais quente" : "desligada"
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
            }

            Rectangle {
                width: 36
                height: 20
                radius: 2
                color: Display.nightLight ? "#F2B84B" : Theme.line

                Rectangle {
                    x: Display.nightLight ? parent.width - width - 2 : 2
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
                    onTapped: Display.toggleNight()
                }
            }
        }
    }
}
