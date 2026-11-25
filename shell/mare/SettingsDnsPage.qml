import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "Dns.js" as Dns

// Página "DNS" dos Ajustes: o servidor de nomes de todas as redes de uma vez.
// As opções (o da rede, os provedores públicos e "outro"), a validação dos
// endereços e os textos vêm do Dns.js; quem lê e troca é o scripts/mare-dns,
// que responde em JSON. A troca grava o DNS global do NetworkManager (um
// arquivo em /etc, por um ajudante que roda pelo pkexec) e não mexe em
// nenhuma conexão; "da rede" apaga o arquivo.
// A janela de senha é do Maré (PolkitDialog) e ficaria atrás desta folha: ao
// ler "elevating" a página fecha os Ajustes; o script termina a troca, testa
// se um nome resolve e reabre a folha aqui, com o resultado no `estado`.
ColumnLayout {
    id: page

    spacing: 0

    property var dns: Dns.stateOf(null)
    property bool loaded: false
    // A opção pedida, enquanto a troca corre.
    property string asking: ""
    // O recado em vigor: { text, error, back } (veja Dns.noteOf e Dns.testNote).
    property var note: ({ text: "", error: false, back: false })
    // O `at` do último resultado já contado, para não repeti-lo a cada leitura.
    property real toldAt: 0
    property bool testing: false
    property bool loosing: false
    property bool typing: false

    readonly property string script: Quickshell.shellPath("scripts/mare-dns")
    readonly property bool busy: asking !== "" || dns.pending !== "" || loosing
    readonly property string current: Dns.selected(dns, asking)
    readonly property var typed: Dns.parse(otherInput.text)

    function refresh() {
        if (!stateProc.running)
            stateProc.running = true;
    }

    // O que o script contou: o estado e, se há um resultado novo, o recado.
    function take(data) {
        dns = Dns.stateOf(data);
        if (!loaded && dns.opcao === "outro" && otherInput.text === "")
            otherInput.text = dns.arquivo.join(" ");
        loaded = true;
        const last = data && typeof data === "object" ? (data.acao !== undefined ? data : data.last) : null;
        if (last && typeof last === "object" && last.at !== toldAt && last.acao !== "soltar") {
            toldAt = last.at;
            note = Dns.noteOf(last);
        }
    }

    function choose(id) {
        if (busy)
            return;
        if (id === "outro") {
            typing = true;
            otherInput.forceActiveFocus();
            return;
        }
        typing = false;
        if (id !== dns.opcao)
            start(id, "");
    }

    function start(id, text) {
        if (busy || actProc.running)
            return;
        const r = Dns.argsFor(id, text);
        if (r.error !== "") {
            note = ({ text: r.error, error: true, back: false });
            return;
        }
        note = ({ text: "", error: false, back: false });
        asking = id;
        actProc.command = ["bash", script].concat(r.args);
        actProc.running = true;
    }

    function test() {
        if (testing || busy)
            return;
        testing = true;
        testProc.running = true;
    }

    function loose() {
        if (busy)
            return;
        loosing = true;
        looseProc.running = true;
    }

    Component.onCompleted: refresh()

    // Uma VPN que liga, o cabo que sai: os servidores em uso mudam por fora.
    Timer {
        interval: page.dns.pending !== "" ? 1000 : 5000
        repeat: true
        running: true
        onTriggered: page.refresh()
    }

    Process {
        id: stateProc
        command: ["bash", page.script, "estado"]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    page.take(JSON.parse(text));
                } catch (e) {
                    page.loaded = true;
                }
            }
        }
    }

    // A troca. O script responde uma linha de JSON por vez: {"elevating":true}
    // quando a janela de senha vai aparecer, e no fim o estado com o resultado.
    Process {
        id: actProc
        environment: ({ "MARE_DNS_SHELL": page.script.replace(/\/scripts\/mare-dns$/, "") })
        stdout: SplitParser {
            onRead: line => {
                let data = null;
                try {
                    data = JSON.parse(line);
                } catch (e) {
                    return;
                }
                if (data === null || typeof data !== "object")
                    return;
                if (data.elevating === true) {
                    // Sai da frente da janela de senha. A folha (e esta página)
                    // some; o script termina a troca e reabre a folha.
                    Qt.callLater(() => Mare.close());
                    return;
                }
                page.asking = "";
                page.take(data);
            }
        }
        onExited: lost.restart()
    }

    // O script saiu sem dizer o resultado.
    Timer {
        id: lost
        interval: 400
        onTriggered: {
            if (page.asking === "" || actProc.running)
                return;
            page.asking = "";
            page.note = ({ text: "o mare-dns não respondeu", error: true, back: false });
            page.refresh();
        }
    }

    Process {
        id: testProc
        command: ["bash", page.script, "testar"]
        stdout: StdioCollector {
            onStreamFinished: {
                let result = null;
                try {
                    result = JSON.parse(text);
                } catch (e) {}
                page.testing = false;
                page.note = Dns.testNote(result, page.dns.opcao);
            }
        }
    }

    Process {
        id: looseProc
        command: ["bash", page.script, "soltar"]
        stdout: StdioCollector {
            onStreamFinished: {
                let data = null;
                try {
                    data = JSON.parse(text);
                } catch (e) {}
                page.loosing = false;
                if (data !== null && typeof data === "object") {
                    page.take(data);
                    page.note = ({ text: typeof data.message === "string" ? data.message : "", error: data.ok !== true, back: false });
                } else {
                    page.note = ({ text: "o mare-dns não respondeu", error: true, back: false });
                }
            }
        }
    }

    component Label: Text {
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.capitalization: Font.AllUppercase
        font.letterSpacing: Theme.labelSpacing
    }

    component Hairline: Rectangle {
        Layout.fillWidth: true
        implicitHeight: 1
        color: Theme.hairline
    }

    // ── o que vale agora ───────────────────────────────────────────────

    RowLayout {
        Layout.fillWidth: true
        Layout.topMargin: 18
        spacing: 14

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 8

            Text {
                Layout.fillWidth: true
                wrapMode: Text.WordWrap
                text: !page.loaded ? "lendo…"
                    : page.dns.pending !== "" && page.asking === "" ? "uma troca está em curso…"
                    : Dns.summary(page.dns)
                color: Theme.textSoft
                font.family: Theme.serif
                font.pixelSize: 16
                font.italic: true
                font.weight: Font.Light
            }

            Label { text: "em uso agora" }

            Text {
                Layout.fillWidth: true
                wrapMode: Text.WrapAnywhere
                text: Dns.serversLine(page.dns.servidores)
                color: Theme.text
                font.family: Theme.mono
                font.pixelSize: 12
            }
        }

        PillButton {
            Layout.alignment: Qt.AlignTop
            opacity: page.busy ? 0.5 : 1
            text: page.testing ? "testando…" : "testar"
            onClicked: page.test()
        }
    }

    // O recado da última troca ou do teste, com o caminho de volta.
    RowLayout {
        Layout.fillWidth: true
        Layout.topMargin: 12
        visible: page.note.text !== ""
        spacing: 12

        Text {
            Layout.fillWidth: true
            wrapMode: Text.WordWrap
            text: page.note.text
            color: page.note.error ? Theme.copper : Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 11
        }

        PillButton {
            visible: page.note.back && page.dns.opcao !== "rede"
            implicitHeight: 28
            primary: true
            opacity: page.busy ? 0.5 : 1
            text: "voltar para o da rede"
            onClicked: page.start("rede", "")
        }
    }

    // Sem o ajudante que escreve em /etc, a página só mostra.
    Text {
        Layout.fillWidth: true
        Layout.topMargin: 12
        visible: page.loaded && !page.dns.ajudante
        wrapMode: Text.WordWrap
        text: "falta o ajudante do Mk (/usr/lib/mk/mk-dns): rode o mk-update. sem ele não dá para trocar o DNS daqui."
        color: Theme.copper
        font.family: Theme.mono
        font.pixelSize: 11
    }

    // A conexão em uso tem DNS fixo próprio (de um Mk antigo, que trocava por
    // conexão, ou posto à mão): "da rede" não é o do roteador até soltar.
    RowLayout {
        Layout.fillWidth: true
        Layout.topMargin: 12
        visible: page.dns.conexao !== "" && page.dns.opcao === "rede"
        spacing: 12

        Text {
            Layout.fillWidth: true
            wrapMode: Text.WordWrap
            text: "A conexão " + page.dns.conexao + " guarda um DNS fixo só dela e ignora o do roteador. Soltar tira esse DNS dela, sem derrubar a conexão."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 12
            lineHeight: 1.35
        }

        PillButton {
            implicitHeight: 28
            opacity: page.busy ? 0.5 : 1
            text: page.loosing ? "soltando…" : "soltar"
            onClicked: page.loose()
        }
    }

    // ── as opções ──────────────────────────────────────────────────────

    Label {
        Layout.topMargin: 22
        Layout.bottomMargin: 8
        text: "servidor de nomes"
    }

    Hairline {}

    Repeater {
        model: Dns.providers

        ColumnLayout {
            id: entry
            required property var modelData
            readonly property bool on: page.current === modelData.id
            readonly property bool other: modelData.id === "outro"

            Layout.fillWidth: true
            spacing: 0

            Item {
                Layout.fillWidth: true
                implicitHeight: 62

                Rectangle {
                    anchors.fill: parent
                    color: entry.on || rowHover.hovered ? Theme.wash : "transparent"

                    Behavior on color {
                        ColorAnimation { duration: Motion.fast }
                    }
                }

                Rectangle {
                    visible: entry.on
                    width: 2
                    height: parent.height
                    color: Theme.copper
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 14
                    anchors.rightMargin: 12
                    spacing: 14

                    Column {
                        Layout.fillWidth: true
                        Layout.alignment: Qt.AlignVCenter
                        spacing: 5

                        Text {
                            width: parent.width
                            text: entry.modelData.name
                            color: entry.on ? Theme.text : Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 14
                            elide: Text.ElideRight
                        }

                        Text {
                            width: parent.width
                            text: entry.modelData.desc
                            color: Theme.textDim
                            font.family: Theme.sans
                            font.pixelSize: 11
                            elide: Text.ElideRight
                        }
                    }

                    Text {
                        visible: entry.modelData.servers.length > 0
                        text: entry.modelData.servers.slice(0, 2).join("  ·  ")
                        color: Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }

                    Text {
                        visible: text !== ""
                        text: page.asking === entry.modelData.id ? "trocando…" : entry.on ? "em uso" : ""
                        color: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }
                }

                HoverHandler {
                    id: rowHover
                    cursorShape: Qt.PointingHandCursor
                }

                TapHandler {
                    onTapped: page.choose(entry.modelData.id)
                }
            }

            // "outro": os endereços escritos, conferidos a cada tecla.
            RowLayout {
                visible: entry.other && (page.typing || entry.on)
                Layout.fillWidth: true
                Layout.leftMargin: 14
                Layout.rightMargin: 12
                Layout.topMargin: 4
                Layout.bottomMargin: 10
                spacing: 10

                Item {
                    Layout.fillWidth: true
                    implicitHeight: 32

                    // O campo é um só na página; mora na linha de "outro".
                    TextInput {
                        id: field
                        anchors.fill: parent
                        anchors.bottomMargin: 5
                        verticalAlignment: TextInput.AlignVCenter
                        color: Theme.text
                        selectionColor: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 12
                        clip: true
                        selectByMouse: true
                        text: otherInput.text
                        onTextEdited: otherInput.text = text
                        cursorDelegate: Rectangle {
                            width: 2
                            color: Theme.copper
                            visible: field.activeFocus
                        }
                        Keys.onReturnPressed: page.start("outro", otherInput.text)
                        Keys.onEnterPressed: page.start("outro", otherInput.text)

                        Connections {
                            target: otherInput
                            function onFocusAsked() {
                                if (entry.other)
                                    field.forceActiveFocus();
                            }
                        }

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: field.text === ""
                            text: "192.168.1.1  fd00::53"
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 12
                        }
                    }

                    Rectangle {
                        anchors.bottom: parent.bottom
                        width: parent.width
                        height: 1
                        color: otherInput.text.trim() !== "" && page.typed.error !== "" ? Theme.copper
                            : field.activeFocus ? Theme.line : Theme.hairline
                    }
                }

                PillButton {
                    primary: page.typed.error === "" && !page.busy
                    opacity: page.typed.error !== "" || page.busy ? 0.5 : 1
                    text: page.asking === "outro" ? "trocando…" : "aplicar"
                    onClicked: page.start("outro", otherInput.text)
                }
            }

            Text {
                visible: entry.other && (page.typing || entry.on) && otherInput.text.trim() !== ""
                Layout.fillWidth: true
                Layout.leftMargin: 14
                Layout.bottomMargin: 10
                wrapMode: Text.WordWrap
                text: page.typed.error !== "" ? page.typed.error : Dns.serversLine(page.typed.servers)
                color: page.typed.error !== "" ? Theme.copper : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 11
            }

            Hairline {}
        }
    }

    // O texto de "outro", fora do Repeater (as linhas são refeitas; ele fica).
    QtObject {
        id: otherInput
        property string text: ""
        signal focusAsked
        function forceActiveFocus() {
            Qt.callLater(focusAsked);
        }
    }

    // ── o que fica de fora ─────────────────────────────────────────────

    Text {
        Layout.fillWidth: true
        Layout.topMargin: 14
        wrapMode: Text.WordWrap
        text: Dns.limits
        color: Theme.textFaint
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
    }

    Text {
        Layout.fillWidth: true
        Layout.topMargin: 8
        wrapMode: Text.WordWrap
        text: "A troca vale para todas as redes e pede a senha de administrador: fica em /etc/NetworkManager/conf.d/90-mk-dns.conf, e \"da rede\" apaga o arquivo."
        color: Theme.textFaint
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
    }
}
