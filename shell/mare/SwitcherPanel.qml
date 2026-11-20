import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Hyprland
import Quickshell.Wayland

// Alt+Tab: cartões com miniatura ao vivo de cada janela, na ordem de uso
// (a mais recente primeiro). Tab avança, soltar o Alt ou Enter abre, Esc cancela.
Item {
    id: panel

    readonly property int maxCards: 7
    readonly property int cardWidth: 172

    readonly property var windows: Hyprland.toplevels.values
        .filter(t => t.workspace && t.workspace.id > 0)
        .slice()
        .sort((a, b) => focusOrder(a) - focusOrder(b))
    readonly property int count: windows.length
    readonly property int sel: count > 0 ? ((Mare.switchIndex % count) + count) % count : 0
    readonly property int start: Math.max(0, Math.min(sel - 3, count - maxCards))
    readonly property var shown: windows.slice(start, start + maxCards)

    implicitWidth: Math.max(320, 44 + shown.length * (cardWidth + 12) - 12)
    implicitHeight: 218

    focus: true
    Component.onCompleted: {
        Hyprland.refreshToplevels();
        forceActiveFocus();
    }

    function focusOrder(t) {
        const id = t.lastIpcObject ? t.lastIpcObject.focusHistoryID : undefined;
        return id === undefined ? 999 : id;
    }

    function confirm(index) {
        const t = windows[index === undefined ? sel : index];
        Mare.close();
        if (!t)
            return;
        Hypr.focusWindow(t.address);
    }

    Keys.onPressed: event => {
        if (event.key === Qt.Key_Tab) {
            Mare.switchStep(1);
            event.accepted = true;
        } else if (event.key === Qt.Key_Backtab) {
            Mare.switchStep(-1);
            event.accepted = true;
        } else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
            confirm();
            event.accepted = true;
        } else if (event.key === Qt.Key_Escape) {
            Mare.close();
            event.accepted = true;
        }
    }

    // Soltar o Alt confirma, como no Alt+Tab de sempre.
    Keys.onReleased: event => {
        if (event.key === Qt.Key_Alt) {
            confirm();
            event.accepted = true;
        }
    }

    ColumnLayout {
        x: 22
        y: 18
        width: parent.width - 44
        spacing: 14

        Row {
            spacing: 12

            Repeater {
                model: panel.shown

                Rectangle {
                    id: card

                    required property var modelData
                    required property int index
                    readonly property int realIndex: panel.start + index
                    readonly property bool selected: realIndex === panel.sel
                    readonly property var binding: modelData.workspace ? Projects.bindings[String(modelData.workspace.id)] || null : null
                    readonly property color accent: binding ? Theme.accentFor(binding.name) : Theme.neutral
                    readonly property string appClass: modelData.lastIpcObject && modelData.lastIpcObject.class ? modelData.lastIpcObject.class : ""
                    readonly property var entry: appClass !== "" ? DesktopEntries.heuristicLookup(appClass) : null

                    width: panel.cardWidth
                    height: 150
                    radius: 2
                    color: Theme.surfaceLow
                    border.width: 1.5
                    border.color: selected ? accent : Theme.line
                    transform: Translate {
                        y: card.selected ? -4 : 0

                        Behavior on y {
                            NumberAnimation {
                                duration: Motion.fast + 40
                                easing.type: Easing.BezierSpline
                                easing.bezierCurve: Motion.curve
                            }
                        }
                    }

                    Behavior on border.color {
                        ColorAnimation { duration: Motion.fast + 40 }
                    }

                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: 10
                        spacing: 8

                        Rectangle {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 84
                            radius: 6
                            color: Theme.ground
                            clip: true

                            ScreencopyView {
                                id: thumb
                                anchors.fill: parent
                                captureSource: card.modelData.wayland
                                live: false
                                constraintSize: Qt.size(width, height)
                            }

                            Image {
                                anchors.centerIn: parent
                                visible: !thumb.hasContent
                                source: card.entry ? Quickshell.iconPath(card.entry.icon, true) : ""
                                sourceSize.width: 36
                                sourceSize.height: 36
                            }
                        }

                        Text {
                            Layout.fillWidth: true
                            // Janelas desenhadas pelo próprio Quickshell (widgets, players) se
                            // apresentam pelo título; o resto pelo nome do app.
                            text: card.entry && card.appClass !== "org.quickshell" ? card.entry.name : (card.modelData.title || card.appClass)
                            color: Theme.text
                            font.family: Theme.mono
                            font.pixelSize: 12
                            elide: Text.ElideRight
                        }

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: 6

                            Rectangle {
                                implicitWidth: 6
                                implicitHeight: 6
                                radius: 3
                                color: card.accent
                            }

                            Text {
                                Layout.fillWidth: true
                                text: (card.binding ? card.binding.name : "livre") + " · " + (card.modelData.workspace ? card.modelData.workspace.id : "")
                                color: Theme.textDim
                                font.family: Theme.mono
                                font.pixelSize: 11
                                elide: Text.ElideRight
                            }
                        }
                    }

                    HoverHandler {
                        cursorShape: Qt.PointingHandCursor
                    }

                    TapHandler {
                        onTapped: panel.confirm(card.realIndex)
                    }
                }
            }
        }

        Text {
            text: panel.count === 0
                ? "Nenhuma janela aberta"
                : "Tab avança · soltar Alt abre · " + (panel.sel + 1) + " de " + panel.count + " janelas"
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 12
        }
    }
}
