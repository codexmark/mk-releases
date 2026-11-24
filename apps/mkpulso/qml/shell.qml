//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkPulso: o monitor de sistema do Mk. A janela é só a vista; quem lê o
// sistema e executa as ações é o coletor (../bin/mkpulso-coletor, em Go).
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        title: "MkPulso"
        color: Theme.ground
        minimumSize: Qt.size(820, 480)
        // O tamanho dela, ou o que couber: numa tela pequena (1366x768) ela
        // fica inteira abaixo da ilha, com folga em volta.
        implicitWidth: Math.min(1220, (window.screen ? window.screen.width : 1220) - 60)
        implicitHeight: Math.min(780, (window.screen ? window.screen.height : 780) - 110)
        App { anchors.fill: parent }
    }

    Connections {
        target: Quickshell
        function onLastWindowClosed() { Qt.quit(); }
    }
}
