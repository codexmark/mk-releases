import QtQuick
import QtQuick.Layouts
import Quickshell
import "Pulso.js" as Pulso

// A janela inteira: as cinco áreas à esquerda, a página escolhida à direita e,
// no pé da barra, o pulso da máquina (CPU, memória, processos).
Rectangle {
    id: app
    color: Theme.ground

    // MKPULSO_AREA abre direto numa área (o lançador passa: `mkpulso servicos`).
    property string area: {
        const pedida = Quickshell.env("MKPULSO_AREA") || "";
        return ["processos", "desempenho", "sistema", "inicio", "servicos"].indexOf(pedida) >= 0 ? pedida : "processos";
    }
    readonly property var areas: [
        { secao: "monitor", id: "processos", nome: "Processos" },
        { secao: "monitor", id: "desempenho", nome: "Desempenho" },
        { secao: "monitor", id: "sistema", nome: "Sistema" },
        { secao: "controle", id: "inicio", nome: "Inicialização" },
        { secao: "controle", id: "servicos", nome: "Serviços" }
    ]
    readonly property var r: Coletor.retrato

    // ── a barra ────────────────────────────────────────────────────────
    Rectangle {
        id: barra
        width: 208
        anchors.top: parent.top
        anchors.bottom: parent.bottom
        color: Theme.ground

        Rectangle {
            anchors.right: parent.right
            width: 1
            height: parent.height
            color: Theme.hairline
        }

        Column {
            anchors.fill: parent
            anchors.margins: 22
            spacing: 0

            Row {
                spacing: 8
                Text {
                    text: "Mk"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 24
                }
                Text {
                    anchors.baseline: parent.children[0].baseline
                    text: "pulso"
                    color: Theme.copper
                    font.family: Theme.serif
                    font.pixelSize: 17
                    font.italic: true
                }
            }

            Item { width: 1; height: 26 }

            Repeater {
                model: app.areas

                Column {
                    id: item
                    required property var modelData
                    required property int index
                    width: barra.width - 44
                    readonly property bool primeira: index === 0 || app.areas[index - 1].secao !== modelData.secao

                    Item { width: 1; height: item.primeira && item.index > 0 ? 18 : 0 }
                    Rotulo {
                        visible: item.primeira
                        text: (item.modelData.secao === "monitor" ? "01  " : "02  ") + item.modelData.secao
                        bottomPadding: 8
                    }
                    Rectangle {
                        width: parent.width
                        height: 32
                        color: app.area === item.modelData.id ? Theme.wash : toque.containsMouse ? Theme.boneA(0.03) : Qt.rgba(0, 0, 0, 0)

                        Rectangle {
                            visible: app.area === item.modelData.id
                            width: 2
                            height: parent.height
                            color: Theme.copper
                        }
                        Text {
                            x: 12
                            anchors.verticalCenter: parent.verticalCenter
                            text: item.modelData.nome
                            color: app.area === item.modelData.id ? Theme.text : Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 13
                        }
                        MouseArea {
                            id: toque
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: app.area = item.modelData.id
                        }
                    }
                }
            }
        }

        // O pulso da máquina.
        Column {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            anchors.margins: 22
            spacing: 7

            Rectangle { width: parent.width; height: 1; color: Theme.hairline }
            Item { width: 1; height: 3 }

            Repeater {
                model: [
                    { k: "cpu", v: app.r && app.r.cpu ? Pulso.pct(app.r.cpu.pct) : "—" },
                    { k: "mem", v: app.r && app.r.mem && app.r.mem.total_kb > 0 ? Pulso.pct(app.r.mem.usado_kb / app.r.mem.total_kb * 100) : "—" },
                    { k: "proc", v: app.r && app.r.totais ? String(app.r.totais.processos) : "—" }
                ]
                Item {
                    required property var modelData
                    width: barra.width - 44
                    height: 14
                    Rotulo { text: parent.modelData.k; anchors.verticalCenter: parent.verticalCenter }
                    Text {
                        anchors.right: parent.right
                        anchors.verticalCenter: parent.verticalCenter
                        text: parent.modelData.v
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                }
            }
            Rotulo {
                topPadding: 6
                text: app.r ? app.r.maquina : ""
            }
        }
    }

    // ── a página ───────────────────────────────────────────────────────
    Item {
        anchors.left: barra.right
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: parent.bottom

        // O coletor não respondeu: a janela diz o que houve em vez de ficar vazia.
        Text {
            anchors.centerIn: parent
            visible: !Coletor.pronto
            text: Coletor.erro !== "" ? Coletor.erro : "lendo o sistema…"
            color: Coletor.erro !== "" ? Theme.copper : Theme.textFaint
            font.family: Theme.serif
            font.pixelSize: 16
            font.italic: true
        }

        Loader {
            anchors.fill: parent
            anchors.margins: 26
            active: Coletor.pronto
            source: ({ processos: "Processos.qml", desempenho: "Desempenho.qml", sistema: "Sistema.qml",
                       inicio: "Inicio.qml", servicos: "Servicos.qml" })[app.area]
        }
    }
}
