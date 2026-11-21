// Maré: a ilha no alto da tela; tudo nasce dela.
// Rodar: qs -p <este diretório>   (instalado: qs -p ~/.config/mk/shell)
import QtQuick
import Quickshell

ShellRoot {
    Variants {
        model: Quickshell.screens

        Monitor {}
    }

    // A tela de bloqueio (Super + L), uma para todas as telas.
    LockScreen {}

    // Aviso de bateria fraca: a referência é o que carrega o singleton.
    readonly property var batteryWatch: BatteryWatch
}
