import QtQuick
import QtQuick.Layouts
import Quickshell

// Papéis de parede de <imagens>/Wallpapers (Wallpaper.dir), 12 por página.
// Clique aplica; a interface toma o matiz da imagem escolhida.
Item {
    id: panel

    readonly property int perPage: 12
    property int page: Math.max(0, Math.floor(Wallpaper.files.indexOf(Wallpaper.current) / perPage))
    readonly property int pages: Math.max(1, Math.ceil(Wallpaper.files.length / perPage))
    readonly property var shown: Wallpaper.files.slice(page * perPage, page * perPage + perPage)

    implicitWidth: 4 * 150 + 3 * 10 + 44
    implicitHeight: column.implicitHeight + 40

    Component.onCompleted: Wallpaper.refresh()

    ColumnLayout {
        id: column
        x: 22
        y: 20
        width: parent.width - 44
        spacing: 12

        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Text {
                text: "Papel de parede"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }

            Rectangle {
                visible: Wallpaper.primary !== ""
                width: 12
                height: 12
                radius: 6
                color: Wallpaper.primary
            }

            Item { Layout.fillWidth: true }

            PillButton {
                text: "aleatório"
                onClicked: Wallpaper.random()
            }

            PillButton {
                text: "‹"
                onClicked: panel.page = (panel.page - 1 + panel.pages) % panel.pages
            }

            Text {
                text: (panel.page + 1) + "/" + panel.pages
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
            }

            PillButton {
                text: "›"
                onClicked: panel.page = (panel.page + 1) % panel.pages
            }
        }

        GridLayout {
            columns: 4
            rowSpacing: 10
            columnSpacing: 10

            Repeater {
                model: panel.shown

                Rectangle {
                    id: thumb
                    required property string modelData
                    readonly property bool current: modelData === Wallpaper.current

                    Layout.preferredWidth: 150
                    Layout.preferredHeight: 86
                    radius: 2
                    color: Theme.surfaceLow
                    border.width: current ? 2 : 1
                    border.color: current ? Projects.accent : hover.hovered ? Theme.textDim : Theme.line
                    clip: true

                    Image {
                        anchors.fill: parent
                        anchors.margins: thumb.current ? 3 : 1
                        source: "file://" + thumb.modelData
                        fillMode: Image.PreserveAspectCrop
                        asynchronous: true
                        sourceSize.width: 300
                        sourceSize.height: 172
                    }

                    HoverHandler {
                        id: hover
                        cursorShape: Qt.PointingHandCursor
                    }

                    TapHandler {
                        onTapped: Wallpaper.set(thumb.modelData)
                    }
                }
            }
        }

        Text {
            visible: Wallpaper.files.length === 0
            text: "Coloque imagens em " + Wallpaper.dir.replace(Quickshell.env("HOME"), "~") + "."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 13
        }
    }
}
