import QtQuick
import Quickshell
import Quickshell.Hyprland
import Quickshell.Wayland

// Uma barra e uma forma por monitor. Os painéis só abrem no monitor em foco.
Scope {
    id: scope

    required property var modelData
    readonly property bool focused: Quickshell.screens.length === 1
        || (Hyprland.focusedMonitor !== null && Hyprland.focusedMonitor.name === modelData.name)

    Backdrop {
        screen: scope.modelData
    }

    // Encaixe entre a ilha e o painel deste monitor: a ilha diz onde vai
    // estar, o painel diz de que largura precisa. O painel fica dentro da
    // borda de baixo da ilha, e a ilha alarga quando ele é mais largo.
    Bar {
        id: bar
        screen: scope.modelData
        sheetW: tide.sheetW
    }

    Tide {
        id: tide
        screen: scope.modelData
        active: scope.focused
        islandX: bar.islandX
        islandW: bar.islandW
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

    // Clique fora do painel fecha o painel: uma janela transparente pega o
    // clique na tela inteira, menos na ilha (lá quem decide é o item clicado:
    // troca de painel, ou fecha o que está aberto). O painel fica por cima
    // dela (Overlay). Com um app em tela cheia ela fica por baixo do app, e
    // aí quem fecha é o Esc.
    // (Antes isto era um HyprlandFocusGrab, que também entregava o teclado.
    // O Hyprland entrega o foco a uma janela qualquer da lista do grab, e
    // quando sorteava a ilha o que se digitava não chegava a ninguém. Agora o
    // painel pede o teclado ele mesmo: veja Tide.qml.)
    PanelWindow {
        id: outside
        screen: scope.modelData
        visible: Mare.mode !== "rest" && Mare.mode !== "picker" && !Mare.isTransient && !Mare.sheetMode
        color: "transparent"
        anchors {
            top: true
            bottom: true
            left: true
            right: true
        }
        // A tela inteira, também atrás da ilha (veja Mare.ignoreZone).
        exclusionMode: ExclusionMode.Normal
        exclusiveZone: Mare.ignoreZone
        WlrLayershell.layer: WlrLayer.Top
        WlrLayershell.namespace: "mare-outside"
        WlrLayershell.keyboardFocus: WlrKeyboardFocus.None

        mask: Region {
            item: everywhere
            Region {
                item: islandHole
                intersection: Intersection.Subtract
            }
        }

        Item {
            id: everywhere
            anchors.fill: parent
        }
        Item {
            id: islandHole
            x: bar.islandX
            width: bar.islandW
            height: bar.barH
        }

        TapHandler {
            acceptedButtons: Qt.AllButtons
            onTapped: Mare.close()
        }
    }
}
