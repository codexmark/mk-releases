//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkPorto: o gerente de conexões SSH do Mk. Um catálogo dos hosts do
// ~/.ssh/config, com o estado de cada um; conectar é abrir um terminal com
// `ssh apelido`. A janela é só a vista: quem lê o config, sonda as portas e
// grava é o motor (../bin/mkporto-motor, em Go). Nunca pede nem guarda senha.
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        // É por este título que o lançador acha a janela. Ela é uma janela de
        // trabalho: não há regra de flutuar para "MkPorto".
        title: "MkPorto"
        color: Theme.ground
        // Cabe lado a lado em meia tela: abaixo de 720 de largura o detalhe
        // deixa de ficar ao lado da lista (veja App.qml).
        minimumSize: Qt.size(520, 420)
        // O tamanho dela, ou o que couber: numa tela pequena (1366x768) ela
        // fica inteira abaixo da ilha, com folga em volta.
        implicitWidth: Math.min(900, (window.screen ? window.screen.width : 900) - 60)
        implicitHeight: Math.min(600, (window.screen ? window.screen.height : 600) - 110)
        App {
            anchors.fill: parent
            // A área Máquinas só se atualiza sozinha com a janela à vista.
            janelaVisivel: window.visible
        }
    }

    Connections {
        target: Quickshell
        function onLastWindowClosed() { Qt.quit(); }
    }
}
