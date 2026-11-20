import QtQuick
import Quickshell

// Ícone de uma notificação: a imagem que ela trouxe, o ícone do app,
// ou a inicial do app num círculo na cor do projeto.
Rectangle {
    id: icon

    property var notification: null
    readonly property string source: {
        if (!notification)
            return "";
        if (notification.image !== "")
            return notification.image;
        if (notification.appIcon !== "")
            return notification.appIcon.startsWith("/") || notification.appIcon.indexOf("://") > 0
                ? notification.appIcon : Quickshell.iconPath(notification.appIcon, true);
        return "";
    }

    implicitWidth: 36
    implicitHeight: 36
    radius: 2
    color: Theme.surfaceLow
    border.width: 1
    border.color: Theme.line
    clip: true

    Image {
        anchors.fill: parent
        anchors.margins: icon.notification && icon.notification.image !== "" ? 0 : 6
        visible: icon.source !== ""
        source: icon.source
        fillMode: Image.PreserveAspectCrop
        sourceSize.width: 72
        sourceSize.height: 72
    }

    Text {
        anchors.centerIn: parent
        visible: icon.source === ""
        text: icon.notification && icon.notification.appName ? icon.notification.appName[0].toUpperCase() : "•"
        color: Projects.accent
        font.family: Theme.sans
        font.pixelSize: 15
        font.weight: Font.DemiBold
    }
}
