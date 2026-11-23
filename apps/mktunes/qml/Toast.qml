import QtQuick

// Aviso curto que some sozinho em 3 s; um novo substitui o anterior.
Rectangle {
    id: toast
    function show(text) {
        label.text = text;
        toast.opacity = 1;
        hide.restart();
    }
    opacity: 0
    visible: opacity > 0
    width: Math.min(label.implicitWidth + 32, 560)
    height: 40
    radius: 20
    color: Theme.raised
    border.color: Theme.line
    Behavior on opacity { NumberAnimation { duration: 160 } }
    Accessible.role: Accessible.AlertMessage
    Accessible.name: label.text
    Text {
        id: label
        anchors.centerIn: parent
        width: parent.width - 32
        horizontalAlignment: Text.AlignHCenter
        elide: Text.ElideMiddle
        color: Theme.text
        font.family: Theme.sans
        font.pixelSize: 13
    }
    Timer { id: hide; interval: 3000; onTriggered: toast.opacity = 0 }
}
