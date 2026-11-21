pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// O estado da VPN para a ilha e o painel de rede: há túnel aberto? Qual?
// Um só `nmcli monitor` para o shell inteiro (a barra existe uma vez por
// monitor), que avisa quando a rede muda; o relógio lento é só garantia.
// Ligar, importar e esquecer ficam na página VPN dos Ajustes.
Singleton {
    id: root

    property var status: ({ up: false, count: 0, name: "", kind: "", iface: "", ip: "" })
    readonly property bool up: status.up === true

    function refresh() {
        statusProc.running = true;
    }

    Process {
        id: statusProc
        command: ["bash", Quickshell.shellPath("scripts/mare-vpn"), "status"]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    root.status = JSON.parse(text);
                } catch (e) {}
            }
        }
    }

    Process {
        running: true
        command: ["bash", Quickshell.shellPath("scripts/mare-vpn"), "watch"]
        stdout: SplitParser {
            onRead: line => debounce.restart()
        }
    }

    Timer {
        id: debounce
        interval: 400
        onTriggered: root.refresh()
    }

    Timer {
        interval: 60000
        repeat: true
        running: true
        triggeredOnStart: true
        onTriggered: root.refresh()
    }
}
