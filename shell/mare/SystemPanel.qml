import QtQuick
import QtQuick.Layouts

// Sistema: quatro blocos com valor atual e o último minuto em gráfico,
// mais o disco e um atalho para o btop.
Item {
    id: panel

    readonly property color warn: "#FF8F66"

    implicitWidth: 520
    implicitHeight: column.implicitHeight + 40

    component Tile: Rectangle {
        id: tile

        property string label: ""
        property string value: ""
        property string detail: ""
        property var history: []
        property bool hot: false
        property bool normalize: false

        Layout.fillWidth: true
        implicitHeight: 104
        radius: 2
        color: Theme.surfaceLow
        border.width: 1
        border.color: hot ? panel.warn : Theme.line

        ColumnLayout {
            anchors.fill: parent
            anchors.margins: 12
            spacing: 4

            RowLayout {
                Layout.fillWidth: true

                Text {
                    text: tile.label
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 11
                    Layout.fillWidth: true
                }

                Text {
                    text: tile.detail
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
            }

            Text {
                text: tile.value
                color: tile.hot ? panel.warn : Theme.text
                font.family: Theme.sans
                font.pixelSize: 20
                font.weight: Font.DemiBold
            }

            Sparkline {
                Layout.fillWidth: true
                Layout.fillHeight: true
                values: tile.history
                normalize: tile.normalize
                color: tile.hot ? panel.warn : Projects.accent
            }
        }
    }

    ColumnLayout {
        id: column
        x: 20
        y: 20
        width: parent.width - 40
        spacing: 10

        RowLayout {
            Layout.fillWidth: true

            Text {
                text: "Sistema"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
                Layout.fillWidth: true
            }

            PillButton {
                text: "btop"
                onClicked: {
                    Projects.openIn("", "btop");
                    Mare.close();
                }
            }
        }

        GridLayout {
            Layout.fillWidth: true
            columns: 2
            rowSpacing: 10
            columnSpacing: 10

            Tile {
                label: "CPU"
                value: Math.round(Stats.cpu * 100) + "%"
                history: Stats.cpuHistory
                hot: Stats.cpu > 0.9
            }

            Tile {
                label: "Memória"
                value: Math.round(Stats.mem * 100) + "%"
                detail: Stats.memUsedGb.toFixed(1).replace(".", ",") + " / " + Math.round(Stats.memTotalGb) + " GB"
                history: Stats.memHistory
                hot: Stats.mem > 0.9
            }

            Tile {
                label: "Temperatura"
                value: Stats.temp > 0 ? Math.round(Stats.temp) + "°C" : "—"
                history: Stats.tempHistory
                hot: Stats.temp > 90
            }

            Tile {
                label: "Rede"
                value: "↓ " + Stats.rate(Stats.rxRate)
                detail: "↑ " + Stats.rate(Stats.txRate)
                history: Stats.netHistory
                normalize: true
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 12

            Text {
                text: "Disco /"
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 12
            }

            Rectangle {
                Layout.fillWidth: true
                implicitHeight: 4
                radius: 2
                color: Theme.line

                Rectangle {
                    width: parent.width * Stats.disk
                    height: parent.height
                    radius: 2
                    color: Stats.disk > 0.95 ? panel.warn : Projects.accent
                }
            }

            Text {
                text: Math.round(Stats.diskUsedGb) + " / " + Math.round(Stats.diskTotalGb) + " GB"
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 11
            }
        }
    }
}
