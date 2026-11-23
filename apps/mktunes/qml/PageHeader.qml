import QtQuick
import QtQuick.Layouts

// Topo de página: voltar (quando há para onde), rótulo e título grande em serifa.
ColumnLayout {
    id: header
    property string label
    property string title
    property bool canGoBack: false
    signal back()
    spacing: 4

    RowLayout {
        spacing: 8
        Glyph { visible: header.canGoBack; glyph: Theme.icon.back; label: "Voltar"; size: 32; onClicked: header.back() }
        Caption { text: header.label; accent: true }
    }
    Text {
        Layout.fillWidth: true
        visible: header.title !== ""
        text: header.title
        color: Theme.text
        font.family: Theme.serif
        font.pixelSize: 40
        elide: Text.ElideRight
    }
}
