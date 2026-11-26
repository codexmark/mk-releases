//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkFiles: o gerenciador de arquivos do Mk. A janela é só a vista, o teclado
// e o mouse; quem mexe no disco é o motor (../bin/mkfiles-motor, em Go). O
// contrato entre os dois está em ../CONTRATO.md.
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        title: "MkFiles"
        color: Theme.ground
        // Cabe lado a lado em meia tela, e cede em larguras pequenas.
        minimumSize: Qt.size(520, 420)
        // O tamanho dela, ou o que couber na tela, abaixo da ilha.
        implicitWidth: Math.min(1000, (window.screen ? window.screen.width : 1000) - 60)
        implicitHeight: Math.min(640, (window.screen ? window.screen.height : 640) - 110)
        App { anchors.fill: parent }
    }

    Connections {
        target: Quickshell
        function onLastWindowClosed() { Qt.quit(); }
    }
}
