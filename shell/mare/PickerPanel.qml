import QtQuick
import QtQuick.Layouts

// Escolha um projeto de ~/Work para este workspace (os que já estão em outro
// não aparecem) ou deixe livre. Só abre quando chamado (toggle picker): num
// workspace vazio quem convida a começar é a folha do Backdrop.
Item {
    id: panel

    readonly property var taken: Object.keys(Projects.bindings).map(k => Projects.bindings[k].name)
    readonly property var choices: Projects.dirs.filter(d => taken.indexOf(d.name) < 0).slice(0, 7)

    implicitWidth: 460
    implicitHeight: column.implicitHeight + 44

    Component.onCompleted: Projects.refreshDirs()

    function pick(name, dir) {
        Projects.bind(Projects.focusedId, name, dir);
        Projects.openIn(dir, "");
        Mare.close();
    }

    ColumnLayout {
        id: column
        x: 22
        y: 22
        width: parent.width - 44
        spacing: 12

        RowLayout {
            spacing: 10

            Text {
                text: "Workspace " + Projects.focusedId
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 24
                font.weight: Font.Light
            }

            Text {
                text: Projects.windowCount > 0 ? "sem projeto" : "novo"
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
            }
        }

        Text {
            Layout.fillWidth: true
            text: "O que vai acontecer aqui? Escolha um projeto de ~/Work ou deixe livre."
            color: Theme.textSoft
            font.family: Theme.sans
            font.pixelSize: 14
            wrapMode: Text.WordWrap
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            Repeater {
                // desc: a pasta onde o repositório está, relativa a ~/Work (ex.: projetos/kram-all)
                model: panel.choices.map(d => ({ name: d.name, dir: d.dir, desc: d.dir.replace(Projects.workDir + "/", "").replace(/\/?[^\/]+$/, "") || "~/Work" }))
                    .concat([{ name: "livre", dir: "", desc: "sem projeto" }])

                Rectangle {
                    id: row
                    required property var modelData
                    Layout.fillWidth: true
                    implicitHeight: 44
                    radius: 2
                    color: hover.hovered ? Theme.raised : "transparent"

                    Behavior on color {
                        ColorAnimation { duration: Motion.fast }
                    }

                    RowLayout {
                        anchors.fill: parent
                        anchors.leftMargin: 12
                        anchors.rightMargin: 12
                        spacing: 12

                        Rectangle {
                            width: 18
                            height: 7
                            radius: 4
                            color: Theme.accentFor(row.modelData.name)
                        }

                        Text {
                            text: row.modelData.name
                            color: Theme.text
                            font.family: Theme.sans
                            font.pixelSize: 14
                            elide: Text.ElideRight
                            Layout.fillWidth: true
                        }

                        Text {
                            text: row.modelData.desc
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 12
                            elide: Text.ElideLeft
                            horizontalAlignment: Text.AlignRight
                            Layout.maximumWidth: 190
                        }
                    }

                    HoverHandler {
                        id: hover
                        cursorShape: Qt.PointingHandCursor
                    }

                    TapHandler {
                        onTapped: panel.pick(row.modelData.name, row.modelData.dir)
                    }
                }
            }
        }

        Text {
            visible: Projects.dirs.length === 0
            text: "Nenhum projeto em ~/Work ainda."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 13
        }
    }
}
