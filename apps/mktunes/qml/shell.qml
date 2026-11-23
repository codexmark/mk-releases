//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkTunes no desktop: a janela é só a vista; quem toca é o mktunesd.
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        title: "MkTunes"
        color: Theme.ground
        minimumSize: Qt.size(900, 600)
        implicitWidth: 1280
        implicitHeight: 820
        App { anchors.fill: parent }
    }

    // Fechar a janela encerra este processo; a música segue no mktunesd.
    Connections {
        target: Quickshell
        function onLastWindowClosed() { Qt.quit(); }
    }
}
