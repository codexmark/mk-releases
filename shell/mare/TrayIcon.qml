import QtQuick
import Quickshell
import Quickshell.Services.SystemTray

// Um app da bandeja (Discord, Steam, Nextcloud...). Clique abre o app,
// botão direito abre o menu dele, roda do mouse repassa para o app.
Rectangle {
    id: tray

    required property var item
    required property var barWindow

    width: 26
    height: 26
    radius: 2
    color: hover.hovered ? Theme.raised : "transparent"

    Behavior on color {
        ColorAnimation { duration: Motion.fast }
    }

    Image {
        anchors.centerIn: parent
        source: tray.item.icon
        sourceSize.width: 16
        sourceSize.height: 16
        opacity: tray.item.status === Status.Passive ? 0.6 : 1
    }

    // Pedindo atenção: um ponto na cor do projeto.
    Rectangle {
        visible: tray.item.status === Status.NeedsAttention
        anchors.top: parent.top
        anchors.right: parent.right
        anchors.margins: 3
        width: 5
        height: 5
        radius: 3
        color: Projects.accent
    }

    function openMenu() {
        const p = tray.mapToItem(null, 0, 0);
        tray.item.display(tray.barWindow, p.x, p.y);
    }

    HoverHandler {
        id: hover
        cursorShape: Qt.PointingHandCursor
    }

    TapHandler {
        acceptedButtons: Qt.LeftButton | Qt.RightButton | Qt.MiddleButton
        onTapped: (point, button) => {
            if (button === Qt.RightButton || (button === Qt.LeftButton && tray.item.onlyMenu))
                tray.openMenu();
            else if (button === Qt.MiddleButton)
                tray.item.secondaryActivate();
            else
                tray.item.activate();
        }
    }

    WheelHandler {
        onWheel: event => tray.item.scroll(event.angleDelta.y, false)
    }
}
