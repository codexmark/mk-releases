import QtQuick

// Botão redondo com um símbolo. `label` é o que o leitor de tela diz e a dica mostra.
Rectangle {
    id: button
    property string glyph
    property string label
    property bool primary: false
    property bool active: false
    property int size: 36
    property int glyphSize: Math.round(size * 0.45)
    signal clicked()

    width: size
    height: size
    radius: size / 2
    color: primary ? Theme.copper : (hover.hovered ? Theme.wash : "transparent")
    opacity: enabled ? 1 : 0.35
    Accessible.role: Accessible.Button
    Accessible.name: label
    Accessible.onPressAction: button.clicked()

    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { onTapped: button.clicked() }

    Text {
        anchors.centerIn: parent
        text: button.glyph
        color: button.primary ? Theme.ground : (button.active ? Theme.copper : Theme.text)
        font.family: Theme.symbols
        font.pixelSize: button.glyphSize
    }

    ToolTipText { text: button.label; shown: hover.hovered && button.label !== "" && !button.primary }
}
