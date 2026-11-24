//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkCorte: o editor de cortes de vídeo do Mk. A janela guarda a edição e toca
// a pré-visualização; quem olha os arquivos e chama o ffmpeg é o motor
// (../bin/mkcorte-motor, em Go).
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        title: "MkCorte"
        color: Theme.ground
        minimumSize: Qt.size(520, 420)
        // O tamanho dela, ou o que couber: numa tela pequena (1366x768) ela
        // fica inteira abaixo da ilha, com folga em volta.
        implicitWidth: Math.min(1120, (window.screen ? window.screen.width : 1120) - 60)
        implicitHeight: Math.min(740, (window.screen ? window.screen.height : 740) - 110)

        Editor {
            anchors.fill: parent
            anchors.margins: 20
        }
    }

    Connections {
        target: Quickshell
        function onLastWindowClosed() { Qt.quit(); }
    }
}
