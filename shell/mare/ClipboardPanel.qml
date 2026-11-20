import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

// Histórico da área de transferência (cliphist). Digite para filtrar,
// ↑/↓ escolhe, Enter copia de volta, Del apaga o item, Esc fecha.
Item {
    id: panel

    readonly property int rows: 8
    property var entries: []
    property string query: ""
    property int sel: 0
    property bool armedWipe: false

    readonly property var results: {
        const q = query.trim().toLowerCase();
        const list = q === "" ? entries : entries.filter(e => e.text.toLowerCase().indexOf(q) >= 0);
        return list.slice(0, rows);
    }

    implicitWidth: 620
    implicitHeight: 18 + 46 + 10 + Math.max(1, results.length) * 42 + 10 + 34 + 18

    onQueryChanged: sel = 0
    Component.onCompleted: {
        load();
        input.forceActiveFocus();
    }

    function load() {
        lister.running = true;
    }

    // cliphist list: "<id>\t<prévia>"; imagens vêm como [[ binary data ... ]]
    function parse(text) {
        entries = text.split("\n").filter(l => l.indexOf("\t") > 0).map(l => {
            const tab = l.indexOf("\t");
            const raw = l.substring(tab + 1);
            const img = raw.match(/^\[\[ binary data .*?(png|jpe?g|webp|gif|bmp)\s*(\d+x\d+)?/i);
            return {
                line: l,
                id: l.substring(0, tab),
                text: img ? "imagem " + img[1] + (img[2] ? " " + img[2].replace("x", "×") : "") : raw.replace(/\s+/g, " ").trim(),
                image: img !== null
            };
        });
    }

    function copy(index) {
        const e = results[index];
        if (!e || !/^\d+$/.test(e.id))
            return;
        Quickshell.execDetached(["sh", "-c", "cliphist decode " + e.id + " | wl-copy"]);
        Mare.close();
    }

    function remove(index) {
        const e = results[index];
        if (!e)
            return;
        deleter.input = e.line;
        deleter.running = true;
    }

    function wipe() {
        if (!armedWipe) {
            armedWipe = true;
            disarm.restart();
            return;
        }
        armedWipe = false;
        wiper.running = true;
    }

    Timer {
        id: disarm
        interval: 3000
        onTriggered: panel.armedWipe = false
    }

    Process {
        id: lister
        command: ["cliphist", "list"]
        stdout: StdioCollector {
            onStreamFinished: panel.parse(text)
        }
    }

    // cliphist delete lê a linha exata pela entrada padrão.
    Process {
        id: deleter
        property string input: ""
        command: ["sh", "-c", "printf '%s\\n' \"$1\" | cliphist delete", "sh", input]
        onExited: panel.load()
    }

    Process {
        id: wiper
        command: ["cliphist", "wipe"]
        onExited: panel.load()
    }

    ColumnLayout {
        x: 18
        y: 18
        width: parent.width - 36
        spacing: 10

        Rectangle {
            Layout.fillWidth: true
            implicitHeight: 46
            radius: 2
            color: Theme.surfaceLow
            border.width: 1
            border.color: Theme.line

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 14
                anchors.rightMargin: 14
                spacing: 10

                Text {
                    text: "󰅌"
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 16
                }

                TextInput {
                    id: input
                    Layout.fillWidth: true
                    color: Theme.text
                    selectionColor: Projects.accent
                    font.family: Theme.sans
                    font.pixelSize: 16
                    clip: true
                    onTextChanged: panel.query = text

                    Text {
                        visible: input.text === ""
                        text: "Buscar no que você copiou"
                        color: Theme.textFaint
                        font: input.font
                    }

                    Keys.onPressed: event => {
                        if (event.key === Qt.Key_Down || event.key === Qt.Key_Tab) {
                            panel.sel = Math.min(panel.sel + 1, panel.results.length - 1);
                            event.accepted = true;
                        } else if (event.key === Qt.Key_Up || event.key === Qt.Key_Backtab) {
                            panel.sel = Math.max(panel.sel - 1, 0);
                            event.accepted = true;
                        } else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                            panel.copy(panel.sel);
                            event.accepted = true;
                        } else if (event.key === Qt.Key_Delete) {
                            panel.remove(panel.sel);
                            event.accepted = true;
                        } else if (event.key === Qt.Key_Escape) {
                            Mare.close();
                            event.accepted = true;
                        }
                    }
                }
            }
        }

        Repeater {
            model: panel.results

            Rectangle {
                id: row
                required property var modelData
                required property int index
                Layout.fillWidth: true
                implicitHeight: 40
                radius: 2
                color: index === panel.sel ? Theme.raised : "transparent"

                Behavior on color {
                    ColorAnimation { duration: Motion.fast }
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 12
                    anchors.rightMargin: 12
                    spacing: 12

                    Text {
                        text: row.modelData.image ? "󰋩" : "󰦨"
                        color: row.index === panel.sel ? Projects.accent : Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 14
                    }

                    Text {
                        text: row.modelData.text
                        color: Theme.text
                        font.family: row.modelData.image ? Theme.sans : Theme.mono
                        font.pixelSize: 13
                        elide: Text.ElideRight
                        Layout.fillWidth: true
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onPositionChanged: panel.sel = row.index
                    onClicked: panel.copy(row.index)
                }
            }
        }

        Text {
            visible: panel.results.length === 0
            text: panel.entries.length === 0 ? "Nada copiado ainda." : "Nada encontrado para \"" + panel.query + "\""
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 13
            Layout.leftMargin: 12
            Layout.preferredHeight: 42
            verticalAlignment: Text.AlignVCenter
        }

        RowLayout {
            Layout.fillWidth: true

            Text {
                text: panel.entries.length + " itens · Enter copia · Del apaga"
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 11
                Layout.fillWidth: true
            }

            PillButton {
                text: panel.armedWipe ? "confirmar" : "limpar tudo"
                selected: panel.armedWipe
                onClicked: panel.wipe()
            }
        }
    }
}
