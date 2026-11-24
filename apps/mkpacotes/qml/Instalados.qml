import QtQuick
import QtQuick.Layouts
import "Pacotes.js" as Pacotes

// Instalados: tudo o que há na máquina, com filtros (pedidos por você, do AUR,
// órfãos, com atualização), busca e ordem. Clicar numa linha abre a ficha;
// marcar várias deixa remover de uma vez.
Item {
    id: pagina

    property string busca: ""
    property string filtro: "todos"
    property string coluna: "nome"
    property bool decrescente: false
    property string escolhido: ""
    property var marcas: ({})

    readonly property var contagens: Pacotes.contagens(Coletor.pacotes)
    readonly property var visiveis: Pacotes.ordenar(Pacotes.filtrar(Coletor.pacotes, filtro, busca), coluna, decrescente)
    readonly property var marcados: Pacotes.marcados(marcas).filter(n => Coletor.instalado(n) !== null)
    readonly property real agora: Date.now() / 1000

    function ordenarPor(c) {
        if (coluna === c) {
            decrescente = !decrescente;
        } else {
            coluna = c;
            decrescente = c !== "nome"; // tamanho e data começam do maior e do mais novo
        }
    }

    // Depois de uma ação, o que saiu deixa de estar marcado e escolhido.
    Connections {
        target: Coletor
        function onLeituraChanged() {
            const novas = {};
            for (const n of pagina.marcados)
                novas[n] = true;
            pagina.marcas = novas;
            if (pagina.escolhido !== "" && Coletor.instalado(pagina.escolhido) === null)
                pagina.escolhido = "";
        }
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 14

        RowLayout {
            Layout.fillWidth: true
            spacing: 14
            Cabecalho {
                numero: "01"
                titulo: "Instalados"
                frase: "O que você pediu, o que veio junto e o que sobrou."
            }
            Item { Layout.fillWidth: true }
            Campo { Layout.alignment: Qt.AlignBottom; Layout.preferredWidth: 220; dica: "procurar nos instalados"; onTextChanged: pagina.busca = text }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 14
            Segmento {
                opcoes: [
                    { v: "todos", texto: "todos " + pagina.contagens.todos },
                    { v: "pedidos", texto: "pedidos " + pagina.contagens.pedidos },
                    { v: "aur", texto: "do aur " + pagina.contagens.aur },
                    { v: "orfaos", texto: "órfãos " + pagina.contagens.orfaos },
                    { v: "atualizacoes", texto: "atualizáveis " + pagina.contagens.atualizacoes }
                ]
                valor: pagina.filtro
                onEscolhido: v => pagina.filtro = v
            }
            Item { Layout.fillWidth: true }
            Segmento {
                opcoes: [
                    { v: "nome", texto: "nome" + (pagina.coluna === "nome" ? (pagina.decrescente ? " ↓" : " ↑") : "") },
                    { v: "tamanho_bytes", texto: "tamanho" + (pagina.coluna === "tamanho_bytes" ? (pagina.decrescente ? " ↓" : " ↑") : "") },
                    { v: "instalado_em_s", texto: "data" + (pagina.coluna === "instalado_em_s" ? (pagina.decrescente ? " ↓" : " ↑") : "") }
                ]
                valor: pagina.coluna
                onEscolhido: v => pagina.ordenarPor(v)
            }
        }

        Text {
            Layout.fillWidth: true
            visible: Coletor.aviso !== ""
            text: Coletor.aviso
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
            elide: Text.ElideRight
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 14

            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                color: Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: Theme.hairline

                ListView {
                    id: lista
                    anchors.fill: parent
                    anchors.margins: 1
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    model: pagina.visiveis
                    // Com a ficha aberta a tabela estreita: as colunas menos
                    // importantes saem, e a ficha já diz o que elas diziam.
                    readonly property bool comDescricao: width >= 760
                    readonly property bool comData: width >= 520
                    readonly property bool comVersao: width >= 430

                    delegate: Item {
                        id: linha
                        required property var modelData
                        readonly property bool atual: pagina.escolhido === modelData.nome
                        width: lista.width
                        height: 34

                        Rectangle {
                            anchors.fill: parent
                            color: linha.atual ? Theme.copperA(0.1) : passa.containsMouse ? Theme.boneA(0.03) : Qt.rgba(0, 0, 0, 0)
                        }
                        Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                        Rectangle { visible: linha.atual; width: 2; height: parent.height; color: Theme.copper }
                        MouseArea {
                            id: passa
                            anchors.fill: parent
                            hoverEnabled: true
                            onClicked: pagina.escolhido = linha.atual ? "" : linha.modelData.nome
                        }
                        Marca {
                            x: 13
                            anchors.verticalCenter: parent.verticalCenter
                            marcado: pagina.marcas[linha.modelData.nome] === true
                            ligado: Pacotes.nomeValido(linha.modelData.nome)
                            onTrocado: pagina.marcas = Pacotes.alterna(pagina.marcas, linha.modelData.nome)
                        }
                        Text {
                            id: oNome
                            x: 40
                            width: Math.min(230, parent.width - 40 - direita.width - 20)
                            anchors.verticalCenter: parent.verticalCenter
                            text: linha.modelData.nome
                            color: linha.modelData.motivo === "pedido" ? Theme.text : Theme.textSoft
                            font.family: Theme.mono
                            font.pixelSize: 11
                            elide: Text.ElideRight
                        }
                        Text {
                            visible: lista.comDescricao
                            anchors.left: oNome.right
                            anchors.leftMargin: 14
                            anchors.right: direita.left
                            anchors.rightMargin: 14
                            anchors.verticalCenter: parent.verticalCenter
                            text: linha.modelData.descricao
                            color: Theme.textDim
                            font.family: Theme.sans
                            font.pixelSize: 12
                            elide: Text.ElideRight
                        }
                        Row {
                            id: direita
                            anchors.right: parent.right
                            anchors.rightMargin: 14
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 0
                            Text {
                                visible: lista.comVersao
                                width: 150
                                horizontalAlignment: Text.AlignRight
                                text: linha.modelData.atualizacao ? linha.modelData.versao + " → " + linha.modelData.atualizacao : linha.modelData.versao
                                color: linha.modelData.atualizacao ? Theme.copper : Theme.textDim
                                font.family: Theme.mono
                                font.pixelSize: 10
                                elide: Text.ElideMiddle
                            }
                            Text {
                                width: 74
                                horizontalAlignment: Text.AlignRight
                                text: linha.modelData.orfao ? "órfão" : Pacotes.origem(linha.modelData.origem)
                                color: linha.modelData.orfao || linha.modelData.origem === "aur" ? Theme.copper : Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 10
                            }
                            Text {
                                width: 76
                                horizontalAlignment: Text.AlignRight
                                text: Pacotes.tamanho(linha.modelData.tamanho_bytes)
                                color: Theme.textSoft
                                font.family: Theme.mono
                                font.pixelSize: 10
                            }
                            Text {
                                visible: lista.comData
                                width: 92
                                horizontalAlignment: Text.AlignRight
                                text: Pacotes.quando(linha.modelData.instalado_em_s, pagina.agora)
                                color: Theme.textDim
                                font.family: Theme.mono
                                font.pixelSize: 10
                            }
                        }
                    }

                    Text {
                        anchors.centerIn: parent
                        visible: lista.count === 0
                        text: pagina.filtro === "orfaos" && pagina.busca === "" ? "nenhum órfão: nada sobrando"
                            : pagina.filtro === "atualizacoes" && pagina.busca === "" ? "tudo em dia, pelo que a última sincronização sabe"
                            : "nenhum pacote assim"
                        color: Theme.textFaint
                        font.family: Theme.serif
                        font.pixelSize: 14
                        font.italic: true
                    }
                }
            }

            Detalhe {
                Layout.preferredWidth: 340
                Layout.fillHeight: true
                visible: pagina.escolhido !== ""
                nome: pagina.escolhido
                onAbrir: nome => pagina.escolhido = nome
                onFechar: pagina.escolhido = ""
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 10
            Text {
                Layout.fillWidth: true
                elide: Text.ElideRight
                text: pagina.marcados.length > 0 ? Pacotes.plural(pagina.marcados.length, "marcado", "marcados") + ": " + pagina.marcados.join(", ")
                    : Pacotes.plural(pagina.visiveis.length, "pacote", "pacotes") + (Coletor.lendo ? "  ·  relendo…" : "")
                        + "  ·  clique numa linha para ver a ficha; marque o quadrado para escolher vários"
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 11
            }
            Botao {
                visible: pagina.marcados.length > 0
                texto: "desmarcar"
                onAcionado: pagina.marcas = ({})
            }
            Botao {
                visible: pagina.filtro === "orfaos" && pagina.contagens.orfaos > 0 && pagina.marcados.length === 0
                texto: "remover os órfãos"
                forte: true
                onAcionado: Acao.planejar("orfaos", [], false)
            }
            Botao {
                texto: pagina.marcados.length > 1 ? "remover os " + pagina.marcados.length + " marcados" : "remover o marcado"
                visible: pagina.marcados.length > 0
                forte: true
                onAcionado: Acao.planejar("remover", pagina.marcados, false)
            }
        }
    }
}
