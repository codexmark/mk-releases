import QtQuick
import QtQuick.Layouts

// OSD: a pílula vira uma barra de nível por um instante quando o volume ou
// o brilho mudam pelas teclas.
Item {
    implicitWidth: 300
    implicitHeight: 40

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: 16
        anchors.rightMargin: 16
        spacing: 12

        Text {
            text: Mare.osdKind === "brightness" ? "󰃠" : Mare.osdMuted ? "󰖁" : "󰕾"
            color: Projects.accent
            font.family: Theme.mono
            font.pixelSize: 16
        }

        Rectangle {
            Layout.fillWidth: true
            implicitHeight: 4
            radius: 2
            color: Theme.line

            Rectangle {
                width: parent.width * (Mare.osdMuted ? 0 : Mare.osdValue)
                height: parent.height
                radius: 2
                color: Projects.accent

                Behavior on width {
                    NumberAnimation {
                        duration: Motion.fast
                        easing.type: Easing.BezierSpline
                        easing.bezierCurve: Motion.curve
                    }
                }
            }
        }

        Text {
            text: Mare.osdMuted ? "mudo" : Math.round(Mare.osdValue * 100) + "%"
            color: Theme.textSoft
            font.family: Theme.mono
            font.pixelSize: 12
            Layout.preferredWidth: 38
            horizontalAlignment: Text.AlignRight
        }
    }
}
