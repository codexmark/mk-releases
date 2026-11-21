import QtQuick
import Quickshell
import Quickshell.Hyprland

// Uma barra e uma forma por monitor. Os painéis só abrem no monitor em foco.
Scope {
    id: scope

    required property var modelData
    readonly property bool focused: Quickshell.screens.length === 1
        || (Hyprland.focusedMonitor !== null && Hyprland.focusedMonitor.name === modelData.name)

    Backdrop {
        screen: scope.modelData
    }

    Bar {
        id: bar
        screen: scope.modelData
    }

    Tide {
        id: tide
        screen: scope.modelData
        active: scope.focused
    }

    SettingsWindow {
        id: settings
        screen: scope.modelData
        active: scope.focused
    }

    // Boas-vindas (primeiro login; Welcome.qml decide quando).
    WelcomeWindow {
        screen: scope.modelData
        active: scope.focused
    }

    // Clique fora da linha e da barra fecha o painel aberto.
    HyprlandFocusGrab {
        windows: [tide, bar, settings]
        // O seletor e os passageiros (aviso, OSD) não prendem o foco.
        active: scope.focused && Mare.mode !== "rest" && Mare.mode !== "picker" && !Mare.isTransient && !rearm.running
        onCleared: {
            // Sair de um painel com teclado (a busca) para um sem teclado solta
            // o teclado, e o Hyprland desfaz o grab. Isso não é um clique fora:
            // o painel novo fica aberto e o grab é armado de novo.
            if (Date.now() - Mare.switchedAt < 400)
                rearm.restart();
            else
                Mare.close();
        }
    }

    Timer {
        id: rearm
        interval: 80
    }
}
