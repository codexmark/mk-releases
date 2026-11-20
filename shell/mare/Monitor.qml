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

    // Clique fora da linha e da barra fecha o painel aberto.
    HyprlandFocusGrab {
        windows: [tide, bar, settings]
        // O seletor e os passageiros (aviso, OSD) não prendem o foco.
        active: scope.focused && Mare.mode !== "rest" && Mare.mode !== "picker" && !Mare.isTransient
        onCleared: Mare.close()
    }
}
