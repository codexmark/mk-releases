import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "Emoji.js" as Emoji

// Emoji: digite para procurar (em português ou inglês), as setas andam pela
// grade, Enter escolhe, Tab troca de categoria, Ctrl + T troca o tom de pele,
// Esc fecha. Escolher copia e, se houver wtype, digita na janela de antes;
// Shift + Enter (ou o botão direito) só copia.
Item {
    id: panel

    readonly property int cols: 14
    readonly property int rows: 6
    readonly property int cell: 44
    property string query: ""
    property string tab: "rostos"
    property int sel: 0
    // Dá para digitar na janela de antes? (o wtype é opcional)
    property bool canType: false

    // Os últimos escolhidos, do mais novo ao mais antigo: duas fileiras. Lidos
    // uma vez, ao abrir: ligados ao Settings, trocar o tom (que regrava o
    // ajustes.json) refazia a lista e levava o cursor de volta ao primeiro.
    property var recent: []
    // Tom de pele: 0 = o amarelo padrão, 1 a 5 = do mais claro ao mais escuro.
    readonly property int tone: {
        const t = Number(Settings.get("emoji.tone"));
        return t >= 1 && t <= 5 ? Math.floor(t) : 0;
    }
    readonly property var tabs: [{ key: "recentes", name: "recentes" }].concat(Emoji.groups)
    readonly property bool searching: query.trim() !== ""
    readonly property var results: searching ? Emoji.search(query)
        : tab === "recentes" ? Emoji.fromChars(recent) : Emoji.list(tab)
    readonly property var current: results[sel] || null

    implicitWidth: cols * cell + 44
    implicitHeight: column.implicitHeight + 40

    onResultsChanged: {
        sel = 0;
        grid.contentY = 0;
    }
    onSelChanged: grid.positionViewAtIndex(sel, GridView.Contain)
    Component.onCompleted: {
        // Só uma lista de textos serve (o ajustes.json pode ter sido mexido à mão).
        const saved = Settings.get("emoji.recent");
        recent = Array.isArray(saved) ? saved.filter(c => typeof c === "string") : [];
        tab = recent.length > 0 ? "recentes" : "rostos";
        input.forceActiveFocus();
    }

    function move(delta) {
        const n = results.length;
        const next = sel + delta;
        if (next >= 0 && next < n)
            sel = next;
        // Para baixo sem ninguém embaixo, havendo mais uma fileira: o último.
        else if (next >= n && Math.floor(sel / cols) < Math.floor((n - 1) / cols))
            sel = n - 1;
    }

    function stepTab(step) {
        const keys = tabs.map(t => t.key);
        showTab(keys[(keys.indexOf(tab) + step + keys.length) % keys.length]);
    }

    // Trocar de categoria sai da busca.
    function showTab(key) {
        tab = key;
        if (input.text !== "")
            input.text = "";
    }

    function setTone(n) {
        Settings.set("emoji.tone", (n + 6) % 6);
    }

    // Copia (e digita na janela, se `paste`). Quem faz é o mare-emoji, solto
    // do Maré: ele espera o painel fechar e o teclado voltar para a janela.
    function pick(index, paste) {
        const e = results[index];
        if (!e)
            return;
        Settings.set("emoji.recent", [e.s].concat(recent.filter(c => c !== e.s)).slice(0, cols * 2));
        Quickshell.execDetached(["bash", Quickshell.shellPath("scripts/mare-emoji"), "pick",
            Emoji.withTone(e.c, tone), paste && canType ? "colar" : "copiar", e.n]);
        Mare.close();
    }

    Process {
        command: ["bash", Quickshell.shellPath("scripts/mare-emoji"), "tem-wtype"]
        running: true
        onExited: code => panel.canType = code === 0
    }

    ColumnLayout {
        id: column
        x: 22
        y: 22
        width: parent.width - 44
        spacing: 0

        PanelHeader {
            Layout.fillWidth: true
            index: "01"
            title: "emoji"
            meta: panel.searching ? panel.results.length + (panel.results.length === 1 ? " encontrado" : " encontrados")
                : (panel.tabs.find(t => t.key === panel.tab) || panel.tabs[1]).name + " · " + panel.results.length
        }

        // A busca, em serifa grande, como a do launcher.
        TextInput {
            id: input
            Layout.fillWidth: true
            Layout.topMargin: 10
            color: Theme.text
            selectionColor: Theme.copper
            selectedTextColor: Theme.ground
            font.family: Theme.serif
            font.pixelSize: 26
            font.weight: Font.Light
            cursorDelegate: Rectangle {
                width: 2
                color: Theme.copper
            }
            clip: true
            onTextChanged: panel.query = text

            Text {
                visible: input.text === ""
                text: "coração, festa, joinha, brasil"
                color: Theme.textFaint
                font.family: Theme.serif
                font.pixelSize: 26
                font.weight: Font.Light
                font.italic: true
            }

            Keys.onPressed: event => {
                let used = true;
                if (event.key === Qt.Key_T && (event.modifiers & Qt.ControlModifier))
                    panel.setTone(panel.tone + 1);
                else if (event.key === Qt.Key_Right)
                    panel.move(1);
                else if (event.key === Qt.Key_Left)
                    panel.move(-1);
                else if (event.key === Qt.Key_Down)
                    panel.move(panel.cols);
                else if (event.key === Qt.Key_Up)
                    panel.move(-panel.cols);
                else if (event.key === Qt.Key_PageDown)
                    panel.sel = Math.min(panel.sel + panel.cols * panel.rows, Math.max(0, panel.results.length - 1));
                else if (event.key === Qt.Key_PageUp)
                    panel.sel = Math.max(panel.sel - panel.cols * panel.rows, 0);
                else if (event.key === Qt.Key_Tab)
                    panel.stepTab(1);
                else if (event.key === Qt.Key_Backtab)
                    panel.stepTab(-1);
                else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter)
                    panel.pick(panel.sel, !(event.modifiers & Qt.ShiftModifier));
                else if (event.key === Qt.Key_Escape)
                    Mare.close();
                else
                    used = false;
                event.accepted = used;
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.topMargin: 12
            implicitHeight: 1
            color: Theme.line
        }

        // Categorias. Durante a busca ficam apagadas: ela procura em todas.
        Row {
            Layout.fillWidth: true
            Layout.topMargin: 4
            // O primeiro rótulo alinha com a busca (cada aba tem 7 de folga).
            Layout.leftMargin: -7
            spacing: 0

            Repeater {
                model: panel.tabs

                Item {
                    id: tabItem
                    required property var modelData
                    readonly property bool current: !panel.searching && panel.tab === modelData.key
                    width: tabLabel.implicitWidth + 14
                    height: 28

                    Text {
                        id: tabLabel
                        anchors.centerIn: parent
                        text: tabItem.modelData.name
                        color: tabItem.current ? Theme.copper : tabHover.containsMouse ? Theme.text : Theme.textDim
                        opacity: panel.searching ? 0.55 : 1
                        font.family: Theme.mono
                        font.pixelSize: 9
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: 1.2

                        Behavior on color {
                            ColorAnimation { duration: Motion.fast }
                        }
                    }

                    Rectangle {
                        visible: tabItem.current
                        anchors.bottom: parent.bottom
                        x: 7
                        width: parent.width - 14
                        height: 1
                        color: Theme.copper
                    }

                    MouseArea {
                        id: tabHover
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: {
                            panel.showTab(tabItem.modelData.key);
                            input.forceActiveFocus();
                        }
                    }
                }
            }
        }

        // A grade. Altura fixa: o painel não muda de forma enquanto se digita.
        Item {
            Layout.fillWidth: true
            Layout.topMargin: 6
            implicitHeight: panel.rows * panel.cell

            GridView {
                id: grid
                anchors.fill: parent
                cellWidth: panel.cell
                cellHeight: panel.cell
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                keyNavigationEnabled: false
                model: panel.results

                delegate: Item {
                    id: tile
                    required property var modelData
                    required property int index
                    readonly property bool current: index === panel.sel
                    width: panel.cell
                    height: panel.cell

                    Rectangle {
                        anchors.fill: parent
                        anchors.margins: 2
                        radius: 2
                        color: tile.current ? Theme.wash : "transparent"
                        border.width: 1
                        border.color: tile.current ? Theme.copper : "transparent"
                    }

                    Text {
                        anchors.centerIn: parent
                        text: Emoji.withTone(tile.modelData.c, panel.tone)
                        font.family: "Noto Color Emoji"
                        font.pixelSize: 24
                    }

                    // Só segue o mouse quando ele se move, como no launcher.
                    MouseArea {
                        anchors.fill: parent
                        hoverEnabled: true
                        acceptedButtons: Qt.LeftButton | Qt.RightButton
                        cursorShape: Qt.PointingHandCursor
                        onPositionChanged: panel.sel = tile.index
                        onClicked: mouse => panel.pick(tile.index, mouse.button === Qt.LeftButton)
                    }
                }
            }

            // Onde se está na categoria, quando ela não cabe inteira.
            Rectangle {
                visible: grid.visibleArea.heightRatio < 1
                x: parent.width + 6
                y: grid.visibleArea.yPosition * parent.height
                width: 2
                height: Math.max(12, grid.visibleArea.heightRatio * parent.height)
                color: Theme.line
            }

            Text {
                visible: panel.results.length === 0
                anchors.centerIn: parent
                text: panel.searching ? "nada encontrado para “" + panel.query.trim() + "”"
                    : "o que você escolher fica guardado aqui"
                color: Theme.textDim
                font.family: Theme.serif
                font.pixelSize: 15
                font.italic: true
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.topMargin: 8
            implicitHeight: 1
            color: Theme.hairline
        }

        // O nome do emoji sob o cursor e, à direita, o tom de pele.
        RowLayout {
            Layout.fillWidth: true
            Layout.topMargin: 6
            spacing: 12

            Text {
                Layout.fillWidth: true
                text: panel.current ? panel.current.n : ""
                color: Theme.textSoft
                font.family: Theme.serif
                font.pixelSize: 15
                font.italic: true
                elide: Text.ElideRight
            }

            Text {
                text: "tom"
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 9
                font.capitalization: Font.AllUppercase
                font.letterSpacing: Theme.labelSpacing
            }

            Row {
                spacing: 2

                Repeater {
                    model: 6

                    Item {
                        id: swatch
                        required property int index
                        readonly property bool current: index === panel.tone
                        width: 24
                        height: 26

                        Text {
                            anchors.centerIn: parent
                            text: Emoji.withTone("✋~", swatch.index)
                            opacity: swatch.current || swatchHover.containsMouse ? 1 : 0.55
                            font.family: "Noto Color Emoji"
                            font.pixelSize: 14
                        }

                        Rectangle {
                            visible: swatch.current
                            anchors.bottom: parent.bottom
                            x: 4
                            width: parent.width - 8
                            height: 1
                            color: Theme.copper
                        }

                        MouseArea {
                            id: swatchHover
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: {
                                panel.setTone(swatch.index);
                                input.forceActiveFocus();
                            }
                        }
                    }
                }
            }
        }

        Text {
            Layout.fillWidth: true
            Layout.topMargin: 6
            text: (panel.canType ? "enter cola na janela · shift + enter só copia" : "enter copia")
                + " · tab categoria · ctrl + t tom"
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 9
            font.capitalization: Font.AllUppercase
            font.letterSpacing: 1.2
            elide: Text.ElideRight
        }
    }
}
