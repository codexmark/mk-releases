import QtQuick

// Rótulo em mono maiúsculo espaçado, como os do Maré ("01 · INÍCIO").
Text {
    property bool accent: false
    color: accent ? Theme.copper : Theme.textFaint
    font.family: Theme.mono
    font.pixelSize: 11
    font.letterSpacing: Theme.labelSpacing
    font.capitalization: Font.AllUppercase
}
