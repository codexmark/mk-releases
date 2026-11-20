import QtQuick
import QtQuick.Layouts

// Painel do projeto: branch, mudanças, containers, portas e atalhos.
Item {
    implicitWidth: 560
    implicitHeight: column.implicitHeight + 40

    ColumnLayout {
        id: column
        x: 22
        y: 20
        width: parent.width - 44
        spacing: 16

        RowLayout {
            Layout.fillWidth: true
            spacing: 12

            Text {
                text: Projects.name
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 24
                font.weight: Font.Light
            }

            Text {
                text: Projects.info.branch
                color: Projects.accent
                font.family: Theme.mono
                font.pixelSize: 12
            }

            Item { Layout.fillWidth: true }

            Text {
                text: Projects.info.dirty > 0 ? Projects.info.dirty + " modificados" : "limpo"
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 16

            ColumnLayout {
                Layout.preferredWidth: 1
                Layout.fillWidth: true
                Layout.alignment: Qt.AlignTop
                spacing: 8

                Text {
                    text: "Containers"
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 11
                }

                Repeater {
                    model: Projects.info.containers.length > 0
                        ? Projects.info.containers
                        : [{ name: "nenhum container", status: "", running: false }]

                    RowLayout {
                        required property var modelData
                        Layout.fillWidth: true
                        spacing: 8

                        Rectangle {
                            width: 6
                            height: 6
                            radius: 3
                            color: modelData.running ? Projects.accent : "#4A515B"
                        }

                        Text {
                            text: modelData.name
                            color: Theme.text
                            font.family: Theme.mono
                            font.pixelSize: 12
                        }

                        Item { Layout.fillWidth: true }

                        Text {
                            text: modelData.status
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 11
                            elide: Text.ElideRight
                            Layout.maximumWidth: 110
                        }
                    }
                }
            }

            ColumnLayout {
                Layout.preferredWidth: 1
                Layout.fillWidth: true
                Layout.alignment: Qt.AlignTop
                spacing: 8

                Text {
                    text: "Portas abertas"
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 11
                }

                Flow {
                    Layout.fillWidth: true
                    spacing: 6

                    Repeater {
                        model: Projects.info.ports.length > 0 ? Projects.info.ports : ["nenhuma"]

                        Rectangle {
                            required property string modelData
                            width: portText.implicitWidth + 18
                            height: 22
                            radius: 2
                            color: "transparent"
                            border.width: 1
                            border.color: Theme.line

                            Text {
                                id: portText
                                anchors.centerIn: parent
                                text: parent.modelData
                                color: Theme.text
                                font.family: Theme.mono
                                font.pixelSize: 12
                            }
                        }
                    }
                }

                Text {
                    text: "Runtime (mise)"
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 11
                    Layout.topMargin: 4
                }

                Text {
                    text: Projects.info.runtime !== "" ? Projects.info.runtime : "padrão do sistema"
                    color: Theme.text
                    font.family: Theme.mono
                    font.pixelSize: 12
                    elide: Text.ElideRight
                    Layout.fillWidth: true
                }
            }
        }

        Row {
            spacing: 8

            PillButton {
                text: "lazygit"
                primary: true
                onClicked: {
                    Projects.openHere("lazygit");
                    Mare.close();
                }
            }

            PillButton {
                text: "lazydocker"
                onClicked: {
                    Projects.openHere("lazydocker");
                    Mare.close();
                }
            }

            PillButton {
                text: "terminal aqui"
                onClicked: {
                    Projects.openHere("");
                    Mare.close();
                }
            }
        }
    }
}
