//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkUtil: dois utilitários pequenos do Mk num app só. O lançador diz qual
// janela abrir (MKUTIL_JANELA): MkInstalar, que instala um AppImage, um pacote
// ou um arquivo compactado, ou MkComprimir, que encolhe vídeos e imagens. A
// janela é só a vista; quem faz o trabalho é o motor (../bin/mkutil-motor, em Go).
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        readonly property bool comprimir: Motor.janela === "comprimir"
        title: comprimir ? "MkComprimir" : "MkInstalar"
        color: Theme.ground
        minimumSize: Qt.size(680, 440)
        // O tamanho dela, ou o que couber: numa tela pequena (1366x768) ela
        // fica inteira abaixo da ilha, com folga em volta.
        implicitWidth: Math.min(comprimir ? 940 : 820, (window.screen ? window.screen.width : 940) - 60)
        implicitHeight: Math.min(comprimir ? 640 : 580, (window.screen ? window.screen.height : 640) - 110)

        Loader {
            anchors.fill: parent
            anchors.margins: 24
            source: window.comprimir ? "Comprimir.qml" : "Instalar.qml"
        }
    }

    Connections {
        target: Quickshell
        function onLastWindowClosed() { Qt.quit(); }
    }
}
