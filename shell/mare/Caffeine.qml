pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// Cafeína: enquanto ligada, o computador não descansa sozinho (nem descanso
// de tela, nem bloqueio, nem tela apagada). É um inibidor de ociosidade do
// systemd, que o hypridle respeita; desligar (ou o Maré sair) o solta. Não
// fica guardada: ao entrar de novo na sessão ela começa desligada.
Singleton {
    id: root

    property bool on: false

    function toggle() {
        on = !on;
    }

    Process {
        running: root.on
        command: ["systemd-inhibit", "--what=idle", "--who=Mk", "--why=cafeína ligada nos Controles", "sleep", "infinity"]
        // Se o inibidor cair sozinho, a chave não fica mentindo.
        onExited: root.on = false
    }
}
