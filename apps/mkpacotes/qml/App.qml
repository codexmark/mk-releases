import QtQuick
import QtQuick.Layouts
import Quickshell
import "Pacotes.js" as Pacotes

// A janela inteira: as quatro áreas à esquerda, a página escolhida à direita,
// o resumo da máquina no pé da barra e, por cima de tudo, a folha de
// confirmação quando há uma ação.
Rectangle {
    id: app
    color: Theme.ground

    // MKPACOTES_AREA abre direto numa área (o lançador passa: `mkpacotes procurar`).
    property string area: {
        const pedida = Quickshell.env("MKPACOTES_AREA") || "";
        return ["instalados", "procurar", "atualizacoes", "historico"].indexOf(pedida) >= 0 ? pedida : "instalados";
    }
    readonly property var areas: [
        { secao: "pacotes", id: "instalados", nome: "Instalados" },
        { secao: "pacotes", id: "procurar", nome: "Procurar" },
        { secao: "cuidado", id: "atualizacoes", nome: "Atualizações" },
        { secao: "cuidado", id: "historico", nome: "Histórico" }
    ]
    readonly property var t: Coletor.totais
    readonly property var e: Coletor.estado

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
                    text: "pacotes"
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
                    readonly property int pendentes: modelData.id === "atualizacoes" && app.t ? app.t.atualizacoes : 0

                    Item { width: 1; height: item.primeira && item.index > 0 ? 18 : 0 }
                    Rotulo {
                        visible: item.primeira
                        text: (item.modelData.secao === "pacotes" ? "01  " : "02  ") + item.modelData.secao
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
                        Text {
                            visible: item.pendentes > 0
                            anchors.right: parent.right
                            anchors.rightMargin: 10
                            anchors.verticalCenter: parent.verticalCenter
                            text: item.pendentes
                            color: Theme.copper
                            font.family: Theme.mono
                            font.pixelSize: 10
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

        // O resumo da máquina.
        Column {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            anchors.margins: 22
            spacing: 7

            Text {
                visible: app.e !== null && app.e.travado === true
                width: parent.width
                wrapMode: Text.WordWrap
                text: "o pacman está travado: outro está rodando"
                color: Theme.copper
                font.family: Theme.mono
                font.pixelSize: 10
            }
            Rectangle { width: parent.width; height: 1; color: Theme.hairline }
            Item { width: 1; height: 3 }

            Repeater {
                model: [
                    { k: "pacotes", v: app.t ? String(app.t.pacotes) : "—" },
                    { k: "do aur", v: app.t ? String(app.t.aur) : "—" },
                    { k: "órfãos", v: app.t ? String(app.t.orfaos) : "—" },
                    { k: "em disco", v: app.t ? Pacotes.tamanho(app.t.tamanho_bytes) : "—" },
                    { k: "cache", v: app.e && app.e.cache ? Pacotes.tamanho(app.e.cache.bytes) : "—" }
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
            width: parent.width - 80
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.WordWrap
            visible: !Coletor.pronto
            text: Coletor.erro !== "" ? Coletor.erro : "lendo os pacotes…"
            color: Coletor.erro !== "" ? Theme.copper : Theme.textFaint
            font.family: Theme.serif
            font.pixelSize: 16
            font.italic: true
        }

        Loader {
            anchors.fill: parent
            anchors.margins: 26
            active: Coletor.pronto
            source: ({ instalados: "Instalados.qml", procurar: "Procurar.qml", atualizacoes: "Atualizacoes.qml",
                       historico: "Historico.qml" })[app.area]
        }
    }

    // ── a folha de confirmação, por cima de tudo ───────────────────────
    Folha {
        anchors.fill: parent
        visible: Acao.aberta
    }
}
