import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

// Página "Apps padrão" dos Ajustes: que app abre cada tipo de coisa. Cada
// linha mostra os apps instalados que servem para aquilo; o de cobre é o
// padrão de hoje, e clicar em outro troca na hora. Quem lê os .desktop e fala
// com o xdg-mime é o scripts/mare-default-apps (os padrões ficam em
// ~/.config/mimeapps.list, onde todo app procura). O terminal é a exceção: não
// há padrão do XDG para ele, então a escolha fica no ajustes.json
// ("apps.terminal") e quem a segue é o Projects.qml.
ColumnLayout {
    id: page
    spacing: 0

    // O que o script devolveu: papéis por id, terminais e se há xdg-terminal-exec.
    property var roles: ({})
    property var terminals: []
    property bool terminalExec: true
    property bool loaded: false
    property string failure: ""
    // O papel sendo trocado agora ("" = nenhum).
    property string pending: ""

    readonly property var sections: [
        { name: "DIA A DIA", rows: [
            { id: "navegador", label: "Navegador", desc: "Links e páginas da web" },
            { id: "arquivos", label: "Arquivos", desc: "Pastas, pendrives e “mostrar na pasta”" },
            { id: "terminal", label: "Terminal", desc: "Super + Enter, lazygit, lazydocker e o que o Maré abre" },
            { id: "editor", label: "Editor de texto", desc: "Texto puro, scripts e código" } ] },
        { name: "MÍDIA E DOCUMENTOS", rows: [
            { id: "imagens", label: "Imagens", desc: "Fotos, capturas de tela, PNG, JPEG, WebP" },
            { id: "video", label: "Vídeo", desc: "MP4, MKV, WebM e as gravações da tela" },
            { id: "musica", label: "Música", desc: "MP3, FLAC, Ogg e outros áudios" },
            { id: "pdf", label: "PDF", desc: "Documentos em PDF" } ] },
        { name: "CORREIO", rows: [
            { id: "email", label: "E-mail", desc: "Links de e-mail (mailto:)" } ] }
    ]

    // Algum papel está com um app de terminal (Neovim, Yazi) como padrão?
    readonly property bool terminalAppChosen: {
        for (const id in roles) {
            const r = roles[id];
            if (r.apps.some(a => a.id === r.current && a.terminal))
                return true;
        }
        return false;
    }

    // As escolhas de uma linha: [{ v, label }].
    function optionsFor(id) {
        if (id === "terminal")
            return terminals.filter(t => t.installed || t.id === Projects.terminal)
                .map(t => ({ v: t.id, label: t.name }));
        const r = roles[id];
        return r ? r.apps.map(a => ({ v: a.id, label: a.name })) : [];
    }

    function currentOf(id) {
        if (id === "terminal")
            return Projects.terminal;
        return roles[id] ? roles[id].current : "";
    }

    // O padrão de hoje quando ele não está entre as escolhas (um app que não
    // se encaixa no papel, ou que foi desinstalado): mostrado por nome.
    function strayOf(id) {
        const r = roles[id];
        if (id === "terminal" || !r || r.current === "" || r.apps.some(a => a.id === r.current))
            return "";
        return r.currentName;
    }

    function pick(id, value) {
        if (value === currentOf(id))
            return;
        if (id === "terminal") {
            Settings.set("apps.terminal", value);
            failure = "";
            syncTerminalList();
            return;
        }
        // Uma troca por vez: o xdg-mime reescreve o mimeapps.list inteiro.
        if (setter.running)
            return;
        failure = "";
        pending = id;
        setter.command = ["bash", Quickshell.shellPath("scripts/mare-default-apps"), "set", id, value];
        setter.running = true;
    }

    // O terminal escolhido no alto do xdg-terminals.list. Se o script ainda
    // está gravando uma escolha anterior, o onExited dele chama de novo: ligar
    // um Process que já roda não faz nada, e a segunda escolha se perderia.
    function syncTerminalList() {
        if (terminalList.running || terminalList.written === Projects.terminal)
            return;
        terminalList.written = Projects.terminal;
        terminalList.command = ["bash", Quickshell.shellPath("scripts/mare-default-apps"), "terminal", terminalList.written];
        terminalList.running = true;
    }

    function take(text) {
        try {
            const data = JSON.parse(text);
            const next = {};
            for (const r of data.roles)
                next[r.id] = r;
            roles = next;
            terminals = data.terminals;
            terminalExec = data.terminalExec;
            loaded = true;
            return true;
        } catch (e) {
            return false;
        }
    }

    Component.onCompleted: lister.running = true

    Process {
        id: lister
        command: ["bash", Quickshell.shellPath("scripts/mare-default-apps"), "list"]
        stdout: StdioCollector {
            onStreamFinished: {
                if (!page.take(text))
                    page.failure = "não deu para ler os apps instalados";
            }
        }
    }

    // `set` devolve a mesma lista, já com o padrão novo.
    Process {
        id: setter
        stdout: StdioCollector {
            onStreamFinished: {
                if (text.trim() !== "")
                    page.take(text);
            }
        }
        stderr: StdioCollector {
            id: setterErr
        }
        onExited: (code, status) => {
            if (code !== 0)
                page.failure = setterErr.text.trim().replace(/^mare-default-apps: /, "") || "não deu para trocar o padrão";
            page.pending = "";
        }
    }

    // O terminal escolhido também vai para o xdg-terminals.list, para os apps
    // de terminal abertos por outros apps.
    Process {
        id: terminalList
        // O último terminal mandado para o arquivo ("" = nenhum ainda).
        property string written: ""
        onExited: (code, status) => {
            if (code !== 0)
                page.failure = "o terminal mudou no Mk, mas não deu para gravar o ~/.config/xdg-terminals.list";
            page.syncTerminalList();
        }
    }

    // Rótulo de seção: mono pequeno em maiúsculas espaçadas.
    component Section: Text {
        Layout.topMargin: 22
        Layout.bottomMargin: 4
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.letterSpacing: Theme.labelSpacing
    }

    // Uma linha: o papel à esquerda, os apps que servem à direita (encostados
    // na borda; se não couberem numa linha, quebram).
    component Line: Item {
        id: ln
        property var row: ({})
        readonly property var options: page.optionsFor(row.id)
        readonly property string current: page.currentOf(row.id)
        readonly property string stray: page.strayOf(row.id)
        readonly property real labelWidth: Math.min(230, width * 0.36)

        Layout.fillWidth: true
        implicitHeight: Math.max(62, flow.implicitHeight + 28, labels.implicitHeight + 22)
        opacity: page.pending === row.id ? 0.55 : 1

        Behavior on opacity {
            NumberAnimation { duration: Motion.fast }
        }

        Rectangle {
            width: parent.width
            height: 1
            color: Theme.hairline
        }

        Column {
            id: labels
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            width: ln.labelWidth
            spacing: 4

            Row {
                spacing: 8

                Text {
                    text: ln.row.label
                    color: Theme.text
                    font.family: Theme.sans
                    font.pixelSize: 14
                }
                // Mudado em relação à fábrica (só o terminal tem fábrica aqui).
                Rectangle {
                    visible: ln.row.id === "terminal" && Projects.terminal !== "ghostty"
                    anchors.verticalCenter: parent.verticalCenter
                    width: 5
                    height: 5
                    radius: 3
                    color: Theme.copper
                }
            }
            Text {
                width: parent.width
                text: ln.row.desc
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 12
                wrapMode: Text.WordWrap
            }
            Text {
                width: parent.width
                visible: ln.stray !== ""
                text: "agora: " + ln.stray
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 10
                elide: Text.ElideRight
            }
        }

        // Da esquerda para a direita, na ordem da lista, mas encostado na
        // borda direita: o Flow tem a largura toda que sobra (é com ela que
        // ele quebra) e é empurrado pela folga da sua linha mais larga. Com
        // layoutDirection: Qt.RightToLeft a ordem sairia de trás para frente.
        Flow {
            id: flow
            anchors.verticalCenter: parent.verticalCenter
            width: Math.max(0, parent.width - ln.labelWidth - 24)
            x: parent.width - childrenRect.width
            spacing: 6

            Repeater {
                model: ln.options

                PillButton {
                    required property var modelData
                    text: modelData.label
                    selected: modelData.v === ln.current
                    onClicked: page.pick(ln.row.id, modelData.v)
                }
            }
        }

        // Sem escolhas: ainda lendo, ou nada instalado que sirva.
        Text {
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            visible: ln.options.length === 0
            text: !page.loaded ? "lendo os apps…" : ln.stray !== "" ? "nenhum outro app instalado para isto"
                : "nenhum app instalado para isto"
            color: Theme.textFaint
            font.family: Theme.serif
            font.pixelSize: 13
            font.italic: true
        }
    }

    Repeater {
        model: page.sections

        ColumnLayout {
            id: section
            required property var modelData
            required property int index

            Layout.fillWidth: true
            spacing: 0

            Section {
                Layout.topMargin: section.index === 0 ? 18 : 22
                text: section.modelData.name
            }

            Repeater {
                model: section.modelData.rows

                Line {
                    required property var modelData
                    row: modelData
                }
            }
        }
    }

    Rectangle {
        Layout.fillWidth: true
        implicitHeight: 1
        color: Theme.hairline
    }

    // O que deu errado na última troca.
    Text {
        Layout.fillWidth: true
        Layout.topMargin: 14
        visible: page.failure !== ""
        text: page.failure
        color: Theme.copper
        font.family: Theme.sans
        font.pixelSize: 12
        wrapMode: Text.WordWrap
    }

    // Um app de terminal como padrão precisa de quem abra o terminal para ele.
    Text {
        Layout.fillWidth: true
        Layout.topMargin: 14
        visible: page.loaded && page.terminalAppChosen && !page.terminalExec
        text: "Um dos padrões é um app de terminal. Para ele abrir a partir de outros apps (um clique duplo no gerenciador de arquivos), instale o xdg-terminal-exec."
        color: Theme.textDim
        font.family: Theme.sans
        font.pixelSize: 12
        wrapMode: Text.WordWrap
    }

    Text {
        Layout.fillWidth: true
        Layout.topMargin: 14
        text: "Os padrões ficam em ~/.config/mimeapps.list, onde todo app procura. O terminal é do Mk: vale para Super + Enter e para o que o Maré abre. A variável $TERMINAL e o terminal de emergência (Super + Shift + Enter) continuam no Ghostty."
        color: Theme.textFaint
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
        wrapMode: Text.WordWrap
    }
}
