//@ pragma UseQApplication
import QtQuick
import Quickshell

// MkEscrita: um editor de Markdown sem distração. Uma coluna de texto e mais
// nada; o resto (busca, sumário, pré-visualização) aparece quando se pede. A
// janela é só a vista: quem lê e grava o arquivo, guarda os rascunhos e
// converte o Markdown é o motor (../bin/mkescrita-motor, em Go).
ShellRoot {
    // App instalado: sem recarregar a cada arquivo salvo.
    settings.watchFiles: false

    FloatingWindow {
        id: window
        // A janela nasce com o título fixo "MkEscrita" (é por ele que o
        // Hyprland a reconhece e que o lançador a acha); depois de assentada,
        // o título passa a dizer o documento e se há mudança não salva.
        property bool assentada: false
        title: assentada ? pagina.titulo : "MkEscrita"
        color: Theme.ground
        minimumSize: Qt.size(520, 360)
        // O tamanho dela, ou o que couber: numa tela pequena (1366x768) ela
        // fica inteira abaixo da ilha, com folga em volta.
        implicitWidth: Math.min(1040, (window.screen ? window.screen.width : 1040) - 60)
        implicitHeight: Math.min(760, (window.screen ? window.screen.height : 760) - 110)

        Timer {
            interval: 1500
            running: true
            onTriggered: window.assentada = true
        }

        Pagina {
            id: pagina
            anchors.fill: parent
            onSair: Qt.quit()
            onTelaCheia: window.fullscreen = !window.fullscreen
            // Fecharam a janela com texto por salvar: ela volta, com a pergunta.
            onReaparecer: window.visible = true
        }
    }

    // Fechar a janela não encerra o app por conta própria: quem decide é a
    // página (grava o que falta, ou pergunta) e só então ela pede para sair.
    Connections {
        target: Quickshell
        function onLastWindowClosed() { pagina.pedeFechar(true); }
    }
}
