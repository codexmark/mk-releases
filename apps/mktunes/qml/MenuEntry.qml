import QtQuick
import QtQuick.Controls

// Item de menu no visual do Maré.
MenuItem {
    id: entry
    property string glyph
    implicitHeight: 36
    contentItem: Row {
        spacing: 12
        leftPadding: 4
        Text {
            width: 18
            text: entry.glyph
            color: entry.enabled ? Theme.textSoft : Theme.textFaint
            font.family: Theme.symbols
            font.pixelSize: 15
            anchors.verticalCenter: parent.verticalCenter
        }
        Text {
            text: entry.text
            color: entry.enabled ? Theme.text : Theme.textFaint
            font.family: Theme.sans
            font.pixelSize: 13
            anchors.verticalCenter: parent.verticalCenter
        }
    }
    background: Rectangle { radius: 6; color: entry.highlighted ? Theme.wash : "transparent" }
}
