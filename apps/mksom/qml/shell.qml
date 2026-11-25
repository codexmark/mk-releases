//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkSom: o equalizador da saída de som e o teste de microfone do Mk. A janela
// é só a vista: quem escreve a configuração do filter-chain, liga, ajusta e
// mede é o motor (../bin/mksom-motor, em Go), só com o que vem no PipeWire.
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        // É por este título que o lançador acha a janela e que a regra de
        // flutuar do Hyprland a reconhece.
        title: "MkSom"
        color: Theme.ground
        // Cabe lado a lado numa tela pequena: em 420×420 os controles
        // estreitam, os rótulos encurtam e a curva some (veja Som.js).
        minimumSize: Qt.size(400, 400)
        // O tamanho dela, ou o que couber na tela, abaixo da ilha.
        implicitWidth: Math.min(560, (window.screen ? window.screen.width : 560) - 60)
        implicitHeight: Math.min(520, (window.screen ? window.screen.height : 520) - 110)
        App { anchors.fill: parent }
    }

    Connections {
        target: Quickshell
        function onLastWindowClosed() { Qt.quit(); }
    }
}
