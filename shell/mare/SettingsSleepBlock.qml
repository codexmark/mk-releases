import QtQuick
import QtQuick.Layouts
import Quickshell

// O fim da página Energia dos Ajustes, abaixo de "Ao dormir": o aviso de uma
// placa de vídeo que não volta do suspender e o preparo da hibernação. O que a
// máquina tem vem do scripts/mare-sleep, por Settings.caps.sono. Daqui nada
// muda a escolha da pessoa: o aviso só diz as saídas.
// Preparar roda num terminal (bin/mk-hibernate): pede senha, mexe no boot e
// mostra o que faz.
ColumnLayout {
    id: block
    spacing: 0

    readonly property var sono: Settings.caps && Settings.caps.sono ? Settings.caps.sono : null
    readonly property string choice: Settings.get("power.sleep")
    readonly property bool ready: sono !== null && sono.hibernateReady === true
    readonly property bool pending: sono !== null && sono.pending === true
    readonly property var risky: sono !== null && sono.risky ? sono.risky : []

    // O mk-hibernate do PATH, o do repositório ao lado deste shell (sessão de
    // teste) ou o instalado em /usr/share/mk.
    function prepare() {
        const script = "for m in \"$(command -v mk-hibernate)\" "
            + Projects.shellQuote(Quickshell.shellPath("../../bin/mk-hibernate"))
            + " /usr/share/mk/bin/mk-hibernate; do [ -x \"$m\" ] && break; done; "
            + "sudo \"$m\" setup; echo; read -r -p 'Enter fecha. ' _";
        Mare.close();
        Hypr.exec(Projects.terminalLine("", "bash -c " + Projects.shellQuote(script), "Mk · hibernação"));
    }

    // O preparo acontece fora daqui: relê enquanto a página está à vista.
    Timer {
        interval: 4000
        repeat: true
        running: block.visible && block.sono !== null && !block.ready
        onTriggered: Settings.refreshCaps()
    }

    component Line: Text {
        Layout.fillWidth: true
        Layout.topMargin: 14
        wrapMode: Text.WordWrap
        color: Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 13
        lineHeight: 1.3
    }

    // A placa que não acorda: só com suspender escolhido.
    Line {
        visible: block.risky.length > 0 && block.choice === "suspender"
        color: Theme.copper
        text: "Esta placa de vídeo (" + block.risky.join(", ") + ") é conhecida por não voltar do suspender: a tela fica preta e só reiniciando. "
            + (block.ready ? "Escolha hibernar ou não dormir."
                : "Escolha não dormir, ou prepare a hibernação e escolha hibernar.")
    }

    // Suspender proibido no sistema (sleep.conf): o Mk não suspende.
    Line {
        visible: block.sono !== null && block.sono.canSuspend === false && block.choice === "suspender"
        color: Theme.copper
        text: "O suspender está desligado neste sistema (sleep.conf): com esta escolha o computador não dorme."
    }

    // Hibernar escolhido e a hibernação deixou de estar pronta.
    Line {
        visible: block.sono !== null && !block.ready && block.choice === "hibernar"
        color: Theme.copper
        text: "A hibernação não está pronta: com esta escolha o computador não dorme."
    }

    Line {
        visible: block.pending
        text: "A hibernação está preparada. Reinicie o computador para ela valer; depois, hibernar aparece em Ao dormir."
    }

    RowLayout {
        visible: block.sono !== null && !block.ready && !block.pending
        Layout.fillWidth: true
        Layout.topMargin: 14
        spacing: 16

        Line {
            Layout.topMargin: 0
            text: "Para hibernar falta preparar: um arquivo de swap do tamanho da memória e uma opção no boot. Abre um terminal e pede a senha."
        }
        PillButton {
            text: "preparar a hibernação"
            onClicked: block.prepare()
        }
    }
}
