import QtQuick

// Um rótulo de folha: mono, maiúsculas, espaçado.
Text {
    color: Theme.textFaint
    font.family: Theme.mono
    font.pixelSize: 9
    font.capitalization: Font.AllUppercase
    font.letterSpacing: Theme.labelSpacing
}
