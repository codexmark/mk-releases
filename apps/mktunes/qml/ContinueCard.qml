import QtQuick
import QtQuick.Layouts

// "Continuar ouvindo": a música que ficou pausada (inclusive a da última vez que o app abriu),
// com o ponto em que parou. Tocar retoma dali.
Rectangle {
    id: card
    readonly property var now: daemon.now
    readonly property var progress: daemon.progress
    implicitHeight: 112
    radius: 14
    color: hover.hovered ? Theme.raised : Theme.surface
    border.color: Theme.hairline
    Behavior on color { ColorAnimation { duration: 120 } }
    Accessible.role: Accessible.Button
    Accessible.name: "Continuar ouvindo " + (now.title || "")

    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { onTapped: daemon.send({ cmd: "toggle" }) }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 16
        spacing: 16
        Cover { Layout.preferredWidth: 80; Layout.preferredHeight: 80; source: card.now.artwork || "" }
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 4
            Text { Layout.fillWidth: true; text: card.now.title || ""; color: Theme.text; font.family: Theme.serif; font.pixelSize: 22; elide: Text.ElideRight }
            Text { Layout.fillWidth: true; text: card.now.artist || ""; color: Theme.textDim; font.family: Theme.sans; font.pixelSize: 13; elide: Text.ElideRight }
            RowLayout {
                Layout.fillWidth: true
                Layout.topMargin: 6
                spacing: 10
                Rectangle {
                    Layout.fillWidth: true
                    Layout.maximumWidth: 360
                    height: 3
                    radius: 2
                    color: Theme.line
                    visible: card.progress.duration > 0
                    Rectangle { width: parent.width * Math.min(1, card.progress.position / Math.max(1, card.progress.duration)); height: parent.height; radius: 2; color: Theme.copper }
                }
                Text {
                    visible: card.progress.duration > 0
                    text: "parou em " + Theme.clock(card.progress.position) + " de " + Theme.clock(card.progress.duration)
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
                Item { Layout.fillWidth: true }
            }
        }
        Rectangle {
            width: 48; height: 48; radius: 24
            color: Theme.copper
            Text { anchors.centerIn: parent; text: Theme.icon.play; color: Theme.ground; font.family: Theme.symbols; font.pixelSize: 24 }
        }
    }
}
