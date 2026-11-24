import QtQuick
import QtQuick.Layouts
import "Pacotes.js" as Pacotes

// A ficha de um pacote (`detalhe`): o que é, de onde veio, do que depende,
// quem depende dele, os arquivos e, do AUR, os votos e se está desatualizado.
// Embaixo, o que dá para fazer com ele.
Rectangle {
    id: painel

    property string nome: ""
    property var d: null
    property string erro: ""
    signal abrir(string nome)
    signal fechar()

    color: Theme.surface
    border.width: 1
    border.color: Theme.hairline

    function ler() {
        if (nome === "") {
            d = null;
            return;
        }
        const pedido = nome;
        Coletor.pedir(["detalhe", "--", pedido], r => {
            if (painel.nome !== pedido)
                return;
            painel.erro = r.erro || "";
            painel.d = r.erro ? null : r;
        });
    }
    onNomeChanged: { d = null; erro = ""; ler(); }
    Component.onCompleted: ler()
    Connections {
        target: Coletor
        function onLeituraChanged() { painel.ler(); }
    }

    // Uma linha "rótulo: valor".
    component Par: Column {
        property string k: ""
        property string v: ""
        property bool destaque: false
        width: parent ? parent.width : 0
        spacing: 2
        visible: v !== ""
        Rotulo { text: parent.k }
        Text {
            width: parent.width
            text: parent.v
            color: parent.destaque ? Theme.copper : Theme.textSoft
            font.family: Theme.sans
            font.pixelSize: 12
            wrapMode: Text.Wrap
        }
    }

    // Uma lista de nomes de pacote em que dá para clicar.
    component Nomes: Column {
        id: nomes
        property string k: ""
        property var lista: []
        property int max: 40
        width: parent ? parent.width : 0
        spacing: 4
        visible: lista.length > 0
        Rotulo { text: nomes.k + "  " + nomes.lista.length }
        Flow {
            width: parent.width
            spacing: 6
            Repeater {
                model: nomes.lista.slice(0, nomes.max)
                Text {
                    id: umNome
                    required property var modelData
                    readonly property string alvo: typeof modelData === "string" ? modelData : modelData.nome
                    readonly property bool falta: typeof modelData !== "string" && modelData.instalado === false
                    text: alvo
                    color: toque.containsMouse ? Theme.copper : falta ? Theme.textFaint : Theme.textSoft
                    font.family: Theme.mono
                    font.pixelSize: 10
                    MouseArea {
                        id: toque
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: painel.abrir(umNome.alvo)
                    }
                }
            }
            Text {
                visible: nomes.lista.length > nomes.max
                text: "e mais " + (nomes.lista.length - nomes.max)
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 10
            }
        }
    }

    Text {
        anchors.centerIn: parent
        width: parent.width - 40
        horizontalAlignment: Text.AlignHCenter
        wrapMode: Text.WordWrap
        visible: painel.d === null
        text: painel.erro !== "" ? painel.erro : "lendo a ficha…"
        color: painel.erro !== "" ? Theme.copper : Theme.textFaint
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
    }

    Text {
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: 12
        z: 2
        text: "fechar"
        color: fecha.containsMouse ? Theme.copper : Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.capitalization: Font.AllUppercase
        font.letterSpacing: Theme.labelSpacing
        MouseArea {
            id: fecha
            anchors.fill: parent
            anchors.margins: -6
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            onClicked: painel.fechar()
        }
    }

    Flickable {
        id: rolo
        anchors.fill: parent
        anchors.margins: 18
        anchors.bottomMargin: 62
        visible: painel.d !== null
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        contentHeight: corpo.height

        Column {
            id: corpo
            width: rolo.width
            spacing: 12

            Column {
                width: parent.width
                spacing: 3
                Rotulo { text: painel.d ? (painel.d.instalado ? "instalado" : "não instalado") + "  ·  " + Pacotes.origem(painel.d.origem) : "" }
                Text {
                    width: parent.width - 50
                    text: painel.d ? painel.d.nome : ""
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 22
                    font.weight: Font.Light
                    wrapMode: Text.WrapAnywhere
                }
                Text {
                    width: parent.width
                    text: painel.d ? painel.d.versao + (painel.d.atualizacao ? "  →  " + painel.d.atualizacao : "") : ""
                    color: painel.d && painel.d.atualizacao ? Theme.copper : Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                    wrapMode: Text.WrapAnywhere
                }
            }
            Text {
                width: parent.width
                text: painel.d ? painel.d.descricao : ""
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 13
                wrapMode: Text.WordWrap
            }
            Rectangle { width: parent.width; height: 1; color: Theme.hairline }

            Par {
                k: "situação"
                v: !painel.d || !painel.d.instalado ? ""
                    : Pacotes.motivo(painel.d.motivo) + (painel.d.orfao ? "  ·  órfão: nada mais depende dele" : "")
                        + (painel.d.protegido ? "  ·  parte da base do sistema" : "")
                destaque: painel.d !== null && painel.d.orfao === true
            }
            Par {
                k: "no aur"
                v: !painel.d || !painel.d.aur ? ""
                    : Pacotes.plural(painel.d.aur.votos, "voto", "votos")
                        + (painel.d.aur.mantenedor ? "  ·  mantido por " + painel.d.aur.mantenedor : "  ·  sem mantenedor")
                        + (painel.d.aur.desatualizado_desde_s ? "  ·  marcado como desatualizado em " + Pacotes.data(painel.d.aur.desatualizado_desde_s) : "")
                        + (painel.d.instalado && painel.d.aur.versao !== painel.d.versao ? "  ·  lá está a " + painel.d.aur.versao : "")
                destaque: painel.d !== null && painel.d.aur !== null && (painel.d.aur.desatualizado_desde_s !== null || painel.d.aur.mantenedor === null)
            }
            Par { k: "aur"; v: painel.d && painel.d.erro_aur ? painel.d.erro_aur : ""; destaque: true }
            Par {
                k: "tamanho"
                v: !painel.d ? "" : painel.d.tamanho_bytes === null && painel.d.baixar_bytes === null ? ""
                    : Pacotes.tamanho(painel.d.tamanho_bytes) + " em disco" + (painel.d.baixar_bytes !== null ? "  ·  " + Pacotes.tamanho(painel.d.baixar_bytes) + " para baixar" : "")
            }
            Par { k: "instalado em"; v: painel.d && painel.d.instalado_em_s ? Qt.formatDateTime(new Date(painel.d.instalado_em_s * 1000), "dd/MM/yyyy HH:mm") : "" }
            Par { k: painel.d && painel.d.origem === "aur" && !painel.d.instalado ? "mudou em" : "construído em"; v: painel.d && painel.d.construido_em_s ? Pacotes.data(painel.d.construido_em_s) : "" }
            Par { k: "licença"; v: painel.d ? (painel.d.licencas || []).join(", ") : "" }
            Par { k: "mantenedor"; v: painel.d && painel.d.mantenedor ? painel.d.mantenedor : "" }
            Par { k: "página"; v: painel.d ? painel.d.url || "" : "" }

            Nomes { k: "depende de"; lista: painel.d ? painel.d.depende || [] : [] }
            Nomes { k: "opcionais"; lista: painel.d ? painel.d.opcionais || [] : [] }
            Nomes { k: "quem depende dele"; lista: painel.d ? painel.d.requerido_por || [] : [] }
            Nomes { k: "opcional para"; lista: painel.d ? painel.d.opcional_para || [] : [] }
            Par { k: "fornece"; v: painel.d ? (painel.d.fornece || []).join("  ") : "" }
            Par { k: "conflita com"; v: painel.d ? (painel.d.conflita || []).join("  ") : "" }

            Column {
                width: parent.width
                spacing: 4
                visible: painel.d !== null && painel.d.arquivos !== null
                Rotulo { text: painel.d && painel.d.arquivos ? "arquivos  " + painel.d.arquivos.total : "" }
                Repeater {
                    model: painel.d && painel.d.arquivos ? painel.d.arquivos.executaveis.slice(0, 30) : []
                    Text {
                        required property var modelData
                        width: corpo.width
                        text: modelData
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 10
                        elide: Text.ElideMiddle
                    }
                }
                Text {
                    visible: painel.d !== null && painel.d.arquivos !== null && painel.d.arquivos.executaveis.length > 30
                    text: painel.d && painel.d.arquivos ? "e mais " + (painel.d.arquivos.executaveis.length - 30) + " executáveis" : ""
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
                Text {
                    visible: painel.d !== null && painel.d.arquivos !== null && painel.d.arquivos.executaveis.length === 0
                    text: "nenhum executável em /usr/bin"
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }
        }
    }

    // ── o que dá para fazer ────────────────────────────────────────────
    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.margins: 1
        height: 52
        visible: painel.d !== null
        color: Theme.surface

        Rectangle { width: parent.width; height: 1; color: Theme.hairline }
        Row {
            anchors.right: parent.right
            anchors.rightMargin: 17
            anchors.verticalCenter: parent.verticalCenter
            spacing: 8
            Botao {
                visible: painel.d !== null && painel.d.instalado && painel.d.aur !== null && Pacotes.nomeValido(painel.nome)
                    && painel.d.aur.versao !== painel.d.versao
                texto: "atualizar do aur"
                onAcionado: Acao.planejar("instalar", [painel.nome], false)
            }
            Botao {
                visible: painel.d !== null && painel.d.instalado
                texto: "remover"
                forte: true
                ligado: Pacotes.nomeValido(painel.nome)
                onAcionado: Acao.planejar("remover", [painel.nome], false)
            }
            Botao {
                visible: painel.d !== null && !painel.d.instalado
                texto: "instalar"
                forte: true
                ligado: Pacotes.nomeValido(painel.nome)
                onAcionado: Acao.planejar("instalar", [painel.nome], false)
            }
        }
    }
}
