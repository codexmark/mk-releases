import QtQuick
import Quickshell
import Quickshell.Wayland

// O bloqueio de sessão do Wayland: enquanto Lock.locked, cada tela ganha uma
// LockSurface e nada mais recebe teclado ou mouse. Se o Maré cair com a tela
// bloqueada, o Hyprland segura o bloqueio (misc.allow_session_lock_restore
// deixa um Maré novo assumir).
WlSessionLock {
    locked: Lock.locked

    WlSessionLockSurface {
        color: Theme.ground

        LockSurface {
            anchors.fill: parent
        }
    }
}
