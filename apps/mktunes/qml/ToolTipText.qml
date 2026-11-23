import QtQuick

// Dica pequena acima do dono, depois de meio segundo com o mouse em cima.
Rectangle {
    id: tip
    property string text
    property bool shown: false
    visible: opacity > 0
    opacity: 0
    z: 100
    anchors.bottom: parent.top
    anchors.bottomMargin: 6
    anchors.horizontalCenter: parent.horizontalCenter
    width: label.implicitWidth + 16
    height: label.implicitHeight + 8
    radius: 6
    color: Theme.raised
    border.color: Theme.line

    states: State { when: tip.shown; PropertyChanges { tip.opacity: 1 } }
    transitions: Transition { SequentialAnimation { PauseAnimation { duration: tip.shown ? 500 : 0 } NumberAnimation { property: "opacity"; duration: 120 } } }

    Text {
        id: label
        anchors.centerIn: parent
        text: tip.text
        color: Theme.text
        font.family: Theme.sans
        font.pixelSize: 12
    }
}
