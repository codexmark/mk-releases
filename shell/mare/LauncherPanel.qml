import QtQuick
import "Calc.js" as Calc
import QtQuick.Layouts
import Quickshell
import Quickshell.Hyprland

// Launcher: comandos do projeto em foco, projetos de ~/Work e aplicativos,
// numa busca só. ↑/↓ escolhe, Enter abre, Esc fecha.
Item {
    id: panel

    readonly property int rows: 7
    property string query: ""
    property int sel: 0

    readonly property var apps: DesktopEntries.applications.values
        .filter(e => !e.noDisplay)
        .map(e => ({
            title: e.name,
            detail: e.genericName || e.comment || "",
            icon: Quickshell.iconPath(e.icon, true),
            glyph: "",
            accent: Theme.textSoft,
            run: () => e.execute()
        }))
        .sort((a, b) => a.title.localeCompare(b.title))

    // No projeto em foco quando há um; senão, na pasta pessoal.
    readonly property bool inProject: Projects.current !== null && !Projects.free
    readonly property var projectCommands: [
        { cmd: "lazygit", what: "git" },
        { cmd: "lazydocker", what: "containers" },
        { cmd: "terminal", what: "terminal" }
    ].map(c => ({
        title: c.cmd,
        detail: inProject ? c.what + " · " + Projects.name : c.what + " em ~",
        icon: "",
        glyph: "$",
        accent: Projects.accent,
        run: () => Projects.openHere(c.cmd === "terminal" ? "" : c.cmd)
    }))

    // Painéis do Maré que não têm botão na barra.
    readonly property var shellCommands: [
        { title: "ajustes", detail: "ilha, fundo, janelas, teclado e mouse", glyph: "󰒓", panel: "settings" },
        { title: "capturar a tela", detail: "área, janela, tela inteira ou vídeo", glyph: "󰄀", panel: "capture" },
        { title: "papel de parede", detail: "trocar a imagem e as cores", glyph: "󰸉", panel: "wallpaper" },
        { title: "área de transferência", detail: "histórico do que você copiou", glyph: "󰅌", panel: "clipboard" },
        { title: "emoji", detail: "procurar, copiar e colar na janela", glyph: "󰇲", panel: "emoji" },
        { title: "sistema", detail: "CPU, memória, temperatura, rede", glyph: "󰍛", panel: "system" },
        { title: "tela", detail: "brilho e luz noturna", glyph: "󰃠", panel: "display" },
        { title: "rede", detail: "Wi-Fi e Ethernet", glyph: "󰖩", panel: "network" },
        { title: "bluetooth", detail: "aparelhos e pareamento", glyph: "󰂯", panel: "bluetooth" },
        { title: "sessão", detail: "bloquear, sair, reiniciar, desligar", glyph: "󰐥", panel: "session" },
        { title: "boas-vindas", detail: "o passeio pelo Mk, de novo", glyph: "\u{f02d7}", welcome: true }
    ].map(c => ({
        title: c.title,
        detail: c.detail,
        icon: "",
        glyph: c.glyph,
        accent: Projects.accent,
        run: () => c.welcome ? Welcome.open() : Mare.open(c.panel)
    }))

    readonly property var projects: Projects.dirs.map(d => {
        const ws = Object.keys(Projects.bindings).find(k => Projects.bindings[k].name === d.name);
        return {
            title: d.name,
            detail: ws ? "projeto · workspace " + ws : "projeto · abrir em workspace novo",
            icon: "",
            glyph: "→",
            accent: Theme.accentFor(d.name),
            run: () => panel.goToProject(d, ws)
        };
    })

    readonly property var results: {
        const q = query.trim().toLowerCase();
        const all = projectCommands.concat(shellCommands, projects, apps);
        if (q === "")
            return all.slice(0, rows);
        const found = all
            .map(item => ({ item: item, score: Math.max(score(q, item.title), score(q, item.detail) * 0.5) }))
            .filter(r => r.score > 0)
            .sort((a, b) => b.score - a.score)
            .slice(0, rows)
            .map(r => r.item);
        // Uma conta? O resultado vem primeiro; Enter copia.
        const value = Calc.evaluate(query);
        if (value === null)
            return found;
        const shown = Calc.format(value);
        return [{
            title: "= " + shown,
            detail: "enter copia o resultado",
            icon: "",
            glyph: "󰃬",
            accent: Theme.copper,
            run: () => Quickshell.execDetached(["wl-copy", shown.replace("−", "-")])
        }].concat(found).slice(0, rows);
    }

    implicitWidth: 620
    // A forma acompanha o número de resultados enquanto você digita.
    implicitHeight: column.implicitHeight + 40

    onQueryChanged: sel = 0
    // A busca fica carregada o tempo todo (Tide.qml), para abrir na hora:
    // cada vez que aparece, começa limpa e com o cursor no campo.
    function shown() {
        // Só mexe no que mudou: trocar o texto refaz a lista de resultados,
        // e refazer as linhas é o que pesa numa máquina lenta.
        if (input.text !== "")
            input.text = "";
        sel = 0;
        Projects.refreshDirs();
        input.forceActiveFocus();
    }

    // Prefixo > início de palavra > trecho > letras na ordem.
    function score(q, text) {
        const s = (text || "").toLowerCase();
        const i = s.indexOf(q);
        if (i === 0)
            return 100;
        if (i > 0)
            return " -_.".indexOf(s[i - 1]) >= 0 ? 80 : 60;
        let j = 0;
        for (let k = 0; k < s.length && j < q.length; k++)
            if (s[k] === q[j])
                j++;
        return j === q.length ? 30 : 0;
    }

    // Projeto já ligado a um workspace: vai até ele. Senão, usa o primeiro
    // workspace livre, liga ao projeto e abre um terminal lá.
    function goToProject(d, ws) {
        if (ws) {
            Hypr.workspace(ws);
            return;
        }
        const used = Hyprland.workspaces.values.map(w => w.id);
        let id = 1;
        while (used.indexOf(id) >= 0 || Projects.bindings[String(id)])
            id++;
        Projects.bind(id, d.name, d.dir);
        Hypr.workspace(id);
        Projects.openIn(d.dir, "");
    }

    function launch(index) {
        const item = results[index];
        if (!item)
            return;
        Mare.close();
        // Um comando que abre outro painel o faz depois de fechar este.
        Qt.callLater(item.run);
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
            title: "buscar"
            meta: Projects.current && !Projects.free ? "folha " + Theme.wsName(Projects.focusedId) + " · " + Projects.name : "Mk"
        }

        // A busca, em serifa grande, como na prancha da direção Folha.
        TextInput {
            id: input
            Layout.fillWidth: true
            Layout.topMargin: 10
            color: Theme.text
            selectionColor: Theme.copper
            selectedTextColor: Theme.ground
            font.family: Theme.serif
            font.pixelSize: 30
            font.weight: Font.Light
            cursorDelegate: Rectangle {
                width: 2
                color: Theme.copper
            }
            clip: true
            onTextChanged: panel.query = text

            Text {
                visible: input.text === ""
                text: "apps, projetos, comandos"
                color: Theme.textFaint
                font.family: Theme.serif
                font.pixelSize: 30
                font.weight: Font.Light
                font.italic: true
            }

            Keys.onPressed: event => {
                if (event.key === Qt.Key_Down || event.key === Qt.Key_Tab) {
                    panel.sel = Math.min(panel.sel + 1, panel.results.length - 1);
                    event.accepted = true;
                } else if (event.key === Qt.Key_Up || event.key === Qt.Key_Backtab) {
                    panel.sel = Math.max(panel.sel - 1, 0);
                    event.accepted = true;
                } else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                    panel.launch(panel.sel);
                    event.accepted = true;
                } else if (event.key === Qt.Key_Escape) {
                    Mare.close();
                    event.accepted = true;
                }
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.topMargin: 12
            Layout.bottomMargin: 4
            implicitHeight: 1
            color: Theme.line
        }

        // Resultados numerados, com linha guia até o detalhe.
        Repeater {
            model: panel.results

            Item {
                id: row
                required property var modelData
                required property int index
                readonly property bool current: index === panel.sel
                Layout.fillWidth: true
                implicitHeight: 44

                Rectangle {
                    anchors.fill: parent
                    color: row.current ? Theme.wash : "transparent"

                    Behavior on color {
                        ColorAnimation { duration: Motion.fast }
                    }
                }

                Rectangle {
                    visible: row.current
                    width: 2
                    height: parent.height
                    color: Theme.copper
                }

                Rectangle {
                    visible: row.index > 0
                    width: parent.width
                    height: 1
                    color: Theme.hairline
                    opacity: 0.6
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 14
                    anchors.rightMargin: 10
                    spacing: 14

                    Text {
                        text: String(row.index + 1).padStart(2, "0")
                        color: row.current ? Theme.copper : Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 11
                        font.letterSpacing: 1.2
                    }

                    Item {
                        implicitWidth: 20
                        implicitHeight: 20

                        Image {
                            anchors.centerIn: parent
                            visible: row.modelData.icon !== ""
                            source: row.modelData.icon
                            sourceSize.width: 20
                            sourceSize.height: 20
                        }

                        Text {
                            anchors.centerIn: parent
                            visible: row.modelData.icon === ""
                            text: row.modelData.glyph
                            color: row.current ? Theme.copper : Theme.textDim
                            font.family: Theme.icons
                            font.pixelSize: 15
                        }
                    }

                    Text {
                        text: row.modelData.title
                        color: row.current ? Theme.text : Theme.textSoft
                        font.family: Theme.sans
                        font.pixelSize: 14
                        elide: Text.ElideRight
                        Layout.maximumWidth: 240
                    }

                    Rectangle {
                        Layout.fillWidth: true
                        Layout.minimumWidth: 12
                        implicitHeight: 1
                        color: Theme.hairline
                    }

                    Text {
                        text: row.modelData.detail
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: 1.2
                        elide: Text.ElideRight
                        horizontalAlignment: Text.AlignRight
                        Layout.maximumWidth: 230
                    }
                }

                // Só segue o mouse quando ele se move: o ponteiro parado sobre a
                // lista ao abrir não rouba a seleção do teclado.
                MouseArea {
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onPositionChanged: panel.sel = row.index
                    onClicked: panel.launch(row.index)
                }
            }
        }

        Text {
            visible: panel.results.length === 0
            Layout.topMargin: 12
            Layout.bottomMargin: 8
            text: "nada encontrado para “" + panel.query + "”"
            color: Theme.textDim
            font.family: Theme.serif
            font.pixelSize: 15
            font.italic: true
        }
    }
}
