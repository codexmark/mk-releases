//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkCalc: a calculadora do Mk. A janela é só a vista e o teclado; quem faz a
// conta e guarda o histórico é o motor (../bin/mkcalc-motor, em Go).
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        title: "MkCalc"
        color: Theme.ground
        // Pequena, e funciona estreita: o teclado encolhe junto.
        minimumSize: Qt.size(240, 380)
        // O tamanho dela, ou o que couber na tela, abaixo da ilha.
        implicitWidth: Math.min(340, (window.screen ? window.screen.width : 340) - 60)
        implicitHeight: Math.min(520, (window.screen ? window.screen.height : 520) - 110)
        App { anchors.fill: parent }
    }

    Connections {
        target: Quickshell
        function onLastWindowClosed() { Qt.quit(); }
    }
}
