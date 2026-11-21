import QtQuick
import QtQuick.Layouts
import Quickshell

// Capturar a tela: uma área, a janela, a tela inteira, ou gravar um vídeo.
// (Print sozinho continua sendo o atalho rápido: área para a área de
// transferência.)
Item {
    id: panel

    implicitWidth: 440
    implicitHeight: column.implicitHeight + 40

    readonly property var actions: [
        { key: "copiar", glyph: "󰆏", title: "área, só copiar", detail: "PRINT" },
        { key: "area", glyph: "󰩭", title: "área", detail: "salva e copia" },
        { key: "janela", glyph: "󰖯", title: "janela em foco", detail: "salva e copia" },
        { key: "tela", glyph: "󰍹", title: "tela inteira", detail: "SHIFT + PRINT" }
    ]

    ColumnLayout {
        id: column
        x: 22
        y: 20
        width: parent.width - 44
        spacing: 0

        PanelHeader {
            Layout.fillWidth: true
            Layout.bottomMargin: 8
            index: "01"
            title: "Capturar"
            meta: Capture.recording ? "gravando " + Capture.clock() : "imagens / capturas"
            metaColor: Capture.recording ? Theme.copper : Theme.textDim
        }

        Repeater {
            model: panel.actions

            Item {
                id: row
                required property var modelData
                Layout.fillWidth: true
                implicitHeight: 40

                Rectangle {
                    anchors.fill: parent
                    color: hover.hovered ? Theme.wash : "transparent"
                }
                Rectangle {
                    anchors.bottom: parent.bottom
                    width: parent.width
                    height: 1
                    color: Theme.hairline
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 10
                    anchors.rightMargin: 10
                    spacing: 14

                    Text {
                        text: row.modelData.glyph
                        color: Theme.copper
                        font.family: Theme.icons
                        font.pixelSize: 16
                    }
                    Text {
                        Layout.fillWidth: true
                        text: row.modelData.title
                        color: Theme.text
                        font.family: Theme.sans
                        font.pixelSize: 13
                    }
                    Text {
                        text: row.modelData.detail
                        color: Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 10
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: 1.2
                    }
                }

                HoverHandler {
                    id: hover
                    cursorShape: Qt.PointingHandCursor
                }
                TapHandler {
                    onTapped: {
                        Mare.close();
                        Capture.shot(row.modelData.key);
                    }
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.topMargin: 14
            spacing: 8

            // Sem gravador a linha apaga e diz por quê.
            opacity: Capture.available || Capture.recording ? 1 : 0.45

            PillButton {
                primary: !Capture.recording
                selected: Capture.recording
                text: Capture.recording ? "parar a gravação" : "gravar a tela"
                onClicked: {
                    if (!Capture.available && !Capture.recording)
                        return;
                    Capture.toggleRecording();
                    Mare.close();
                }
            }

            Text {
                Layout.fillWidth: true
                text: Capture.available || Capture.recording ? "vídeo com o som do sistema" : Capture.reason
                color: Theme.textFaint
                font.family: Theme.serif
                font.pixelSize: 13
                font.italic: true
                elide: Text.ElideRight
            }
        }
    }
}
