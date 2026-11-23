import QtQuick
import QtQuick.Layouts

// Cartão de prateleira: capa quadrada (redonda para artista), título e subtítulo.
Item {
    id: card
    property var item: ({})
    property int size: 168
    signal activated()
    signal menu(var anchorItem)
    width: size
    height: size + 52
    Accessible.role: Accessible.Button
    Accessible.name: (item.title || "") + ", " + (item.subtitle || "")

    HoverHandler { id: hover; cursorShape: Qt.PointingHandCursor }
    TapHandler { acceptedButtons: Qt.LeftButton; onTapped: card.activated() }
    TapHandler { acceptedButtons: Qt.RightButton; onTapped: if (card.item.kind === "song") card.menu(card) }

    Cover {
        id: art
        width: card.size
        height: card.size
        source: card.item.artwork || ""
        round: card.item.kind === "artist"
        scale: hover.hovered ? 1.03 : 1
        Behavior on scale { NumberAnimation { duration: 140; easing.type: Easing.OutCubic } }
        Rectangle {
            visible: card.item.kind === "song" && hover.hovered
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            anchors.margins: 8
            width: 36; height: 36; radius: 18
            color: Theme.copper
            Text { anchors.centerIn: parent; text: Theme.icon.play; color: Theme.ground; font.family: Theme.symbols; font.pixelSize: 18 }
        }
    }
    Text {
        anchors.top: art.bottom
        anchors.topMargin: 8
        width: card.size
        text: card.item.title || ""
        color: Theme.text
        font.family: Theme.sans
        font.pixelSize: 13
        font.weight: Font.Medium
        elide: Text.ElideRight
        horizontalAlignment: card.item.kind === "artist" ? Text.AlignHCenter : Text.AlignLeft
    }
    Text {
        anchors.top: art.bottom
        anchors.topMargin: 28
        width: card.size
        text: card.item.subtitle || ""
        color: Theme.textDim
        font.family: Theme.sans
        font.pixelSize: 12
        elide: Text.ElideRight
        horizontalAlignment: card.item.kind === "artist" ? Text.AlignHCenter : Text.AlignLeft
    }
}
