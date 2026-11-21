import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

// Página "Controle" dos Ajustes: o controle Xbox e o driver xone
// (scripts/mare-gamepad). O kernel já faz os botões funcionarem; o xone é o
// que faz o fone plugado no controle virar dispositivo de som. É um módulo
// de fora do kernel, então a página só oferece: quem instala é você, num
// terminal que ela abre.
ColumnLayout {
    id: page
    spacing: 0

    readonly property string script: Quickshell.shellPath("scripts/mare-gamepad")
    // O que o mare-gamepad contou por último (null = ainda lendo).
    property var info: null
    readonly property bool installed: info !== null && info.installed === true
    readonly property var controllers: info !== null && info.controllers ? info.controllers : []

    function refresh() {
        if (!statProc.running)
            statProc.running = true;
    }

    // Instalar e remover rodam num terminal: pedem senha e mostram o que fazem.
    function run(action) {
        Mare.close();
        Hypr.exec(Projects.terminalLine("", "bash " + Projects.shellQuote(page.script) + " " + action, "Mk · controle"));
    }

    Component.onCompleted: refresh()

    Process {
        id: statProc
        command: ["bash", page.script, "status"]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    page.info = JSON.parse(text);
                } catch (e) {
                    page.info = ({});
                }
            }
        }
    }

    // A instalação acontece fora daqui: relê de tempos em tempos.
    Timer {
        interval: 4000
        repeat: true
        running: true
        onTriggered: page.refresh()
    }

    // Rótulo de seção: mono pequeno em maiúsculas espaçadas.
    component Section: Text {
        Layout.topMargin: 22
        Layout.bottomMargin: 6
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.letterSpacing: Theme.labelSpacing
    }

    component Line: Text {
        Layout.fillWidth: true
        wrapMode: Text.WordWrap
        color: Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 13
        lineHeight: 1.3
    }

    Section {
        text: "CONTROLE LIGADO"
    }
    Text {
        Layout.fillWidth: true
        text: page.info === null ? "lendo…"
            : page.controllers.length > 0 ? page.controllers.join(", ")
            : "nenhum controle Xbox no cabo"
        color: page.controllers.length > 0 ? Theme.text : Theme.textDim
        font.family: Theme.serif
        font.pixelSize: 16
        font.italic: page.controllers.length === 0
        wrapMode: Text.WordWrap
    }

    Section {
        text: "FONE NO CONTROLE"
    }
    RowLayout {
        Layout.fillWidth: true
        spacing: 16

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 4

            Text {
                text: page.installed
                    ? (page.info.loaded ? "driver xone em uso" : "driver xone instalado")
                    : "driver do kernel (só os botões)"
                color: page.installed ? Theme.copper : Theme.text
                font.family: Theme.sans
                font.pixelSize: 15
            }
            Line {
                text: page.installed
                    ? "O fone plugado no controle aparece no painel de som. Se não aparecer, tire e ponha o cabo, ou reinicie."
                    : "Os botões já funcionam. Para o fone plugado no controle virar saída e microfone, falta o driver xone."
            }
        }

        PillButton {
            visible: page.info !== null && !page.installed
            enabled: page.info !== null && page.info.yay === true
            opacity: enabled ? 1 : 0.4
            text: "instalar o xone"
            onClicked: {
                if (enabled)
                    page.run("install");
            }
        }
        PillButton {
            visible: page.installed
            text: "remover"
            onClicked: page.run("remove")
        }
    }

    Section {
        text: "ANTES DE INSTALAR"
    }
    Line {
        text: "O xone não é do kernel: é compilado na sua máquina a cada kernel novo (as atualizações ficam um pouco mais lentas) e marca o kernel como modificado. Ele baixa um firmware da Microsoft, de licença própria, para o adaptador sem fio. Com ele instalado, o driver do kernel para controles Xbox fica desligado."
    }
    Line {
        Layout.topMargin: 8
        visible: page.info !== null && page.info.yay !== true
        color: Theme.copper
        text: "Falta o yay: o xone vem do AUR."
    }
    Line {
        Layout.topMargin: 8
        color: Theme.textDim
        text: "Testado com um controle Xbox One no cabo. Pelo Bluetooth e pelo adaptador sem fio, ninguém testou."
    }
}
